package immich

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Test helper functions and types
************************************************************************************************/

func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		apiURL        string
		resetStacks   bool
		replaceStacks bool
		dryRun        bool
		wantErr       bool
	}{
		{
			name:          "valid config",
			apiKey:        "test-key",
			apiURL:        "http://test.com",
			resetStacks:   false,
			replaceStacks: false,
			dryRun:        false,
			wantErr:       false,
		},
		{
			name:          "missing api key",
			apiKey:        "",
			apiURL:        "http://test.com",
			resetStacks:   false,
			replaceStacks: false,
			dryRun:        false,
			wantErr:       true,
		},
		{
			name:          "missing api url",
			apiKey:        "test-key",
			apiURL:        "",
			resetStacks:   false,
			replaceStacks: false,
			dryRun:        false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			client := NewClient(tt.apiURL, tt.apiKey, tt.resetStacks, tt.replaceStacks, tt.dryRun, true, false, false, nil, "", "", logrus.New())

			// Assert
			if tt.wantErr {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
				assert.Equal(t, tt.apiKey, client.apiKey)
				assert.Equal(t, tt.apiURL+"/api", client.apiURL)
			}
		})
	}
}

func TestFetchAssets(t *testing.T) {
	tests := []struct {
		name      string
		client    *Client
		size      int
		stacksMap map[string]utils.TStack
		wantErr   bool
	}{
		{
			name: "invalid client",
			client: &Client{
				apiKey: "invalid",
				apiURL: "invalid",
				logger: logrus.New(),
				client: &http.Client{},
			},
			size:      10,
			stacksMap: make(map[string]utils.TStack),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			assets, err := tt.client.FetchAssets(tt.size, tt.stacksMap)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, assets)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, assets)
			}
		})
	}
}

type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// mockTransportSeq allows returning different responses for sequential requests
type mockTransportSeq struct {
	responses []*http.Response
	errors    []error
	index     int
}

func (m *mockTransportSeq) RoundTrip(*http.Request) (*http.Response, error) {
	if m.index >= len(m.responses) {
		// Return last response if we've exhausted the list
		idx := len(m.responses) - 1
		if idx >= 0 && m.errors != nil && idx < len(m.errors) && m.errors[idx] != nil {
			return nil, m.errors[idx]
		}
		if idx >= 0 {
			return m.responses[idx], nil
		}
		return nil, nil
	}
	resp := m.responses[m.index]
	var err error
	if m.errors != nil && m.index < len(m.errors) {
		err = m.errors[m.index]
	}
	m.index++
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func TestFetchAllStacks(t *testing.T) {
	tests := []struct {
		name           string
		stacksResponse string
		expectedMap    map[string]string // assetID -> stackID mapping
		wantErr        bool
	}{
		{
			name: "stacks indexed by all asset IDs not just primary",
			stacksResponse: `[
				{
					"id": "stack-123",
					"primaryAssetId": "asset-a",
					"assets": [
						{"id": "asset-a"},
						{"id": "asset-b"},
						{"id": "asset-c"}
					]
				},
				{
					"id": "stack-456",
					"primaryAssetId": "asset-x",
					"assets": [
						{"id": "asset-x"},
						{"id": "asset-y"}
					]
				}
			]`,
			expectedMap: map[string]string{
				"asset-a": "stack-123",
				"asset-b": "stack-123",
				"asset-c": "stack-123",
				"asset-x": "stack-456",
				"asset-y": "stack-456",
			},
			wantErr: false,
		},
		{
			name: "empty stacks",
			stacksResponse: `[]`,
			expectedMap:    map[string]string{},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logrus.New(),
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(tt.stacksResponse)),
						},
					},
				},
			}

			// Act
			stacksMap, err := client.FetchAllStacks()

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, stacksMap)

				// Verify that ALL assets (primary and children) are indexed
				for assetID, expectedStackID := range tt.expectedMap {
					stack, exists := stacksMap[assetID]
					assert.True(t, exists, "Asset %s should be in stacksMap", assetID)
					if exists {
						assert.Equal(t, expectedStackID, stack.ID,
							"Asset %s should map to stack %s", assetID, expectedStackID)
					}
				}

				// Verify map size matches expected
				assert.Equal(t, len(tt.expectedMap), len(stacksMap),
					"stacksMap should contain entries for all assets in all stacks")
			}
		})
	}
}

func TestModifyStack(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		assets  []string
		wantErr bool
	}{
		{
			name: "empty assets",
			client: &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logrus.New(),
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			},
			assets:  []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := tt.client.ModifyStack(tt.assets)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/************************************************************************************************
** Tests for UUID validation function
************************************************************************************************/

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid UUIDs
		{"valid lowercase", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"valid mixed case", "550e8400-E29B-41d4-A716-446655440000", true},

		// Invalid UUIDs
		{"empty string", "", false},
		{"too short", "550e8400-e29b", false},
		{"too long", "550e8400-e29b-41d4-a716-446655440000-extra", false},
		{"invalid hex char G", "550e8400-e29b-41d4-a716-44665544000G", false},
		{"invalid hex char Z", "550e8400-e29b-41d4-a716-44665544000Z", false},
		{"missing dashes", "550e8400e29b41d4a716446655440000", false},
		{"wrong dash position", "550e840-0e29b-41d4-a716-446655440000", false},
		{"dash in wrong place", "550e84000e29b-41d4-a716-44665544000", false},
		{"album name", "My Vacation Photos", false},
		{"spaces instead of dashes", "550e8400 e29b 41d4 a716 446655440000", false},
		{"partial uuid", "550e8400-e29b-41d4", false},
		{"just dashes", "------------------------------------", false},
		{"numbers only wrong length", "12345678901234567890123456789012345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUUID(tt.input)
			assert.Equal(t, tt.expected, result, "isUUID(%q) should return %v", tt.input, tt.expected)
		})
	}
}

/************************************************************************************************
** Tests for album filter resolution
************************************************************************************************/

func TestResolveAlbumFilters(t *testing.T) {
	tests := []struct {
		name           string
		filters        []string
		albumsResponse string
		expected       []string
		wantErr        bool
		errContains    string
	}{
		{
			name:     "empty filters",
			filters:  []string{},
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "nil filters",
			filters:  nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "single UUID passthrough",
			filters:  []string{"550e8400-e29b-41d4-a716-446655440000"},
			expected: []string{"550e8400-e29b-41d4-a716-446655440000"},
			wantErr:  false,
		},
		{
			name:     "multiple UUIDs passthrough",
			filters:  []string{"550e8400-e29b-41d4-a716-446655440000", "660e8400-e29b-41d4-a716-446655440001"},
			expected: []string{"550e8400-e29b-41d4-a716-446655440000", "660e8400-e29b-41d4-a716-446655440001"},
			wantErr:  false,
		},
		{
			name:    "single name resolved",
			filters: []string{"Vacation"},
			albumsResponse: `[
				{"id": "album-uuid-vacation", "albumName": "Vacation"},
				{"id": "album-uuid-work", "albumName": "Work"}
			]`,
			expected: []string{"album-uuid-vacation"},
			wantErr:  false,
		},
		{
			name:    "mixed UUID and name",
			filters: []string{"550e8400-e29b-41d4-a716-446655440000", "Vacation"},
			albumsResponse: `[
				{"id": "album-uuid-vacation", "albumName": "Vacation"}
			]`,
			expected: []string{"550e8400-e29b-41d4-a716-446655440000", "album-uuid-vacation"},
			wantErr:  false,
		},
		{
			name:    "album not found",
			filters: []string{"NonExistent"},
			albumsResponse: `[
				{"id": "album-uuid-1", "albumName": "Vacation"}
			]`,
			wantErr:     true,
			errContains: "album not found",
		},
		{
			name:    "case sensitive - lowercase not found",
			filters: []string{"vacation"},
			albumsResponse: `[
				{"id": "album-uuid-1", "albumName": "Vacation"}
			]`,
			wantErr:     true,
			errContains: "album not found",
		},
		{
			name:    "multiple names resolved",
			filters: []string{"Vacation", "Work"},
			albumsResponse: `[
				{"id": "album-uuid-vacation", "albumName": "Vacation"},
				{"id": "album-uuid-work", "albumName": "Work"}
			]`,
			expected: []string{"album-uuid-vacation", "album-uuid-work"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with mock transport
			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logrus.New(),
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(tt.albumsResponse)),
						},
					},
				},
			}

			// Act
			result, err := client.resolveAlbumFilters(tt.filters)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

/************************************************************************************************
** Tests for date validation in FetchAssets
************************************************************************************************/

func TestFetchAssetsDateValidation(t *testing.T) {
	tests := []struct {
		name        string
		takenAfter  string
		takenBefore string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty dates - no validation",
			takenAfter:  "",
			takenBefore: "",
			wantErr:     false,
		},
		{
			name:        "valid takenAfter only",
			takenAfter:  "2024-01-01T00:00:00Z",
			takenBefore: "",
			wantErr:     false,
		},
		{
			name:        "valid takenBefore only",
			takenAfter:  "",
			takenBefore: "2024-12-31T23:59:59Z",
			wantErr:     false,
		},
		{
			name:        "both valid dates",
			takenAfter:  "2024-01-01T00:00:00Z",
			takenBefore: "2024-12-31T23:59:59Z",
			wantErr:     false,
		},
		{
			name:        "valid date with timezone offset",
			takenAfter:  "2024-01-01T00:00:00+05:30",
			takenBefore: "",
			wantErr:     false,
		},
		{
			name:        "invalid takenAfter - date only",
			takenAfter:  "2024-01-01",
			takenBefore: "",
			wantErr:     true,
			errContains: "invalid takenAfter date format",
		},
		{
			name:        "invalid takenBefore - human readable",
			takenAfter:  "",
			takenBefore: "Jan 1, 2024",
			wantErr:     true,
			errContains: "invalid takenBefore date format",
		},
		{
			name:        "invalid takenAfter - random string",
			takenAfter:  "not-a-date",
			takenBefore: "",
			wantErr:     true,
			errContains: "invalid takenAfter date format",
		},
		{
			name:        "invalid takenBefore - unix timestamp",
			takenAfter:  "",
			takenBefore: "1704067200",
			wantErr:     true,
			errContains: "invalid takenBefore date format",
		},
		{
			name:        "invalid takenAfter - missing timezone",
			takenAfter:  "2024-01-01T00:00:00",
			takenBefore: "",
			wantErr:     true,
			errContains: "invalid takenAfter date format",
		},
		{
			name:        "inverted dates - takenAfter after takenBefore",
			takenAfter:  "2024-12-31T23:59:59Z",
			takenBefore: "2024-01-01T00:00:00Z",
			wantErr:     true,
			errContains: "takenAfter",
		},
		{
			name:        "same date - takenAfter equals takenBefore",
			takenAfter:  "2024-06-15T12:00:00Z",
			takenBefore: "2024-06-15T12:00:00Z",
			wantErr:     true,
			errContains: "takenAfter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with mock transport that returns empty assets
			client := &Client{
				apiKey:            "test",
				apiURL:            "http://test/api",
				logger:            logrus.New(),
				filterTakenAfter:  tt.takenAfter,
				filterTakenBefore: tt.takenBefore,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{"assets": {"items": [], "nextPage": ""}}`)),
						},
					},
				},
			}

			// Act
			_, err := client.FetchAssets(10, make(map[string]utils.TStack))

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/************************************************************************************************
** Tests for FetchAssets album filter building and deduplication
************************************************************************************************/

func TestFetchAssetsWithAlbumFilters(t *testing.T) {
	// Standard assets response
	assetsResponse := `{"assets": {"items": [
		{"id": "asset-1", "originalFileName": "photo1.jpg"},
		{"id": "asset-2", "originalFileName": "photo2.jpg"}
	], "nextPage": ""}}`

	tests := []struct {
		name           string
		filterAlbumIDs []string
		responses      []string
		expectedCount  int
	}{
		{
			name:           "no album filter",
			filterAlbumIDs: nil,
			responses:      []string{assetsResponse},
			expectedCount:  2,
		},
		{
			name:           "single album filter",
			filterAlbumIDs: []string{"550e8400-e29b-41d4-a716-446655440000"},
			responses:      []string{assetsResponse},
			expectedCount:  2,
		},
		{
			name:           "multiple album filters - same assets deduped",
			filterAlbumIDs: []string{"550e8400-e29b-41d4-a716-446655440000", "660e8400-e29b-41d4-a716-446655440001"},
			responses:      []string{assetsResponse, assetsResponse}, // Both albums return same assets
			expectedCount:  2,                                        // Should be deduped to 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build responses slice
			var httpResponses []*http.Response
			for _, resp := range tt.responses {
				httpResponses = append(httpResponses, &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(resp)),
				})
			}

			client := &Client{
				apiKey:         "test",
				apiURL:         "http://test/api",
				logger:         logrus.New(),
				filterAlbumIDs: tt.filterAlbumIDs,
				client: &http.Client{
					Transport: &mockTransportSeq{responses: httpResponses},
				},
			}

			// Act
			assets, err := client.FetchAssets(10, make(map[string]utils.TStack))

			// Assert
			require.NoError(t, err)
			assert.Len(t, assets, tt.expectedCount)
		})
	}
}

func TestFetchAssetsDeduplication(t *testing.T) {
	// First album returns asset-1 and asset-2
	album1Response := `{"assets": {"items": [
		{"id": "asset-1", "originalFileName": "photo1.jpg"},
		{"id": "asset-2", "originalFileName": "photo2.jpg"}
	], "nextPage": ""}}`

	// Second album returns asset-2 and asset-3 (asset-2 is duplicate)
	album2Response := `{"assets": {"items": [
		{"id": "asset-2", "originalFileName": "photo2.jpg"},
		{"id": "asset-3", "originalFileName": "photo3.jpg"}
	], "nextPage": ""}}`

	client := &Client{
		apiKey:         "test",
		apiURL:         "http://test/api",
		logger:         logrus.New(),
		filterAlbumIDs: []string{"550e8400-e29b-41d4-a716-446655440000", "660e8400-e29b-41d4-a716-446655440001"},
		client: &http.Client{
			Transport: &mockTransportSeq{
				responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(album1Response))},
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(album2Response))},
				},
			},
		},
	}

	// Act
	assets, err := client.FetchAssets(10, make(map[string]utils.TStack))

	// Assert
	require.NoError(t, err)
	assert.Len(t, assets, 3, "Should have 3 unique assets (asset-1, asset-2, asset-3)")

	// Verify specific assets
	assetIDs := make(map[string]bool)
	for _, asset := range assets {
		assetIDs[asset.ID] = true
	}
	assert.True(t, assetIDs["asset-1"])
	assert.True(t, assetIDs["asset-2"])
	assert.True(t, assetIDs["asset-3"])
}

func TestFetchAssetsPagination(t *testing.T) {
	page1 := `{"assets": {"items": [{"id": "asset-1"}], "nextPage": "2"}}`
	page2 := `{"assets": {"items": [{"id": "asset-2"}], "nextPage": "3"}}`
	page3 := `{"assets": {"items": [{"id": "asset-3"}], "nextPage": ""}}`

	tests := []struct {
		name          string
		responses     []string
		expectedCount int
	}{
		{
			name:          "single page - empty nextPage",
			responses:     []string{`{"assets": {"items": [{"id": "asset-1"}], "nextPage": ""}}`},
			expectedCount: 1,
		},
		{
			name:          "single page - zero nextPage",
			responses:     []string{`{"assets": {"items": [{"id": "asset-1"}], "nextPage": "0"}}`},
			expectedCount: 1,
		},
		{
			name:          "multiple pages",
			responses:     []string{page1, page2, page3},
			expectedCount: 3,
		},
		{
			name:          "invalid nextPage stops pagination",
			responses:     []string{`{"assets": {"items": [{"id": "asset-1"}], "nextPage": "invalid"}}`},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var httpResponses []*http.Response
			for _, resp := range tt.responses {
				httpResponses = append(httpResponses, &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(resp)),
				})
			}

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logrus.New(),
				client: &http.Client{
					Transport: &mockTransportSeq{responses: httpResponses},
				},
			}

			// Act
			assets, err := client.FetchAssets(10, make(map[string]utils.TStack))

			// Assert
			require.NoError(t, err)
			assert.Len(t, assets, tt.expectedCount)
		})
	}
}

func TestFetchAssetsStackEnrichment(t *testing.T) {
	assetsResponse := `{"assets": {"items": [
		{"id": "asset-1", "originalFileName": "photo1.jpg"},
		{"id": "asset-2", "originalFileName": "photo2.jpg"}
	], "nextPage": ""}}`

	// Create stacksMap with stack info for asset-1
	stacksMap := map[string]utils.TStack{
		"asset-1": {
			ID:             "stack-123",
			PrimaryAssetID: "asset-1",
		},
	}

	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: logrus.New(),
		client: &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(assetsResponse)),
				},
			},
		},
	}

	// Act
	assets, err := client.FetchAssets(10, stacksMap)

	// Assert
	require.NoError(t, err)
	assert.Len(t, assets, 2)

	// asset-1 should have stack info
	var asset1, asset2 *utils.TAsset
	for i := range assets {
		if assets[i].ID == "asset-1" {
			asset1 = &assets[i]
		} else if assets[i].ID == "asset-2" {
			asset2 = &assets[i]
		}
	}

	require.NotNil(t, asset1)
	require.NotNil(t, asset2)
	assert.NotNil(t, asset1.Stack, "asset-1 should have stack info")
	assert.Equal(t, "stack-123", asset1.Stack.ID)
	assert.Nil(t, asset2.Stack, "asset-2 should not have stack info")
}

func TestFetchAssetsAlbumResolutionError(t *testing.T) {
	// When album name can't be resolved, FetchAssets should return error
	client := &Client{
		apiKey:         "test",
		apiURL:         "http://test/api",
		logger:         logrus.New(),
		filterAlbumIDs: []string{"NonExistentAlbum"}, // Name, not UUID
		client: &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)), // Empty albums list
				},
			},
		},
	}

	// Act
	assets, err := client.FetchAssets(10, make(map[string]utils.TStack))

	// Assert
	assert.Error(t, err)
	assert.Nil(t, assets)
	assert.Contains(t, err.Error(), "album not found")
}

/************************************************************************************************
** Tests for resolveAlbumFilters API error handling
************************************************************************************************/

func TestResolveAlbumFiltersAPIError(t *testing.T) {
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: logrus.New(),
		client: &http.Client{
			Transport: &mockTransport{
				err: io.ErrUnexpectedEOF, // Simulate network error
			},
		},
	}

	// Act - try to resolve a name (not UUID) which requires API call
	result, err := client.resolveAlbumFilters([]string{"SomeAlbumName"})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to resolve album names")
}

/************************************************************************************************
** Tests for NewClient with filter parameters
************************************************************************************************/

func TestNewClientWithFilterParams(t *testing.T) {
	tests := []struct {
		name              string
		filterAlbumIDs    []string
		filterTakenAfter  string
		filterTakenBefore string
	}{
		{
			name:              "with all filter params",
			filterAlbumIDs:    []string{"album-1", "album-2"},
			filterTakenAfter:  "2024-01-01T00:00:00Z",
			filterTakenBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:              "with only album filter",
			filterAlbumIDs:    []string{"album-1"},
			filterTakenAfter:  "",
			filterTakenBefore: "",
		},
		{
			name:              "with only date filters",
			filterAlbumIDs:    nil,
			filterTakenAfter:  "2024-01-01T00:00:00Z",
			filterTakenBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:              "with no filters",
			filterAlbumIDs:    nil,
			filterTakenAfter:  "",
			filterTakenBefore: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(
				"http://test.com",
				"test-key",
				false, false, false, false, false, false,
				tt.filterAlbumIDs,
				tt.filterTakenAfter,
				tt.filterTakenBefore,
				logrus.New(),
			)

			require.NotNil(t, client)
			assert.Equal(t, tt.filterAlbumIDs, client.filterAlbumIDs)
			assert.Equal(t, tt.filterTakenAfter, client.filterTakenAfter)
			assert.Equal(t, tt.filterTakenBefore, client.filterTakenBefore)
		})
	}
}
