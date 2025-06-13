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
	defaultHTTPTimeout      = 600 * time.Second
	maxIdleConns           = 100
	maxIdleConnsPerHost    = 100
	idleConnTimeout        = 90 * time.Second
	retryBaseDelay         = 500 * time.Millisecond
	maxRetries             = 3
)

/**************************************************************************************************
** Client represents an Immich API client with standard http package implementation.
** It handles all API interactions with the Immich server including authentication,
** request retries, and response handling.
**************************************************************************************************/
type Client struct {
	client        *http.Client
	apiURL        string
	apiKey        string
	resetStacks   bool
	replaceStacks bool
	dryRun        bool
	withArchived  bool
	withDeleted   bool
	logger        *logrus.Logger
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
** @param logger - Logger instance for output
** @return *Client - Configured Immich client instance
**************************************************************************************************/
func NewClient(apiURL, apiKey string, resetStacks bool, replaceStacks bool, dryRun bool, withArchived bool, withDeleted bool, logger *logrus.Logger) *Client {
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
		client:        client,
		apiURL:        baseURL,
		apiKey:        apiKey,
		resetStacks:   resetStacks,
		replaceStacks: replaceStacks,
		dryRun:        dryRun,
		withArchived:  withArchived,
		withDeleted:   withDeleted,
		logger:        logger,
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
** Single-asset stacks are automatically deleted.
**
** @return map[string]stacker.Stack - Map of stacks indexed by primary asset ID
** @return error - Any error that occurred during the fetch
**************************************************************************************************/
func (c *Client) FetchAllStacks(shouldRemoveSingleAssetStacks bool) (map[string]utils.TStack, error) {
	var stacks []utils.TStack
	if err := c.doRequest(http.MethodGet, "/stacks", nil, &stacks); err != nil {
		return nil, fmt.Errorf("error fetching stacks: %w", err)
	}

	// Handle single-asset stacks and reset if needed
	for _, stack := range stacks {
		if c.resetStacks {
			c.logger.Infof("🔄 Resetting stack %s", stack.PrimaryAssetID)
			if err := c.DeleteStack(stack.ID, utils.REASON_RESET_STACK); err != nil {
				c.logger.Errorf("Error deleting stack: %v", err)
			}
		} else if shouldRemoveSingleAssetStacks && len(stack.Assets) <= 1 {
			if err := c.DeleteStack(stack.ID, utils.REASON_DELETE_STACK_WITH_ONE_ASSET); err != nil {
				c.logger.Errorf("Error deleting stack: %v", err)
			}
		}
	}

	if c.resetStacks {
		if c.dryRun {
			return nil, nil
		}
		c.logger.Warningf(`⚠️ Done resetting stacks.`)
		c.resetStacks = false
		return map[string]utils.TStack{}, nil
	}

	// Log stack statistics
	stackCounts := make(map[int]int)
	for _, stack := range stacks {
		stackCounts[len(stack.Assets)]++
	}

	for count, num := range stackCounts {
		c.logger.Infof("📚 %d assets in a stack with %d assets", num, count)
	}

	// Create lookup map
	stacksMap := make(map[string]utils.TStack)
	for _, stack := range stacks {
		stacksMap[stack.PrimaryAssetID] = stack
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
	var allAssets []utils.TAsset
	page := 1

	c.logger.Infof("⬇️  Fetching assets:")
	for {
		c.logger.Debugf("Fetching page %d", page)
		var response utils.TSearchResponse
		if err := c.doRequest(http.MethodPost, "/search/metadata", map[string]interface{}{
			"size":         size,
			"page":         page,
			"order":        "asc",
			"type":         "IMAGE",
			"isVisible":    true,
			"withStacked":  true,
			"withArchived": c.withArchived,
			"withDeleted":  c.withDeleted,
		}, &response); err != nil {
			c.logger.Errorf("Error fetching assets: %v", err)
			return nil, fmt.Errorf("error fetching assets: %w", err)
		}

		// Enrich assets with stack information
		for i := range response.Assets.Items {
			asset := &response.Assets.Items[i]
			if stack, ok := stacksMap[asset.ID]; ok {
				asset.Stack = &stack
			}
		}

		allAssets = append(allAssets, response.Assets.Items...)

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
		c.logger.Infof("\t🟢 Success! Stack created (dry run)")
		return nil
	}

	if err := c.doRequest(http.MethodPost, "/stacks", map[string]interface{}{
		"assetIds": assetIDs,
	}, nil); err != nil {
		c.logger.Errorf("Error modifying stack: %v", err)
		return fmt.Errorf("error modifying stack: %w", err)
	}

	c.logger.Info("\t🟢 Success! Stack created")
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

	c.logger.Infof("🗑️  Fetching trashed assets:")
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
	c.logger.Infof("🗑️  %d trashed assets found", len(allTrashedAssets))

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
		c.logger.Infof("🗑️  Would move %d assets to trash (dry run)", len(assetIDs))
		for _, assetID := range assetIDs {
			c.logger.Infof("\t- Asset ID: %s", assetID)
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

	c.logger.Infof("🗑️  Successfully moved %d assets to trash", len(assetIDs))
	return nil
}
