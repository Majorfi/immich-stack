package stacker

import (
	"fmt"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** applyAdvancedCriteria generates grouping keys for an asset using advanced criteria logic.
** It correctly handles OR groups where each matching criterion creates separate grouping opportunities.
**
** @param asset - The utils.TAsset to apply criteria to.
** @param groups - A slice of utils.TCriteriaGroup defining how to group assets.
** @return []string - A slice of grouping keys that the asset matches. Empty if no groups match.
** @return error - An error if any extractor function returns an error.
**************************************************************************************************/
func applyAdvancedCriteria(asset utils.TAsset, groups []utils.TCriteriaGroup) ([]string, error) {
	var groupingKeys []string

	// Process each criteria group
	for groupIdx, group := range groups {
		if group.Operator == "OR" {
			// For OR groups, each matching criterion creates its own grouping opportunity
			// This allows assets to be grouped by ANY of the criteria, creating multiple potential stacks
			for criteriaIdx, criterion := range group.Criteria {
				extractor, ok := getExtractor(criterion.Key)
				if !ok {
					return nil, fmt.Errorf("unknown criteria key: %s", criterion.Key)
				}

				value, err := extractor(asset, criterion)
				if err != nil {
					return nil, err
				}

				if value != "" {
					// Create a unique key for this specific criterion match
					groupKey := fmt.Sprintf("group_%d_or_%d_%s:%s", groupIdx, criteriaIdx, criterion.Key, value)
					groupingKeys = append(groupingKeys, groupKey)
				}
			}
		} else {
			// Default to AND logic - all criteria in the group must match
			var groupValues []string
			var criteriaKeys []string
			groupMatches := true

			for _, criterion := range group.Criteria {
				extractor, ok := getExtractor(criterion.Key)
				if !ok {
					return nil, fmt.Errorf("unknown criteria key: %s", criterion.Key)
				}

				value, err := extractor(asset, criterion)
				if err != nil {
					return nil, err
				}

				if value == "" {
					groupMatches = false
					break
				}

				groupValues = append(groupValues, value)
				criteriaKeys = append(criteriaKeys, criterion.Key)
			}

			// If all criteria match, create a single grouping key
			if groupMatches && len(groupValues) > 0 {
				groupKey := fmt.Sprintf("group_%d_and:", groupIdx)
				for i, key := range criteriaKeys {
					if i > 0 {
						groupKey += "|"
					}
					groupKey += fmt.Sprintf("%s=%s", key, groupValues[i])
				}
				groupingKeys = append(groupingKeys, groupKey)
			}
		}
	}

	return groupingKeys, nil
}
