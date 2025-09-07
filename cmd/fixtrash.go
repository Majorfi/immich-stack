/**************************************************************************************************
** Fix-trash command implementation for the Immich CLI application.
** Handles stack-aware trash operations to maintain consistency.
**************************************************************************************************/

package main

import (
	"path/filepath"
	"strings"

	"github.com/majorfi/immich-stack/pkg/immich"
	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

/**************************************************************************************************
** Main execution logic for fixing incomplete trash operations. Identifies trashed assets
** and moves their stack-related assets to trash to maintain consistency.
**
** @param cmd - Cobra command instance
** @param args - Command line arguments
**************************************************************************************************/
func runFixTrash(cmd *cobra.Command, args []string) {
	logger := loadEnv()

	/**********************************************************************************************
	** Support multiple API keys (comma-separated).
	**********************************************************************************************/
	apiKeys := utils.RemoveEmptyStrings(func(keys []string) []string {
		for i, key := range keys {
			keys[i] = strings.TrimSpace(key)
		}
		return keys
	}(strings.Split(apiKey, ",")))
	if len(apiKeys) == 0 {
		logger.Fatalf("No API key(s) provided.")
	}

	for i, key := range apiKeys {
		if i > 0 {
			logger.Infof("\n")
		}
		client := immich.NewClient(apiURL, key, false, false, dryRun, withArchived, withDeleted, false, logger)
		if client == nil {
			logger.Errorf("Invalid client for API key: %s", key)
			continue
		}
		user, err := client.GetCurrentUser()
		if err != nil {
			logger.Errorf("Failed to fetch user for API key: %s: %v", key, err)
			continue
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
			continue
		}

		if len(trashedAssets) == 0 {
			logger.Info("No trashed assets found. Nothing to fix.")
			continue
		}

		logger.Infof("🗑️  Found %d trashed assets", len(trashedAssets))

		existingStacks, err := client.FetchAllStacks()
		if err != nil {
			logger.Errorf("Error fetching stacks: %v", err)
			continue
		}

		allAssets, err := client.FetchAssets(1000, existingStacks)
		if err != nil {
			logger.Errorf("Error fetching all assets: %v", err)
			continue
		}

		/**********************************************************************************************
		** Find assets that should be trashed using reverse criteria matching.
		** For each trashed asset, combine it with all active assets and run stacker criteria
		** to find which active assets would group with the trashed ones.
		**********************************************************************************************/
		assetsToTrash := make(map[string]utils.TAsset)
		// Track which trashed asset caused each asset to be marked for deletion
		trashedAssetMapping := make(map[string]string) // assetID -> trashed asset filename

		// Filter active (non-trashed) assets
		activeAssets := make([]utils.TAsset, 0)
		for _, asset := range allAssets {
			if !asset.IsTrashed {
				activeAssets = append(activeAssets, asset)
			}
		}

		logger.Infof("📊 Analyzing against %d active assets...", len(activeAssets))

		for idx, trashedAsset := range trashedAssets {
			// Show progress every 50 assets or in debug mode
			if logger.IsLevelEnabled(logrus.DebugLevel) || (idx > 0 && idx%50 == 0) {
				logger.Infof("   Analyzing trashed asset %d/%d...", idx+1, len(trashedAssets))
			}
			logger.Debugf("Analyzing trashed asset: %s", trashedAsset.OriginalFileName)

			// Create a combined asset list: trashed asset + all active assets
			combinedAssets := make([]utils.TAsset, 0, len(activeAssets)+1)
			combinedAssets = append(combinedAssets, trashedAsset)
			combinedAssets = append(combinedAssets, activeAssets...)

			// Run stacker criteria on the combined list
			stacks, err := stacker.StackBy(combinedAssets, criteria, parentFilenamePromote, parentExtPromote, logger)
			if err != nil {
				logger.Errorf("Error using stacker criteria for asset %s: %v", trashedAsset.OriginalFileName, err)
				continue
			}

			// Find groups that contain our trashed asset
			for _, stack := range stacks {
				containsTrashedAsset := false
				for _, asset := range stack {
					if asset.ID == trashedAsset.ID {
						containsTrashedAsset = true
						break
					}
				}

				// If this group contains the trashed asset, all other assets in the group should be trashed
				if containsTrashedAsset && len(stack) > 1 {
					logger.Debugf("Stack found with %d assets (1 in trash, %d active):", len(stack), len(stack)-1)
					logger.Debugf("  🗑️  %s (already in trash)", trashedAsset.OriginalFileName)
					for _, relatedAsset := range stack {
						if relatedAsset.ID != trashedAsset.ID && !relatedAsset.IsTrashed {
							assetsToTrash[relatedAsset.ID] = relatedAsset
							trashedAssetMapping[relatedAsset.ID] = trashedAsset.OriginalFileName
							logger.Debugf("  ➡️  %s (active → will trash)", relatedAsset.OriginalFileName)
						}
					}
				}
			}
		}

		/**********************************************************************************************
		** Move the identified assets to trash.
		**********************************************************************************************/
		if len(assetsToTrash) == 0 {
			logger.Info("✅ No related assets found that need to be moved to trash.")
			continue
		}

		logger.Infof("✅ Analysis complete: %d trashed → %d related assets to trash", len(trashedAssets), len(assetsToTrash))

		// Group by file extension for summary
		extensionCount := make(map[string]int)
		assetIDs := make([]string, 0, len(assetsToTrash))

		// In debug mode, show detailed mapping
		if logger.IsLevelEnabled(logrus.DebugLevel) {
			logger.Debug("\n📋 Summary of assets to trash:")
			// Group by the trashed asset that caused them to be marked
			groupedByTrashed := make(map[string][]utils.TAsset)
			for _, asset := range assetsToTrash {
				trashedName := trashedAssetMapping[asset.ID]
				groupedByTrashed[trashedName] = append(groupedByTrashed[trashedName], asset)
			}

			for trashedName, relatedAssets := range groupedByTrashed {
				relatedAssetNames := make([]string, 0, len(relatedAssets))
				for _, asset := range relatedAssets {
					relatedAssetNames = append(relatedAssetNames, asset.OriginalFileName)
				}
				logger.Debugf("Stack with %s (in trash): %s\n", trashedName, strings.Join(relatedAssetNames, ", "))
			}
		}

		for _, asset := range assetsToTrash {
			ext := filepath.Ext(asset.OriginalFileName)
			if ext == "" {
				ext = "(no extension)"
			}
			extensionCount[ext]++
			assetIDs = append(assetIDs, asset.ID)
		}

		// Show summary by file type
		if len(extensionCount) > 0 {
			logger.Info("📁 Assets to trash by type:")
			for ext, count := range extensionCount {
				logger.Infof("   - %s files: %d", strings.ToUpper(strings.TrimPrefix(ext, ".")), count)
			}
		}

		if err := client.TrashAssets(assetIDs); err != nil {
			logger.Errorf("Error moving assets to trash: %v", err)
		}
	}
}
