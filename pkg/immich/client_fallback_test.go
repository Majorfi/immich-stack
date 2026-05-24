package immich

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**************************************************************************************************
** pathRouterMockTransport routes responses based on the request URL path and query string.
** Lets a single test simulate multiple Immich endpoints (e.g., /stacks returns 500 while
** /search/metadata returns 200 with mock data).
**************************************************************************************************/
type pathRouterMockTransport struct {
	mu      sync.Mutex
	calls   []string
	handler func(req *http.Request) (statusCode int, body string)
}

func (p *pathRouterMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target := req.URL.Path
	if req.URL.RawQuery != "" {
		target = req.URL.Path + "?" + req.URL.RawQuery
	}
	p.mu.Lock()
	p.calls = append(p.calls, target)
	p.mu.Unlock()

	statusCode, body := p.handler(req)
	return &http.Response{
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func (p *pathRouterMockTransport) callsSnapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.calls))
	copy(out, p.calls)
	return out
}

func newSilentLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

/**************************************************************************************************
** TestFetchAllStacksFallbackOn5xx verifies that a 5xx response from /stacks triggers the
** hybrid fallback path, which then succeeds via /search/metadata + /assets/{id} and produces
** a correct stacks map.
**************************************************************************************************/
func TestFetchAllStacksFallbackOn5xx(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		path := req.URL.Path
		query := req.URL.RawQuery
		switch {
		case path == "/api/stacks" && query == "":
			return http.StatusInternalServerError, `{"message":"Internal server error"}`
		case strings.HasPrefix(path, "/api/stacks") && strings.Contains(query, "primaryAssetId="):
			return http.StatusOK, `[]`
		case path == "/api/search/metadata":
			return http.StatusOK, `{"assets":{"items":[{"id":"a1"},{"id":"a2"}],"nextPage":""}}`
		case strings.HasPrefix(path, "/api/assets/"):
			assetID := strings.TrimPrefix(path, "/api/assets/")
			return http.StatusOK, fmt.Sprintf(
				`{"id":"%s","stack":{"id":"s1","primaryAssetId":"a1","assetCount":2}}`,
				assetID,
			)
		}
		return http.StatusNotFound, `{"error":"no mock for ` + path + `"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	stacksMap, err := client.FetchAllStacks()
	require.NoError(t, err, "FetchAllStacks should succeed via hybrid fallback")
	require.NotNil(t, stacksMap)

	assert.Equal(t, "s1", stacksMap["a1"].ID, "asset a1 should resolve to stack s1")
	assert.Equal(t, "s1", stacksMap["a2"].ID, "asset a2 should resolve to stack s1")
	assert.Len(t, stacksMap, 2, "map should hold 2 entries (a1, a2)")

	calls := transport.callsSnapshot()
	hasInitialStacksCall := false
	hasSearchCall := false
	for _, c := range calls {
		if c == "/api/stacks" {
			hasInitialStacksCall = true
		}
		if c == "/api/search/metadata" {
			hasSearchCall = true
		}
	}
	assert.True(t, hasInitialStacksCall, "initial GET /stacks should be attempted before fallback")
	assert.True(t, hasSearchCall, "fallback should call /search/metadata")
}

/**************************************************************************************************
** TestFetchAllStacksFallbackRecoversArchivedPrimary verifies that an archived primary (whose
** /assets/{id} response strips the stack field) is recovered via the phase-2
** /stacks?primaryAssetId fallback.
**************************************************************************************************/
func TestFetchAllStacksFallbackRecoversArchivedPrimary(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		path := req.URL.Path
		query := req.URL.RawQuery
		switch {
		case path == "/api/stacks" && query == "":
			return http.StatusInternalServerError, `{"message":"Internal server error"}`
		case strings.HasPrefix(path, "/api/stacks") && strings.Contains(query, "primaryAssetId=archived-primary"):
			return http.StatusOK, `[{
				"id":"s-archived",
				"primaryAssetId":"archived-primary",
				"assets":[
					{"id":"archived-primary"},
					{"id":"archived-child"}
				]
			}]`
		case strings.HasPrefix(path, "/api/stacks") && strings.Contains(query, "primaryAssetId="):
			return http.StatusOK, `[]`
		case path == "/api/search/metadata":
			return http.StatusOK, `{"assets":{"items":[
				{"id":"normal-a"},
				{"id":"archived-primary"},
				{"id":"archived-child"}
			],"nextPage":""}}`
		case path == "/api/assets/normal-a":
			return http.StatusOK, `{"id":"normal-a","stack":null}`
		case path == "/api/assets/archived-primary":
			return http.StatusOK, `{"id":"archived-primary","stack":null}`
		case path == "/api/assets/archived-child":
			return http.StatusOK, `{"id":"archived-child","stack":null}`
		}
		return http.StatusNotFound, `{"error":"no mock for ` + path + `"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	stacksMap, err := client.FetchAllStacks()
	require.NoError(t, err)
	require.NotNil(t, stacksMap)

	assert.Equal(t, "s-archived", stacksMap["archived-primary"].ID,
		"archived primary should be recovered via phase 2 fallback")
	assert.Equal(t, "s-archived", stacksMap["archived-child"].ID,
		"archived child should be recovered via phase 2 fallback (from the stack's member list)")
	_, hasNormalA := stacksMap["normal-a"]
	assert.False(t, hasNormalA, "normal-a is not in any stack, should not appear in map")
}

/**************************************************************************************************
** TestFetchAllStacksFallbackMergesPartialPhase1Stack verifies the phase 2 merge logic when
** phase 1 has already discovered a stack via a non-archived child. The archived primary's
** /assets/{id} returns stack=null (the Immich quirk), but phase 2's /stacks?primaryAssetId
** call returns the full member list — the merge must add the missing primary to the existing
** stack instead of skipping it.
**************************************************************************************************/
func TestFetchAllStacksFallbackMergesPartialPhase1Stack(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		path := req.URL.Path
		query := req.URL.RawQuery
		switch {
		case path == "/api/stacks" && query == "":
			return http.StatusInternalServerError, `{"message":"Internal server error"}`
		case strings.HasPrefix(path, "/api/stacks") && strings.Contains(query, "primaryAssetId=archived-primary"):
			return http.StatusOK, `[{
				"id":"s-mixed",
				"primaryAssetId":"archived-primary",
				"assets":[
					{"id":"archived-primary","isArchived":true,"visibility":"archive"},
					{"id":"normal-child","isArchived":false,"visibility":"timeline"}
				]
			}]`
		case strings.HasPrefix(path, "/api/stacks") && strings.Contains(query, "primaryAssetId="):
			return http.StatusOK, `[]`
		case path == "/api/search/metadata":
			return http.StatusOK, `{"assets":{"items":[
				{"id":"archived-primary"},
				{"id":"normal-child"}
			],"nextPage":""}}`
		case path == "/api/assets/archived-primary":
			return http.StatusOK, `{"id":"archived-primary","isArchived":true,"visibility":"archive","stack":null}`
		case path == "/api/assets/normal-child":
			return http.StatusOK, `{"id":"normal-child","isArchived":false,"visibility":"timeline","stack":{"id":"s-mixed","primaryAssetId":"archived-primary","assetCount":2}}`
		}
		return http.StatusNotFound, `{"error":"no mock for ` + path + `"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	stacksMap, err := client.FetchAllStacks()
	require.NoError(t, err)
	require.NotNil(t, stacksMap)

	assert.Equal(t, "s-mixed", stacksMap["normal-child"].ID,
		"normal child must resolve to s-mixed (phase 1 discovers this)")
	assert.Equal(t, "s-mixed", stacksMap["archived-primary"].ID,
		"archived primary must resolve to s-mixed (phase 2 must merge, not skip)")
	assert.Len(t, stacksMap, 2,
		"both members of the partially-discovered stack should be in the final map")
}

/**************************************************************************************************
** TestFetchAllStacksNoFallbackOn4xx verifies that a 4xx response from /stacks does NOT trigger
** the fallback — only 5xx responses do (since 4xx indicates client-side problems, not the
** large-library JSON limit).
**************************************************************************************************/
func TestFetchAllStacksNoFallbackOn4xx(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		return http.StatusUnauthorized, `{"message":"Unauthorized"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	_, err := client.FetchAllStacks()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401", "error should mention the 401")

	calls := transport.callsSnapshot()
	assert.Len(t, calls, 1, "should NOT trigger fallback on 4xx")
	assert.Equal(t, "/api/stacks", calls[0])
}

/**************************************************************************************************
** TestFetchAllStacksFallbackBothFail verifies that when /stacks returns 5xx AND the hybrid
** fallback also fails, FetchAllStacks returns an error that surfaces both failures.
**************************************************************************************************/
func TestFetchAllStacksFallbackBothFail(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		return http.StatusInternalServerError, `{"message":"Internal server error"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	_, err := client.FetchAllStacks()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/stacks", "error should mention initial /stacks failure")
	assert.Contains(t, err.Error(), "hybrid", "error should mention hybrid fallback failure")
}

/**************************************************************************************************
** TestFetchAllStacksFallbackPartialResult verifies that when the hybrid path completes with
** some per-asset failures (e.g., transient 4xx on individual /assets/{id} calls), it returns
** a PartialResultError. FetchAllStacks then treats that as a soft failure: the partial map
** is returned to the caller with nil error, since "incomplete map" is strictly better than
** "no stack info at all" when the primary /stacks endpoint already failed.
**************************************************************************************************/
func TestFetchAllStacksFallbackPartialResult(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		path := req.URL.Path
		query := req.URL.RawQuery
		switch {
		case path == "/api/stacks" && query == "":
			return http.StatusInternalServerError, `{"message":"Internal server error"}`
		case strings.HasPrefix(path, "/api/stacks") && strings.Contains(query, "primaryAssetId="):
			return http.StatusOK, `[]`
		case path == "/api/search/metadata":
			return http.StatusOK, `{"assets":{"items":[{"id":"a-ok"},{"id":"a-fail"}],"nextPage":""}}`
		case path == "/api/assets/a-ok":
			return http.StatusOK, `{"id":"a-ok","stack":{"id":"s1","primaryAssetId":"a-ok","assetCount":1}}`
		case path == "/api/assets/a-fail":
			return http.StatusNotFound, `{"error":"asset gone"}`
		}
		return http.StatusNotFound, `{"error":"no mock for ` + path + `"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	stacksMap, err := client.FetchAllStacks()
	require.NoError(t, err, "FetchAllStacks should accept PartialResultError as a soft failure")
	require.NotNil(t, stacksMap)
	assert.Equal(t, "s1", stacksMap["a-ok"].ID, "successful asset should be in partial result")
}

/**************************************************************************************************
** TestFetchAllStacksHybridReturnsPartialResultError verifies that fetchAllStacksHybrid itself
** returns a typed *PartialResultError when there are per-asset failures, so callers that don't
** want partial results can distinguish via errors.As.
**************************************************************************************************/
func TestFetchAllStacksHybridReturnsPartialResultError(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		path := req.URL.Path
		switch {
		case path == "/api/search/metadata":
			return http.StatusOK, `{"assets":{"items":[{"id":"a-ok"},{"id":"a-fail"}],"nextPage":""}}`
		case path == "/api/assets/a-ok":
			return http.StatusOK, `{"id":"a-ok","stack":null}`
		case path == "/api/assets/a-fail":
			return http.StatusNotFound, `{"error":"asset gone"}`
		case strings.HasPrefix(path, "/api/stacks"):
			return http.StatusOK, `[]`
		}
		return http.StatusNotFound, `{"error":"no mock for ` + path + `"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	_, err := client.fetchAllStacksHybrid(2)
	require.Error(t, err, "hybrid should report partial failure")

	var partial *PartialResultError
	require.ErrorAs(t, err, &partial, "error should be inspectable as *PartialResultError")
	assert.Equal(t, 1, partial.Phase1Failed, "should report exactly 1 phase-1 failure")
	assert.Equal(t, 0, partial.Phase2Failed, "phase 2 had no failures (no archived primary candidates)")
}

/**************************************************************************************************
** TestFetchAllStacksFallbackBothFailErrorsArePreserved verifies that when both /stacks and the
** hybrid fallback fail with APIErrors, both are reachable via errors.As (thanks to errors.Join
** preserving the full chain), letting callers inspect the underlying status codes.
**************************************************************************************************/
func TestFetchAllStacksFallbackBothFailErrorsArePreserved(t *testing.T) {
	handler := func(req *http.Request) (int, string) {
		return http.StatusInternalServerError, `{"message":"Internal server error"}`
	}

	transport := &pathRouterMockTransport{handler: handler}
	client := &Client{
		apiKey: "test",
		apiURL: "http://test/api",
		logger: newSilentLogger(),
		client: &http.Client{Transport: transport},
	}

	_, err := client.FetchAllStacks()
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "an APIError should be reachable via errors.As")
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

/**************************************************************************************************
** TestDoRequestWithUpstreamRetryAttemptsGuard verifies that calling with attempts <= 0 still
** issues exactly one request (defensive guard against silent no-op success).
**************************************************************************************************/
func TestDoRequestWithUpstreamRetryAttemptsGuard(t *testing.T) {
	for _, attempts := range []int{0, -1, -100} {
		t.Run(fmt.Sprintf("attempts=%d", attempts), func(t *testing.T) {
			callCount := 0
			handler := func(req *http.Request) (int, string) {
				callCount++
				return http.StatusOK, `{}`
			}
			transport := &pathRouterMockTransport{handler: handler}
			client := &Client{
				apiKey: "test",
				apiURL: "http://test/api",
				logger: newSilentLogger(),
				client: &http.Client{Transport: transport},
			}

			var result struct{}
			err := client.doRequestWithUpstreamRetry(http.MethodGet, "/probe", nil, &result, attempts)
			require.NoError(t, err)
			assert.Equal(t, 1, callCount, "attempts <= 0 must still issue one request")
		})
	}
}

/**************************************************************************************************
** TestIsLikelyLargeLibrary5xx verifies the predicate that decides whether to trigger the
** fallback. Only 5xx APIErrors should match; 4xx, plain errors, and nil should not.
**************************************************************************************************/
func TestIsLikelyLargeLibrary5xx(t *testing.T) {
	mkAPIErr := func(code int) error {
		return &APIError{StatusCode: code, Status: http.StatusText(code), Body: "{}"}
	}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "500 response", err: mkAPIErr(500), want: true},
		{name: "502 response", err: mkAPIErr(502), want: true},
		{name: "503 response", err: mkAPIErr(503), want: true},
		{name: "504 response", err: mkAPIErr(504), want: true},
		{name: "599 boundary", err: mkAPIErr(599), want: true},
		{name: "600 out of range", err: mkAPIErr(600), want: false},
		{name: "499 just under 5xx", err: mkAPIErr(499), want: false},
		{name: "401 response", err: mkAPIErr(401), want: false},
		{name: "404 response", err: mkAPIErr(404), want: false},
		{name: "wrapped 500", err: fmt.Errorf("ctx: %w", mkAPIErr(500)), want: true},
		{name: "plain error with '500' in message", err: fmt.Errorf("connection failed with code 500"), want: false},
		{name: "network error", err: fmt.Errorf("error making request after 3 retries: connection refused"), want: false},
		{name: "nil error", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyLargeLibrary5xx(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
