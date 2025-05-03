package stacker

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** sortStack sorts a stack of assets based on filename and extension priority.
** The order is:
** 1. Promoted filenames (PARENT_FILENAME_PROMOTE, comma-separated, order matters)
** 2. Promoted extensions (PARENT_EXT_PROMOTE, comma-separated, order matters)
** 3. Extension priority (jpeg > jpg > png > others)
** 4. Alphabetical order (case-sensitive)
**
** @param stack - List of assets to sort
** @return []Asset - Sorted list of assets
**************************************************************************************************/
func sortStack(stack []utils.TAsset, parentFilenamePromote string, parentExtPromote string) []utils.TAsset {
	promoteSubstrings := parsePromoteList(parentFilenamePromote)
	if len(promoteSubstrings) == 0 {
		promoteSubstrings = utils.DefaultParentFilenamePromote
	}

	promoteExtensions := parsePromoteList(parentExtPromote)
	if len(promoteExtensions) == 0 {
		promoteExtensions = utils.DefaultParentExtPromote
	}

	sort.SliceStable(stack, func(i, j int) bool {
		iPromoteIdx := getPromoteIndex(stack[i].OriginalFileName, promoteSubstrings)
		jPromoteIdx := getPromoteIndex(stack[j].OriginalFileName, promoteSubstrings)
		if iPromoteIdx != jPromoteIdx {
			return iPromoteIdx < jPromoteIdx
		}

		extI := strings.ToLower(filepath.Ext(stack[i].OriginalFileName))
		extJ := strings.ToLower(filepath.Ext(stack[j].OriginalFileName))
		iExtPromoteIdx := getPromoteIndex(extI, promoteExtensions)
		jExtPromoteIdx := getPromoteIndex(extJ, promoteExtensions)
		if iExtPromoteIdx != jExtPromoteIdx {
			return iExtPromoteIdx < jExtPromoteIdx
		}

		rankI := getExtensionRank(extI)
		rankJ := getExtensionRank(extJ)
		if rankI != rankJ {
			return rankI > rankJ
		}

		return stack[i].OriginalFileName < stack[j].OriginalFileName
	})

	return stack
}

/**************************************************************************************************
** StackBy groups photos into stacks based on configured criteria.
** Photos that match the same criteria values are grouped together.
**
** @param assets - List of assets to group into stacks
** @param criteria - List of criteria to use for grouping
** @return [][]Asset - List of stacks, where each stack is a list of assets
** @return error - Any error that occurred during stacking
**************************************************************************************************/
func StackBy(assets []utils.TAsset, criteria string, parentFilenamePromote string, parentExtPromote string) ([][]utils.TAsset, error) {
	if len(assets) == 0 {
		return nil, nil
	}

	stackingCriteria, err := getCriteriaConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get criteria config: %w", err)
	}

	groups := make(map[string][]utils.TAsset, len(assets)/2)
	for _, asset := range assets {
		values, err := applyCriteria(asset, stackingCriteria)
		if err != nil {
			return nil, fmt.Errorf("failed to apply criteria to asset %s: %w", asset.OriginalFileName, err)
		}

		key := strings.Join(values, "|")
		if key == "" {
			continue
		}

		groups[key] = append(groups[key], asset)
	}

	var result [][]utils.TAsset
	for _, group := range groups {
		if len(group) > 1 {
			result = append(result, sortStack(group, parentFilenamePromote, parentExtPromote))
		}
	}

	return result, nil
}
