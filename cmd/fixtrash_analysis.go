/**************************************************************************************************
** Pure decision logic for the fix-trash command: given trashed and active assets, decide
** which active assets should follow their stack members into the trash. No API calls here,
** so everything is unit-testable with plain fixtures.
**************************************************************************************************/

package main

import (
	"io"
	"time"

	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** timestampAfter reports whether timestamp a is strictly after timestamp b.
** Immich returns RFC3339 timestamps; parsing them makes the comparison correct even when
** timezone offsets differ. If either side fails to parse, it falls back to a lexicographic
** comparison, which matches the previous behavior for uniform-format values.
**
** @param a - First RFC3339 timestamp
** @param b - Second RFC3339 timestamp
** @return bool - True if a is after b
**************************************************************************************************/
func timestampAfter(a, b string) bool {
	timeA, errA := time.Parse(time.RFC3339Nano, a)
	timeB, errB := time.Parse(time.RFC3339Nano, b)
	if errA != nil || errB != nil {
		return a > b
	}
	return timeA.After(timeB)
}

/**************************************************************************************************
** isReplacedByNewerCopy reports whether a trashed asset appears to have been re-uploaded:
** an active asset with the same filename exists and any of its timestamps is newer. Such
** trashed assets must not trigger a cascade, otherwise the fresh copy's companions would be
** dragged into the trash.
**
** @param trashed - The trashed asset to check
** @param activeByFilename - Active assets indexed by OriginalFileName
** @return bool - True if a newer active copy exists
**************************************************************************************************/
func isReplacedByNewerCopy(trashed utils.TAsset, activeByFilename map[string][]utils.TAsset) bool {
	for _, active := range activeByFilename[trashed.OriginalFileName] {
		if timestampAfter(active.FileCreatedAt, trashed.FileCreatedAt) ||
			timestampAfter(active.FileModifiedAt, trashed.FileModifiedAt) ||
			timestampAfter(active.UpdatedAt, trashed.UpdatedAt) {
			return true
		}
	}
	return false
}

/**************************************************************************************************
** findStackRelatedAssets finds the active assets that would stack with a trashed asset and
** should therefore follow it into the trash. Replaced trashed assets (newer active copy with
** the same filename) are excluded first, then a single StackBy run over the remaining trashed
** assets plus all active assets yields the groups to cascade.
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
		if isReplacedByNewerCopy(trashed, activeByFilename) {
			logger.Debugf("  🔄 Skipping %s - appears to be replaced (newer version exists)", trashed.OriginalFileName)
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
			assetsToTrash[asset.ID] = asset
			triggeredBy[asset.ID] = triggerName
			logger.Debugf("  ➡️  %s (active → will trash, stacks with %s)", asset.OriginalFileName, triggerName)
		}
	}
	return assetsToTrash, triggeredBy, replacedCount, nil
}
