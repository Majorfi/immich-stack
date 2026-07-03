/**************************************************************************************************
** Orphaned-DNG detection for the fix-trash command: DNG files with no JPG companion are
** flagged for trash unless they already sit in a stack containing a JPG.
**
** NOTE: behavior intentionally preserved as-is from the original inline implementation,
** including the Leica-specific prefix normalization (DO0/DL0/DL/L) and the JPG-only
** companion check. Changing these semantics is tracked separately.
**************************************************************************************************/

package main

import (
	"path/filepath"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** orphanedDNGTrigger is the label recorded in the triggeredBy map for DNGs flagged by this
** pass; logTrashSummary branches on it to group them separately in the output.
**************************************************************************************************/
const orphanedDNGTrigger = "orphaned DNG"

/**************************************************************************************************
** hasJPGExt reports whether a filename has a .jpg/.jpeg extension — the only companion
** formats the orphan check recognizes (part of the pinned behavior).
**************************************************************************************************/
func hasJPGExt(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return ext == ".jpg" || ext == ".jpeg"
}

/**************************************************************************************************
** normalizeDNGBaseName normalizes base filenames so camera variants map to the same key:
** suffixes after "_" or "~" are dropped, and the Leica prefixes DO0/DL0/DL/L are stripped
** when followed by digits (DO01001336, DL01001336 and L1001336 all become 1001336).
**
** @param baseName - Filename without extension
** @return string - Normalized base name
**************************************************************************************************/
func normalizeDNGBaseName(baseName string) string {
	if idx := strings.Index(baseName, "_"); idx > 0 {
		baseName = baseName[:idx]
	}
	if idx := strings.Index(baseName, "~"); idx > 0 {
		baseName = baseName[:idx]
	}

	if strings.HasPrefix(baseName, "DO0") && len(baseName) > 3 {
		return baseName[3:]
	}
	if strings.HasPrefix(baseName, "DL0") && len(baseName) > 3 {
		return baseName[3:]
	}
	if strings.HasPrefix(baseName, "DL") && len(baseName) > 2 {
		if baseName[2] >= '0' && baseName[2] <= '9' {
			return baseName[2:]
		}
	}
	if strings.HasPrefix(baseName, "L") && len(baseName) > 1 {
		if baseName[1] >= '0' && baseName[1] <= '9' {
			return baseName[1:]
		}
	}
	return baseName
}

/**************************************************************************************************
** dngGroupingBaseName derives the grouping key for an asset filename: the extension is
** stripped, and for multi-extension names like L1000746.edit.jpg everything after the first
** dot is dropped.
**
** @param fileName - Original asset filename
** @return string - Base name before normalization
**************************************************************************************************/
func dngGroupingBaseName(fileName string) string {
	parts := strings.Split(fileName, ".")
	if len(parts) > 2 {
		return parts[0]
	}
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

/**************************************************************************************************
** findOrphanedDNGs finds active DNG assets that have no JPG companion sharing the same
** normalized base name and are not already stacked with a JPG.
**
** @param activeAssets - Assets not in the trash
** @param logger - For debug traces
** @return map[string]utils.TAsset - Orphaned DNGs to trash, by ID
** @return int - Number of DNGs skipped because they are already stacked with a JPG
**************************************************************************************************/
func findOrphanedDNGs(activeAssets []utils.TAsset, logger *logrus.Logger) (map[string]utils.TAsset, int) {
	/**********************************************************************************************
	** Pre-pass: stacks that already contain a JPG. A DNG in such a stack is not orphaned even
	** without a same-base-name JPG.
	**********************************************************************************************/
	stackHasJPG := make(map[string]bool)
	assetsByBaseName := make(map[string][]utils.TAsset)
	for _, asset := range activeAssets {
		if asset.Stack != nil && asset.Stack.ID != "" && hasJPGExt(asset.OriginalFileName) {
			stackHasJPG[asset.Stack.ID] = true
		}
		baseName := dngGroupingBaseName(asset.OriginalFileName)
		normalizedName := normalizeDNGBaseName(baseName)
		assetsByBaseName[normalizedName] = append(assetsByBaseName[normalizedName], asset)
		logger.Debugf("  Normalized %s -> %s (via base: %s)", asset.OriginalFileName, normalizedName, baseName)
	}

	orphans := make(map[string]utils.TAsset)
	skippedStackedCount := 0
	for _, assets := range assetsByBaseName {
		hasDNG := false
		hasJPG := false
		var dngAsset utils.TAsset

		for _, asset := range assets {
			if strings.ToLower(filepath.Ext(asset.OriginalFileName)) == ".dng" {
				hasDNG = true
				dngAsset = asset
			} else if hasJPGExt(asset.OriginalFileName) {
				hasJPG = true
			}
		}

		if hasDNG && !hasJPG {
			if dngAsset.Stack != nil && dngAsset.Stack.ID != "" && stackHasJPG[dngAsset.Stack.ID] {
				skippedStackedCount++
				logger.Debugf("  ✅ Skipping DNG %s - already in stack with JPG", dngAsset.OriginalFileName)
			} else {
				orphans[dngAsset.ID] = dngAsset
				logger.Debugf("  🔍 Found orphaned DNG: %s (no corresponding JPG)", dngAsset.OriginalFileName)
			}
		}
	}
	return orphans, skippedStackedCount
}
