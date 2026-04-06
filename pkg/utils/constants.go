package utils

import "strings"

/**************************************************************************************************
** TimeFormat is the standard format for all time values in the application.
** It uses RFC3339Nano format to ensure consistent precision across all time operations.
**************************************************************************************************/
const TimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

// orOp is used to create a pointer to the "OR" operator string for TCriteriaExpression.
var orOp = "OR"

/**************************************************************************************************
** DefaultCriteria is kept for reference and legacy/custom criteria usage.
**************************************************************************************************/
var DefaultCriteria = []TCriteria{
	{
		Key: "originalFileName",
		Regex: &TRegex{
			Key:   `^(.+?)(?:_[a-z])?\.`,
			Index: 1,
		},
	},
	{
		Key: "localDateTime",
		Delta: &TDelta{
			Milliseconds: 5000,
		},
	},
}

/**************************************************************************************************
** DefaultCriteriaOR is the default OR-based criteria expression. Assets are grouped if they
** match EITHER:
**   1. The same base filename (stripping any _a–_z suffix before the extension), OR
**   2. Were captured within 5000ms of each other.
**
** This ensures front/back scans (e.g. photo.jpg + photo_a.jpg + photo_b.jpg) are grouped
** even when scanned at different times.
**************************************************************************************************/
var DefaultCriteriaOR = &TCriteriaExpression{
	Operator: &orOp,
	Children: []TCriteriaExpression{
		{Criteria: &TCriteria{
			Key:   "originalFileName",
			Regex: &TRegex{Key: `^(.+?)(?:_[a-z])?\.`, Index: 1},
		}},
		{Criteria: &TCriteria{
			Key:   "localDateTime",
			Delta: &TDelta{Milliseconds: 5000},
		}},
	},
}

/**************************************************************************************************
** DefaultParentFilenamePromote is the default parent filename promote for grouping photos.
** It promotes the filename of the original filename.
**************************************************************************************************/
var DefaultParentFilenamePromote = []string{"", "a", "b"}
var DefaultParentFilenamePromoteString = strings.Join(DefaultParentFilenamePromote, ",")

/**************************************************************************************************
** DefaultParentExtPromote is the default parent extension promote for grouping photos.
** It promotes the extension of the filename.
**************************************************************************************************/
var DefaultParentExtPromote = []string{".jpg", ".png", ".jpeg", ".heic", ".dng"}
var DefaultParentExtPromoteString = strings.Join(DefaultParentExtPromote, ",")

/**************************************************************************************************
** Reason messages
**************************************************************************************************/
var REASON_DELETE_STACK_WITH_ONE_ASSET = "deleting stack with only one asset"
var REASON_REPLACE_CHILD_STACK_WITH_NEW_ONE = "replacing child stack with new one"
var REASON_RESET_STACK = "resetting stack"
