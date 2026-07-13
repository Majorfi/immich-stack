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
	"maps"
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

func hasExtension(set map[string]bool, fileName string) bool {
	return set[strings.ToLower(filepath.Ext(fileName))]
}

func isRAWFile(fileName string) bool {
	return hasExtension(rawExtensions, fileName)
}

/**************************************************************************************************
** parseOrphanExtensions turns the RAW_ORPHAN_EXTENSIONS setting into the set of extensions
** the orphan pass may flag. Empty means every known RAW format. Entries are matched against
** rawExtensions so a typo cannot silently extend the pass to non-RAW files; unknown entries
** are warned about and ignored. This only restricts the candidates — companion detection
** still treats every RAW format as non-developed.
**
** @param raw - Comma-separated extensions, with or without leading dots (e.g. "dng,nef")
** @param logger - For warnings on unknown entries
** @return map[string]bool - Allowed candidate extensions, keyed as ".ext"
**************************************************************************************************/
func parseOrphanExtensions(raw string, logger *logrus.Logger) map[string]bool {
	if raw == "" {
		return maps.Clone(rawExtensions)
	}
	allowed := make(map[string]bool)
	for _, entry := range splitCommaList(raw) {
		ext := "." + strings.TrimPrefix(strings.ToLower(entry), ".")
		if !rawExtensions[ext] {
			logger.Warnf("⚠️  RAW_ORPHAN_EXTENSIONS: %q is not a known RAW extension, ignoring", entry)
			continue
		}
		allowed[ext] = true
	}
	return allowed
}

/**************************************************************************************************
** findOrphanedRAWs finds active RAW assets that have no developed companion: neither in
** their stacking group (computed with the user's criteria), nor in their existing Immich
** stack. RAW assets that the criteria match but that end up in no group are companionless
** singletons (the stacker drops single-asset groups); RAW assets the criteria cannot
** evaluate are never flagged, and neither are archived RAWs — archived assets protect
** their group but are never trashed.
**
** @param activeAssets - Assets not in the trash
** @param criteria - Stacking criteria JSON (empty = defaults), same as pass 1
** @param parentFilenamePromote - Parent promotion list (shared with the stacker command)
** @param parentExtPromote - Extension promotion list (shared with the stacker command)
** @param orphanExtensions - RAW_ORPHAN_EXTENSIONS value restricting the candidates
**                           (empty = all; all-invalid entries disable the pass)
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
	orphanExtensions string,
	logger *logrus.Logger,
) (map[string]utils.TAsset, int, error) {
	candidateExtensions := parseOrphanExtensions(orphanExtensions, logger)
	if len(candidateExtensions) == 0 {
		return nil, 0, nil
	}

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
	** Candidates: every allowed RAW in an all-RAW group, plus every allowed RAW that the
	** criteria matched but that grouped with nothing (the stacker drops singletons). A RAW
	** the criteria cannot evaluate is left alone: its absence from the groups says nothing
	** about its companions.
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
			if !asset.IsArchived && hasExtension(candidateExtensions, asset.OriginalFileName) {
				candidates[asset.ID] = asset
			}
		}
	}

	ungroupedRAWs := make([]utils.TAsset, 0)
	for _, asset := range activeAssets {
		if !grouped[asset.ID] && !asset.IsArchived && hasExtension(candidateExtensions, asset.OriginalFileName) {
			ungroupedRAWs = append(ungroupedRAWs, asset)
		}
	}
	if len(ungroupedRAWs) > 0 {
		matchedIDs, err := stacker.MatchedAssetIDs(ungroupedRAWs, criteria)
		if err != nil {
			return nil, 0, err
		}
		for _, asset := range ungroupedRAWs {
			if matchedIDs[asset.ID] {
				candidates[asset.ID] = asset
			} else {
				logger.Debugf("  ⏭️  Skipping RAW %s - the criteria cannot evaluate it", asset.OriginalFileName)
			}
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
