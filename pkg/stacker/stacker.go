package stacker

import (
	"fmt"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

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

	criteriaConfig, err := getCriteriaConfig(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to get criteria config: %w", err)
	}

	// Handle different criteria modes
	switch criteriaConfig.Mode {
	case "advanced":
		if criteriaConfig.Expression != nil {
			return stackByAdvanced(assets, criteriaConfig, parentFilenamePromote, parentExtPromote, logger)
		} else if len(criteriaConfig.Groups) > 0 {
			return stackByLegacyGroups(assets, criteriaConfig, parentFilenamePromote, parentExtPromote, logger)
		}
		return nil, fmt.Errorf("advanced mode specified but no expression or groups provided")
	case "legacy":
		fallthrough
	default:
		// Use legacy criteria for backward compatibility
		return stackByLegacy(assets, criteriaConfig.Legacy, parentFilenamePromote, parentExtPromote, logger)
	}
}

/**************************************************************************************************
** stackByLegacy handles traditional criteria-based stacking using a simple list of criteria.
** This is the original stacking logic that groups assets based on matching criteria values.
**************************************************************************************************/
func stackByLegacy(assets []utils.TAsset, stackingCriteria []utils.TCriteria, parentFilenamePromote string, parentExtPromote string, logger *logrus.Logger) ([][]utils.TAsset, error) {
	// Precompile regex patterns from legacy criteria
	if err := PrecompileRegexes(stackingCriteria); err != nil {
		return nil, fmt.Errorf("failed to precompile legacy criteria regexes: %w", err)
	}
	// Pre-compute promotion key maps for O(1) lookup
	promotionMaps := buildPromotionMaps(stackingCriteria)

	// Find delimiters for originalFileName criteria
	delimiters := findOriginalNameDelimiters(stackingCriteria)

	// Debug logging
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		listOfCriteria := make([]string, len(stackingCriteria))
		for i, c := range stackingCriteria {
			listOfCriteria[i] = c.Key
		}
		logger.Debugf("Legacy criteria stacking with criteria: %s", listOfCriteria)
		logger.Debugf("Parent filename promote: %s", parentFilenamePromote)
		logger.Debugf("Parent extension promote: %s", parentExtPromote)
		logger.Debugf("Delimiters: %v", delimiters)
	}

	groups := make(map[string][]utils.TAsset, len(assets)/2)
	// Thread-safe map to store promotion data: assetID -> (criteriaKey -> promoteValue)
	promoteData := &safePromoteData{
		data: make(map[string]map[string]string),
	}

	// Pre-allocate string builder for efficiency
	var keyBuilder strings.Builder
	keyBuilder.Grow(512) // Pre-allocate reasonable size for keys

	for _, asset := range assets {
		values, assetPromoteValues, err := applyCriteriaWithPromote(asset, stackingCriteria)
		if err != nil {
			return nil, fmt.Errorf("failed to apply criteria to asset %s: %w", asset.OriginalFileName, err)
		}

		key := buildGroupKey(values, &keyBuilder)
		if key == "" {
			continue
		}

		if logger.IsLevelEnabled(logrus.DebugLevel) {
			logger.WithFields(logrus.Fields{"stack": key}).Debugf("Asset %s", asset.OriginalFileName)
		}

		groups[key] = append(groups[key], asset)

		// Store promotion values if any
		if len(assetPromoteValues) > 0 {
			promoteData.Set(asset.ID, assetPromoteValues)
		}
	}
	
	// Merge groups that should be together based on time proximity
	groups, err := mergeTimeBasedGroups(groups, stackingCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to merge time-based groups: %w", err)
	}

	// Count how many valid stacks we'll have (groups with 2+ assets)
	validStackCount := 0
	for _, group := range groups {
		if len(group) > 1 {
			validStackCount++
		}
	}

	result := make([][]utils.TAsset, 0, validStackCount)
	for _, group := range groups {
		if len(group) > 1 {
			result = append(result, sortStack(group, parentFilenamePromote, parentExtPromote, delimiters, stackingCriteria, promoteData, promotionMaps))
		}
	}

	logStackingResults("Legacy criteria stacking", len(result), len(assets), logger)

	return result, nil
}

/**************************************************************************************************
** stackByAdvanced handles expression-based stacking using nested logical expressions.
** This allows complex AND/OR/NOT logic for advanced asset filtering and grouping.
**************************************************************************************************/
func stackByAdvanced(assets []utils.TAsset, config CriteriaConfig, parentFilenamePromote string, parentExtPromote string, logger *logrus.Logger) ([][]utils.TAsset, error) {
	if config.Expression == nil {
		return nil, fmt.Errorf("advanced mode requires a criteria expression")
	}

	// Debug logging
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		if config.Expression != nil {
			logger.Debugf("Advanced criteria (expression-based) stacking with expression evaluation")
		} else {
			logger.Debugf("Advanced criteria (groups-based) stacking with %d groups", len(config.Groups))
		}
		logger.Debugf("Parent filename promote: %s", parentFilenamePromote)
		logger.Debugf("Parent extension promote: %s", parentExtPromote)
	}

	// Precompile regex patterns from the expression leaves to avoid first-hit compilation
	if err := PrecompileRegexes(config.Expression); err != nil {
		return nil, fmt.Errorf("failed to precompile expression regexes: %w", err)
	}

	// Build criteria list from expression for delimiter detection and regex promotion
	exprCriteria := flattenCriteriaFromExpression(config.Expression)

	// Pre-compute promotion key maps for O(1) lookup
	promotionMaps := buildPromotionMaps(exprCriteria)

	// Find delimiters for originalFileName criteria
	delimiters := findOriginalNameDelimiters(exprCriteria)

	// Group assets by their expression-based grouping keys
	stackGroups := make(map[string][]utils.TAsset)
	promoteData := &safePromoteData{data: make(map[string]map[string]string)}

	for _, asset := range assets {
		// Check if asset matches the expression
		matches, err := EvaluateExpression(config.Expression, asset)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression for asset %s: %w", asset.OriginalFileName, err)
		}

		if !matches {
			continue // Skip assets that don't match the expression
		}

		// Build grouping key based on matching criteria values
		key, err := buildExpressionGroupingKey(asset, config.Expression, exprCriteria)
		if err != nil {
			return nil, fmt.Errorf("failed to build grouping key for asset %s: %w", asset.OriginalFileName, err)
		}

		if key == "" {
			continue // Skip assets with empty grouping keys
		}

		if logger.IsLevelEnabled(logrus.DebugLevel) {
			logger.Debugf("Asset %s (%s) -> grouping key: %s", asset.OriginalFileName, asset.ID, key)
		}

		// Add asset to the appropriate group
		stackGroups[key] = append(stackGroups[key], asset)

		// Collect promotion values for sorting within each group
		_, promVals, _ := applyCriteriaWithPromote(asset, exprCriteria)
		if len(promVals) > 0 {
			promoteData.Set(asset.ID, promVals)
		}
	}

	// Merge groups that should be together based on time proximity
	stackGroups, err := mergeTimeBasedGroups(stackGroups, exprCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to merge time-based groups: %w", err)
	}

	// Convert groups to stacks (filter out groups with < 2 assets)
	result := make([][]utils.TAsset, 0, len(stackGroups))

	for key, group := range stackGroups {
		if len(group) < 2 {
			if logger.IsLevelEnabled(logrus.DebugLevel) {
				logger.Debugf("Skipping group with key %s (only %d asset)", key, len(group))
			}
			continue // Skip groups with fewer than 2 assets
		}

		// Sort the group using existing sorting pipeline
		sorted := sortStack(group, parentFilenamePromote, parentExtPromote, delimiters, exprCriteria, promoteData, promotionMaps)
		result = append(result, sorted)

		if logger.IsLevelEnabled(logrus.DebugLevel) {
			logger.Debugf("Formed stack with %d assets from key: %s", len(sorted), key)
		}
	}

	logStackingResults("Advanced criteria (expression-based)", len(result), len(assets), logger)

	return result, nil
}

/**************************************************************************************************
** stackByLegacyGroups handles group-based stacking using OR/AND logic between criteria groups.
** This is the intermediate complexity level between legacy and full expression-based stacking.
**************************************************************************************************/
func stackByLegacyGroups(assets []utils.TAsset, config CriteriaConfig, parentFilenamePromote string, parentExtPromote string, logger *logrus.Logger) ([][]utils.TAsset, error) {
	if len(config.Groups) == 0 {
		return nil, fmt.Errorf("groups-based mode requires at least one criteria group")
	}

	// Debug logging
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.Debugf("Advanced criteria (groups-based) stacking with %d groups", len(config.Groups))
		logger.Debugf("Parent filename promote: %s", parentFilenamePromote)
		logger.Debugf("Parent extension promote: %s", parentExtPromote)
	}

	// Precompile regex patterns from groups
	if err := PrecompileRegexes(config.Groups); err != nil {
		return nil, fmt.Errorf("failed to precompile group regexes: %w", err)
	}

	// Flatten criteria across groups for delimiter detection and regex promotion
	groupCriteria := flattenCriteriaFromGroups(config.Groups)

	// Pre-compute promotion key maps for O(1) lookup
	promotionMaps := buildPromotionMaps(groupCriteria)

	// Find delimiters for originalFileName criteria
	delimiters := findOriginalNameDelimiters(groupCriteria)

	// For groups mode with OR semantics, we need to build a connectivity graph
	// where assets are connected if they share any grouping keys from OR groups
	assetKeys := make(map[string][]string) // assetID -> list of grouping keys
	promoteData := &safePromoteData{data: make(map[string]map[string]string)}
	matchingAssets := make([]utils.TAsset, 0)

	for _, asset := range assets {
		groupKeys, err := applyAdvancedCriteria(asset, config.Groups)
		if err != nil {
			return nil, fmt.Errorf("failed to apply advanced criteria to asset %s: %w", asset.OriginalFileName, err)
		}

		if len(groupKeys) > 0 {
			assetKeys[asset.ID] = groupKeys
			matchingAssets = append(matchingAssets, asset)

			if logger.IsLevelEnabled(logrus.DebugLevel) {
				logger.Debugf("Asset %s (%s) -> grouping keys: %v", asset.OriginalFileName, asset.ID, groupKeys)
			}

			// Record promotion values for assets that appear in any group
			_, promVals, _ := applyCriteriaWithPromote(asset, groupCriteria)
			if len(promVals) > 0 {
				promoteData.Set(asset.ID, promVals)
			}
		}
	}

	if len(matchingAssets) == 0 {
		logStackingResults("Advanced criteria (groups-based)", 0, len(assets), logger)
		return nil, nil
	}

	// Build connected components using union semantics for OR groups
	components := buildConnectedComponents(matchingAssets, assetKeys, logger)

	// Convert components to result format and sort each component
	result := make([][]utils.TAsset, 0, len(components))

	for _, component := range components {
		if len(component) > 1 {
			sorted := sortStack(component, parentFilenamePromote, parentExtPromote, delimiters, groupCriteria, promoteData, promotionMaps)
			result = append(result, sorted)

			if logger.IsLevelEnabled(logrus.DebugLevel) {
				logger.Debugf("Formed stack with %d assets in connected component", len(sorted))
			}
		} else if logger.IsLevelEnabled(logrus.DebugLevel) {
			logger.Debugf("Skipping component with only 1 asset")
		}
	}

	logStackingResults("Advanced criteria (groups-based)", len(result), len(assets), logger)

	return result, nil
}
