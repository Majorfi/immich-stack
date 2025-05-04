package stacker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** getCriteriaConfig retrieves the criteria configuration from environment variables.
** If CRITERIA env var is not set, returns the default criteria configuration.
**
** @return []Criteria - List of criteria to use for stacking
** @return error - Any error that occurred during configuration retrieval
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
** applyCriteria applies the configured criteria to an asset.
** Returns a list of strings that uniquely identify the asset based on the criteria.
**
** @param asset - The asset to apply criteria to
** @param criteria - List of criteria to apply
** @return []string - List of strings that uniquely identify the asset
** @return error - Any error that occurred during criteria application
**************************************************************************************************/
func applyCriteria(asset utils.TAsset, criteria []utils.TCriteria) ([]string, error) {
	extractors := map[string]func(asset utils.TAsset, c utils.TCriteria) (string, error){
		"id":               func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.ID, nil },
		"deviceAssetId":    func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceAssetID, nil },
		"deviceId":         func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.DeviceID, nil },
		"duration":         func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Duration, nil },
		"fileCreatedAt":    func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.FileCreatedAt, nil },
		"fileModifiedAt":   func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.FileModifiedAt, nil },
		"hasMetadata":      func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.HasMetadata), nil },
		"isArchived":       func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsArchived), nil },
		"isFavorite":       func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsFavorite), nil },
		"isOffline":        func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsOffline), nil },
		"isTrashed":        func(a utils.TAsset, _ utils.TCriteria) (string, error) { return boolToString(a.IsTrashed), nil },
		"localDateTime":    func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.LocalDateTime, nil },
		"originalFileName": extractOriginalFileName,
		"originalPath":     func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.OriginalPath, nil },
		"ownerId":          func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.OwnerID, nil },
		"type":             func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Type, nil },
		"updatedAt":        func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.UpdatedAt, nil },
		"checksum":         func(a utils.TAsset, _ utils.TCriteria) (string, error) { return a.Checksum, nil },
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
** extractOriginalFileName extracts the base name from the asset's original file name,
** discarding the extension first, then applying split logic if specified in the criteria.
**************************************************************************************************/
func extractOriginalFileName(asset utils.TAsset, c utils.TCriteria) (string, error) {
	baseName := asset.OriginalFileName
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

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
** boolToString converts a boolean value to its string representation ("true" or "false").
**************************************************************************************************/
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
