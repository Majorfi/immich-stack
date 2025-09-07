package stacker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

/****************************************************************************************************
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

/****************************************************************************************************
** evaluateSingleCriteria evaluates a single criteria against an asset.
** This is a helper function for the recursive expression evaluator.
**
** @param criteria - The single criteria to evaluate
** @param asset - The asset to evaluate against
** @return bool - True if the asset matches the criteria
** @return error - An error if evaluation fails
****************************************************************************************************/
func evaluateSingleCriteria(criteria utils.TCriteria, asset utils.TAsset) (bool, error) {
	// Use the existing extractor logic from applyCriteria
	extractors := map[string]func(asset utils.TAsset, c utils.TCriteria) (string, error){
		"id":            func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.ID, nil },
		"deviceAssetId": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceAssetID, nil },
		"deviceId":      func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceID, nil },
		"duration":      func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Duration, nil },
		"fileCreatedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.FileCreatedAt, c.Delta)
		},
		"fileModifiedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.FileModifiedAt, c.Delta)
		},
		"hasMetadata": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.HasMetadata), nil },
		"isArchived":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsArchived), nil },
		"isFavorite":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsFavorite), nil },
		"isOffline":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsOffline), nil },
		"isTrashed":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsTrashed), nil },
		"localDateTime": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.LocalDateTime, c.Delta)
		},
		"originalFileName": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			value, _, err := extractOriginalFileName(a, c)
			return value, err
		},
		"originalPath": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			value, _, err := extractOriginalPath(a, c)
			return value, err
		},
		"ownerId": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.OwnerID, nil },
		"type":    func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Type, nil },
		"updatedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.UpdatedAt, c.Delta)
		},
		"checksum": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Checksum, nil },
	}

	extractor, ok := extractors[criteria.Key]
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
	booleanFields := map[string]bool{
		"hasMetadata": true,
		"isArchived":  true,
		"isFavorite":  true,
		"isOffline":   true,
		"isTrashed":   true,
	}

	if booleanFields[criteria.Key] {
		return value == "true", nil
	}

	// If there's a regex pattern, validate it against the extracted value
	if criteria.Regex != nil {
		// For generic fields (like type), we need to validate the regex pattern
		// (originalFileName and originalPath extractors handle this internally)
		if criteria.Key != "originalFileName" && criteria.Key != "originalPath" {
			re, compileErr := regexp.Compile(criteria.Regex.Key)
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
** extractTimeWithDelta parses a time string and applies a specified time delta if
** configured. The input time string is expected to be in RFC3339Nano format. If a
** delta is provided (non-nil and with non-zero milliseconds), the time is truncated
** to the largest multiple of the delta interval that is less than or equal to the
** original time. The result is returned as a string formatted according to
** utils.TimeFormat in UTC. If the input time string is empty, an empty string is
** returned.
**
** @param timeStr - The time string to parse (RFC3339Nano format).
** @param delta - A pointer to a TDelta struct specifying the time delta to apply. Can be
**                nil or have zero milliseconds, in which case the original time (after
**                parsing and UTC conversion) is formatted and returned.
** @return string - The formatted time string (UTC, utils.TimeFormat) after applying the
**                  delta, or an empty string if the input was empty.
** @return error - An error if parsing the time string fails, or nil otherwise.
**************************************************************************************************/
func extractTimeWithDelta(timeStr string, delta *utils.TDelta) (string, error) {
	if timeStr == "" {
		return "", nil
	}

	t, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse time %s: %w", timeStr, err)
	}

	if delta == nil || delta.Milliseconds == 0 {
		return t.UTC().Format(utils.TimeFormat), nil
	}

	// Truncate to the nearest delta interval
	ms := t.UnixNano() / int64(time.Millisecond)
	interval := int64(delta.Milliseconds)
	truncatedMs := (ms / interval) * interval

	truncatedTime := time.Unix(0, truncatedMs*int64(time.Millisecond)).UTC()
	return truncatedTime.Format(utils.TimeFormat), nil
}

/**************************************************************************************************
** applyCriteriaWithPromote generates a list of identifying strings for a given asset based on a
** set of criteria, and also extracts promotion values from regex criteria if specified.
** Each criterion specifies a key (e.g., "originalFileName", "localDateTime") and optional
** parameters (like time delta, split rules, or regex with promotion). The function iterates
** through the criteria, extracts the corresponding value from the asset, applies any
** transformations, and collects non-empty values into a slice of strings. This slice serves
** as a unique key for grouping the asset. Additionally, it collects promotion values from
** regex criteria that have promote_index configured.
**
** @param asset - The utils.TAsset to apply criteria to.
** @param criteria - A slice of utils.TCriteria defining how to extract and transform
**                   asset properties.
** @return []string - A slice of strings that collectively identify the asset based on
**                    the applied criteria. Empty strings resulting from extractors are
**                    omitted.
** @return map[string]string - A map of criteria key to promotion value for regex criteria
**                              with promote_index configured.
** @return error - An error if an unknown criteria key is encountered or if any
**                 extractor function returns an error.
**************************************************************************************************/
func applyCriteriaWithPromote(asset utils.TAsset, criteria []utils.TCriteria) ([]string, map[string]string, error) {
	result := make([]string, 0, len(criteria))
	// NOTE: promoteValues keyed by criteria.Key. If multiple regex promotions exist for the
	// same key (e.g., two different filename regexes), later matches may overwrite earlier ones.
	// TODO: Consider keying by criteria identifier or pattern if this becomes an issue.
	promoteValues := make(map[string]string)

	for _, c := range criteria {
		var value string
		var promoteValue string
		var err error

		// Handle special cases that can return promotion values
		switch c.Key {
		case "originalFileName":
			value, promoteValue, err = extractOriginalFileName(asset, c)
		case "originalPath":
			value, promoteValue, err = extractOriginalPath(asset, c)
		default:
			// For other extractors, we need to use the old signature
			extractors := map[string]func(asset utils.TAsset, c utils.TCriteria) (string, error){
				"id":            func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.ID, nil },
				"deviceAssetId": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceAssetID, nil },
				"deviceId":      func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceID, nil },
				"duration":      func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Duration, nil },
				"fileCreatedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
					return extractTimeWithDelta(a.FileCreatedAt, c.Delta)
				},
				"fileModifiedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
					return extractTimeWithDelta(a.FileModifiedAt, c.Delta)
				},
				"hasMetadata": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.HasMetadata), nil },
				"isArchived":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsArchived), nil },
				"isFavorite":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsFavorite), nil },
				"isOffline":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsOffline), nil },
				"isTrashed":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsTrashed), nil },
				"localDateTime": func(a utils.TAsset, c utils.TCriteria) (string, error) {
					return extractTimeWithDelta(a.LocalDateTime, c.Delta)
				},
				"ownerId": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.OwnerID, nil },
				"type":    func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Type, nil },
				"updatedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
					return extractTimeWithDelta(a.UpdatedAt, c.Delta)
				},
				"checksum": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Checksum, nil },
			}

			extractor, ok := extractors[c.Key]
			if !ok {
				return nil, nil, fmt.Errorf("unknown criteria key: %s", c.Key)
			}
			value, err = extractor(asset, c)
		}

		if err != nil {
			return nil, nil, err
		}

		if value != "" {
			result = append(result, value)
		}

		// Store promotion value if present (including empty strings, which are valid promote values)
		if c.Regex != nil && c.Regex.PromoteIndex != nil {
			promoteValues[c.Key] = promoteValue
		}
	}

	return result, promoteValues, nil
}

/**************************************************************************************************
** extractOriginalFileName extracts and processes the original file name from an asset
** according to the provided criteria. If using regex parameters, the regex is applied to
** the full filename including extension. If using split parameters, the extension is
** removed first, then the base name is split by the delimiters.
**
** @param asset - The utils.TAsset from which to extract the original file name.
** @param c - The utils.TCriteria containing potential split or regex parameters.
** @return string - The processed original file name (base name, potentially split or matched).
** @return string - The promotion value if regex promote_index is specified, empty otherwise.
** @return error - An error if the split index is out of range for the resulting parts,
**                 if regex compilation fails, or if the regex index is out of range.
**************************************************************************************************/
func extractOriginalFileName(asset utils.TAsset, c utils.TCriteria) (string, string, error) {
	// Handle regex processing if configured - use full filename including extension
	if c.Regex != nil && c.Regex.Key != "" {
		regex, err := regexp.Compile(c.Regex.Key)
		if err != nil {
			return "", "", fmt.Errorf("failed to compile regex %q: %w", c.Regex.Key, err)
		}

		matches := regex.FindStringSubmatch(asset.OriginalFileName)
		if matches == nil {
			// No match found - in legacy mode, this means the asset may be excluded from stacking
			// or grouped by other criteria if multiple criteria are specified.
			// For complex regex filtering, consider using advanced mode.
			return "", "", nil
		}

		if c.Regex.Index < 0 || c.Regex.Index >= len(matches) {
			return "", "", fmt.Errorf("regex capture group index %d out of range for %q (found %d groups)",
				c.Regex.Index, asset.OriginalFileName, len(matches)-1)
		}

		// Extract promotion value if promote_index is specified
		promoteValue := ""
		if c.Regex.PromoteIndex != nil {
			if *c.Regex.PromoteIndex < 0 || *c.Regex.PromoteIndex >= len(matches) {
				return "", "", fmt.Errorf("regex promote capture group index %d out of range for %q (found %d groups)",
					*c.Regex.PromoteIndex, asset.OriginalFileName, len(matches)-1)
			}
			promoteValue = matches[*c.Regex.PromoteIndex]
		}

		return matches[c.Regex.Index], promoteValue, nil
	}

	// For split mode, remove extension first
	baseName := asset.OriginalFileName
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

	// Handle delimiter-based split processing if configured
	if c.Split != nil && len(c.Split.Delimiters) > 0 {
		parts := []string{baseName}
		for _, delim := range c.Split.Delimiters {
			temp := []string{}
			for _, part := range parts {
				temp = append(temp, strings.Split(part, delim)...)
			}
			parts = temp
		}
		if c.Split.Index < 0 || c.Split.Index >= len(parts) {
			return "", "", fmt.Errorf("split index %d out of range for %q", c.Split.Index, baseName)
		}
		baseName = parts[c.Split.Index]
	}

	return baseName, "", nil
}

/**************************************************************************************************
** extractOriginalPath extracts and processes the original path from an asset according
** to the provided criteria. If the criteria include split parameters (delimiters and an
** index), the path is split by those delimiters, and the part at the specified index
** is returned. Alternatively, if regex parameters are provided, the path is processed
** using regular expressions to extract specific capture groups. The function handles
** both forward slashes and backslashes as path separators by always normalizing them
** to forward slashes.
**
** @param asset - The utils.TAsset from which to extract the original path.
** @param c - The utils.TCriteria containing potential split or regex parameters.
** @return string - The processed original path (potentially split or matched).
** @return string - The promotion value if regex promote_index is specified, empty otherwise.
** @return error - An error if the split index is out of range for the resulting parts,
**                 if regex compilation fails, or if the regex index is out of range.
**************************************************************************************************/
func extractOriginalPath(asset utils.TAsset, c utils.TCriteria) (string, string, error) {
	// Always normalize path separators to forward slashes
	path := strings.ReplaceAll(asset.OriginalPath, "\\", "/")

	// Handle regex processing if configured
	if c.Regex != nil && c.Regex.Key != "" {
		regex, err := regexp.Compile(c.Regex.Key)
		if err != nil {
			return "", "", fmt.Errorf("failed to compile regex %q: %w", c.Regex.Key, err)
		}

		matches := regex.FindStringSubmatch(path)
		if matches == nil {
			// No match found - in legacy mode, this means the asset may be excluded from stacking
			// or grouped by other criteria if multiple criteria are specified.
			// For complex regex filtering, consider using advanced mode.
			return "", "", nil
		}

		if c.Regex.Index < 0 || c.Regex.Index >= len(matches) {
			return "", "", fmt.Errorf("regex capture group index %d out of range for %q (found %d groups)",
				c.Regex.Index, path, len(matches)-1)
		}

		// Extract promotion value if promote_index is specified
		promoteValue := ""
		if c.Regex.PromoteIndex != nil {
			if *c.Regex.PromoteIndex < 0 || *c.Regex.PromoteIndex >= len(matches) {
				return "", "", fmt.Errorf("regex promote capture group index %d out of range for %q (found %d groups)",
					*c.Regex.PromoteIndex, path, len(matches)-1)
			}
			promoteValue = matches[*c.Regex.PromoteIndex]
		}

		return matches[c.Regex.Index], promoteValue, nil
	}

	// Handle delimiter-based split processing if configured
	if c.Split != nil && len(c.Split.Delimiters) > 0 {
		parts := []string{path}
		for _, delim := range c.Split.Delimiters {
			temp := []string{}
			for _, part := range parts {
				temp = append(temp, strings.Split(part, delim)...)
			}
			parts = temp
		}
		if c.Split.Index < 0 || c.Split.Index >= len(parts) {
			return "", "", fmt.Errorf("split index %d out of range for %q", c.Split.Index, path)
		}
		path = parts[c.Split.Index]
	}
	return path, "", nil
}

/**************************************************************************************************
** boolToString converts a boolean value to its string representation. It returns "true"
** for a true input and "false" for a false input.
**
** @param b - The boolean value to convert.
** @return string - The string "true" or "false".
**************************************************************************************************/
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

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
	extractors := map[string]func(asset utils.TAsset, c utils.TCriteria) (string, error){
		"id":            func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.ID, nil },
		"deviceAssetId": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceAssetID, nil },
		"deviceId":      func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceID, nil },
		"duration":      func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Duration, nil },
		"fileCreatedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.FileCreatedAt, c.Delta)
		},
		"fileModifiedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.FileModifiedAt, c.Delta)
		},
		"hasMetadata": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.HasMetadata), nil },
		"isArchived":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsArchived), nil },
		"isFavorite":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsFavorite), nil },
		"isOffline":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsOffline), nil },
		"isTrashed":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsTrashed), nil },
		"localDateTime": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.LocalDateTime, c.Delta)
		},
		"originalFileName": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			value, _, err := extractOriginalFileName(a, c)
			return value, err
		},
		"originalPath": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			value, _, err := extractOriginalPath(a, c)
			return value, err
		},
		"ownerId": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.OwnerID, nil },
		"type":    func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Type, nil },
		"updatedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.UpdatedAt, c.Delta)
		},
		"checksum": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Checksum, nil },
	}

	var groupingKeys []string

	// Process each criteria group
	for groupIdx, group := range groups {
		if group.Operator == "OR" {
			// For OR groups, each matching criterion creates its own grouping opportunity
			// This allows assets to be grouped by ANY of the criteria, creating multiple potential stacks
			for criteriaIdx, criterion := range group.Criteria {
				extractor, ok := extractors[criterion.Key]
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
				extractor, ok := extractors[criterion.Key]
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

/**************************************************************************************************
** ParseCriteria is a small public wrapper around getCriteriaConfig for testing and callers
** that need to parse a criteria string directly. It honors the provided string and falls
** back to the CRITERIA environment variable only when the string is empty.
**************************************************************************************************/
func ParseCriteria(criteria string) (CriteriaConfig, error) {
    return getCriteriaConfig(criteria)
}
