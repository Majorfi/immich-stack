package immich

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**************************************************************************************************
** concurrencyProbe is an HTTP transport that counts in-flight DELETE /stacks/{id} calls so a
** test can assert that the bounded-concurrency semaphore in FetchAllStacks actually caps
** parallelism at the configured value. The same goroutine pattern is used by the main stacker
** loop in cmd/, so validating it here covers both code paths.
**
** Each DELETE increments `inFlight`, sleeps briefly to force overlap, then decrements.
** `peak` tracks the running max via atomic CAS to be race-free.
**************************************************************************************************/
type concurrencyProbe struct {
	inFlight    atomic.Int64
	peak        atomic.Int64
	delay       time.Duration
	stacksBody  string
	deleteCount atomic.Int64
}

func (p *concurrencyProbe) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/stacks"):
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(p.stacksBody)),
		}, nil
	case req.Method == http.MethodDelete && strings.Contains(req.URL.Path, "/stacks/"):
		p.deleteCount.Add(1)
		now := p.inFlight.Add(1)
		for {
			cur := p.peak.Load()
			if now <= cur || p.peak.CompareAndSwap(cur, now) {
				break
			}
		}
		time.Sleep(p.delay)
		p.inFlight.Add(-1)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}, nil
}

func TestFetchAllStacksResetRespectsStackConcurrency(t *testing.T) {
	// Build a fake /stacks response with 40 stacks so we have enough work to observe parallelism.
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 40; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"id":"stack-`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(`","primaryAssetId":"asset-`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(`","assets":[{"id":"asset-`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(`"}]}`)
	}
	sb.WriteString("]")
	stacksBody := sb.String()

	for _, tt := range []struct {
		name        string
		concurrency int
	}{
		{name: "sequential (1)", concurrency: 1},
		{name: "moderate (5)", concurrency: 5},
		{name: "high (15)", concurrency: 15},
	} {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			probe := &concurrencyProbe{
				delay:      20 * time.Millisecond,
				stacksBody: stacksBody,
			}
			client := &Client{
				apiKey:           "test",
				apiURL:           "http://test/api",
				logger:           logger,
				resetStacks:      true,
				stackConcurrency: tt.concurrency,
				client:           &http.Client{Transport: probe},
			}

			_, err := client.FetchAllStacks()
			require.NoError(t, err)

			require.EqualValues(t, 40, probe.deleteCount.Load(),
				"every stack should be deleted exactly once")

			peak := probe.peak.Load()
			assert.LessOrEqual(t, int(peak), tt.concurrency,
				"peak in-flight DELETE /stacks/{id} (%d) exceeded configured limit (%d)",
				peak, tt.concurrency)

			// Sanity: with concurrency > 1 we want real overlap, otherwise the test isn't
			// proving anything beyond sequential behavior.
			if tt.concurrency > 1 {
				assert.Greater(t, int(peak), 1,
					"with concurrency=%d the probe should observe ≥ 2 in-flight calls (got %d) — either the delay is too small or the parallelism is broken",
					tt.concurrency, peak)
			}
		})
	}
}

func TestFetchAllStacksResetGuardsZeroConcurrency(t *testing.T) {
	// Regression test: a Client constructed directly (bypassing NewClient) with
	// stackConcurrency = 0 must not deadlock on an unbuffered semaphore. The defensive
	// `max(c.stackConcurrency, 1)` inside FetchAllStacks keeps it safe.
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	probe := &concurrencyProbe{
		delay:      1 * time.Millisecond,
		stacksBody: `[{"id":"s1","primaryAssetId":"a1","assets":[{"id":"a1"}]}]`,
	}
	client := &Client{
		apiKey:           "test",
		apiURL:           "http://test/api",
		logger:           logger,
		resetStacks:      true,
		stackConcurrency: 0,
		client:           &http.Client{Transport: probe},
	}

	done := make(chan struct{})
	go func() {
		_, _ = client.FetchAllStacks()
		close(done)
	}()
	select {
	case <-done:
		assert.EqualValues(t, 1, probe.deleteCount.Load())
	case <-time.After(5 * time.Second):
		t.Fatal("FetchAllStacks deadlocked with stackConcurrency=0 — defensive guard missing or broken")
	}
}

