package immich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

// HTTP client configuration constants
const (
	defaultHTTPTimeout  = 600 * time.Second
	maxIdleConns        = 100
	maxIdleConnsPerHost = 100
	idleConnTimeout     = 90 * time.Second
	retryBaseDelay      = 500 * time.Millisecond
	maxRetries          = 3
)

/**************************************************************************************************
** Client represents an Immich API client with standard http package implementation.
** It handles all API interactions with the Immich server including authentication,
** request retries, and response handling.
**************************************************************************************************/
type Client struct {
	client                  *http.Client
	apiURL                  string
	apiKey                  string
	resetStacks             bool
	replaceStacks           bool
	dryRun                  bool
	withArchived            bool
	withDeleted             bool
	removeSingleAssetStacks bool
	filterAlbumIDs          []string
	filterTakenAfter        string
	filterTakenBefore       string
	logger                  *logrus.Logger
}

/**************************************************************************************************
** NewClient creates a new Immich client with standard http package.
** It configures the client with retry logic and proper headers.
**
** @param apiURL - Base URL of the Immich API
** @param apiKey - API key for authentication
** @param resetStacks - Whether to reset all existing stacks
** @param replaceStacks - Whether to replace existing stacks
** @param dryRun - Whether to perform a dry run without making changes
** @param withArchived - Whether to include archived assets
** @param withDeleted - Whether to include deleted assets
** @param removeSingleAssetStacks - Whether to remove stacks with only one asset
** @param filterAlbumIDs - Filter by album IDs (empty slice means no filter)
** @param filterTakenAfter - Filter assets taken after this date (empty means no filter)
** @param filterTakenBefore - Filter assets taken before this date (empty means no filter)
** @param logger - Logger instance for output
** @return *Client - Configured Immich client instance
**************************************************************************************************/
func NewClient(apiURL, apiKey string, resetStacks bool, replaceStacks bool, dryRun bool, withArchived bool, withDeleted bool, removeSingleAssetStacks bool, filterAlbumIDs []string, filterTakenAfter string, filterTakenBefore string, logger *logrus.Logger) *Client {
	if apiKey == "" {
		return nil
	}

	if apiURL == "" {
		return nil
	}

	if logger == nil {
		return nil
	}

	parsedURL, err := url.Parse(apiURL)
	if err != nil || parsedURL.Host == "" {
		return nil
	}

	baseURL := fmt.Sprintf("%s://%s/api", parsedURL.Scheme, parsedURL.Host)

	client := &http.Client{
		Timeout: defaultHTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     idleConnTimeout,
		},
	}

	return &Client{
		client:                  client,
		apiURL:                  baseURL,
		apiKey:                  apiKey,
		resetStacks:             resetStacks,
		replaceStacks:           replaceStacks,
		dryRun:                  dryRun,
		withArchived:            withArchived,
		withDeleted:             withDeleted,
		removeSingleAssetStacks: removeSingleAssetStacks,
		filterAlbumIDs:          filterAlbumIDs,
		filterTakenAfter:        filterTakenAfter,
		filterTakenBefore:       filterTakenBefore,
		logger:                  logger,
	}
}

/**************************************************************************************************
** doRequest handles the HTTP request with retry logic and proper error handling.
** It's a helper function to reduce code duplication across API calls.
**
** @param method - HTTP method (GET, POST, etc.)
** @param path - API endpoint path
** @param body - Request body (optional)
** @param result - Pointer to store response data
** @return error - Any error that occurred during the request
**************************************************************************************************/
func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.apiURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := c.client.Do(req)
		if err != nil {
			if i == maxRetries-1 {
				return fmt.Errorf("error making request after %d retries: %w", maxRetries, err)
			}
			time.Sleep(retryBaseDelay * time.Duration(i+1))
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil {
				if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
					return fmt.Errorf("error decoding response: %w", err)
				}
			}
			return nil
		}

		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response: %s - %s", resp.Status, string(body))
	}

	return fmt.Errorf("failed after %d retries", maxRetries)
}

/**************************************************************************************************
** FetchAllStacks retrieves all stacks from Immich and handles stack management.
** If resetStacks is true, it will delete all existing stacks.
** If removeSingleAssetStacks is true, single-asset stacks are automatically deleted.
**
** @return map[string]stacker.Stack - Map of stacks indexed by primary asset ID
** @return error - Any error that occurred during the fetch
**************************************************************************************************/
func (c *Client) FetchAllStacks() (map[string]utils.TStack, error) {
	var stacks []utils.TStack
	if err := c.doRequest(http.MethodGet, "/stacks", nil, &stacks); err != nil {
		return nil, fmt.Errorf("error fetching stacks: %w", err)
	}

	// Log info when starting reset stacks operation
	if c.resetStacks {
		if len(stacks) > 0 {
			c.logger.Infof("🔄 Starting reset stacks operation - will delete %d existing stacks", len(stacks))
		} else {
			c.logger.Infof("🔄 Reset stacks operation - no existing stacks to delete")
		}
	}

	// Handle single-asset stacks and reset if needed
	for _, stack := range stacks {
		if c.resetStacks {
			c.logger.Debugf("🔄 Resetting stack %s", stack.PrimaryAssetID)
			if err := c.DeleteStack(stack.ID, utils.REASON_RESET_STACK); err != nil {
				c.logger.Errorf("Error deleting stack: %v", err)
			}
		} else if c.removeSingleAssetStacks && len(stack.Assets) <= 1 {
			if err := c.DeleteStack(stack.ID, utils.REASON_DELETE_STACK_WITH_ONE_ASSET); err != nil {
				c.logger.Errorf("Error deleting stack: %v", err)
			}
		}
	}

	if c.resetStacks {
		if c.dryRun {
			return nil, nil
		}
		c.logger.Warnf(`⚠️ Done resetting stacks.`)
		c.resetStacks = false
		return map[string]utils.TStack{}, nil
	}

	// Log stack statistics only in debug mode
	if c.logger.IsLevelEnabled(logrus.DebugLevel) {
		stackCounts := make(map[int]int)
		for _, stack := range stacks {
			stackCounts[len(stack.Assets)]++
		}

		for count, num := range stackCounts {
			c.logger.Debugf("📚 %d assets in a stack with %d assets", num, count)
		}
	}

	stacksMap := make(map[string]utils.TStack)
	for _, stack := range stacks {
		for _, asset := range stack.Assets {
			stacksMap[asset.ID] = stack
		}
	}

	c.logger.Infof("📚 Fetched %d stacks", len(stacks))
	return stacksMap, nil
}

/**************************************************************************************************
** FetchAssets retrieves all assets from Immich with pagination support.
** Assets are enriched with their stack information if available.
**
** @param size - Number of assets per page
** @param stacksMap - Map of existing stacks for enrichment
** @return []stacker.Asset - List of all assets
** @return error - Any error that occurred during the fetch
**************************************************************************************************/
func (c *Client) FetchAssets(size int, stacksMap map[string]utils.TStack) ([]utils.TAsset, error) {
	// Resolve album filters (names to UUIDs) once
	resolvedAlbumIDs, err := c.resolveAlbumFilters(c.filterAlbumIDs)
	if err != nil {
		return nil, err
	}

	c.logger.Infof("⬇️  Fetching assets:")

	// If multiple albums, fetch each separately (OR logic) and deduplicate
	var albumFilters [][]string
	if len(resolvedAlbumIDs) > 1 {
		for _, albumID := range resolvedAlbumIDs {
			albumFilters = append(albumFilters, []string{albumID})
		}
	} else if len(resolvedAlbumIDs) == 1 {
		albumFilters = [][]string{resolvedAlbumIDs}
	} else {
		albumFilters = [][]string{nil} // No album filter
	}

	seen := make(map[string]bool)
	var allAssets []utils.TAsset

	for _, albumFilter := range albumFilters {
		page := 1
		for {
			c.logger.Debugf("Fetching page %d", page)
			var response utils.TSearchResponse

			payload := map[string]interface{}{
				"size":         size,
				"page":         page,
				"order":        "asc",
				"type":         "IMAGE",
				"isVisible":    true,
				"withStacked":  true,
				"withArchived": c.withArchived,
				"withDeleted":  c.withDeleted,
			}
			if len(albumFilter) > 0 {
				payload["albumIds"] = albumFilter
			}
			if c.filterTakenAfter != "" {
				payload["takenAfter"] = c.filterTakenAfter
			}
			if c.filterTakenBefore != "" {
				payload["takenBefore"] = c.filterTakenBefore
			}

			if err := c.doRequest(http.MethodPost, "/search/metadata", payload, &response); err != nil {
				c.logger.Errorf("Error fetching assets: %v", err)
				return nil, fmt.Errorf("error fetching assets: %w", err)
			}

			// Enrich assets with stack information and deduplicate
			for i := range response.Assets.Items {
				asset := &response.Assets.Items[i]
				if seen[asset.ID] {
					continue
				}
				seen[asset.ID] = true
				if stack, ok := stacksMap[asset.ID]; ok {
					asset.Stack = &stack
				}
				allAssets = append(allAssets, *asset)
			}

			// Handle string nextPage: empty string means no more pages
			if response.Assets.NextPage == "" || response.Assets.NextPage == "0" {
				break
			}
			nextPageInt, err := strconv.Atoi(response.Assets.NextPage)
			if err != nil || nextPageInt == 0 {
				break
			}
			page = nextPageInt
		}
	}

	c.logger.Infof("🌄 %d assets fetched", len(allAssets))
	return allAssets, nil
}

/**************************************************************************************************
** DeleteStack removes a stack from Immich.
** In dry run mode, it only logs the action without making changes.
**
** @param stackID - ID of the stack to delete
** @param reason - Reason for deletion (for logging)
** @return error - Any error that occurred during deletion
**************************************************************************************************/
func (c *Client) DeleteStack(stackID string, reason string) error {
	reasonMsg := ""
	if reason != utils.REASON_DELETE_STACK_WITH_ONE_ASSET {
		reasonMsg = "\t"
	}

	if c.dryRun {

		c.logger.Warnf("%sDeleted Stack %s (dry run) - %s", reasonMsg, stackID, reason)
		return nil
	}

	if err := c.doRequest(http.MethodDelete, fmt.Sprintf("/stacks/%s", stackID), nil, nil); err != nil {
		c.logger.Errorf("Error deleting stack: %v", err)
		return fmt.Errorf("error deleting stack: %w", err)
	}

	c.logger.Infof("%sDeleted Stack %s - %s", reasonMsg, stackID, reason)
	return nil
}

/**************************************************************************************************
** ModifyStack creates or updates a stack in Immich.
** In dry run mode, it only logs the action without making changes.
**
** @param assetIDs - Array of asset IDs to include in the stack
** @return error - Any error that occurred during modification
**************************************************************************************************/
func (c *Client) ModifyStack(assetIDs []string) error {
	if c.dryRun {
		return nil
	}

	if err := c.doRequest(http.MethodPost, "/stacks", map[string]interface{}{
		"assetIds": assetIDs,
	}, nil); err != nil {
		c.logger.Errorf("\t❌ Stack operation failed: %v", err)
		return fmt.Errorf("error modifying stack: %w", err)
	}

	c.logger.Debug("\t✅ API call successful")
	return nil
}

/**************************************************************************************************
** ListDuplicates finds and logs duplicate assets based on OriginalFileName and LocalDateTime.
** It groups assets by the combination of these fields and logs all groups with more than one
** asset, showing their IDs and file names for review.
**
** @param allAssets - List of assets to check for duplicates
** @return error - Any error that occurred during the check
**************************************************************************************************/
func (c *Client) ListDuplicates(allAssets []utils.TAsset) error {
	if len(allAssets) == 0 {
		c.logger.Warn("No assets provided for duplicate check.")
		return nil
	}

	// Map to group assets by OriginalFileName + LocalDateTime
	groups := make(map[string][]utils.TAsset)
	for _, asset := range allAssets {
		key := asset.OriginalFileName + "|" + asset.LocalDateTime
		groups[key] = append(groups[key], asset)
	}

	found := false
	for key, assets := range groups {
		if len(assets) > 1 {
			found = true
			c.logger.Warnf("Duplicate group: %s (%d assets)", key, len(assets))
			for _, asset := range assets {
				c.logger.Warnf("  - ID: %s, FileName: %s, LocalDateTime: %s", asset.ID, asset.OriginalFileName, asset.LocalDateTime)
			}
		}
	}

	if !found {
		c.logger.Info("No duplicates found based on OriginalFileName and LocalDateTime.")
	}
	return nil
}

/**************************************************************************************************
** GetCurrentUser fetches the current user info using the API key (GET /users/me).
** Returns the user as utils.TUserResponse or an error.
**************************************************************************************************/
func (c *Client) GetCurrentUser() (utils.TUserResponse, error) {
	var user utils.TUserResponse
	if err := c.doRequest(http.MethodGet, "/users/me", nil, &user); err != nil {
		c.logger.Errorf("Error fetching current user: %v", err)
		return user, fmt.Errorf("error fetching current user: %w", err)
	}
	return user, nil
}

/**************************************************************************************************
** FetchTrashedAssets retrieves only assets that are in the trash.
** This function specifically filters for assets where IsTrashed is true.
**
** @param size - Number of assets per page
** @return []utils.TAsset - List of trashed assets
** @return error - Any error that occurred during the fetch
**************************************************************************************************/
func (c *Client) FetchTrashedAssets(size int) ([]utils.TAsset, error) {
	var allTrashedAssets []utils.TAsset
	page := 1

	c.logger.Debugf("🗑️  Fetching trashed assets:")
	for {
		c.logger.Debugf("Fetching trashed assets page %d", page)
		var response utils.TSearchResponse
		if err := c.doRequest(http.MethodPost, "/search/metadata", map[string]interface{}{
			"size":         size,
			"page":         page,
			"order":        "asc",
			"type":         "IMAGE",
			"isVisible":    true,
			"withStacked":  true,
			"withArchived": false,
			"withDeleted":  true,
		}, &response); err != nil {
			c.logger.Errorf("Error fetching trashed assets: %v", err)
			return nil, fmt.Errorf("error fetching trashed assets: %w", err)
		}

		// Filter for only trashed assets
		for _, asset := range response.Assets.Items {
			if asset.IsTrashed {
				allTrashedAssets = append(allTrashedAssets, asset)
			}
		}

		// Handle string nextPage: empty string means no more pages
		if response.Assets.NextPage == "" || response.Assets.NextPage == "0" {
			break
		}
		nextPageInt, err := strconv.Atoi(response.Assets.NextPage)
		if err != nil || nextPageInt == 0 {
			break
		}
		page = nextPageInt
	}
	c.logger.Debugf("🗑️  %d trashed assets found", len(allTrashedAssets))

	return allTrashedAssets, nil
}

/**************************************************************************************************
** TrashAssets moves the specified assets to trash using the DELETE API with force=false.
** In dry run mode, it only logs the action without making changes.
**
** @param assetIDs - Array of asset IDs to move to trash
** @return error - Any error that occurred during the operation
**************************************************************************************************/
func (c *Client) TrashAssets(assetIDs []string) error {
	if len(assetIDs) == 0 {
		return nil
	}

	if c.dryRun {
		c.logger.Infof("🗑️  Moving %d assets to trash... (dry run)", len(assetIDs))
		for _, assetID := range assetIDs {
			c.logger.Debugf("\t- Asset ID: %s", assetID)
		}
		return nil
	}

	if err := c.doRequest(http.MethodDelete, "/assets", map[string]interface{}{
		"force": false,
		"ids":   assetIDs,
	}, nil); err != nil {
		c.logger.Errorf("Error moving assets to trash: %v", err)
		return fmt.Errorf("error moving assets to trash: %w", err)
	}

	c.logger.Infof("🗑️  Moving %d assets to trash... done", len(assetIDs))
	return nil
}

/**************************************************************************************************
** FetchAlbums fetches all albums for the authenticated user.
**
** @return []utils.TAlbum - List of albums
** @return error - Error if the request failed
**************************************************************************************************/
func (c *Client) FetchAlbums() ([]utils.TAlbum, error) {
	var albums []utils.TAlbum
	if err := c.doRequest(http.MethodGet, "/albums", nil, &albums); err != nil {
		return nil, fmt.Errorf("failed to fetch albums: %w", err)
	}
	return albums, nil
}

/**************************************************************************************************
** isUUID checks if a string is a valid UUID format.
**************************************************************************************************/
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

/**************************************************************************************************
** resolveAlbumFilters resolves album filters that may be names or UUIDs to actual UUIDs.
** If a filter value is already a UUID, it's used directly. Otherwise, it's treated as an
** album name and resolved by fetching albums from the API.
**
** @param filters - List of album IDs or names
** @return []string - List of resolved album UUIDs
** @return error - Error if album name resolution fails
**************************************************************************************************/
func (c *Client) resolveAlbumFilters(filters []string) ([]string, error) {
	if len(filters) == 0 {
		return nil, nil
	}

	var resolved []string
	var namesToResolve []string

	for _, filter := range filters {
		if isUUID(filter) {
			resolved = append(resolved, filter)
		} else {
			namesToResolve = append(namesToResolve, filter)
		}
	}

	if len(namesToResolve) == 0 {
		return resolved, nil
	}

	albums, err := c.FetchAlbums()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve album names: %w", err)
	}

	for _, name := range namesToResolve {
		found := false
		for _, album := range albums {
			if album.AlbumName == name {
				resolved = append(resolved, album.ID)
				found = true
				c.logger.Debugf("Resolved album name %q to ID %s", name, album.ID)
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("album not found: %q", name)
		}
	}

	return resolved, nil
}

/**************************************************************************************************
** FetchAlbumAssets fetches all assets in a specific album.
**
** @param albumID - Album identifier
** @return []utils.TAsset - List of assets in the album
** @return error - Error if the request failed
**************************************************************************************************/
func (c *Client) FetchAlbumAssets(albumID string) ([]utils.TAsset, error) {
	var response struct {
		Assets []utils.TAsset `json:"assets"`
	}
	if err := c.doRequest(http.MethodGet, fmt.Sprintf("/albums/%s", albumID), nil, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch album assets: %w", err)
	}
	return response.Assets, nil
}

/**************************************************************************************************
** CreateAlbum creates a new album with the given name and description.
**
** @param name - Album name
** @param description - Album description
** @return *utils.TAlbum - Created album
** @return error - Error if the request failed
**************************************************************************************************/
func (c *Client) CreateAlbum(name, description string) (*utils.TAlbum, error) {
	if c.dryRun {
		c.logger.Infof("[DRY RUN] Would create album: %s", name)
		return &utils.TAlbum{
			ID:          "dry-run-id",
			AlbumName:   name,
			Description: description,
		}, nil
	}

	payload := map[string]string{
		"albumName":   name,
		"description": description,
	}

	var album utils.TAlbum
	if err := c.doRequest(http.MethodPost, "/albums", payload, &album); err != nil {
		return nil, fmt.Errorf("failed to create album: %w", err)
	}

	return &album, nil
}

/**************************************************************************************************
** AddAssetsToAlbum adds assets to an album.
**
** @param albumID - Album identifier
** @param assetIDs - List of asset IDs to add
** @return error - Error if the request failed
**************************************************************************************************/
func (c *Client) AddAssetsToAlbum(albumID string, assetIDs []string) error {
	if len(assetIDs) == 0 {
		return nil
	}

	if c.dryRun {
		c.logger.Infof("[DRY RUN] Would add %d assets to album %s", len(assetIDs), albumID)
		return nil
	}

	payload := map[string]interface{}{
		"ids": assetIDs,
	}

	if err := c.doRequest(http.MethodPut, fmt.Sprintf("/albums/%s/assets", albumID), payload, nil); err != nil {
		return fmt.Errorf("failed to add assets to album: %w", err)
	}

	return nil
}

/**************************************************************************************************
** RemoveAssetsFromAlbum removes assets from an album.
**
** @param albumID - Album identifier
** @param assetIDs - List of asset IDs to remove
** @return error - Error if the request failed
**************************************************************************************************/
func (c *Client) RemoveAssetsFromAlbum(albumID string, assetIDs []string) error {
	if len(assetIDs) == 0 {
		return nil
	}

	if c.dryRun {
		c.logger.Infof("[DRY RUN] Would remove %d assets from album %s", len(assetIDs), albumID)
		return nil
	}

	payload := map[string]interface{}{
		"ids": assetIDs,
	}

	if err := c.doRequest(http.MethodDelete, fmt.Sprintf("/albums/%s/assets", albumID), payload, nil); err != nil {
		return fmt.Errorf("failed to remove assets from album: %w", err)
	}

	return nil
}

/**************************************************************************************************
** UpdateAlbum updates an album's properties (used for archiving).
**
** @param albumID - Album identifier
** @param updates - Map of properties to update
** @return error - Error if the request failed
**************************************************************************************************/
func (c *Client) UpdateAlbum(albumID string, updates map[string]interface{}) error {
	if c.dryRun {
		c.logger.Infof("[DRY RUN] Would update album %s", albumID)
		return nil
	}

	if err := c.doRequest(http.MethodPatch, fmt.Sprintf("/albums/%s", albumID), updates, nil); err != nil {
		return fmt.Errorf("failed to update album: %w", err)
	}

	return nil
}
