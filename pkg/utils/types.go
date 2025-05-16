package utils

/**************************************************************************************************
** TDelta represents a time delta configuration for comparing time-based values.
** It allows for a buffer when comparing timestamps.
**************************************************************************************************/
type TDelta struct {
	Milliseconds int `json:"milliseconds"` // Number of milliseconds to allow as difference
}

/**************************************************************************************************
** TCriteria represents a single criterion for grouping photos. It defines how to extract
** and process values from assets for comparison and grouping.
**************************************************************************************************/
type TCriteria struct {
	Key   string  `json:"key"`             // Field name to extract from asset
	Split *TSplit `json:"split,omitempty"` // Optional split operation
	Regex *TRegex `json:"regex,omitempty"` // Optional regex operation
	Delta *TDelta `json:"delta,omitempty"` // Optional time delta for time-based fields
}

/**************************************************************************************************
** TSplit represents a split operation on a key value. It splits the value by a delimiter
** and selects a specific part by index.
**************************************************************************************************/
type TSplit struct {
	/**********************************************************************************************
	** Delimiters is a list of delimiters to split the string sequentially (e.g., ["~", "."]).
	** Index is the part to select after all splits.
	**********************************************************************************************/
	Delimiters []string `json:"delimiters"`
	Index      int      `json:"index"`
}

/**************************************************************************************************
** TRegex represents a regex operation on a key value. It applies a regular expression
** and selects a specific capture group by index.
**************************************************************************************************/
type TRegex struct {
	Key   string `json:"key"`   // Regular expression pattern
	Index int    `json:"index"` // Index of capture group to select
}

/**************************************************************************************************
** TAsset represents an Immich asset with all its metadata and properties.
** This structure matches the Immich API response format.
**************************************************************************************************/
type TAsset struct {
	ID               string  `json:"id"`               // Unique identifier
	DeviceAssetID    string  `json:"deviceAssetId"`    // Original device asset ID
	DeviceID         string  `json:"deviceId"`         // Device identifier
	OriginalFileName string  `json:"originalFileName"` // Original file name
	OriginalPath     string  `json:"originalPath"`     // Original file path
	LocalDateTime    string  `json:"localDateTime"`    // Local capture time
	FileCreatedAt    string  `json:"fileCreatedAt"`    // File creation time
	FileModifiedAt   string  `json:"fileModifiedAt"`   // File modification time
	HasMetadata      bool    `json:"hasMetadata"`      // Whether asset has metadata
	IsArchived       bool    `json:"isArchived"`       // Whether asset is archived
	IsFavorite       bool    `json:"isFavorite"`       // Whether asset is favorited
	IsOffline        bool    `json:"isOffline"`        // Whether asset is offline
	IsTrashed        bool    `json:"isTrashed"`        // Whether asset is trashed
	OwnerID          string  `json:"ownerId"`          // Owner identifier
	Type             string  `json:"type"`             // Asset type
	UpdatedAt        string  `json:"updatedAt"`        // Last update time
	Checksum         string  `json:"checksum"`         // File checksum
	Duration         string  `json:"duration"`         // Duration (for videos)
	Stack            *TStack `json:"stack,omitempty"`  // Associated stack if any
}

/**************************************************************************************************
** TStack represents an Immich stack as defined in the Immich OpenAPI spec (StackResponseDto).
** Contains a primary asset and all associated assets in the stack.
**************************************************************************************************/
type TStack struct {
	ID             string   `json:"id"`             // Stack identifier
	PrimaryAssetID string   `json:"primaryAssetId"` // Primary asset identifier
	Assets         []TAsset `json:"assets"`         // All assets in the stack
}

/**************************************************************************************************
** TSearchResponse represents the response from Immich search API.
** Contains paginated results and next page information.
**************************************************************************************************/
type TSearchResponse struct {
	Assets struct {
		Items    []TAsset `json:"items"`    // List of assets in current page
		NextPage string   `json:"nextPage"` // Next page token or empty if last page
	} `json:"assets"`
}

/**************************************************************************************************
** TUserResponse represents a user as returned by the Immich API (UserResponseDto).
** This structure matches the Immich API response format for /users/me.
**************************************************************************************************/
type TUserResponse struct {
	AvatarColor      string `json:"avatarColor"`
	Email            string `json:"email"`
	ID               string `json:"id"`
	Name             string `json:"name"`
	ProfileChangedAt string `json:"profileChangedAt"`
	ProfileImagePath string `json:"profileImagePath"`
}
