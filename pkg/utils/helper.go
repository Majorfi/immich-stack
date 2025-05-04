package utils

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
