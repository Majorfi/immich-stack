package stacker

import "strings"

/**************************************************************************************************
** parsePromoteList parses a comma-separated list from an environment variable into a slice.
** Trims whitespace and ignores empty entries.
**************************************************************************************************/
func parsePromoteList(list string) []string {
	if list == "" {
		return nil
	}
	parts := strings.Split(list, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

/**************************************************************************************************
** getPromoteIndex returns the index of the first promote substring/extension found in the value.
** If none found, returns len(promoteList) (lowest priority).
**************************************************************************************************/
func getPromoteIndex(value string, promoteList []string) int {
	for idx, promote := range promoteList {
		if promote == "" {
			continue
		}
		if strings.Contains(value, promote) {
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
