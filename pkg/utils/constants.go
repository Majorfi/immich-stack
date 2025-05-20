package utils

import "strings"

/**************************************************************************************************
** TimeFormat is the standard format for all time values in the application.
** It uses RFC3339Nano format to ensure consistent precision across all time operations.
**************************************************************************************************/
const TimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

/**************************************************************************************************
** DefaultCriteria is the default criteria for grouping photos. It groups photos by:
** 1. Original filename (before extension)
** 2. Local capture time (with no delta as default)
**************************************************************************************************/
var DefaultCriteria = []TCriteria{
	{
		Key: "originalFileName",
		Split: &TSplit{ // We want to split sequentially on "~" and then "."
			Delimiters: []string{"~", "."},
			Index:      0,
		},
	},
	{
		Key: "localDateTime",
		Delta: &TDelta{
			Milliseconds: 0, // No delta by default
		},
	},
}

/**************************************************************************************************
** DefaultParentFilenamePromote is the default parent filename promote for grouping photos.
** It promotes the filename of the original filename.
**************************************************************************************************/
var DefaultParentFilenamePromote = []string{"edit", "crop", "hdr", "biggestNumber"}
var DefaultParentFilenamePromoteString = strings.Join(DefaultParentFilenamePromote, ",")

/**************************************************************************************************
** DefaultParentExtPromote is the default parent extension promote for grouping photos.
** It promotes the extension of the filename.
**************************************************************************************************/
var DefaultParentExtPromote = []string{".jpg", ".png", ".jpeg", ".dng"}
var DefaultParentExtPromoteString = strings.Join(DefaultParentExtPromote, ",")

/**************************************************************************************************
** Reason messages
**************************************************************************************************/
var REASON_DELETE_STACK_WITH_ONE_ASSET = "deleting stack with only one asset"
var REASON_REPLACE_CHILD_STACK_WITH_NEW_ONE = "replacing child stack with new one"
var REASON_RESET_STACK = "resetting stack"
