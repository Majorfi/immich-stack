package stacker

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** CriteriaConfig holds the processed criteria configuration, either from legacy format
** or advanced format.
**************************************************************************************************/
type CriteriaConfig struct {
	Mode       string                     // "legacy" or "advanced"
	Groups     []utils.TCriteriaGroup     // Criteria groups for stacking (legacy)
	Legacy     []utils.TCriteria          // Legacy format for backward compatibility
	Expression *utils.TCriteriaExpression // New nested expression format
}


/**************************************************************************************************
** getCriteriaConfig parses the provided criteria string and returns the configuration.
** If the criteria string is empty, it falls back to the CRITERIA environment variable.
** If both are empty, it returns the default criteria configuration.
** It supports both legacy array format and advanced object format.
**
** @param criteria - The criteria string to parse (from CLI flag or other source)
** @return CriteriaConfig - The processed criteria configuration
** @return error - An error if parsing the criteria string fails, or nil otherwise.
**************************************************************************************************/
func getCriteriaConfig(criteria string) (CriteriaConfig, error) {
	criteriaOverride := criteria
	// Fall back to environment variable if criteria parameter is empty
	if criteriaOverride == "" {
		criteriaOverride = os.Getenv("CRITERIA")
	}
	if criteriaOverride == "" {
		return CriteriaConfig{
			Mode:   "legacy",
			Legacy: utils.DefaultCriteria,
		}, nil
	}

	// First, try to parse as advanced criteria format
	var advancedCriteria utils.TAdvancedCriteria
	if err := json.Unmarshal([]byte(criteriaOverride), &advancedCriteria); err == nil && advancedCriteria.Mode != "" {
		// Successfully parsed as advanced format
		return CriteriaConfig{
			Mode:       advancedCriteria.Mode,
			Groups:     advancedCriteria.Groups,
			Expression: advancedCriteria.Expression,
		}, nil
	}

	// Fallback to legacy array format
	var legacyCriteria []utils.TCriteria
	if err := json.Unmarshal([]byte(criteriaOverride), &legacyCriteria); err != nil {
		return CriteriaConfig{}, fmt.Errorf("failed to parse criteria as either advanced or legacy format: %w", err)
	}

	return CriteriaConfig{
		Mode:   "legacy",
		Legacy: legacyCriteria,
	}, nil
}






/**************************************************************************************************
** ParseCriteria is a small public wrapper around getCriteriaConfig for testing and callers
** that need to parse a criteria string directly. It honors the provided string and falls
** back to the CRITERIA environment variable only when the string is empty.
**************************************************************************************************/
func ParseCriteria(criteria string) (CriteriaConfig, error) {
	return getCriteriaConfig(criteria)
}
