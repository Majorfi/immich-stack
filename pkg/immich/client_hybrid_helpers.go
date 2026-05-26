package immich

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** getAssetWithRetry fetches a single asset via GET /assets/{id} with retries on transient
** upstream errors (502/503/504). Unlike /search/metadata, this endpoint returns the full
** asset including its stack reference (when applicable).
**************************************************************************************************/
func (c *Client) getAssetWithRetry(assetID string, attempts int) (utils.TAsset, error) {
	var asset utils.TAsset
	err := c.doRequestWithUpstreamRetry(http.MethodGet, "/assets/"+assetID, nil, &asset, attempts)
	return asset, err
}

/**************************************************************************************************
** getStackByPrimaryWithRetry calls GET /stacks?primaryAssetId=X with retries on transient
** upstream errors. Returns the stack list (empty if the asset is not a primary).
**************************************************************************************************/
func (c *Client) getStackByPrimaryWithRetry(assetID string, attempts int) ([]utils.TStack, error) {
	var stacks []utils.TStack
	err := c.doRequestWithUpstreamRetry(http.MethodGet, "/stacks?primaryAssetId="+assetID, nil, &stacks, attempts)
	return stacks, err
}

/**************************************************************************************************
** isTransientUpstreamError returns true for nginx upstream errors (502/503/504) that are
** typically caused by Immich being slow to respond under load. These are worth retrying.
** Other errors (4xx, 500, transport errors) propagate immediately.
**************************************************************************************************/
func isTransientUpstreamError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusBadGateway ||
		apiErr.StatusCode == http.StatusServiceUnavailable ||
		apiErr.StatusCode == http.StatusGatewayTimeout
}

/**************************************************************************************************
** doRequestWithUpstreamRetry wraps doRequest with N attempts on transient upstream errors
** (502/503/504). The base doRequest only retries transport errors, so upstream nginx hiccups
** under high concurrency would otherwise propagate as hard failures. Used by all hybrid-path
** callers (per-asset lookup, per-primary lookup, search pagination).
**************************************************************************************************/
func (c *Client) doRequestWithUpstreamRetry(method, path string, body interface{}, result interface{}, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		err := c.doRequest(method, path, body, result)
		if err == nil {
			return nil
		}
		if !isTransientUpstreamError(err) {
			return err
		}
		lastErr = err
		time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
	}
	return lastErr
}

/**************************************************************************************************
** fetchAllAssetIDsViaSearch enumerates all asset IDs across all pages of /search/metadata.
** Older Immich versions ignore withArchived on this endpoint, so we run two separate searches:
** one default (timeline assets) and one with visibility=archive (archived assets). Results
** are deduplicated.
**************************************************************************************************/
func (c *Client) fetchAllAssetIDsViaSearch() ([]string, error) {
	seen := make(map[string]bool)
	var ids []string

	visibilityFilters := []interface{}{nil, "archive"}
	for _, assetType := range c.assetTypesForSearch() {
		for _, vis := range visibilityFilters {
			page := 1
			const pageSize = 1000
			for {
				var response utils.TSearchResponse
				payload := map[string]interface{}{
					"size":        pageSize,
					"page":        page,
					"order":       "asc",
					"type":        assetType,
					"withStacked": true,
					"withDeleted": true,
				}
				if vis != nil {
					payload["visibility"] = vis
				}
				if err := c.doRequestWithUpstreamRetry(http.MethodPost, "/search/metadata", payload, &response, 4); err != nil {
					return nil, fmt.Errorf("error fetching asset IDs (type=%s, visibility=%v): %w", assetType, vis, err)
				}
				for _, asset := range response.Assets.Items {
					if seen[asset.ID] {
						continue
					}
					seen[asset.ID] = true
					ids = append(ids, asset.ID)
				}
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
	}
	return ids, nil
}
