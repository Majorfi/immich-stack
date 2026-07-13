/**************************************************************************************************
** Fix-trash command implementation for the Immich CLI application.
** Handles stack-aware trash operations to maintain consistency.
**************************************************************************************************/

package main

import (
	"path/filepath"
	"strings"

	"github.com/majorfi/immich-stack/pkg/immich"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

/**************************************************************************************************
** Entry point for the fix-trash command: warns about settings that have no effect here,
** splits the comma-separated API keys, and runs fixTrashForAPIKey for each of them.
**
** @param cmd - Cobra command instance
** @param args - Command line arguments
**************************************************************************************************/
func runFixTrash(cmd *cobra.Command, args []string) {
	logger := loadEnv()

	/**********************************************************************************************
	** Warn about settings that have no effect on this run.
	**********************************************************************************************/
	if len(filterAlbumIDs) > 0 || filterTakenAfter != "" || filterTakenBefore != "" {
		logger.Warnf("Filter flags (--filter-album-ids, --filter-taken-after, --filter-taken-before) have no effect on the fix-trash command")
	}
	if !trashOrphanedRAWs && rawOrphanExtensions != "" {
		logger.Warnf("--raw-orphan-extensions (RAW_ORPHAN_EXTENSIONS) has no effect without --trash-orphaned-raws")
	}

	/**********************************************************************************************
	** Support multiple API keys (comma-separated).
	**********************************************************************************************/
	apiKeys := splitCommaList(apiKey)
	if len(apiKeys) == 0 {
		logger.Fatalf("No API key(s) provided.")
	}

	for i, key := range apiKeys {
		if i > 0 {
			logger.Infof("\n")
		}
		fixTrashForAPIKey(key, logger)
	}
}

/**************************************************************************************************
** fixTrashForAPIKey runs the full fix-trash flow for one API key: stack cascade from the
** trashed assets, then the opt-in orphaned-RAW pass. Called per key by the fix-trash
** command, and by the stacker after each run when --fix-trash-after-stacking is set.
** A fresh client is created here on purpose: fix-trash must not inherit the stacker's
** reset/replace/filter options. Archived assets are always fetched (withArchived=true):
** an archived companion must be able to protect its group, but archived assets are never
** trashed by either pass.
**
** @param key - API key of the user to process
** @param logger - Logger instance
**************************************************************************************************/
func fixTrashForAPIKey(key string, logger *logrus.Logger) {
	client := immich.NewClient(apiURL, key, false, false, dryRun, true, withDeleted, false, includeVideos, stackConcurrency, nil, "", "", logger)
	if client == nil {
		logger.Errorf("Invalid client for API key: %s", key)
		return
	}
	user, err := client.GetCurrentUser()
	if err != nil {
		logger.Errorf("Failed to fetch user for API key: %s: %v", key, err)
		return
	}
	logger.Infof("=====================================================================================")
	logger.Infof("Fixing trash for user: %s (%s)", user.Name, user.Email)
	logger.Infof("=====================================================================================")

	/**********************************************************************************************
	** Fetch trashed assets and all assets.
	**********************************************************************************************/
	trashedAssets, err := client.FetchTrashedAssets(1000)
	if err != nil {
		logger.Errorf("Error fetching trashed assets: %v", err)
		return
	}
	trashedAssets = filterOutPartnerAssets(trashedAssets, user.ID, logger)

	if len(trashedAssets) == 0 {
		logger.Info("No trashed assets found. Nothing to fix.")
		return
	}

	logger.Infof("🗑️  Found %d trashed assets", len(trashedAssets))

	existingStacks, err := client.FetchAllStacks()
	if err != nil {
		logger.Errorf("Error fetching stacks: %v", err)
		return
	}

	allAssets, err := client.FetchAssets(1000, existingStacks)
	if err != nil {
		logger.Errorf("Error fetching all assets: %v", err)
		return
	}
	allAssets = filterOutPartnerAssets(allAssets, user.ID, logger)

	activeAssets := make([]utils.TAsset, 0, len(allAssets))
	for _, asset := range allAssets {
		if !asset.IsTrashed {
			activeAssets = append(activeAssets, asset)
		}
	}

	/**********************************************************************************************
	** Find the active assets that would stack with the trashed ones (replaced files are
	** skipped), then the orphaned RAWs.
	**********************************************************************************************/
	logger.Infof("🔍 Analyzing %d trashed assets against %d active assets...", len(trashedAssets), len(activeAssets))
	assetsToTrash, triggeredBy, replacedCount, err := findStackRelatedAssets(
		trashedAssets, activeAssets, criteria, parentFilenamePromote, parentExtPromote, logger)
	if err != nil {
		logger.Errorf("Error using stacker criteria: %v", err)
		return
	}
	if replacedCount > 0 {
		logger.Infof("🔄 Skipped %d trashed assets that still have an active copy", replacedCount)
	}

	if trashOrphanedRAWs {
		logger.Info("🔍 Looking for orphaned RAW files...")
		orphanedRAWs, keptStackedRAWCount, err := findOrphanedRAWs(activeAssets, criteria, parentFilenamePromote, parentExtPromote, rawOrphanExtensions, logger)
		if err != nil {
			logger.Errorf("Error detecting orphaned RAW files (continuing with pass 1 results): %v", err)
		}
		for id, asset := range orphanedRAWs {
			assetsToTrash[id] = asset
			triggeredBy[id] = orphanedRAWTrigger
		}
		if len(orphanedRAWs) > 0 {
			logger.Infof("📸 Found %d orphaned RAW files without a developed companion", len(orphanedRAWs))
		}
		if keptStackedRAWCount > 0 {
			logger.Infof("✅ Kept %d RAW files already stacked with a developed file", keptStackedRAWCount)
		}
	} else {
		logger.Info("⏭️  Orphaned RAW cleanup disabled (enable with --trash-orphaned-raws)")
	}

	/**********************************************************************************************
	** Move the identified assets to trash. The volume warning is non-blocking on purpose:
	** fix-trash is documented for cron usage, so a confirmation gate would break
	** unattended runs.
	**********************************************************************************************/
	if len(assetsToTrash) == 0 {
		logger.Info("✅ No related assets need to be trashed.")
		return
	}

	if len(activeAssets) > 0 && len(assetsToTrash)*10 > len(activeAssets) {
		logger.Warnf("⚠️  About to trash %d assets — more than 10%% of your %d active assets. Review the summary below carefully (use DRY_RUN=true first if unsure).", len(assetsToTrash), len(activeAssets))
	}

	logTrashSummary(logger, assetsToTrash, triggeredBy)

	assetIDs := make([]string, 0, len(assetsToTrash))
	for id := range assetsToTrash {
		assetIDs = append(assetIDs, id)
	}
	if err := client.TrashAssets(assetIDs); err != nil {
		logger.Errorf("Error moving assets to trash: %v", err)
	}
}

/**************************************************************************************************
** logTrashSummary prints the assets about to be trashed, grouped by the trashed asset that
** triggered them (orphaned RAWs first), plus a per-extension count at debug level.
**
** @param logger - Logger instance
** @param assetsToTrash - Assets to trash, by ID
** @param triggeredBy - Trigger label for each asset ID
**************************************************************************************************/
func logTrashSummary(logger *logrus.Logger, assetsToTrash map[string]utils.TAsset, triggeredBy map[string]string) {
	if logger.IsLevelEnabled(logrus.InfoLevel) {
		logger.Infof("📋 Summary of assets to trash (%d):", len(assetsToTrash))
		groupedByTrigger := make(map[string][]string)
		orphanedNames := make([]string, 0)

		for id, asset := range assetsToTrash {
			trigger := triggeredBy[id]
			if trigger == orphanedRAWTrigger {
				orphanedNames = append(orphanedNames, asset.OriginalFileName)
			} else {
				groupedByTrigger[trigger] = append(groupedByTrigger[trigger], asset.OriginalFileName)
			}
		}

		if len(orphanedNames) > 0 {
			logger.Infof("\t📸 Orphaned RAW files (no developed companion): %s\n", strings.Join(orphanedNames, ", "))
		}
		for trigger, names := range groupedByTrigger {
			logger.Infof("\t%s (in trash): %s\n", trigger, strings.Join(names, ", "))
		}
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		extensionCount := make(map[string]int)
		for _, asset := range assetsToTrash {
			ext := filepath.Ext(asset.OriginalFileName)
			if ext == "" {
				ext = "(no extension)"
			}
			extensionCount[ext]++
		}
		logger.Debugf("📁 Assets to trash by type:")
		for ext, count := range extensionCount {
			logger.Debugf("   - %s files: %d", strings.ToUpper(strings.TrimPrefix(ext, ".")), count)
		}
	}
}
