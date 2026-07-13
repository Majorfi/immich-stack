/**************************************************************************************************
** Criteria-match detection: which assets can the configured criteria actually evaluate and
** key. Consumers that reason about StackBy's output need this to distinguish "grouped with
** nothing" (a real singleton) from "never evaluated" (the criteria skipped the asset —
** nothing can be concluded about its companions).
**************************************************************************************************/

package stacker

import (
	"fmt"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** MatchedAssetIDs returns the IDs of the assets the criteria can evaluate: those that
** produce a non-empty grouping key on the same path StackBy would take. Assets absent from
** the result are skipped by StackBy regardless of companions.
**
** @param assets - Assets to check
** @param criteria - Stacking criteria JSON (empty = defaults)
** @return map[string]bool - IDs of the assets the criteria match
** @return error - Any error from criteria parsing or evaluation
**************************************************************************************************/
func MatchedAssetIDs(assets []utils.TAsset, criteria string) (map[string]bool, error) {
	matched := make(map[string]bool, len(assets))
	if len(assets) == 0 {
		return matched, nil
	}

	config, err := getCriteriaConfig(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to get criteria config: %w", err)
	}

	if config.Mode != "advanced" {
		return matchedByLegacyCriteria(assets, config.Legacy)
	}

	// Mirror StackBy's routing: AND-only groups are converted to an expression.
	if config.Expression == nil && len(config.Groups) > 0 && canConvertGroupsToExpression(config.Groups) {
		config.Expression = convertGroupsToExpression(config.Groups)
	}
	if config.Expression != nil {
		return matchedByExpression(assets, config.Expression)
	}
	if len(config.Groups) > 0 {
		return matchedByGroups(assets, config.Groups)
	}
	return nil, fmt.Errorf("advanced mode specified but no expression or groups provided")
}

func matchedByLegacyCriteria(assets []utils.TAsset, criteria []utils.TCriteria) (map[string]bool, error) {
	if err := PrecompileRegexes(criteria); err != nil {
		return nil, fmt.Errorf("failed to precompile legacy criteria regexes: %w", err)
	}
	matched := make(map[string]bool, len(assets))
	var keyBuilder strings.Builder
	for _, asset := range assets {
		values, _, err := applyCriteriaWithPromote(asset, criteria)
		if err != nil {
			return nil, fmt.Errorf("failed to apply criteria to asset %s: %w", asset.OriginalFileName, err)
		}
		if buildGroupKey(values, &keyBuilder) != "" {
			matched[asset.ID] = true
		}
	}
	return matched, nil
}

func matchedByExpression(assets []utils.TAsset, expression *utils.TCriteriaExpression) (map[string]bool, error) {
	if err := PrecompileRegexes(expression); err != nil {
		return nil, fmt.Errorf("failed to precompile expression regexes: %w", err)
	}
	exprCriteria := flattenCriteriaFromExpression(expression)
	matched := make(map[string]bool, len(assets))
	for _, asset := range assets {
		matches, err := EvaluateExpression(expression, asset)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression for asset %s: %w", asset.OriginalFileName, err)
		}
		if !matches {
			continue
		}
		key, err := buildExpressionGroupingKey(asset, expression, exprCriteria)
		if err != nil {
			return nil, fmt.Errorf("failed to build grouping key for asset %s: %w", asset.OriginalFileName, err)
		}
		if key != "" {
			matched[asset.ID] = true
		}
	}
	return matched, nil
}

func matchedByGroups(assets []utils.TAsset, groups []utils.TCriteriaGroup) (map[string]bool, error) {
	if err := PrecompileRegexes(groups); err != nil {
		return nil, fmt.Errorf("failed to precompile group regexes: %w", err)
	}
	matched := make(map[string]bool, len(assets))
	for _, asset := range assets {
		groupKeys, err := applyAdvancedCriteria(asset, groups)
		if err != nil {
			return nil, fmt.Errorf("failed to apply advanced criteria to asset %s: %w", asset.OriginalFileName, err)
		}
		if len(groupKeys) > 0 {
			matched[asset.ID] = true
		}
	}
	return matched, nil
}
