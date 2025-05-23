package stacker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** getCriteriaConfig retrieves the criteria configuration from environment variables.
** If the CRITERIA environment variable is not set, it returns the default criteria
** configuration. Otherwise, it parses the JSON string from the CRITERIA environment
** variable.
**
** @return []utils.TCriteria - A slice of criteria to be used for stacking assets.
** @return error - An error if parsing the CRITERIA environment variable fails, or nil
**                 otherwise.
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
** applyCriteria generates a list of identifying strings for a given asset based on a
** set of criteria. Each criterion specifies a key (e.g., "originalFileName",
** "localDateTime") and optional parameters (like time delta or split rules). The
** function iterates through the criteria, extracts the corresponding value from the
** asset, applies any transformations, and collects non-empty values into a slice of
** strings. This slice serves as a unique key for grouping the asset.
**
** @param asset - The utils.TAsset to apply criteria to.
** @param criteria - A slice of utils.TCriteria defining how to extract and transform
**                   asset properties.
** @return []string - A slice of strings that collectively identify the asset based on
**                    the applied criteria. Empty strings resulting from extractors are
**                    omitted.
** @return error - An error if an unknown criteria key is encountered or if any
**                 extractor function returns an error.
**************************************************************************************************/
func applyCriteria(asset utils.TAsset, criteria []utils.TCriteria) ([]string, error) {
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
		"originalFileName": extractOriginalFileName,
		"originalPath":     extractOriginalPath,
		"ownerId":          func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.OwnerID, nil },
		"type":             func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Type, nil },
		"updatedAt": func(a utils.TAsset, c utils.TCriteria) (string, error) {
			return extractTimeWithDelta(a.UpdatedAt, c.Delta)
		},
		"checksum": func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Checksum, nil },
	}

	result := make([]string, 0, len(criteria))

	for _, c := range criteria {
		extractor, ok := extractors[c.Key]
		if !ok {
			return nil, fmt.Errorf("unknown criteria key: %s", c.Key)
		}
		value, err := extractor(asset, c)
		if err != nil {
			return nil, err
		}
		if value != "" {
			result = append(result, value)
		}
	}

	return result, nil
}

/**************************************************************************************************
** extractOriginalFileName extracts and processes the original file name from an asset
** according to the provided criteria. First, it removes the file extension from the
** asset's OriginalFileName. If the criteria include split parameters (delimiters and
** an index), the base name is further split by those delimiters, and the part at the
** specified index is returned. Alternatively, if regex parameters are provided, the
** base name is processed using regular expressions to extract specific capture groups.
**
** @param asset - The utils.TAsset from which to extract the original file name.
** @param c - The utils.TCriteria containing potential split or regex parameters.
** @return string - The processed original file name (base name, potentially split or matched).
** @return error - An error if the split index is out of range for the resulting parts,
**                 if regex compilation fails, or if the regex index is out of range.
**************************************************************************************************/
func extractOriginalFileName(asset utils.TAsset, c utils.TCriteria) (string, error) {
	baseName := asset.OriginalFileName
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

	// Handle regex processing if configured
	if c.Regex != nil && c.Regex.Key != "" {
		regex, err := regexp.Compile(c.Regex.Key)
		if err != nil {
			return "", fmt.Errorf("failed to compile regex %q: %w", c.Regex.Key, err)
		}

		matches := regex.FindStringSubmatch(baseName)
		if matches == nil {
			return "", nil // No match found, return empty string
		}

		if c.Regex.Index < 0 || c.Regex.Index >= len(matches) {
			return "", fmt.Errorf("regex capture group index %d out of range for %q (found %d groups)",
				c.Regex.Index, baseName, len(matches)-1)
		}

		return matches[c.Regex.Index], nil
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
			return "", fmt.Errorf("split index %d out of range for %q", c.Split.Index, baseName)
		}
		baseName = parts[c.Split.Index]
	}

	return baseName, nil
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
** @return error - An error if the split index is out of range for the resulting parts,
**                 if regex compilation fails, or if the regex index is out of range.
**************************************************************************************************/
func extractOriginalPath(asset utils.TAsset, c utils.TCriteria) (string, error) {
	// Always normalize path separators to forward slashes
	path := strings.ReplaceAll(asset.OriginalPath, "\\", "/")

	// Handle regex processing if configured
	if c.Regex != nil && c.Regex.Key != "" {
		regex, err := regexp.Compile(c.Regex.Key)
		if err != nil {
			return "", fmt.Errorf("failed to compile regex %q: %w", c.Regex.Key, err)
		}

		matches := regex.FindStringSubmatch(path)
		if matches == nil {
			return "", nil // No match found, return empty string
		}

		if c.Regex.Index < 0 || c.Regex.Index >= len(matches) {
			return "", fmt.Errorf("regex capture group index %d out of range for %q (found %d groups)",
				c.Regex.Index, path, len(matches)-1)
		}

		return matches[c.Regex.Index], nil
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
			return "", fmt.Errorf("split index %d out of range for %q", c.Split.Index, path)
		}
		path = parts[c.Split.Index]
	}
	return path, nil
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
