package immich

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetTypesForSearch(t *testing.T) {
	tests := []struct {
		name          string
		includeVideos bool
		want          []string
	}{
		{name: "default is IMAGE only", includeVideos: false, want: []string{"IMAGE"}},
		{name: "includeVideos adds VIDEO", includeVideos: true, want: []string{"IMAGE", "VIDEO"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{includeVideos: tt.includeVideos}
			assert.Equal(t, tt.want, c.assetTypesForSearch())
		})
	}
}

/**************************************************************************************************
** typeTrackingTransport captures the "type" field of each /search/metadata POST and counts
** calls per type. Used to verify FetchAssets dispatches the right number of searches based
** on the includeVideos flag.
**************************************************************************************************/
type typeTrackingTransport struct {
	mu         sync.Mutex
	typesCalls map[string]int
}

func (t *typeTrackingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Decode the asset type filter OUTSIDE the lock — I/O and JSON parsing are slow and
	// holding the mutex through them would serialize concurrent requests unnecessarily.
	var assetType string
	if strings.HasSuffix(req.URL.Path, "/search/metadata") && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		var payload map[string]interface{}
		_ = json.Unmarshal(body, &payload)
		if v, ok := payload["type"].(string); ok {
			assetType = v
		}
	}

	if assetType != "" {
		t.mu.Lock()
		if t.typesCalls == nil {
			t.typesCalls = make(map[string]int)
		}
		t.typesCalls[assetType]++
		t.mu.Unlock()
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"assets":{"items":[],"nextPage":""}}`)),
	}, nil
}

func TestFetchAssetsRespectsIncludeVideosFlag(t *testing.T) {
	tests := []struct {
		name             string
		includeVideos    bool
		expectedTypeKeys []string
	}{
		{name: "default fetches only IMAGE", includeVideos: false, expectedTypeKeys: []string{"IMAGE"}},
		{name: "includeVideos fetches IMAGE and VIDEO", includeVideos: true, expectedTypeKeys: []string{"IMAGE", "VIDEO"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &typeTrackingTransport{}
			client := &Client{
				apiKey:        "test",
				apiURL:        "http://test/api",
				logger:        newSilentLogger(),
				includeVideos: tt.includeVideos,
				client:        &http.Client{Transport: transport},
			}
			_, err := client.FetchAssets(100, nil)
			require.NoError(t, err)

			gotTypes := make([]string, 0, len(transport.typesCalls))
			for k := range transport.typesCalls {
				gotTypes = append(gotTypes, k)
			}
			assert.ElementsMatch(t, tt.expectedTypeKeys, gotTypes,
				"unexpected /search/metadata type filters actually called: %+v", transport.typesCalls)
		})
	}
}

func TestFetchAllAssetIDsViaSearchRespectsIncludeVideosFlag(t *testing.T) {
	// The hybrid fallback enumerates assets via /search/metadata across both visibility
	// passes (timeline + archive). With includeVideos, this cross-product should also span
	// both asset types — i.e., 4 distinct (type, visibility) combinations.
	for _, tt := range []struct {
		name              string
		includeVideos     bool
		expectedTypeKeys  []string
		expectedTotalKeys int
	}{
		{
			name:              "default fetches IMAGE × 2 visibilities",
			includeVideos:     false,
			expectedTypeKeys:  []string{"IMAGE"},
			expectedTotalKeys: 1,
		},
		{
			name:              "includeVideos fetches IMAGE and VIDEO × 2 visibilities",
			includeVideos:     true,
			expectedTypeKeys:  []string{"IMAGE", "VIDEO"},
			expectedTotalKeys: 2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			transport := &typeTrackingTransport{}
			client := &Client{
				apiKey:        "test",
				apiURL:        "http://test/api",
				logger:        newSilentLogger(),
				includeVideos: tt.includeVideos,
				client:        &http.Client{Transport: transport},
			}
			_, err := client.fetchAllAssetIDsViaSearch()
			require.NoError(t, err)

			gotTypes := make([]string, 0, len(transport.typesCalls))
			for k := range transport.typesCalls {
				gotTypes = append(gotTypes, k)
			}
			assert.ElementsMatch(t, tt.expectedTypeKeys, gotTypes,
				"unexpected type filters actually called: %+v", transport.typesCalls)
			assert.Len(t, transport.typesCalls, tt.expectedTotalKeys)
		})
	}
}

func TestFetchTrashedAssetsRespectsIncludeVideosFlag(t *testing.T) {
	for _, tt := range []struct {
		name             string
		includeVideos    bool
		expectedTypeKeys []string
	}{
		{name: "default fetches only IMAGE trash", includeVideos: false, expectedTypeKeys: []string{"IMAGE"}},
		{name: "includeVideos fetches IMAGE and VIDEO trash", includeVideos: true, expectedTypeKeys: []string{"IMAGE", "VIDEO"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			transport := &typeTrackingTransport{}
			client := &Client{
				apiKey:        "test",
				apiURL:        "http://test/api",
				logger:        newSilentLogger(),
				includeVideos: tt.includeVideos,
				client:        &http.Client{Transport: transport},
			}
			_, err := client.FetchTrashedAssets(100)
			require.NoError(t, err)

			gotTypes := make([]string, 0, len(transport.typesCalls))
			for k := range transport.typesCalls {
				gotTypes = append(gotTypes, k)
			}
			assert.ElementsMatch(t, tt.expectedTypeKeys, gotTypes,
				"unexpected /search/metadata type filters actually called: %+v", transport.typesCalls)
		})
	}
}
