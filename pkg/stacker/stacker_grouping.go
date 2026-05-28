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

/**************************************************************************************************
** canConvertGroupsToExpression reports whether a legacy groups config can be losslessly rewritten
** as an expression tree.
**
** AND groups translate directly to AND expression nodes: the matching values are equivalent.
** OR groups in legacy mode use connectivity-graph union semantics (an asset is connected to
** every other asset that shares ANY criterion key), which expression OR does not replicate
** (walkMatchingCriteria only collects values from the first matching OR branch). Configs
** containing any OR group must stay on the legacy code path to preserve that behavior.
**
** Caveat: overlapping AND groups: when two AND groups share a criterion key AND an asset
** can match both simultaneously, the legacy path emits one grouping key per matched group
** (allowing connectivity-graph union across the groups) while the converted OR(AND, AND)
** expression short-circuits at the first matching branch and emits a single key. For configs
** where each AND group has a mutually exclusive predicate (e.g. distinct filename regexes per
** group, the typical pattern that motivated issue #46), this divergence cannot occur because
** no asset matches more than one group. For configs where the same asset could legitimately
** belong to multiple groups, the connectivity-graph union is the documented behavior: but
** the practical effect on stacking is usually the same since both modes feed assets into
** mergeTimeBasedGroups afterwards.
**************************************************************************************************/
func canConvertGroupsToExpression(groups []utils.TCriteriaGroup) bool {
	for _, g := range groups {
		if g.Operator != "" && g.Operator != "AND" {
			return false
		}
	}
	return true
}

/**************************************************************************************************
** convertGroupsToExpression builds an OR(AND(...), AND(...), ...) expression equivalent to a
** list of AND groups. Callers must check canConvertGroupsToExpression first; passing groups
** that contain OR operators will silently misrepresent the original union semantics.
**
** Shapes produced:
**   - 0 groups → nil (caller decides how to handle)
**   - 1 group, 1 criterion → bare leaf (no AND/OR wrapper)
**   - 1 group, N criteria → single AND node
**   - N groups → OR with N AND children
**************************************************************************************************/
func convertGroupsToExpression(groups []utils.TCriteriaGroup) *utils.TCriteriaExpression {
	children := make([]utils.TCriteriaExpression, 0, len(groups))
	for _, g := range groups {
		expr := groupToExpression(g)
		if expr != nil {
			children = append(children, *expr)
		}
	}

	if len(children) == 0 {
		return nil
	}
	if len(children) == 1 {
		return &children[0]
	}

	orOp := "OR"
	return &utils.TCriteriaExpression{
		Operator: &orOp,
		Children: children,
	}
}

/**************************************************************************************************
** groupToExpression converts a single AND criteria group to an expression tree. Returns nil
** for empty groups so the caller can drop them rather than emitting empty AND nodes that the
** evaluator would reject.
**************************************************************************************************/
func groupToExpression(group utils.TCriteriaGroup) *utils.TCriteriaExpression {
	if len(group.Criteria) == 0 {
		return nil
	}

	leaves := make([]utils.TCriteriaExpression, len(group.Criteria))
	for i := range group.Criteria {
		c := group.Criteria[i] // copy to avoid loop-variable address reuse in Go <1.22
		leaves[i] = utils.TCriteriaExpression{Criteria: &c}
	}

	if len(leaves) == 1 {
		return &leaves[0]
	}

	andOp := "AND"
	return &utils.TCriteriaExpression{
		Operator: &andOp,
		Children: leaves,
	}
}
