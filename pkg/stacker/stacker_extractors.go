package stacker

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
)

// Package-level extractor map to avoid allocation on each call
var extractors = map[string]func(asset utils.TAsset, c utils.TCriteria) (string, error){
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
	"hasMetadata": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return utils.BoolToString(a.HasMetadata), nil },
	"isArchived":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return utils.BoolToString(a.IsArchived), nil },
	"isFavorite":  func(a utils.TAsset, _ utils.TCriteria) (string, error) { return utils.BoolToString(a.IsFavorite), nil },
	"isOffline":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return utils.BoolToString(a.IsOffline), nil },
	"isTrashed":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return utils.BoolToString(a.IsTrashed), nil },
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

/**************************************************************************************************
** getExtractor returns the extractor function for a given criteria key.
** This centralizes all extractor logic to avoid duplication across functions.
**
** @param key - The criteria key (e.g., "originalFileName", "localDateTime")
** @return func(utils.TAsset, utils.TCriteria) (string, error) - The extractor function
** @return bool - Whether the extractor was found
**************************************************************************************************/
func getExtractor(key string) (func(utils.TAsset, utils.TCriteria) (string, error), bool) {
	extractor, exists := extractors[key]
	return extractor, exists
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
** @return map[string]string - A map of criteria identifier to promotion value for regex criteria
**                              with promote_index configured. Uses format "key:index" to avoid
**                              collisions when multiple criteria use the same key.
** @return error - An error if an unknown criteria key is encountered or if any
**                 extractor function returns an error.
**************************************************************************************************/
func applyCriteriaWithPromote(asset utils.TAsset, criteria []utils.TCriteria) ([]string, map[string]string, error) {
	result := make([]string, 0, len(criteria))
	// Use criteria index-based keys to avoid collisions when multiple criteria use the same key
	// Format: "key:index" where index is the position in the criteria slice
	promoteValues := make(map[string]string)

	for i, c := range criteria {
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
			// For other extractors, use the shared extractor logic
			extractor, ok := getExtractor(c.Key)
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
		// Use criteria index-based key to avoid collisions between multiple criteria with same key
		if c.Regex != nil && c.Regex.PromoteIndex != nil {
			criteriaIdentifier := buildCriteriaIdentifier(c.Key, i)
			promoteValues[criteriaIdentifier] = promoteValue
		}
	}

	return result, promoteValues, nil
}

/**************************************************************************************************
** extractOriginalFileName extracts and processes the original file name from an asset
** according to the provided criteria. It uses shared helper functions for common operations.
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
		return applyRegexWithPromote(asset.OriginalFileName, c.Regex.Key, c.Regex.Index, c.Regex.PromoteIndex)
	}

	// For split mode, remove extension first
	baseName := asset.OriginalFileName
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

	// Handle delimiter-based split processing if configured
	if c.Split != nil && len(c.Split.Delimiters) > 0 {
		result, err := splitByDelimiters(baseName, c.Split.Delimiters, c.Split.Index)
		return result, "", err
	}

	return baseName, "", nil
}

/**************************************************************************************************
** extractOriginalPath extracts and processes the original path from an asset according
** to the provided criteria. It uses shared helper functions for common operations.
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
		return applyRegexWithPromote(path, c.Regex.Key, c.Regex.Index, c.Regex.PromoteIndex)
	}

	// Handle delimiter-based split processing if configured
	if c.Split != nil && len(c.Split.Delimiters) > 0 {
		result, err := splitByDelimiters(path, c.Split.Delimiters, c.Split.Index)
		return result, "", err
	}

	return path, "", nil
}

/**************************************************************************************************
** applyRegexWithPromote applies a regex pattern to input text and extracts values at specified
** indices. This consolidates the common regex logic used by both filename and path extractors.
**
** @param input - The input string to match against
** @param pattern - The regex pattern to compile and apply
** @param index - The capture group index for the main value
** @param promoteIndex - Optional capture group index for promotion value
** @return string - The matched value at the specified index
** @return string - The promotion value if promoteIndex is specified, empty otherwise
** @return error - Error if regex compilation fails or indices are out of range
**************************************************************************************************/
func applyRegexWithPromote(input string, pattern string, index int, promoteIndex *int) (string, string, error) {
	regex, err := utils.RegexCompile(pattern)
	if err != nil {
		return "", "", fmt.Errorf("failed to compile regex %q: %w", pattern, err)
	}

	matches := regex.FindStringSubmatch(input)
	if matches == nil {
		// No match found - returns empty values. Caller is responsible for handling unmatched cases.
		// If specific behavior is required for unmatched assets, implement it in the calling code.
		// This function does not perform any mode-specific logic.
		return "", "", nil
	}

	if index < 0 || index >= len(matches) {
		return "", "", fmt.Errorf("regex capture group index %d out of range for %q (found %d groups)",
			index, input, len(matches)-1)
	}

	// Extract promotion value if promote_index is specified
	promoteValue := ""
	if promoteIndex != nil {
		if *promoteIndex < 0 || *promoteIndex >= len(matches) {
			return "", "", fmt.Errorf("regex promote capture group index %d out of range for %q (found %d groups)",
				*promoteIndex, input, len(matches)-1)
		}
		promoteValue = matches[*promoteIndex]
	}

	return matches[index], promoteValue, nil
}

/**************************************************************************************************
** splitByDelimiters splits input text by multiple delimiters and returns the part at the
** specified index. This consolidates the common split logic used by both filename and path extractors.
**
** @param input - The input string to split
** @param delimiters - List of delimiters to split by
** @param index - The index of the part to return
** @return string - The part at the specified index
** @return error - Error if the index is out of range
**************************************************************************************************/
func splitByDelimiters(input string, delimiters []string, index int) (string, error) {
	// Early return if no delimiters - just validate index
	if len(delimiters) == 0 {
		if index == 0 {
			return input, nil
		}
		return "", fmt.Errorf("split index %d out of range for %q", index, input)
	}

	parts := []string{input}
	for _, delim := range delimiters {
		temp := []string{}
		for _, part := range parts {
			temp = append(temp, strings.Split(part, delim)...)
		}
		parts = temp
	}
	if index < 0 || index >= len(parts) {
		return "", fmt.Errorf("split index %d out of range for %q", index, input)
	}
	return parts[index], nil
}
