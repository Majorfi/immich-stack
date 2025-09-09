package stacker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

// numericSuffixRegex matches a string that contains only digits (used for extracting numeric suffixes)
var numericSuffixRegex = regexp.MustCompile(`^(\d+)$`)

// buildCriteriaIdentifier creates a unique identifier for a criteria by combining its key and index.
// This prevents collisions when multiple criteria use the same key for promotions.
func buildCriteriaIdentifier(key string, index int) string {
	return fmt.Sprintf("%s:%d", key, index)
}

// safePromoteData provides thread-safe access to promotion data
type safePromoteData struct {
	mu   sync.RWMutex
	data map[string]map[string]string
}

func (s *safePromoteData) Set(assetID string, values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[assetID] = values
}

func (s *safePromoteData) Get(assetID string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values, exists := s.data[assetID]
	return values, exists
}

/**************************************************************************************************
** getRegexPromoteIndex returns the promotion index for an asset based on regex promotion rules.
** It checks each criteria with regex promotion configured and returns the index of the
** promotion value in the promote_keys list.
**
** @param assetID - The ID of the asset to check
** @param promoteData - Thread-safe map of asset ID to promotion values
** @param criteria - The criteria used for stacking
** @param promotionMaps - Pre-computed maps for O(1) promotion key lookup
** @return int - The promotion index (lower is higher priority), or -1 if no match
**************************************************************************************************/
func getRegexPromoteIndex(assetID string, promoteData *safePromoteData, criteria []utils.TCriteria, promotionMaps map[int]map[string]int) int {
	assetPromoteValues, exists := promoteData.Get(assetID)
	if !exists {
		return -1
	}

	// Check each criteria for regex promotion configuration
	lowestIndex := -1
	for i, c := range criteria {
		promoteMap, hasPromoteMap := promotionMaps[i]
		if !hasPromoteMap {
			continue
		}

		criteriaIdentifier := buildCriteriaIdentifier(c.Key, i)
		promoteValue, hasValue := assetPromoteValues[criteriaIdentifier]
		if !hasValue {
			continue
		}

		// O(1) lookup using pre-computed map
		if idx, found := promoteMap[promoteValue]; found {
			// Use the lowest index found across all criteria
			if lowestIndex == -1 || idx < lowestIndex {
				lowestIndex = idx
			}
		}
	}

	return lowestIndex
}

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
** 1. Regex-based promotion (if criteria has regex with promote_index)
** 2. Promoted filenames (PARENT_FILENAME_PROMOTE, comma-separated, order matters)
** 3. Promoted extensions (PARENT_EXT_PROMOTE, comma-separated, order matters)
** 4. Extension priority (jpeg > jpg > png > others)
** 5. Alphabetical order (case-sensitive)
**
** @param stack - List of assets to sort
** @param parentFilenamePromote - Comma-separated list of filename substrings to promote
** @param parentExtPromote - Comma-separated list of extensions to promote
** @param delimiters - Delimiters to use for numeric suffix extraction
** @param stackCriteria - The criteria used to create this stack (for regex promotion)
** @param promoteData - Thread-safe map of asset ID to promotion values from regex criteria
** @param promotionMaps - Pre-computed maps for O(1) promotion key lookup
** @return []Asset - Sorted list of assets
**************************************************************************************************/
func sortStack(stack []utils.TAsset, parentFilenamePromote string, parentExtPromote string, delimiters []string, stackCriteria []utils.TCriteria, promoteData *safePromoteData, promotionMaps map[int]map[string]int) []utils.TAsset {
	promoteSubstrings := parsePromoteList(parentFilenamePromote)
	if len(promoteSubstrings) == 0 && parentFilenamePromote != "" {
		promoteSubstrings = utils.DefaultParentFilenamePromote
	}

	promoteExtensions := parsePromoteList(parentExtPromote)
	if len(promoteExtensions) == 0 {
		promoteExtensions = utils.DefaultParentExtPromote
	}

	// Detect the best match mode based on promote list and filenames
	matchMode := "contains"
	if len(stack) > 0 {
		matchMode = detectPromoteMatchMode(promoteSubstrings, stack[0].OriginalFileName)
	}

	sort.SliceStable(stack, func(i, j int) bool {
		// First, check regex-based promotion
		iRegexPromoteIdx := getRegexPromoteIndex(stack[i].ID, promoteData, stackCriteria, promotionMaps)
		jRegexPromoteIdx := getRegexPromoteIndex(stack[j].ID, promoteData, stackCriteria, promotionMaps)

		// If both have regex promotion values, compare them
		if iRegexPromoteIdx >= 0 && jRegexPromoteIdx >= 0 {
			if iRegexPromoteIdx != jRegexPromoteIdx {
				return iRegexPromoteIdx < jRegexPromoteIdx
			}
		} else if iRegexPromoteIdx >= 0 {
			// i has regex promotion, j doesn't - i comes first
			return true
		} else if jRegexPromoteIdx >= 0 {
			// j has regex promotion, i doesn't - j comes first
			return false
		}

		// Fall back to filename promotion
		iOriginalFileNameNoExt := filepath.Base(stack[i].OriginalFileName)
		jOriginalFileNameNoExt := filepath.Base(stack[j].OriginalFileName)
		iPromoteIdx := getPromoteIndexWithMode(iOriginalFileNameNoExt, promoteSubstrings, matchMode)
		jPromoteIdx := getPromoteIndexWithMode(jOriginalFileNameNoExt, promoteSubstrings, matchMode)
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
			return stackByLegacyGroups(assets, criteriaConfig.Groups, parentFilenamePromote, parentExtPromote, logger)
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
    if err := PrecompileLegacyRegexes(stackingCriteria); err != nil {
        return nil, fmt.Errorf("failed to precompile legacy criteria regexes: %w", err)
    }
	// Pre-compute promotion key maps for O(1) lookup
	promotionMaps := make(map[int]map[string]int) // criteriaIndex -> (promoteKey -> priority)
	for i, c := range stackingCriteria {
		if c.Regex != nil && c.Regex.PromoteIndex != nil && len(c.Regex.PromoteKeys) > 0 {
			promoteMap := make(map[string]int)
			for idx, key := range c.Regex.PromoteKeys {
				promoteMap[key] = idx
			}
			promotionMaps[i] = promoteMap
		}
	}

	// Find delimiters for originalFileName criteria
	var delimiters []string
	for _, c := range stackingCriteria {
		if c.Key == "originalFileName" && c.Split != nil && len(c.Split.Delimiters) > 0 {
			delimiters = c.Split.Delimiters
			break
		}
	}

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

	logger.Infof("Legacy criteria stacking formed %d stacks from %d assets", len(result), len(assets))

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
    if err := PrecompileExpressionRegexes(config.Expression); err != nil {
        return nil, fmt.Errorf("failed to precompile expression regexes: %w", err)
    }

    // Build criteria list from expression for delimiter detection and regex promotion  
    exprCriteria := flattenCriteriaFromExpression(config.Expression)

	// Pre-compute promotion key maps for O(1) lookup
	promotionMaps := make(map[int]map[string]int) // criteriaIndex -> (promoteKey -> priority)
	for i, c := range exprCriteria {
		if c.Regex != nil && c.Regex.PromoteIndex != nil && len(c.Regex.PromoteKeys) > 0 {
			promoteMap := make(map[string]int)
			for idx, key := range c.Regex.PromoteKeys {
				promoteMap[key] = idx
			}
			promotionMaps[i] = promoteMap
		}
	}

	// Find delimiters for originalFileName criteria
	var delimiters []string
	for _, c := range exprCriteria {
		if c.Key == "originalFileName" && c.Split != nil && len(c.Split.Delimiters) > 0 {
			delimiters = c.Split.Delimiters
			break
		}
	}

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

	logger.Infof("Advanced criteria (expression-based) formed %d stacks from %d assets", len(result), len(assets))

	return result, nil
}

/**************************************************************************************************
** stackByLegacyGroups handles group-based stacking using OR/AND logic between criteria groups.
** This is the intermediate complexity level between legacy and full expression-based stacking.
**************************************************************************************************/
func stackByLegacyGroups(assets []utils.TAsset, groups []utils.TCriteriaGroup, parentFilenamePromote string, parentExtPromote string, logger *logrus.Logger) ([][]utils.TAsset, error) {
    if len(groups) == 0 {
        return nil, fmt.Errorf("groups-based mode requires at least one criteria group")
    }

	// Debug logging
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.Debugf("Advanced criteria (groups-based) stacking with %d groups", len(groups))
		logger.Debugf("Parent filename promote: %s", parentFilenamePromote)
		logger.Debugf("Parent extension promote: %s", parentExtPromote)
	}

    // Precompile regex patterns from groups
    if err := PrecompileGroupsRegexes(groups); err != nil {
        return nil, fmt.Errorf("failed to precompile group regexes: %w", err)
    }

    // Flatten criteria across groups for delimiter detection and regex promotion
    groupCriteria := flattenCriteriaFromGroups(groups)

	// Pre-compute promotion key maps for O(1) lookup
	promotionMaps := make(map[int]map[string]int) // criteriaIndex -> (promoteKey -> priority)
	for i, c := range groupCriteria {
		if c.Regex != nil && c.Regex.PromoteIndex != nil && len(c.Regex.PromoteKeys) > 0 {
			promoteMap := make(map[string]int)
			for idx, key := range c.Regex.PromoteKeys {
				promoteMap[key] = idx
			}
			promotionMaps[i] = promoteMap
		}
	}

	// Find delimiters for originalFileName criteria
	var delimiters []string
	for _, c := range groupCriteria {
		if c.Key == "originalFileName" && c.Split != nil && len(c.Split.Delimiters) > 0 {
			delimiters = c.Split.Delimiters
			break
		}
	}

	// For groups mode with OR semantics, we need to build a connectivity graph
	// where assets are connected if they share any grouping keys from OR groups
	assetKeys := make(map[string][]string) // assetID -> list of grouping keys
	promoteData := &safePromoteData{data: make(map[string]map[string]string)}
	matchingAssets := make([]utils.TAsset, 0)

	for _, asset := range assets {
		groupKeys, err := applyAdvancedCriteria(asset, groups)
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
		logger.Infof("Advanced criteria (groups-based) formed 0 stacks from %d assets", len(assets))
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

	logger.Infof("Advanced criteria (groups-based) formed %d stacks from %d assets", len(result), len(assets))

	return result, nil
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
** flattenCriteriaFromExpression returns all leaf criteria contained within an expression.
**************************************************************************************************/
func flattenCriteriaFromExpression(expr *utils.TCriteriaExpression) []utils.TCriteria {
	if expr == nil {
		return nil
	}
	out := make([]utils.TCriteria, 0)
	var walk func(e *utils.TCriteriaExpression)
	walk = func(e *utils.TCriteriaExpression) {
		if e == nil {
			return
		}
		if e.Criteria != nil {
			out = append(out, *e.Criteria)
			return
		}
		for i := range e.Children {
			walk(&e.Children[i])
		}
	}
	walk(expr)
	return out
}

/**************************************************************************************************
** buildExpressionGroupingKey creates a deterministic grouping key for an asset based on the
** leaf criteria values from an expression that actually matched. This enables proper grouping
** where assets with the same criteria values get stacked together.
**
** The key format is "key=value|key=value" in a stable order to avoid collisions.
**
** @param asset - The asset to build a key for
** @param expr - The expression tree to evaluate
** @param criteria - Flattened criteria from the expression for consistent ordering
** @return string - The grouping key, or empty string if no criteria matched
** @return error - Error if evaluation fails
**************************************************************************************************/
func buildExpressionGroupingKey(asset utils.TAsset, expr *utils.TCriteriaExpression, criteria []utils.TCriteria) (string, error) {
	// For expression-based grouping, we need to collect values from leaf criteria
	// that actually contributed to the asset matching the expression
	matchingValues, err := collectMatchingCriteriaValues(asset, expr, criteria)
	if err != nil {
		return "", err
	}

	if len(matchingValues) == 0 {
		return "", nil
	}

	// Build deterministic key in the format: key=value|key=value
	// Use a map to track keys we've already added to avoid duplicates
	var keyParts []string
	addedKeys := make(map[string]bool)

	for _, c := range criteria {
		if value, exists := matchingValues[c.Key]; exists && value != "" {
			keyValue := c.Key + "=" + value
			if !addedKeys[c.Key] {
				keyParts = append(keyParts, keyValue)
				addedKeys[c.Key] = true
			}
		}
	}

	if len(keyParts) == 0 {
		return "", nil
	}

	return strings.Join(keyParts, "|"), nil
}

/**************************************************************************************************
** collectMatchingCriteriaValues walks through an expression tree and collects the actual
** criteria values that contributed to a match for the given asset. This is crucial for
** proper grouping - we only want to group assets that share the same matching criteria values.
**
** @param asset - The asset to evaluate
** @param expr - The expression tree to walk
** @param criteria - All leaf criteria for consistent key ordering
** @return map[string]string - Map of criteria key to extracted value for matching leaves
** @return error - Error if evaluation fails
**************************************************************************************************/
func collectMatchingCriteriaValues(asset utils.TAsset, expr *utils.TCriteriaExpression, criteria []utils.TCriteria) (map[string]string, error) {
	values := make(map[string]string)

	err := walkMatchingCriteria(asset, expr, values)
	if err != nil {
		return nil, err
	}

	return values, nil
}

/**************************************************************************************************
** walkMatchingCriteria recursively walks an expression tree and collects criteria values
** from leaf nodes that evaluate to true. For OR branches, only values from the branch
** that actually matched are included to prevent mixing unrelated criteria.
**
** @param asset - The asset to evaluate
** @param expr - Current expression node
** @param values - Map to collect matching criteria values (modified in-place)
** @return error - Error if evaluation fails
**************************************************************************************************/
func walkMatchingCriteria(asset utils.TAsset, expr *utils.TCriteriaExpression, values map[string]string) error {
	if expr == nil {
		return nil
	}

	// Leaf node: evaluate criteria and collect value if it matches
	if expr.Criteria != nil {
		matches, err := evaluateSingleCriteria(*expr.Criteria, asset)
		if err != nil {
			return err
		}

		if matches {
			// Extract the value for grouping - use processed criteria values for consistent grouping
			// For regex criteria, we want the matched portion, not the full filename
			criteriaValues, _, err := applyCriteriaWithPromote(asset, []utils.TCriteria{*expr.Criteria})
			if err != nil {
				return err
			}

			if len(criteriaValues) > 0 && criteriaValues[0] != "" {
				values[expr.Criteria.Key] = criteriaValues[0]
			}
		}

		return nil
	}

	// Operator node: evaluate based on logical operator
	if expr.Operator == nil || len(expr.Children) == 0 {
		return fmt.Errorf("expression node must have either criteria or operator with children")
	}

	switch *expr.Operator {
	case "AND":
		// For AND: collect values from all children that evaluate to true
		// Use a temporary map to collect all values, then only keep them if all children match
		tempValues := make(map[string]string)
		allMatch := true

		for _, child := range expr.Children {
			childMatches, err := EvaluateExpression(&child, asset)
			if err != nil {
				return err
			}

			if childMatches {
				// Child matches, collect its values in temporary map
				err = walkMatchingCriteria(asset, &child, tempValues)
				if err != nil {
					return err
				}
			} else {
				allMatch = false
			}
		}

		// If all children match, copy temp values to main values map
		if allMatch {
			for k, v := range tempValues {
				values[k] = v
			}
		}

	case "OR":
		// For OR: collect values only from the first child that matches
		// This prevents mixing values from different OR branches in the grouping key
		for _, child := range expr.Children {
			childMatches, err := EvaluateExpression(&child, asset)
			if err != nil {
				return err
			}

			if childMatches {
				// Found a matching branch, collect its values and stop
				// Use a temporary map to collect values from this branch only
				tempValues := make(map[string]string)
				err = walkMatchingCriteria(asset, &child, tempValues)
				if err != nil {
					return err
				}

				// Copy values from the successful branch to the main values map
				for k, v := range tempValues {
					values[k] = v
				}
				break // Only collect from the first matching OR branch
			}
		}

	case "NOT":
		// For NOT: if the child doesn't match, the NOT matches but contributes no values
		if len(expr.Children) != 1 {
			return fmt.Errorf("NOT operator must have exactly one child")
		}

		childMatches, err := EvaluateExpression(&expr.Children[0], asset)
		if err != nil {
			return err
		}

		// NOT matches when child doesn't match, but contributes no grouping values
		// Assets grouped by NOT operations would all have the same "empty" key
		// This is expected behavior - NOT is used for filtering, not grouping
		if !childMatches {
			// NOT expression matches but adds no values to grouping key
			// This means all assets that match via NOT will be grouped together
		}

	default:
		return fmt.Errorf("unknown operator: %s", *expr.Operator)
	}

	return nil
}

/**************************************************************************************************
** buildConnectedComponents creates connected components of assets based on shared grouping keys.
** Assets that share any grouping key are considered connected and will be placed in the same
** component, implementing union semantics for OR groups.
**
** @param assets - List of assets that matched at least one group
** @param assetKeys - Map from asset ID to list of grouping keys for that asset
** @param logger - Logger for debug output
** @return [][]utils.TAsset - List of connected components (each is a potential stack)
**************************************************************************************************/
func buildConnectedComponents(assets []utils.TAsset, assetKeys map[string][]string, logger *logrus.Logger) [][]utils.TAsset {
	if len(assets) == 0 {
		return nil
	}

	// Build a map from grouping keys to assets that have that key
	keyToAssets := make(map[string][]string) // grouping key -> list of asset IDs
	for assetID, keys := range assetKeys {
		for _, key := range keys {
			keyToAssets[key] = append(keyToAssets[key], assetID)
		}
	}

	// Build adjacency list for the connectivity graph
	assetIDToIndex := make(map[string]int)
	for i, asset := range assets {
		assetIDToIndex[asset.ID] = i
	}

	// Create adjacency list where assets are connected if they share any grouping key
	// TODO: For very large groups, consider deduping neighbors per node to reduce list sizes
	// (e.g., use map[int]bool per node before appending). DFS handles duplicates fine but
	// this could improve memory usage for dense graphs.
	adjacency := make([][]int, len(assets))
	for _, assetIDs := range keyToAssets {
		// Connect all assets that share this grouping key
		for i, assetID1 := range assetIDs {
			for j, assetID2 := range assetIDs {
				if i != j {
					idx1, ok1 := assetIDToIndex[assetID1]
					idx2, ok2 := assetIDToIndex[assetID2]
					if ok1 && ok2 {
						adjacency[idx1] = append(adjacency[idx1], idx2)
					}
				}
			}
		}
	}

	// Find connected components using DFS
	visited := make([]bool, len(assets))
	var components [][]utils.TAsset

	for i := 0; i < len(assets); i++ {
		if !visited[i] {
			// Start a new component from this unvisited node
			var component []utils.TAsset
			dfsConnectedComponents(i, adjacency, visited, assets, &component)

			if len(component) > 0 {
				components = append(components, component)
			}
		}
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.Debugf("Built %d connected components from %d assets", len(components), len(assets))
		for i, comp := range components {
			logger.Debugf("Component %d has %d assets", i, len(comp))
		}
	}

	return components
}

/**************************************************************************************************
** dfsConnectedComponents performs depth-first search to find all assets in a connected component.
**
** @param node - Current asset index to explore
** @param adjacency - Adjacency list representation of the connectivity graph
** @param visited - Array tracking which assets have been visited
** @param assets - Original list of assets
** @param component - Current component being built (modified in-place)
**************************************************************************************************/
func dfsConnectedComponents(node int, adjacency [][]int, visited []bool, assets []utils.TAsset, component *[]utils.TAsset) {
	if visited[node] || node >= len(assets) {
		return
	}

	visited[node] = true
	*component = append(*component, assets[node])

	// Visit all connected nodes
	for _, neighbor := range adjacency[node] {
		if !visited[neighbor] {
			dfsConnectedComponents(neighbor, adjacency, visited, assets, component)
		}
	}
}
