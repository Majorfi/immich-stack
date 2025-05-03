package utils

import "strings"

/**************************************************************************************************
** DefaultCriteria is the default criteria for grouping photos. It groups photos by:
** 1. Original filename (before extension)
** 2. Local capture time
**************************************************************************************************/
var DefaultCriteria = []TCriteria{
	{
		Key: "originalFileName",
		Split: &TSplit{ // We only want the first part, so we want to avoid all stuff like img.edit.jpg
			Key:   ".",
			Index: 0,
		},
	},
	{
		Key: "localDateTime",
	},
}

/**************************************************************************************************
** DefaultParentFilenamePromote is the default parent filename promote for grouping photos.
** It promotes the filename of the original filename.
**************************************************************************************************/
var DefaultParentFilenamePromote = []string{"edit", "crop", "hdr"}
var DefaultParentFilenamePromoteString = strings.Join(DefaultParentFilenamePromote, ",")

/**************************************************************************************************
** DefaultParentExtPromote is the default parent extension promote for grouping photos.
** It promotes the extension of the filename.
**************************************************************************************************/
var DefaultParentExtPromote = []string{".jpg", ".png", ".jpeg", ".dng"}
var DefaultParentExtPromoteString = strings.Join(DefaultParentExtPromote, ",")
