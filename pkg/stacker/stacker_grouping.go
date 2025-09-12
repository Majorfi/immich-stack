package stacker

import (
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** buildGroupKey constructs a key from criteria values using a string builder for efficiency.
** The key is built by joining values with '|' separator.
**
** @param values - List of values to join
** @param builder - Pre-allocated string builder to reuse
** @return string - The constructed key
**************************************************************************************************/
func buildGroupKey(values []string, builder *strings.Builder) string {
	builder.Reset()
	for i, v := range values {
		if i > 0 {
			builder.WriteByte('|')
		}
		builder.WriteString(v)
	}
	return builder.String()
}

/**************************************************************************************************
** flattenCriteriaFromGroups returns a flattened list of criteria from all groups.
**************************************************************************************************/
func flattenCriteriaFromGroups(groups []utils.TCriteriaGroup) []utils.TCriteria {
	out := make([]utils.TCriteria, 0)
	for _, g := range groups {
		out = append(out, g.Criteria...)
	}
	return out
}

/**************************************************************************************************
** findOriginalNameDelimiters searches for delimiters in originalFileName criteria.
** This consolidates the repeated delimiter discovery logic used across different stacking modes.
**
** @param criteria - List of criteria to search
** @return []string - Delimiters found, or empty slice if none
**************************************************************************************************/
func findOriginalNameDelimiters(criteria []utils.TCriteria) []string {
	for _, c := range criteria {
		if c.Key == "originalFileName" && c.Split != nil && len(c.Split.Delimiters) > 0 {
			return c.Split.Delimiters
		}
	}
	return nil
}
