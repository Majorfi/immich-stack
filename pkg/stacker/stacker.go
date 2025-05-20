package stacker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** extractLargestNumberSuffix finds a numeric suffix at the end of the base filename (before the
** extension), but ONLY if it appears after a delimiter. If no delimiters are present, always
** return 0. If delimiters are present, split the base filename using them and check the last part
** for a numeric suffix. If no numeric suffix is found after a delimiter, return 0.
**
** @param filename - The filename to analyze
** @param delimiters - Slice of delimiters to split the base filename (required for suffix)
** @return int - The numeric suffix, or 0 if none found or no delimiter present
**************************************************************************************************/
func extractLargestNumberSuffix(filename string, delimiters []string) int {
	base := filename
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	if len(delimiters) == 0 {
		return 0
	}
	parts := []string{base}
	for _, delim := range delimiters {
		temp := []string{}
		for _, part := range parts {
			temp = append(temp, strings.Split(part, delim)...)
		}
		parts = temp
	}
	if len(parts) < 2 {
		return 0
	}
	last := parts[len(parts)-1]
	numericSuffixRegex := regexp.MustCompile(`^(\d+)$`)
	match := numericSuffixRegex.FindStringSubmatch(last)
	if len(match) < 2 {
		return 0
	}
	n, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return n
}

/**************************************************************************************************
** sortStack sorts a stack of assets based on filename and extension priority.
** The order is:
** 1. Promoted filenames (PARENT_FILENAME_PROMOTE, comma-separated, order matters)
** 2. Promoted extensions (PARENT_EXT_PROMOTE, comma-separated, order matters)
** 3. Extension priority (jpeg > jpg > png > others)
** 4. Alphabetical order (case-sensitive)
**
** @param stack - List of assets to sort
** @param delimiters - Delimiters to use for numeric suffix extraction
** @return []Asset - Sorted list of assets
**************************************************************************************************/
func sortStack(stack []utils.TAsset, parentFilenamePromote string, parentExtPromote string, delimiters []string) []utils.TAsset {
	promoteSubstrings := parsePromoteList(parentFilenamePromote)
	if len(promoteSubstrings) == 0 {
		promoteSubstrings = utils.DefaultParentFilenamePromote
	}

	promoteExtensions := parsePromoteList(parentExtPromote)
	if len(promoteExtensions) == 0 {
		promoteExtensions = utils.DefaultParentExtPromote
	}

	sort.SliceStable(stack, func(i, j int) bool {
		iOriginalFileNameNoExt := filepath.Base(stack[i].OriginalFileName)
		jOriginalFileNameNoExt := filepath.Base(stack[j].OriginalFileName)
		iPromoteIdx := getPromoteIndex(iOriginalFileNameNoExt, promoteSubstrings)
		jPromoteIdx := getPromoteIndex(jOriginalFileNameNoExt, promoteSubstrings)
		if iPromoteIdx != jPromoteIdx {
			return iPromoteIdx < jPromoteIdx
		}

		// If both have the same promote index and 'biggestNumber' is in promoteSubstrings, use largest number as priority
		if utils.Contains(promoteSubstrings, "biggestNumber") && iPromoteIdx < len(promoteSubstrings) {
			iNum := extractLargestNumberSuffix(iOriginalFileNameNoExt, delimiters)
			jNum := extractLargestNumberSuffix(jOriginalFileNameNoExt, delimiters)
			if iNum != jNum {
				return iNum > jNum // highest number first
			}
		}

		extI := strings.ToLower(filepath.Ext(iOriginalFileNameNoExt))
		extJ := strings.ToLower(filepath.Ext(jOriginalFileNameNoExt))
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

		return iOriginalFileNameNoExt < jOriginalFileNameNoExt
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
func StackBy(assets []utils.TAsset, criteria string, parentFilenamePromote string, parentExtPromote string, logger *logrus.Logger) ([][]utils.TAsset, error) {
	if len(assets) == 0 {
		return nil, nil
	}

	stackingCriteria, err := getCriteriaConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get criteria config: %w", err)
	}

	// Find delimiters for originalFileName criteria
	var delimiters []string
	for _, c := range stackingCriteria {
		if c.Key == "originalFileName" && c.Split != nil && len(c.Split.Delimiters) > 0 {
			delimiters = c.Split.Delimiters
			break
		}
	}

	// Debugging
	{
		listOfCriteria := []string{}
		for _, c := range stackingCriteria {
			listOfCriteria = append(listOfCriteria, c.Key)
		}
		logger.Debugf("Stacking assets with criteria: %s", listOfCriteria)
		logger.Debugf("Parent filename promote: %s", parentFilenamePromote)
		logger.Debugf("Parent extension promote: %s", parentExtPromote)
		logger.Debugf("Delimiters: %v", delimiters)
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

		// Debugging
		logger.WithFields(logrus.Fields{"stack": key}).Debugf("Asset %s", asset.OriginalFileName)

		groups[key] = append(groups[key], asset)
	}

	var result [][]utils.TAsset
	for _, group := range groups {
		if len(group) > 1 {
			result = append(result, sortStack(group, parentFilenamePromote, parentExtPromote, delimiters))
		}
	}

	return result, nil
}
