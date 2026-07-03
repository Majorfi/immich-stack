/**************************************************************************************************
** Orphaned-RAW detection for the fix-trash command: a RAW file whose stacking group contains
** no developed (non-RAW) companion is flagged for trash, unless it already sits in an Immich
** stack that contains a non-RAW asset.
**
** Grouping reuses the same stacking criteria as the main command and pass 1, so filename
** normalization — including camera-specific patterns like Leica's L/DO0 prefixes — is
** driven by CRITERIA instead of hardcoded rules.
**************************************************************************************************/

package main

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** orphanedRAWTrigger is the label recorded in the triggeredBy map for RAWs flagged by this
** pass; logTrashSummary branches on it to group them separately in the output.
**************************************************************************************************/
const orphanedRAWTrigger = "orphaned RAW"

/**************************************************************************************************
** rawExtensions lists the common camera RAW formats. Any asset whose extension is not in
** this set counts as a developed companion.
**************************************************************************************************/
var rawExtensions = map[string]bool{
	".3fr": true, ".arw": true, ".cr2": true, ".cr3": true, ".crw": true,
	".dcr": true, ".dng": true, ".erf": true, ".fff": true, ".gpr": true,
	".iiq": true, ".kdc": true, ".mef": true, ".mos": true, ".mrw": true,
	".nef": true, ".nrw": true, ".orf": true, ".pef": true, ".raf": true,
	".raw": true, ".rw2": true, ".rwl": true, ".sr2": true, ".srf": true,
	".srw": true, ".x3f": true,
}

func isRAWFile(fileName string) bool {
	return rawExtensions[strings.ToLower(filepath.Ext(fileName))]
}

/**************************************************************************************************
** findOrphanedRAWs finds active RAW assets that have no developed companion: neither in
** their stacking group (computed with the user's criteria), nor in their existing Immich
** stack. RAW assets that end up in no group at all have no companion by definition, since
** the stacker drops single-asset groups.
**
** @param activeAssets - Assets not in the trash
** @param criteria - Stacking criteria JSON (empty = defaults), same as pass 1
** @param parentFilenamePromote - Parent promotion list (shared with the stacker command)
** @param parentExtPromote - Extension promotion list (shared with the stacker command)
** @param logger - For debug traces
** @return map[string]utils.TAsset - Orphaned RAWs to trash, by ID
** @return int - Number of RAWs kept because their Immich stack has a developed asset
** @return error - Any error from the stacker
**************************************************************************************************/
func findOrphanedRAWs(
	activeAssets []utils.TAsset,
	criteria string,
	parentFilenamePromote string,
	parentExtPromote string,
	logger *logrus.Logger,
) (map[string]utils.TAsset, int, error) {
	quietLogger := logrus.New()
	quietLogger.SetOutput(io.Discard)
	stacks, err := stacker.StackBy(activeAssets, criteria, parentFilenamePromote, parentExtPromote, quietLogger)
	if err != nil {
		return nil, 0, err
	}

	/**********************************************************************************************
	** Pre-pass: Immich stacks that already contain a developed asset.
	**********************************************************************************************/
	stackHasDeveloped := make(map[string]bool)
	for _, asset := range activeAssets {
		if asset.Stack != nil && asset.Stack.ID != "" && !isRAWFile(asset.OriginalFileName) {
			stackHasDeveloped[asset.Stack.ID] = true
		}
	}

	/**********************************************************************************************
	** Candidates: every RAW in an all-RAW group, plus every RAW in no group at all.
	**********************************************************************************************/
	candidates := make(map[string]utils.TAsset)
	grouped := make(map[string]bool)
	for _, stack := range stacks {
		hasDeveloped := false
		for _, asset := range stack {
			grouped[asset.ID] = true
			if !isRAWFile(asset.OriginalFileName) {
				hasDeveloped = true
			}
		}
		if hasDeveloped {
			continue
		}
		for _, asset := range stack {
			candidates[asset.ID] = asset
		}
	}
	for _, asset := range activeAssets {
		if !grouped[asset.ID] && isRAWFile(asset.OriginalFileName) {
			candidates[asset.ID] = asset
		}
	}

	orphans := make(map[string]utils.TAsset)
	keptStackedCount := 0
	for id, asset := range candidates {
		if asset.Stack != nil && asset.Stack.ID != "" && stackHasDeveloped[asset.Stack.ID] {
			keptStackedCount++
			logger.Debugf("  ✅ Keeping RAW %s - already in a stack with a developed file", asset.OriginalFileName)
			continue
		}
		orphans[id] = asset
		logger.Debugf("  🔍 Found orphaned RAW: %s (no developed companion)", asset.OriginalFileName)
	}
	return orphans, keptStackedCount, nil
}
