package stacker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** Stacker handles the photo stacking logic, providing functionality to group and sort
** photos based on configurable criteria and sorting rules.
**************************************************************************************************/
type Stacker struct {
	logger *logrus.Logger
}

/**************************************************************************************************
** NewStacker creates a new Stacker instance with the provided logger.
**
** @param logger - Logger instance for output
** @return *Stacker - Configured Stacker instance
**************************************************************************************************/
func NewStacker(logger *logrus.Logger) *Stacker {
	return &Stacker{
		logger: logger,
	}
}

/**************************************************************************************************
** GetCriteriaConfig retrieves the criteria configuration from environment variables.
** If CRITERIA env var is not set, returns the default criteria configuration.
**
** @return []Criteria - List of criteria to use for stacking
** @return error - Any error that occurred during configuration retrieval
**************************************************************************************************/
func (s *Stacker) GetCriteriaConfig() ([]Criteria, error) {
	criteriaOverride := os.Getenv("CRITERIA")
	if criteriaOverride == "" {
		return DefaultCriteria, nil
	}

	var criteria []Criteria
	if err := json.Unmarshal([]byte(criteriaOverride), &criteria); err != nil {
		return nil, err
	}
	return criteria, nil
}

/**************************************************************************************************
** ApplyCriteria applies the configured criteria to an asset.
** Returns a list of strings that uniquely identify the asset based on the criteria.
**
** @param asset - The asset to apply criteria to
** @return []string - List of strings that uniquely identify the asset
** @return error - Any error that occurred during criteria application
**************************************************************************************************/
func (s *Stacker) ApplyCriteria(asset Asset) ([]string, error) {
	criteria := []string{}

	// Use all DefaultCriteria for grouping
	for _, c := range DefaultCriteria {
		var value string
		switch c.Key {
		case "originalFileName":
			baseName := asset.OriginalFileName
			if c.Split != nil {
				parts := strings.Split(baseName, c.Split.Key)
				if len(parts) > c.Split.Index {
					baseName = parts[c.Split.Index]
				}
			}
			value = baseName
		case "localDateTime":
			value = asset.LocalDateTime
		}
		if value != "" {
			criteria = append(criteria, value)
		}
	}

	return criteria, nil
}

/**************************************************************************************************
** StackBy groups photos into stacks based on configured criteria.
** Photos that match the same criteria values are grouped together.
**
** @param assets - List of assets to group into stacks
** @return [][]Asset - List of stacks, where each stack is a list of assets
** @return error - Any error that occurred during stacking
**************************************************************************************************/
func (s *Stacker) StackBy(assets []Asset) ([][]Asset, error) {
	// Group assets by criteria
	groups := make(map[string][]Asset)
	for _, asset := range assets {
		criteria, err := s.ApplyCriteria(asset)
		if err != nil {
			return nil, err
		}

		// Get base filename without extension
		baseName := strings.TrimSuffix(asset.OriginalFileName, filepath.Ext(asset.OriginalFileName))

		key := baseName
		if len(criteria) > 0 {
			key = strings.Join(criteria, "|")
		}

		// Skip empty keys
		if key == "" {
			continue
		}

		groups[key] = append(groups[key], asset)
	}

	// Convert groups to slices and filter single-asset groups
	var result [][]Asset
	for _, group := range groups {
		if len(group) > 1 {
			result = append(result, group)
		}
	}

	// Sort each group by filename
	for i := range result {
		result[i] = s.SortStack(result[i])
	}

	return result, nil
}

/**************************************************************************************************
** SortStack sorts a stack of assets based on filename and extension priority.
** The order is:
** 1. Promoted filenames (PARENT_FILENAME_PROMOTE, comma-separated, order matters)
** 2. Promoted extensions (PARENT_EXT_PROMOTE, comma-separated, order matters)
** 3. Extension priority (jpeg > jpg > png > others)
** 4. Alphabetical order (case-sensitive)
**
** @param stack - List of assets to sort
** @return []Asset - Sorted list of assets
**************************************************************************************************/
func (s *Stacker) SortStack(stack []Asset) []Asset {
	promoteStr := os.Getenv("PARENT_FILENAME_PROMOTE")
	promoteExt := os.Getenv("PARENT_EXT_PROMOTE")

	promoteSubstrings := parsePromoteList(promoteStr)
	promoteExtensions := parsePromoteList(promoteExt)

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
** parsePromoteList parses a comma-separated list from an environment variable into a slice.
** Trims whitespace and ignores empty entries.
**************************************************************************************************/
func parsePromoteList(list string) []string {
	if list == "" {
		return nil
	}
	parts := strings.Split(list, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

/**************************************************************************************************
** getPromoteIndex returns the index of the first promote substring/extension found in the value.
** If none found, returns len(promoteList) (lowest priority).
**************************************************************************************************/
func getPromoteIndex(value string, promoteList []string) int {
	for idx, promote := range promoteList {
		if promote == "" {
			continue
		}
		if strings.Contains(value, promote) {
			return idx
		}
	}
	return len(promoteList)
}

/**************************************************************************************************
** getExtensionRank returns a numeric rank for file extensions.
** Higher rank means higher priority.
**
** @param ext - File extension (with dot)
** @return int - Rank of the extension
**************************************************************************************************/
func getExtensionRank(ext string) int {
	switch ext {
	case ".jpeg":
		return 4
	case ".jpg":
		return 3
	case ".png":
		return 2
	default:
		return 1
	}
}
