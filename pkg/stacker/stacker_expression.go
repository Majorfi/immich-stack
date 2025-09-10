package stacker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
)

// Package-level boolean fields map to avoid reallocation on each call
var booleanFields = map[string]bool{
	"hasMetadata": true,
	"isArchived":  true,
	"isFavorite":  true,
	"isOffline":   true,
	"isTrashed":   true,
}

/**************************************************************************************************
** EvaluateExpression recursively evaluates a nested criteria expression against an asset.
** Returns true if the asset matches the expression, false otherwise.
**
** @param expr - The criteria expression to evaluate
** @param asset - The asset to evaluate against
** @return bool - True if the asset matches the expression
** @return error - An error if evaluation fails
****************************************************************************************************/
func EvaluateExpression(expr *utils.TCriteriaExpression, asset utils.TAsset) (bool, error) {
	if expr == nil {
		return false, errors.New("expression cannot be nil")
	}

	// Leaf node: evaluate single criteria
	if expr.Criteria != nil {
		return evaluateSingleCriteria(*expr.Criteria, asset)
	}

	// Operator node: evaluate children
	if expr.Operator == nil {
		return false, errors.New("expression must have either criteria or operator")
	}

	if len(expr.Children) == 0 {
		return false, errors.New("operator expression must have children")
	}

	switch *expr.Operator {
	case "AND":
		for _, child := range expr.Children {
			result, err := EvaluateExpression(&child, asset)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil // Short-circuit: AND requires all to be true
			}
		}
		return true, nil

	case "OR":
		for _, child := range expr.Children {
			result, err := EvaluateExpression(&child, asset)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil // Short-circuit: OR requires only one to be true
			}
		}
		return false, nil

	case "NOT":
		if len(expr.Children) != 1 {
			return false, errors.New("NOT operator requires exactly one child")
		}
		result, err := EvaluateExpression(&expr.Children[0], asset)
		if err != nil {
			return false, err
		}
		return !result, nil

	default:
		return false, fmt.Errorf("unknown operator: %s", *expr.Operator)
	}
}

/**************************************************************************************************
** evaluateSingleCriteria evaluates a single criteria against an asset.
** This is a helper function for the recursive expression evaluator.
**
** @param criteria - The single criteria to evaluate
** @param asset - The asset to evaluate against
** @return bool - True if the asset matches the criteria
** @return error - An error if evaluation fails
****************************************************************************************************/
func evaluateSingleCriteria(criteria utils.TCriteria, asset utils.TAsset) (bool, error) {
	// Use the shared extractor logic
	extractor, ok := getExtractor(criteria.Key)
	if !ok {
		return false, fmt.Errorf("unknown criteria key: %s", criteria.Key)
	}

	value, err := extractor(asset, criteria)
	if err != nil {
		return false, err
	}

	// For expression evaluation, we need to validate the extracted value
	// Empty values indicate the criteria couldn't be applied (e.g., no match)
	if value == "" {
		return false, nil
	}

	// For boolean fields, treat "true" as matching and "false" as not matching
	// This makes NOT expressions work intuitively: NOT isArchived means "not archived"
	if booleanFields[criteria.Key] {
		return value == "true", nil
	}

	// If there's a regex pattern, validate it against the extracted value
	if criteria.Regex != nil {
		// For generic fields (like type), we need to validate the regex pattern
		// (originalFileName and originalPath extractors handle this internally)
		if criteria.Key != "originalFileName" && criteria.Key != "originalPath" {
			re, compileErr := utils.RegexCompile(criteria.Regex.Key)
			if compileErr != nil {
				return false, fmt.Errorf("invalid regex pattern %s: %w", criteria.Regex.Key, compileErr)
			}
			matches := re.FindStringSubmatch(value)
			if len(matches) == 0 {
				return false, nil // Regex didn't match
			}
		}
	}

	return true, nil
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
