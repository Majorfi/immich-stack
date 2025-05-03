package stacker

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** getCriteriaConfig retrieves the criteria configuration from environment variables.
** If CRITERIA env var is not set, returns the default criteria configuration.
**
** @return []Criteria - List of criteria to use for stacking
** @return error - Any error that occurred during configuration retrieval
**************************************************************************************************/
func getCriteriaConfig() ([]utils.TCriteria, error) {
	criteriaOverride := os.Getenv("CRITERIA")
	if criteriaOverride == "" {
		return utils.DefaultCriteria, nil
	}

	var criteria []utils.TCriteria
	if err := json.Unmarshal([]byte(criteriaOverride), &criteria); err != nil {
		return nil, fmt.Errorf("failed to parse CRITERIA env var: %w", err)
	}
	return criteria, nil
}

/**************************************************************************************************
** applyCriteria applies the configured criteria to an asset.
** Returns a list of strings that uniquely identify the asset based on the criteria.
**
** @param asset - The asset to apply criteria to
** @param criteria - List of criteria to apply
** @return []string - List of strings that uniquely identify the asset
** @return error - Any error that occurred during criteria application
**************************************************************************************************/
func applyCriteria(asset utils.TAsset, criteria []utils.TCriteria) ([]string, error) {
	result := make([]string, 0, len(criteria))

	for _, c := range criteria {
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
			result = append(result, value)
		}
	}

	return result, nil
}
