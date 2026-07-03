package immich

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Tests for TrashAssets batching
************************************************************************************************/

type trashPayload struct {
	Force bool     `json:"force"`
	IDs   []string `json:"ids"`
}

type recordedRequest struct {
	method  string
	path    string
	payload trashPayload
}

// recordingTransport captures every request and returns per-request statuses (200 by
// default) so batching behavior can be asserted.
type recordingTransport struct {
	requests []recordedRequest
	statuses []int
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var payload trashPayload
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
	}
	r.requests = append(r.requests, recordedRequest{method: req.Method, path: req.URL.Path, payload: payload})

	status := http.StatusOK
	if idx := len(r.requests) - 1; idx < len(r.statuses) && r.statuses[idx] != 0 {
		status = r.statuses[idx]
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader("{}")),
	}, nil
}

func newTrashTestClient(transport *recordingTransport, dryRun bool) *Client {
	return &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		dryRun: dryRun,
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}
}

func makeAssetIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("asset-%d", i)
	}
	return ids
}

func TestTrashAssetsBatchesLargeSets(t *testing.T) {
	transport := &recordingTransport{}
	client := newTrashTestClient(transport, false)
	ids := makeAssetIDs(2500)

	err := client.TrashAssets(ids)

	require.NoError(t, err)
	require.Len(t, transport.requests, 3)
	assert.Len(t, transport.requests[0].payload.IDs, 1000)
	assert.Len(t, transport.requests[1].payload.IDs, 1000)
	assert.Len(t, transport.requests[2].payload.IDs, 500)

	var sent []string
	for i, req := range transport.requests {
		assert.False(t, req.payload.Force, "batch %d must use force=false", i)
		assert.Equal(t, http.MethodDelete, req.method)
		assert.Equal(t, "/api/assets", req.path)
		sent = append(sent, req.payload.IDs...)
	}
	assert.Equal(t, ids, sent, "batches must cover all IDs in order")
}

func TestTrashAssetsSingleBatch(t *testing.T) {
	transport := &recordingTransport{}
	client := newTrashTestClient(transport, false)

	err := client.TrashAssets([]string{"a", "b", "c"})

	require.NoError(t, err)
	require.Len(t, transport.requests, 1)
	assert.Equal(t, []string{"a", "b", "c"}, transport.requests[0].payload.IDs)
}

func TestTrashAssetsDryRunMakesNoRequests(t *testing.T) {
	transport := &recordingTransport{}
	client := newTrashTestClient(transport, true)

	err := client.TrashAssets(makeAssetIDs(1500))

	require.NoError(t, err)
	assert.Empty(t, transport.requests)
}

func TestTrashAssetsEmptyInput(t *testing.T) {
	transport := &recordingTransport{}
	client := newTrashTestClient(transport, false)

	err := client.TrashAssets(nil)

	require.NoError(t, err)
	assert.Empty(t, transport.requests)
}

func TestTrashAssetsStopsOnBatchError(t *testing.T) {
	transport := &recordingTransport{statuses: []int{http.StatusOK, http.StatusInternalServerError}}
	client := newTrashTestClient(transport, false)

	err := client.TrashAssets(makeAssetIDs(2500))

	require.Error(t, err)
	assert.Len(t, transport.requests, 2, "third batch must not be attempted after a failure")
}

/************************************************************************************************
** Tests for FetchTrashedAssets flag forwarding
************************************************************************************************/

// archivedFlagTransport records the withArchived value of each /search/metadata request.
type archivedFlagTransport struct {
	withArchivedValues []bool
}

func (a *archivedFlagTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body struct {
		WithArchived bool `json:"withArchived"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return nil, err
	}
	a.withArchivedValues = append(a.withArchivedValues, body.WithArchived)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"assets":{"items":[],"nextPage":""}}`)),
	}, nil
}

func TestFetchTrashedAssetsRespectsWithArchivedFlag(t *testing.T) {
	for _, withArchived := range []bool{false, true} {
		t.Run(fmt.Sprintf("withArchived=%t", withArchived), func(t *testing.T) {
			transport := &archivedFlagTransport{}
			client := &Client{
				apiKey:       "test",
				apiURL:       "http://test/api",
				logger:       newSilentLogger(),
				withArchived: withArchived,
				client:       &http.Client{Transport: transport},
			}

			_, err := client.FetchTrashedAssets(100)

			require.NoError(t, err)
			require.NotEmpty(t, transport.withArchivedValues)
			for _, got := range transport.withArchivedValues {
				assert.Equal(t, withArchived, got, "withArchived must be forwarded to /search/metadata")
			}
		})
	}
}
