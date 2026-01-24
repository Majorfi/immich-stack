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

/************************************************************************************************
** Tests for GetCurrentUser
************************************************************************************************/

func TestGetCurrentUser(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		statusCode   int
		wantErr      bool
		expectedName string
		expectedEmail string
	}{
		{
			name: "successful user fetch",
			response: `{
				"id": "user-123",
				"name": "Test User",
				"email": "test@example.com",
				"isAdmin": false
			}`,
			statusCode:    http.StatusOK,
			wantErr:       false,
			expectedName:  "Test User",
			expectedEmail: "test@example.com",
		},
		{
			name:       "unauthorized - invalid API key",
			response:   `{"message": "Unauthorized", "statusCode": 401}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "server error",
			response:   `{"message": "Internal Server Error"}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.statusCode,
							Body:       io.NopCloser(strings.NewReader(tt.response)),
						},
					},
				},
			}

			user, err := client.GetCurrentUser()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedName, user.Name)
				assert.Equal(t, tt.expectedEmail, user.Email)
			}
		})
	}
}

/************************************************************************************************
** Tests for DeleteStack
************************************************************************************************/

func TestDeleteStack(t *testing.T) {
	tests := []struct {
		name       string
		stackID    string
		reason     string
		dryRun     bool
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful delete",
			stackID:    "stack-123",
			reason:     "test reason",
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "dry run - no API call",
			stackID:    "stack-456",
			reason:     "dry run test",
			dryRun:     true,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "stack not found",
			stackID:    "nonexistent-stack",
			reason:     "should fail",
			dryRun:     false,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "server error",
			stackID:    "stack-789",
			reason:     "server error test",
			dryRun:     false,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "special reason - single asset stack",
			stackID:    "stack-single",
			reason:     "Stack with 1 asset remaining",
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				dryRun: tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.statusCode,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			}

			err := client.DeleteStack(tt.stackID, tt.reason)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/************************************************************************************************
** Tests for ListDuplicates
************************************************************************************************/

func TestListDuplicates(t *testing.T) {
	tests := []struct {
		name   string
		assets []utils.TAsset
	}{
		{
			name:   "empty assets",
			assets: []utils.TAsset{},
		},
		{
			name:   "nil assets",
			assets: nil,
		},
		{
			name: "no duplicates",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-01T10:00:00"},
				{ID: "2", OriginalFileName: "photo2.jpg", LocalDateTime: "2024-01-01T11:00:00"},
				{ID: "3", OriginalFileName: "photo3.jpg", LocalDateTime: "2024-01-01T12:00:00"},
			},
		},
		{
			name: "duplicates found - same filename and datetime",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-01T10:00:00"},
				{ID: "2", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-01T10:00:00"},
				{ID: "3", OriginalFileName: "photo2.jpg", LocalDateTime: "2024-01-01T11:00:00"},
			},
		},
		{
			name: "multiple duplicate groups",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-01T10:00:00"},
				{ID: "2", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-01T10:00:00"},
				{ID: "3", OriginalFileName: "photo2.jpg", LocalDateTime: "2024-01-01T11:00:00"},
				{ID: "4", OriginalFileName: "photo2.jpg", LocalDateTime: "2024-01-01T11:00:00"},
				{ID: "5", OriginalFileName: "photo2.jpg", LocalDateTime: "2024-01-01T11:00:00"},
			},
		},
		{
			name: "same filename different datetime - not duplicates",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-01T10:00:00"},
				{ID: "2", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-02T10:00:00"},
			},
		},
		{
			name: "same datetime different filename - not duplicates",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "photo1.jpg", LocalDateTime: "2024-01-01T10:00:00"},
				{ID: "2", OriginalFileName: "photo2.jpg", LocalDateTime: "2024-01-01T10:00:00"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
			}

			err := client.ListDuplicates(tt.assets)
			assert.NoError(t, err)
		})
	}
}

/************************************************************************************************
** Tests for FetchTrashedAssets
************************************************************************************************/

func TestFetchTrashedAssets(t *testing.T) {
	tests := []struct {
		name          string
		responses     []string
		expectedCount int
		wantErr       bool
	}{
		{
			name: "empty response",
			responses: []string{
				`{"assets": {"items": [], "nextPage": ""}}`,
			},
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name: "single page with trashed assets",
			responses: []string{
				`{"assets": {"items": [
					{"id": "1", "isTrashed": true, "originalFileName": "deleted1.jpg"},
					{"id": "2", "isTrashed": false, "originalFileName": "active.jpg"},
					{"id": "3", "isTrashed": true, "originalFileName": "deleted2.jpg"}
				], "nextPage": ""}}`,
			},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name: "multiple pages",
			responses: []string{
				`{"assets": {"items": [
					{"id": "1", "isTrashed": true, "originalFileName": "deleted1.jpg"}
				], "nextPage": "2"}}`,
				`{"assets": {"items": [
					{"id": "2", "isTrashed": true, "originalFileName": "deleted2.jpg"}
				], "nextPage": ""}}`,
			},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name: "filters only trashed assets",
			responses: []string{
				`{"assets": {"items": [
					{"id": "1", "isTrashed": true},
					{"id": "2", "isTrashed": false},
					{"id": "3", "isTrashed": false},
					{"id": "4", "isTrashed": true}
				], "nextPage": ""}}`,
			},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name: "nextPage is 0 string",
			responses: []string{
				`{"assets": {"items": [
					{"id": "1", "isTrashed": true}
				], "nextPage": "0"}}`,
			},
			expectedCount: 1,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			responses := make([]*http.Response, len(tt.responses))
			for i, resp := range tt.responses {
				responses[i] = &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(resp)),
				}
			}

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				client: &http.Client{
					Transport: &mockTransportSeq{
						responses: responses,
					},
				},
			}

			assets, err := client.FetchTrashedAssets(100)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, assets, tt.expectedCount)
				for _, asset := range assets {
					assert.True(t, asset.IsTrashed, "All returned assets should be trashed")
				}
			}
		})
	}
}

/************************************************************************************************
** Tests for TrashAssets
************************************************************************************************/

func TestTrashAssets(t *testing.T) {
	tests := []struct {
		name       string
		assetIDs   []string
		dryRun     bool
		statusCode int
		wantErr    bool
	}{
		{
			name:       "empty assets - no-op",
			assetIDs:   []string{},
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "nil assets - no-op",
			assetIDs:   nil,
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "successful trash",
			assetIDs:   []string{"asset-1", "asset-2"},
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "dry run - no API call",
			assetIDs:   []string{"asset-1", "asset-2"},
			dryRun:     true,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "server error",
			assetIDs:   []string{"asset-1"},
			dryRun:     false,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "single asset",
			assetIDs:   []string{"asset-single"},
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				dryRun: tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.statusCode,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			}

			err := client.TrashAssets(tt.assetIDs)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/************************************************************************************************
** Tests for FetchAllStacks - additional error cases
************************************************************************************************/

func TestFetchAllStacksErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"message": "Internal Server Error"}`,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   `{"message": "Unauthorized"}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON response",
			statusCode: http.StatusOK,
			response:   `{invalid json`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.statusCode,
							Body:       io.NopCloser(strings.NewReader(tt.response)),
						},
					},
				},
			}

			_, err := client.FetchAllStacks()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/************************************************************************************************
** Tests for FetchAllStacks - reset and single asset stack paths
************************************************************************************************/

func TestFetchAllStacksResetStacks(t *testing.T) {
	tests := []struct {
		name                    string
		resetStacks             bool
		removeSingleAssetStacks bool
		dryRun                  bool
		stacksResponse          string
		expectedMapSize         int
		expectNilMap            bool
	}{
		{
			name:        "reset stacks - deletes all stacks",
			resetStacks: true,
			dryRun:      false,
			stacksResponse: `[
				{"id": "stack-1", "primaryAssetId": "asset-1", "assets": [{"id": "asset-1"}, {"id": "asset-2"}]},
				{"id": "stack-2", "primaryAssetId": "asset-3", "assets": [{"id": "asset-3"}]}
			]`,
			expectedMapSize: 0,
			expectNilMap:    false,
		},
		{
			name:        "reset stacks dry run - returns nil map",
			resetStacks: true,
			dryRun:      true,
			stacksResponse: `[
				{"id": "stack-1", "primaryAssetId": "asset-1", "assets": [{"id": "asset-1"}]}
			]`,
			expectedMapSize: 0,
			expectNilMap:    true,
		},
		{
			name:        "reset stacks with no existing stacks",
			resetStacks: true,
			dryRun:      false,
			stacksResponse: `[]`,
			expectedMapSize: 0,
			expectNilMap:    false,
		},
		{
			name:                    "remove single asset stacks - map still includes all fetched stacks",
			resetStacks:             false,
			removeSingleAssetStacks: true,
			dryRun:                  false,
			stacksResponse: `[
				{"id": "stack-single", "primaryAssetId": "asset-1", "assets": [{"id": "asset-1"}]},
				{"id": "stack-multi", "primaryAssetId": "asset-2", "assets": [{"id": "asset-2"}, {"id": "asset-3"}]}
			]`,
			expectedMapSize: 3,
			expectNilMap:    false,
		},
		{
			name:                    "remove empty stacks - map still includes non-empty stacks",
			resetStacks:             false,
			removeSingleAssetStacks: true,
			dryRun:                  false,
			stacksResponse: `[
				{"id": "stack-empty", "primaryAssetId": "asset-1", "assets": []},
				{"id": "stack-normal", "primaryAssetId": "asset-2", "assets": [{"id": "asset-2"}, {"id": "asset-3"}]}
			]`,
			expectedMapSize: 2,
			expectNilMap:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey:                  "test",
				apiURL:                  "http://test/api",
				logger:                  logger,
				resetStacks:             tt.resetStacks,
				removeSingleAssetStacks: tt.removeSingleAssetStacks,
				dryRun:                  tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(tt.stacksResponse)),
						},
					},
				},
			}

			stacksMap, err := client.FetchAllStacks()

			require.NoError(t, err)

			if tt.expectNilMap {
				assert.Nil(t, stacksMap)
			} else {
				assert.NotNil(t, stacksMap)
				assert.Len(t, stacksMap, tt.expectedMapSize)
			}
		})
	}
}

func TestFetchAllStacksWithDebugLogging(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.DebugLevel)

	stacksResponse := `[
		{"id": "stack-1", "primaryAssetId": "a1", "assets": [{"id": "a1"}, {"id": "a2"}]},
		{"id": "stack-2", "primaryAssetId": "b1", "assets": [{"id": "b1"}, {"id": "b2"}, {"id": "b3"}]},
		{"id": "stack-3", "primaryAssetId": "c1", "assets": [{"id": "c1"}, {"id": "c2"}]}
	]`

	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: logger,
		client: &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(stacksResponse)),
				},
			},
		},
	}

	stacksMap, err := client.FetchAllStacks()

	require.NoError(t, err)
	assert.Len(t, stacksMap, 7)
}

/************************************************************************************************
** Tests for ModifyStack - comprehensive coverage
************************************************************************************************/

func TestModifyStackComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		assetIDs   []string
		dryRun     bool
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful stack creation",
			assetIDs:   []string{"asset-1", "asset-2", "asset-3"},
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "dry run skips API call",
			assetIDs:   []string{"asset-1", "asset-2"},
			dryRun:     true,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "API error returns error",
			assetIDs:   []string{"asset-1", "asset-2"},
			dryRun:     false,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "single asset stack",
			assetIDs:   []string{"asset-1"},
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "large stack",
			assetIDs:   []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10"},
			dryRun:     false,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized error",
			assetIDs:   []string{"asset-1"},
			dryRun:     false,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "not found error",
			assetIDs:   []string{"nonexistent-asset"},
			dryRun:     false,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				dryRun: tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.statusCode,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			}

			err := client.ModifyStack(tt.assetIDs)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "error modifying stack")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/************************************************************************************************
** Tests for NewClient - edge cases
************************************************************************************************/

func TestNewClientEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		apiURL    string
		apiKey    string
		logger    *logrus.Logger
		expectNil bool
	}{
		{
			name:      "nil logger returns nil",
			apiURL:    "http://test.com",
			apiKey:    "test-key",
			logger:    nil,
			expectNil: true,
		},
		{
			name:      "invalid URL - no host",
			apiURL:    "http://",
			apiKey:    "test-key",
			logger:    logrus.New(),
			expectNil: true,
		},
		{
			name:      "invalid URL - malformed",
			apiURL:    "://invalid",
			apiKey:    "test-key",
			logger:    logrus.New(),
			expectNil: true,
		},
		{
			name:      "valid URL with port",
			apiURL:    "http://localhost:8080",
			apiKey:    "test-key",
			logger:    logrus.New(),
			expectNil: false,
		},
		{
			name:      "valid HTTPS URL",
			apiURL:    "https://immich.example.com",
			apiKey:    "test-key",
			logger:    logrus.New(),
			expectNil: false,
		},
		{
			name:      "URL with path - path stripped",
			apiURL:    "http://localhost:8080/some/path",
			apiKey:    "test-key",
			logger:    logrus.New(),
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(
				tt.apiURL,
				tt.apiKey,
				false, false, false, false, false, false,
				nil, "", "",
				tt.logger,
			)

			if tt.expectNil {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
			}
		})
	}
}

/************************************************************************************************
** Tests for FetchTrashedAssets - error paths
************************************************************************************************/

func TestFetchTrashedAssetsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"message": "Internal Server Error"}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON response",
			statusCode: http.StatusOK,
			response:   `{invalid`,
			wantErr:    true,
		},
		{
			name:       "invalid nextPage number",
			statusCode: http.StatusOK,
			response:   `{"assets": {"items": [{"id": "1", "isTrashed": true}], "nextPage": "invalid"}}`,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.statusCode,
							Body:       io.NopCloser(strings.NewReader(tt.response)),
						},
					},
				},
			}

			_, err := client.FetchTrashedAssets(100)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/************************************************************************************************
** Tests for doRequest edge cases
************************************************************************************************/

func TestDoRequestDecodeError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: logger,
		client: &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{invalid json response`)),
				},
			},
		},
	}

	var result map[string]interface{}
	err := client.doRequest(http.MethodGet, "/test", nil, &result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error decoding response")
}

func TestDoRequestNilResult(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: logger,
		client: &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status": "ok"}`)),
				},
			},
		},
	}

	err := client.doRequest(http.MethodDelete, "/test/123", nil, nil)

	assert.NoError(t, err)
}

func TestDoRequestWithBody(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: logger,
		client: &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id": "new-stack"}`)),
				},
			},
		},
	}

	body := map[string]interface{}{
		"assetIds": []string{"asset-1", "asset-2"},
	}

	var result map[string]interface{}
	err := client.doRequest(http.MethodPost, "/stacks", body, &result)

	assert.NoError(t, err)
	assert.Equal(t, "new-stack", result["id"])
}

func TestDoRequestErrorResponse(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: logger,
		client: &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusBadRequest,
					Status:     "400 Bad Request",
					Body:       io.NopCloser(strings.NewReader(`{"message": "Invalid asset ID"}`)),
				},
			},
		},
	}

	err := client.doRequest(http.MethodPost, "/stacks", nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error response")
	assert.Contains(t, err.Error(), "400 Bad Request")
}

/************************************************************************************************
** Album Operations Tests
************************************************************************************************/

func TestFetchAlbumAssets(t *testing.T) {
	tests := []struct {
		name           string
		albumID        string
		mockResponse   string
		mockStatusCode int
		expectedCount  int
		expectError    bool
	}{
		{
			name:           "success with assets",
			albumID:        "album-123",
			mockResponse:   `{"assets": [{"id": "a1", "originalFileName": "photo1.jpg"}, {"id": "a2", "originalFileName": "photo2.jpg"}]}`,
			mockStatusCode: http.StatusOK,
			expectedCount:  2,
			expectError:    false,
		},
		{
			name:           "success with empty album",
			albumID:        "album-empty",
			mockResponse:   `{"assets": []}`,
			mockStatusCode: http.StatusOK,
			expectedCount:  0,
			expectError:    false,
		},
		{
			name:           "album not found",
			albumID:        "nonexistent",
			mockResponse:   `{"message": "Album not found"}`,
			mockStatusCode: http.StatusNotFound,
			expectedCount:  0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.mockStatusCode,
							Body:       io.NopCloser(strings.NewReader(tt.mockResponse)),
						},
					},
				},
			}

			assets, err := client.FetchAlbumAssets(tt.albumID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to fetch album assets")
			} else {
				require.NoError(t, err)
				assert.Len(t, assets, tt.expectedCount)
			}
		})
	}
}

func TestCreateAlbum(t *testing.T) {
	tests := []struct {
		name           string
		albumName      string
		description    string
		dryRun         bool
		mockResponse   string
		mockStatusCode int
		expectError    bool
	}{
		{
			name:           "success",
			albumName:      "My Album",
			description:    "A test album",
			dryRun:         false,
			mockResponse:   `{"id": "new-album-123", "albumName": "My Album", "description": "A test album"}`,
			mockStatusCode: http.StatusCreated,
			expectError:    false,
		},
		{
			name:           "dry run mode",
			albumName:      "Dry Run Album",
			description:    "Should not create",
			dryRun:         true,
			mockResponse:   "",
			mockStatusCode: 0,
			expectError:    false,
		},
		{
			name:           "server error",
			albumName:      "Error Album",
			description:    "",
			dryRun:         false,
			mockResponse:   `{"message": "Internal server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				dryRun: tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.mockStatusCode,
							Body:       io.NopCloser(strings.NewReader(tt.mockResponse)),
						},
					},
				},
			}

			album, err := client.CreateAlbum(tt.albumName, tt.description)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create album")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, album)
				assert.Equal(t, tt.albumName, album.AlbumName)
				if tt.dryRun {
					assert.Equal(t, "dry-run-id", album.ID)
				}
			}
		})
	}
}

func TestAddAssetsToAlbum(t *testing.T) {
	tests := []struct {
		name           string
		albumID        string
		assetIDs       []string
		dryRun         bool
		mockStatusCode int
		expectError    bool
	}{
		{
			name:           "success",
			albumID:        "album-123",
			assetIDs:       []string{"asset-1", "asset-2"},
			dryRun:         false,
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "empty asset list",
			albumID:        "album-123",
			assetIDs:       []string{},
			dryRun:         false,
			mockStatusCode: 0,
			expectError:    false,
		},
		{
			name:           "dry run mode",
			albumID:        "album-123",
			assetIDs:       []string{"asset-1"},
			dryRun:         true,
			mockStatusCode: 0,
			expectError:    false,
		},
		{
			name:           "server error",
			albumID:        "album-123",
			assetIDs:       []string{"asset-1"},
			dryRun:         false,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				dryRun: tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.mockStatusCode,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			}

			err := client.AddAssetsToAlbum(tt.albumID, tt.assetIDs)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to add assets to album")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemoveAssetsFromAlbum(t *testing.T) {
	tests := []struct {
		name           string
		albumID        string
		assetIDs       []string
		dryRun         bool
		mockStatusCode int
		expectError    bool
	}{
		{
			name:           "success",
			albumID:        "album-123",
			assetIDs:       []string{"asset-1", "asset-2"},
			dryRun:         false,
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "empty asset list",
			albumID:        "album-123",
			assetIDs:       []string{},
			dryRun:         false,
			mockStatusCode: 0,
			expectError:    false,
		},
		{
			name:           "dry run mode",
			albumID:        "album-123",
			assetIDs:       []string{"asset-1"},
			dryRun:         true,
			mockStatusCode: 0,
			expectError:    false,
		},
		{
			name:           "server error",
			albumID:        "album-123",
			assetIDs:       []string{"asset-1"},
			dryRun:         false,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				dryRun: tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.mockStatusCode,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			}

			err := client.RemoveAssetsFromAlbum(tt.albumID, tt.assetIDs)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to remove assets from album")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateAlbum(t *testing.T) {
	tests := []struct {
		name           string
		albumID        string
		updates        map[string]interface{}
		dryRun         bool
		mockStatusCode int
		expectError    bool
	}{
		{
			name:           "success - archive album",
			albumID:        "album-123",
			updates:        map[string]interface{}{"isArchived": true},
			dryRun:         false,
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "success - rename album",
			albumID:        "album-123",
			updates:        map[string]interface{}{"albumName": "New Name"},
			dryRun:         false,
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "dry run mode",
			albumID:        "album-123",
			updates:        map[string]interface{}{"isArchived": true},
			dryRun:         true,
			mockStatusCode: 0,
			expectError:    false,
		},
		{
			name:           "server error",
			albumID:        "album-123",
			updates:        map[string]interface{}{"isArchived": true},
			dryRun:         false,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "album not found",
			albumID:        "nonexistent",
			updates:        map[string]interface{}{"isArchived": true},
			dryRun:         false,
			mockStatusCode: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: logger,
				dryRun: tt.dryRun,
				client: &http.Client{
					Transport: &mockTransport{
						response: &http.Response{
							StatusCode: tt.mockStatusCode,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						},
					},
				},
			}

			err := client.UpdateAlbum(tt.albumID, tt.updates)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to update album")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
