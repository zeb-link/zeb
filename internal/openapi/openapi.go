// Package openapi fetches the live Core OpenAPI spec for the drift tests.
//
// The hand-written client is pinned to the spec by tests (see
// internal/api/spec_drift_test.go and internal/cli/sort_values_test.go).
// Those tests read the spec straight from production, so there is no vendored
// copy to keep fresh and no sync machinery. Production regenerates the spec
// from code on every deploy and serves it at /api/v1/openapi.json.
package openapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// DefaultSpecURL is the production spec. Override with ZEB_SPEC_URL to test
// against another Core (for example a local dev server).
const DefaultSpecURL = "https://app.zeblink.io/api/v1/openapi.json"

// SpecURL returns the spec URL the tests should use.
func SpecURL() string {
	if url := os.Getenv("ZEB_SPEC_URL"); url != "" {
		return url
	}
	return DefaultSpecURL
}

var (
	fetchOnce sync.Once
	fetched   []byte
	fetchErr  error
)

// FetchLiveSpec downloads the spec once per process and caches it. A network
// failure (offline, DNS, timeout) returns Unreachable=true so callers can
// skip; an HTTP error status is a hard error, because the server answering
// wrongly is signal, not a connectivity problem.
func FetchLiveSpec() (data []byte, unreachable bool, err error) {
	fetchOnce.Do(func() {
		fetched, fetchErr = fetchSpec(SpecURL())
	})
	if fetchErr != nil {
		var statusErr *httpStatusError
		if errors.As(fetchErr, &statusErr) {
			return nil, false, fetchErr
		}
		return nil, true, fetchErr
	}
	return fetched, false, nil
}

type httpStatusError struct {
	url    string
	status int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("%s returned HTTP %d", e.url, e.status)
}

func fetchSpec(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &httpStatusError{url: url, status: res.StatusCode}
	}
	return io.ReadAll(res.Body)
}
