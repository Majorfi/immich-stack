/**************************************************************************************************
** Pure decision logic for the fix-trash command: given trashed and active assets, decide
** which active assets should follow their stack members into the trash. No API calls here,
** so everything is unit-testable with plain fixtures.
**************************************************************************************************/

package main

import (
	"io"

	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** hasActiveCopy reports whether a trashed asset still has an active copy of the same file:
** same OriginalFileName and same capture time. Trashing one duplicate must not cascade to
** the surviving copy or its companions, so such trashed assets never become triggers.
** File-system timestamps are deliberately not compared: recycled filenames from unrelated
** photos would look like replacements, and equal-timestamp duplicates would not.
**
** @param trashed - The trashed asset to check
** @param activeByFilename - Active assets indexed by OriginalFileName
** @return bool - True if an active copy of the same photo exists
**************************************************************************************************/
func hasActiveCopy(trashed utils.TAsset, activeByFilename map[string][]utils.TAsset) bool {
	for _, active := range activeByFilename[trashed.OriginalFileName] {
		if active.LocalDateTime == trashed.LocalDateTime {
			return true
		}
	}
	return false
}

/**************************************************************************************************
** findStackRelatedAssets finds the active assets that would stack with a trashed asset and
** should therefore follow it into the trash. Trashed assets that still have an active copy
** (same filename and capture time) are excluded first, then a single StackBy run over the
** remaining trashed assets plus all active assets yields the groups to cascade.
**
** @param trashedAssets - Assets currently in the trash
** @param activeAssets - Assets not in the trash
** @param criteria - Stacking criteria JSON (empty = defaults)
** @param parentFilenamePromote - Parent promotion list (shared with the stacker command)
** @param parentExtPromote - Extension promotion list (shared with the stacker command)
** @param logger - For debug traces
** @return map[string]utils.TAsset - Active assets to trash, by ID
** @return map[string]string - Trashed filename that triggered each asset, by ID
** @return int - Number of trashed assets skipped as replaced
** @return error - Any error from the stacker
**************************************************************************************************/
func findStackRelatedAssets(
	trashedAssets []utils.TAsset,
	activeAssets []utils.TAsset,
	criteria string,
	parentFilenamePromote string,
	parentExtPromote string,
	logger *logrus.Logger,
) (map[string]utils.TAsset, map[string]string, int, error) {
	assetsToTrash := make(map[string]utils.TAsset)
	triggeredBy := make(map[string]string)

	activeByFilename := make(map[string][]utils.TAsset)
	for _, asset := range activeAssets {
		activeByFilename[asset.OriginalFileName] = append(activeByFilename[asset.OriginalFileName], asset)
	}

	replacedCount := 0
	triggers := make([]utils.TAsset, 0, len(trashedAssets))
	for _, trashed := range trashedAssets {
		if hasActiveCopy(trashed, activeByFilename) {
			logger.Debugf("  🔄 Skipping %s - an active copy with the same capture time exists", trashed.OriginalFileName)
			replacedCount++
			continue
		}
		triggers = append(triggers, trashed)
	}
	if len(triggers) == 0 {
		return assetsToTrash, triggeredBy, replacedCount, nil
	}

	/**********************************************************************************************
	** One StackBy run over trashed + active assets replaces the previous per-trashed-asset
	** loop: groups only depend on key equality and time buckets, so the combined run yields
	** the same cascades in a single pass. The stacker's own logging is silenced because only
	** the grouping result matters here.
	**********************************************************************************************/
	combined := make([]utils.TAsset, 0, len(triggers)+len(activeAssets))
	combined = append(combined, triggers...)
	combined = append(combined, activeAssets...)

	quietLogger := logrus.New()
	quietLogger.SetOutput(io.Discard)
	stacks, err := stacker.StackBy(combined, criteria, parentFilenamePromote, parentExtPromote, quietLogger)
	if err != nil {
		return nil, nil, replacedCount, err
	}

	/**********************************************************************************************
	** Within combined, trashed assets are exactly the triggers: activeAssets are non-trashed
	** by contract and replaced trashed assets were excluded above.
	**********************************************************************************************/
	for _, stack := range stacks {
		triggerName := ""
		for _, asset := range stack {
			if asset.IsTrashed {
				triggerName = asset.OriginalFileName
				break
			}
		}
		if triggerName == "" {
			continue
		}
		for _, asset := range stack {
			if asset.IsTrashed {
				continue
			}
			if asset.IsArchived {
				logger.Debugf("  ⏭️  Keeping archived asset %s (archived assets are never trashed)", asset.OriginalFileName)
				continue
			}
			assetsToTrash[asset.ID] = asset
			triggeredBy[asset.ID] = triggerName
			logger.Debugf("  ➡️  %s (active → will trash, stacks with %s)", asset.OriginalFileName, triggerName)
		}
	}
	return assetsToTrash, triggeredBy, replacedCount, nil
}
