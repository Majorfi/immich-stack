package stacker

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** AssetWithTime holds an asset with its parsed time for efficient sorting and comparison
**************************************************************************************************/
type AssetWithTime struct {
	Asset      utils.TAsset
	ParsedTime time.Time
}

/**************************************************************************************************
** mergeTimeBasedGroups performs a second pass to merge groups that should be together based
** on time proximity. This fixes the bucketing issue where photos within the delta but in
** different buckets aren't grouped together.
**
** The algorithm:
** 1. Identifies groups that have time-based criteria
** 2. For each set of groups with the same non-time criteria, checks if they should be merged
** 3. Merges groups where any photos are within the time delta of each other
**
** @param groups - The initial groups created by exact key matching
** @param criteria - The criteria used for grouping
** @return map[string][]utils.TAsset - The merged groups
**************************************************************************************************/
func mergeTimeBasedGroups(groups map[string][]utils.TAsset, criteria []utils.TCriteria) (map[string][]utils.TAsset, error) {
	var timeCriteriaIndices []int
	var timeDeltas []int
	hasTimeDelta := false

	for i, c := range criteria {
		if isTimeCriteria(c.Key) && c.Delta != nil && c.Delta.Milliseconds > 0 {
			timeCriteriaIndices = append(timeCriteriaIndices, i)
			timeDeltas = append(timeDeltas, c.Delta.Milliseconds)
			hasTimeDelta = true
		}
	}

	if !hasTimeDelta {
		return groups, nil
	}

	keysByNonTimeComponents := make(map[string][]string)
	for key := range groups {
		nonTimeKey := extractNonTimeComponents(key, criteria, timeCriteriaIndices)
		keysByNonTimeComponents[nonTimeKey] = append(keysByNonTimeComponents[nonTimeKey], key)
	}

	mergedGroups := make(map[string][]utils.TAsset)
	for nonTimeKey, keys := range keysByNonTimeComponents {
		if len(keys) == 1 {
			mergedGroups[keys[0]] = groups[keys[0]]
			continue
		}

		sort.Strings(keys)
		var allAssetsWithTime []AssetWithTime
		for _, key := range keys {
			for _, asset := range groups[key] {
				for _, idx := range timeCriteriaIndices {
					timeStr := getAssetTimeField(asset, criteria[idx].Key)
					if timeStr != "" {
						if parsedTime, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
							allAssetsWithTime = append(allAssetsWithTime, AssetWithTime{
								Asset:      asset,
								ParsedTime: parsedTime,
							})
							break
						}
					}
				}
			}
		}

		if len(allAssetsWithTime) == 0 {
			for _, key := range keys {
				mergedGroups[key] = groups[key]
			}
			continue
		}

		sort.Slice(allAssetsWithTime, func(i, j int) bool {
			return allAssetsWithTime[i].ParsedTime.Before(allAssetsWithTime[j].ParsedTime)
		})

		maxDelta := 0
		for _, delta := range timeDeltas {
			if delta > maxDelta {
				maxDelta = delta
			}
		}

		slidingGroups := performSlidingWindowGrouping(allAssetsWithTime, maxDelta)

		for i, group := range slidingGroups {
			if len(group) > 0 {
				newKey := fmt.Sprintf("%s|timegroup_%d", nonTimeKey, i)
				assets := make([]utils.TAsset, len(group))
				for j, awt := range group {
					assets[j] = awt.Asset
				}
				mergedGroups[newKey] = assets
			}
		}
	}

	return mergedGroups, nil
}

/**************************************************************************************************
** performSlidingWindowGrouping groups assets using a sliding window approach where assets
** are grouped if consecutive assets are within the time delta of each other.
**
** @param assets - Assets sorted by time
** @param deltaMs - Maximum time difference in milliseconds
** @return [][]AssetWithTime - Groups of assets
**************************************************************************************************/
func performSlidingWindowGrouping(assets []AssetWithTime, deltaMs int) [][]AssetWithTime {
	if len(assets) == 0 {
		return nil
	}

	var groups [][]AssetWithTime
	currentGroup := []AssetWithTime{assets[0]}

	for i := 1; i < len(assets); i++ {
		timeDiff := assets[i].ParsedTime.Sub(assets[i-1].ParsedTime)
		if timeDiff.Milliseconds() <= int64(deltaMs) {
			currentGroup = append(currentGroup, assets[i])
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []AssetWithTime{assets[i]}
		}
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

/**************************************************************************************************
** isTimeCriteria checks if a criteria key represents a time-based field
**************************************************************************************************/
func isTimeCriteria(key string) bool {
	switch key {
	case "fileCreatedAt", "fileModifiedAt", "localDateTime", "updatedAt":
		return true
	default:
		return false
	}
}

/**************************************************************************************************
** extractNonTimeComponents extracts the non-time parts of a group key
**************************************************************************************************/
func extractNonTimeComponents(key string, criteria []utils.TCriteria, timeCriteriaIndices []int) string {
	parts := strings.Split(key, "|")

	isTimeIndex := make(map[int]bool)
	for _, idx := range timeCriteriaIndices {
		isTimeIndex[idx] = true
	}

	var nonTimeParts []string
	for i, part := range parts {
		if i >= len(criteria) || !isTimeIndex[i] {
			nonTimeParts = append(nonTimeParts, part)
		}
	}

	if len(nonTimeParts) == 0 {
		return "notimekey"
	}

	return strings.Join(nonTimeParts, "|")
}

/**************************************************************************************************
** getAssetTimeField retrieves the time field value from an asset based on the criteria key
**************************************************************************************************/
func getAssetTimeField(asset utils.TAsset, key string) string {
	switch key {
	case "fileCreatedAt":
		return asset.FileCreatedAt
	case "fileModifiedAt":
		return asset.FileModifiedAt
	case "localDateTime":
		return asset.LocalDateTime
	case "updatedAt":
		return asset.UpdatedAt
	default:
		return ""
	}
}
