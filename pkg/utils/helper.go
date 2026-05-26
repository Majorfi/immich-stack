package utils

import (
	"path/filepath"
)

/**************************************************************************************************
** AreArraysEqual checks if two string arrays contain the same elements, regardless of their order.
** Uses frequency counting to ensure elements appear the same number of times in both arrays.
**
** @param arr1 - First array to compare
** @param arr2 - Second array to compare
** @return bool - True if arrays contain the same elements with same frequencies
**************************************************************************************************/
func AreArraysEqual(arr1, arr2 []string) bool {
	/******************************************************************************************
	** If lengths are different, arrays can't be equal
	******************************************************************************************/
	if len(arr1) != len(arr2) {
		return false
	}

	/******************************************************************************************
	** Create maps to count frequency of each element
	******************************************************************************************/
	freq1 := make(map[string]int)
	freq2 := make(map[string]int)

	/******************************************************************************************
	** Count frequency of elements in first array
	******************************************************************************************/
	for _, item := range arr1 {
		freq1[item]++
	}

	/******************************************************************************************
	** Count frequency of elements in second array
	******************************************************************************************/
	for _, item := range arr2 {
		freq2[item]++
	}

	/******************************************************************************************
	** Compare the two frequency maps
	******************************************************************************************/
	for item, count := range freq1 {
		if freq2[item] != count {
			return false
		}
	}

	/******************************************************************************************
	** Check if freq2 has any elements not in freq1
	******************************************************************************************/
	for item, count := range freq2 {
		if freq1[item] != count {
			return false
		}
	}

	return true
}

/**************************************************************************************************
** RemoveEmptyStrings removes all empty strings from a string array and returns a new array
** without the empty strings. Preserves the order of non-empty strings.
**
** @param arr - Array to process
** @return []string - New array containing only non-empty strings
**************************************************************************************************/
func RemoveEmptyStrings(arr []string) []string {
	result := make([]string, 0, len(arr))

	for _, str := range arr {
		if str != "" {
			result = append(result, str)
		}
	}

	return result
}

/**************************************************************************************************
** Contains checks if a string is present in a slice of strings.
**
** @param list - Slice of strings to search
** @param s - String to search for
** @return bool - True if string is present in slice, false otherwise
**************************************************************************************************/
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

/**************************************************************************************************
** BoolToString converts a boolean value to its string representation. It returns "true"
** for a true input and "false" for a false input.
**
** @param b - The boolean value to convert
** @return string - The string "true" or "false"
**************************************************************************************************/
func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

/**************************************************************************************************
** GetDir extracts the directory path from a file path. Returns the parent directory
** of the given file path.
**
** @param filePath - The full file path
** @return string - The directory path
**************************************************************************************************/
func GetDir(filePath string) string {
	return filepath.Dir(filePath)
}

/**************************************************************************************************
** FilterAssetsByOwner returns the assets whose OwnerID matches the given ownerID. Used to
** exclude partner-shared assets surfaced by /search/metadata when "Show in timeline" is
** enabled on an incoming partner share: those assets are visible to the current user but
** cannot be modified via the Immich stack API (permission denied), so including them in
** stacking attempts wastes API calls and pollutes logs.
**
** Return-value contract:
**   - When ownerID is non-empty (the normal case), a NEW slice is returned containing only
**     the matching assets, in their original order. The input slice is not modified.
**   - When ownerID is empty (defensive default for misconfigured callers), the input slice
**     itself is returned unchanged — no copy is made. Callers that intend to mutate the
**     result must handle this case explicitly.
**
** @param assets - Input assets, typically the output of FetchAssets
** @param ownerID - The current user's UUID (from GetCurrentUser)
** @return []TAsset - Either a new filtered slice, or the input slice when ownerID is empty
**************************************************************************************************/
func FilterAssetsByOwner(assets []TAsset, ownerID string) []TAsset {
	if ownerID == "" {
		return assets
	}
	out := make([]TAsset, 0, len(assets))
	for _, a := range assets {
		if a.OwnerID == ownerID {
			out = append(out, a)
		}
	}
	return out
}
