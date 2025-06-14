package stacker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

/**************************************************************************************************
** parsePromoteList parses a comma-separated list from an environment variable into a slice.
** Trims whitespace but preserves empty strings for negative matching.
** Special keywords like "sequence" are preserved for special handling.
**************************************************************************************************/
func parsePromoteList(list string) []string {
	if list == "" {
		return nil
	}
	parts := strings.Split(list, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		// Preserve empty strings but trim whitespace from non-empty ones
		if p == "" {
			result = append(result, "")
		} else {
			trimmed := strings.TrimSpace(p)
			result = append(result, trimmed)
		}
	}
	return result
}

/**************************************************************************************************
** isSequenceKeyword checks if a promote string is a special sequence keyword.
** Supports formats: "sequence", "sequence:4", "sequence:prefix_", etc.
**************************************************************************************************/
func isSequenceKeyword(promote string) bool {
	return promote == "sequence" || strings.HasPrefix(promote, "sequence:")
}

/**************************************************************************************************
** extractSequencePattern extracts the pattern from a sequence keyword.
** Examples:
** - "sequence" returns ("", 0)
** - "sequence:4" returns ("", 4)
** - "sequence:IMG_" returns ("IMG_", 0)
**************************************************************************************************/
func extractSequencePattern(keyword string) (prefix string, digits int) {
	if keyword == "sequence" {
		return "", 0
	}

	if strings.HasPrefix(keyword, "sequence:") {
		pattern := strings.TrimPrefix(keyword, "sequence:")

		// Check if it's a digit count
		if n, err := strconv.Atoi(pattern); err == nil {
			return "", n
		}

		// Otherwise it's a prefix pattern
		return pattern, 0
	}

	return "", 0
}

/**************************************************************************************************
** getPromoteIndex returns the index of the first promote substring/extension found in the value.
** If none found, returns len(promoteList) (lowest priority).
** Special handling for empty string: acts as negative match for files without other substrings.
**************************************************************************************************/
func getPromoteIndex(value string, promoteList []string) int {
	// Check if we have an empty string in the promote list
	emptyStringIndex := -1
	hasNonEmptyStrings := false
	
	for idx, promote := range promoteList {
		if promote == "" && emptyStringIndex == -1 {
			emptyStringIndex = idx  // Only record the first empty string
		} else if promote != "" && promote != "biggestNumber" {
			hasNonEmptyStrings = true
		}
	}
	
	// Check for matches with non-empty strings
	for idx, promote := range promoteList {
		if promote == "" || promote == "biggestNumber" {
			continue
		}
		if strings.Contains(strings.ToLower(value), strings.ToLower(promote)) {
			return idx
		}
	}
	
	// If we have an empty string, handle it based on whether there are other non-empty strings
	if emptyStringIndex >= 0 {
		if !hasNonEmptyStrings {
			// If only empty string in promote list, it matches all files
			return emptyStringIndex
		}
		
		// If there are other non-empty strings, check for negative matching
		containsAnyPromote := false
		for _, promote := range promoteList {
			if promote != "" && promote != "biggestNumber" {
				if strings.Contains(strings.ToLower(value), strings.ToLower(promote)) {
					containsAnyPromote = true
					break
				}
			}
		}
		if !containsAnyPromote {
			return emptyStringIndex
		}
	}
	
	// If 'biggestNumber' is in the promote list, assign its index to unmatched files
	for idx, promote := range promoteList {
		if promote == "biggestNumber" {
			return idx
		}
	}
	return len(promoteList)
}

/**************************************************************************************************
** getExtensionRank returns a numeric rank for file extensions.
** Higher rank means higher priority.
**
** @param ext - File extension (with dot)
** @return int - Rank of the extension
**************************************************************************************************/
func getExtensionRank(ext string) int {
	switch ext {
	case ".jpeg":
		return 4
	case ".jpg":
		return 3
	case ".png":
		return 2
	default:
		return 1
	}
}

/**************************************************************************************************
** getPromoteIndexWithMode handles promote string matching with different modes.
** Instead of just using strings.Contains, it can match specific patterns in filenames.
**
** In "sequence" mode, it extracts numeric values from sequential patterns like
** "0000", "0001", "0002" or "img1", "img2", "img3" and uses the numeric value
** directly as the sort index, allowing unlimited sequence numbers.
**
** Special handling for empty string in promote list:
** - An empty string ("") acts as a negative match - files that don't contain any other
**   non-empty promote strings will match the empty string's position
** - Example: promoteList = ["", "_edited", "_crop"] means:
**   - Files without "_edited" or "_crop" get index 0 (highest priority)
**   - Files with "_edited" get index 1
**   - Files with "_crop" get index 2
**
** Special handling for "sequence" keyword in promote list:
** - Returns the position in promote list for non-sequence items
** - For "sequence" keyword, returns the index offset by the max non-sequence items
**   to ensure sequences come after explicit promotes
**
** @param value - The filename to check
** @param promoteList - List of promote strings to match
** @param matchMode - How to match: "contains" (default), "sequence", "mixed"
** @return int - Index of the matched promote string, or len(promoteList) if no match
**************************************************************************************************/
func getPromoteIndexWithMode(value string, promoteList []string, matchMode string) int {
	base := filepath.Base(value)

	// Check if we have an empty string in the promote list (for negative matching)
	emptyStringIndex := -1
	hasNonEmptyStrings := false
	
	for idx, promote := range promoteList {
		if promote == "" && emptyStringIndex == -1 {
			emptyStringIndex = idx  // Only record the first empty string
		} else if promote != "" && !isSequenceKeyword(promote) {
			hasNonEmptyStrings = true
		}
	}

	// First, check for exact matches with non-sequence items in the promote list
	for idx, promote := range promoteList {
		if promote == "" || isSequenceKeyword(promote) {
			continue
		}
		if strings.Contains(strings.ToLower(base), strings.ToLower(promote)) {
			return idx
		}
	}

	// If we have an empty string, handle it based on whether there are other non-empty strings
	if emptyStringIndex >= 0 {
		if !hasNonEmptyStrings {
			// If only empty string in promote list, it matches all files
			return emptyStringIndex
		}
		
		// If there are other non-empty strings, check for negative matching
		containsAnyPromote := false
		for _, promote := range promoteList {
			if promote != "" && !isSequenceKeyword(promote) {
				if strings.Contains(strings.ToLower(base), strings.ToLower(promote)) {
					containsAnyPromote = true
					break
				}
			}
		}
		
		// If it doesn't contain any promote strings, return the empty string's index
		if !containsAnyPromote {
			return emptyStringIndex
		}
	}

	// Check if we have a sequence keyword in the promote list
	sequenceIndex := -1
	var sequencePrefix string
	var sequenceDigits int

	for idx, promote := range promoteList {
		if isSequenceKeyword(promote) {
			sequenceIndex = idx
			sequencePrefix, sequenceDigits = extractSequencePattern(promote)
			break
		}
	}

	// If we have a sequence keyword, try to extract sequence number from filename
	if sequenceIndex >= 0 {
		// Try multiple strategies to find the sequence number

		// Strategy 1: Look for numbers after underscores (common in burst photos)
		parts := strings.Split(base, "_")
		for _, part := range parts {
			// If we have a specific prefix requirement, check it
			if sequencePrefix != "" && !strings.HasPrefix(part, sequencePrefix) {
				continue
			}

			// Extract the numeric portion
			numStr := part
			if sequencePrefix != "" {
				numStr = strings.TrimPrefix(part, sequencePrefix)
			}

			// Check if it matches digit requirements
			if sequenceDigits > 0 && len(numStr) != sequenceDigits {
				continue
			}

			// Try to parse as number
			if num, err := strconv.Atoi(numStr); err == nil {
				// Return the sequence index + the number
				// This ensures sequences come after explicit promotes
				return sequenceIndex + num
			}
		}

		// Strategy 2: Use regex to find numbers in the filename
		var numPattern string
		if sequenceDigits > 0 {
			numPattern = fmt.Sprintf(`\d{%d}`, sequenceDigits)
		} else {
			numPattern = `\d+`
		}

		if sequencePrefix != "" {
			// If we have a prefix requirement, only match filenames with that prefix
			if !strings.Contains(base, sequencePrefix) {
				// Return a high value to put non-matching files at the end
				return len(promoteList) + 10000
			}
			numPattern = regexp.QuoteMeta(sequencePrefix) + numPattern
		}

		re := regexp.MustCompile(numPattern)
		if matches := re.FindStringSubmatch(base); len(matches) > 0 {
			numStr := matches[0]
			if sequencePrefix != "" {
				numStr = strings.TrimPrefix(numStr, sequencePrefix)
			}

			if num, err := strconv.Atoi(numStr); err == nil {
				return sequenceIndex + num
			}
		}
	}

	// Handle backward compatibility with old sequence mode
	if matchMode == "sequence" {
		// Try to extract number from promote list pattern
		if len(promoteList) > 0 {
			patternRegex := regexp.MustCompile(`^(.*?)(\d+)(.*?)$`)
			firstMatch := patternRegex.FindStringSubmatch(promoteList[0])
			if len(firstMatch) == 4 {
				prefix := firstMatch[1]
				suffix := firstMatch[3]

				// Look for exact matches in promote list first
				for idx, promote := range promoteList {
					if strings.Contains(base, promote) {
						return idx
					}
				}

				// If no exact match, try to extract pattern
				if shouldUseSequenceMatching(base, promoteList) {
					// For burst photos, try to extract from specific position
					parts := strings.Split(base, "_")
					for _, part := range parts {
						if strings.HasPrefix(part, prefix) && strings.HasSuffix(part, suffix) {
							numStr := part[len(prefix):]
							if len(suffix) > 0 {
								numStr = numStr[:len(numStr)-len(suffix)]
							}

							if num, err := strconv.Atoi(numStr); err == nil {
								return num
							}
						}
					}

					// Try to find pattern anywhere in filename
					matches := patternRegex.FindAllStringSubmatch(base, -1)
					for _, match := range matches {
						if len(match) == 4 && match[1] == prefix && match[3] == suffix {
							if num, err := strconv.Atoi(match[2]); err == nil {
								return num
							}
						}
					}
				}
			}
		}
	}

	// If 'biggestNumber' is in the promote list, assign its index to unmatched files
	for idx, promote := range promoteList {
		if promote == "biggestNumber" {
			return idx
		}
	}

	return len(promoteList)
}

/**************************************************************************************************
** shouldUseSequenceMatching determines if we should use sequence-based matching
** for a filename by checking if the filename structure matches the sequence pattern.
**
** @param filename - The filename to check
** @param promoteList - The promote list to extract pattern from
** @return bool - True if sequence matching is appropriate for this filename
**************************************************************************************************/
func shouldUseSequenceMatching(filename string, promoteList []string) bool {
	if len(promoteList) == 0 {
		return false
	}

	// Extract the pattern structure from the promote list
	patternRegex := regexp.MustCompile(`^(.*?)(\d+)(.*?)$`)

	// Analyze the first item to understand the pattern
	firstMatch := patternRegex.FindStringSubmatch(promoteList[0])
	if len(firstMatch) != 4 {
		return false
	}

	prefix := firstMatch[1]
	numberLen := len(firstMatch[2])
	suffix := firstMatch[3]

	// Check if filename contains any exact matches from promote list
	for _, promote := range promoteList {
		if strings.Contains(filename, promote) {
			return true
		}
	}

	// Check if filename has similar structure (same prefix/suffix pattern)
	base := filepath.Base(filename)

	// If we have a prefix, check if it exists in the filename
	if prefix != "" && !strings.Contains(base, prefix) {
		return false
	}

	// If we have a suffix, check if it exists in the filename
	if suffix != "" && !strings.Contains(base, suffix) {
		return false
	}

	// Check if filename contains a number with similar length between prefix and suffix
	// This helps identify files that follow the pattern even with different numbers
	if prefix != "" || suffix != "" {
		// Build a pattern to find prefix+number+suffix
		escapedPrefix := regexp.QuoteMeta(prefix)
		escapedSuffix := regexp.QuoteMeta(suffix)
		// Look for numbers of any length (not just similar to promote list)
		// This allows handling files like 0999 when promote list only has 0000-0003
		numberPattern := `\d+`
		fullPattern := regexp.MustCompile(escapedPrefix + numberPattern + escapedSuffix)

		if fullPattern.MatchString(base) {
			return true
		}
	}

	// Special case: if promote list has no prefix/suffix (just numbers like "0000", "0001")
	// check if the filename contains these in a structured way (e.g., after underscore)
	if prefix == "" && suffix == "" && numberLen > 0 {
		// Look for the number pattern in common positions (after underscore, at start, etc)
		parts := strings.Split(base, "_")
		for _, part := range parts {
			if matched, _ := regexp.MatchString(fmt.Sprintf(`^\d{%d,}$`, numberLen), part); matched {
				return true
			}
		}
	}

	return false
}

// Compile regex once to avoid recompilation in loops
var fourDigitRegex = regexp.MustCompile(`^\d{4}$`)

/**************************************************************************************************
** detectPromoteMatchMode analyzes the promote list and filenames to determine
** the best matching mode to use.
**
** @param promoteList - List of promote strings
** @param sampleFilename - A sample filename from the stack
** @return string - The match mode to use ("sequence", "mixed", or "contains")
**************************************************************************************************/
func detectPromoteMatchMode(promoteList []string, sampleFilename string) string {
	// Check if promote list contains sequence keyword
	hasSequenceKeyword := false
	hasNonSequenceItems := false

	for _, promote := range promoteList {
		if isSequenceKeyword(promote) {
			hasSequenceKeyword = true
		} else if promote != "" && promote != "biggestNumber" {
			hasNonSequenceItems = true
		}
	}

	// If we have sequence keyword mixed with other items, use mixed mode
	if hasSequenceKeyword && hasNonSequenceItems {
		return "mixed"
	}

	// If only sequence keyword, return mixed (will be handled same way)
	if hasSequenceKeyword {
		return "mixed"
	}

	// Check if promote list represents a traditional sequence pattern
	if isSequencePattern(promoteList) && shouldUseSequenceMatching(sampleFilename, promoteList) {
		return "sequence"
	}

	return "contains"
}

/**************************************************************************************************
** isSequencePattern checks if the promote list represents a sequential pattern.
** It detects patterns like: 0000,0001,0002 or img1,img2,img3 or any pattern with
** a common prefix/suffix and incrementing numbers.
**
** @param promoteList - List of promote strings to analyze
** @return bool - True if it's a sequential pattern
**************************************************************************************************/
func isSequencePattern(promoteList []string) bool {
	if len(promoteList) < 2 {
		return false
	}

	// Try to extract number from each item
	type PatternInfo struct {
		prefix   string
		number   int
		suffix   string
		original string
	}

	patterns := make([]PatternInfo, 0, len(promoteList))

	// Regex to extract prefix, number, and suffix
	// Matches: (prefix)(number)(suffix)
	patternRegex := regexp.MustCompile(`^(.*?)(\d+)(.*?)$`)

	for _, item := range promoteList {
		if item == "biggestNumber" {
			continue
		}

		matches := patternRegex.FindStringSubmatch(item)
		if len(matches) != 4 {
			return false // Not a pattern with number
		}

		num, err := strconv.Atoi(matches[2])
		if err != nil {
			return false
		}

		patterns = append(patterns, PatternInfo{
			prefix:   matches[1],
			number:   num,
			suffix:   matches[3],
			original: item,
		})
	}

	if len(patterns) < 2 {
		return false
	}

	// Check if all items have the same prefix and suffix
	firstPrefix := patterns[0].prefix
	firstSuffix := patterns[0].suffix

	for i := 1; i < len(patterns); i++ {
		if patterns[i].prefix != firstPrefix || patterns[i].suffix != firstSuffix {
			return false
		}
	}

	// Check if numbers are sequential (allowing gaps)
	// Sort by number first
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].number < patterns[j].number
	})

	// Check if it's an ascending sequence
	for i := 1; i < len(patterns); i++ {
		if patterns[i].number <= patterns[i-1].number {
			return false
		}
	}

	return true
}
