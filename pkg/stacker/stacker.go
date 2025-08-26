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

		promoteValue, hasValue := assetPromoteValues[c.Key]
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
	numericSuffixRegex := regexp.MustCompile(`^(\d+)$`)
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

	stackingCriteria, err := getCriteriaConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get criteria config: %w", err)
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

	// Debugging
	if logger.Level == logrus.DebugLevel {
		listOfCriteria := make([]string, len(stackingCriteria))
		for i, c := range stackingCriteria {
			listOfCriteria[i] = c.Key
		}
		logger.Debugf("Stacking assets with criteria: %s", listOfCriteria)
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

		if logger.Level == logrus.DebugLevel {
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

	return result, nil
}
