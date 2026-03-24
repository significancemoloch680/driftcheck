package agentlock

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func runCanary(ctx context.Context, canary Canary) (CanaryResult, error) {
	expected := canary.ExpectedStatus
	if expected == 0 {
		expected = http.StatusOK
	}

	timeout := time.Duration(canary.TimeoutMillis) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, canary.URL, nil)
	if err != nil {
		return CanaryResult{}, fmt.Errorf("build canary request failed: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return CanaryResult{
			Name:           canary.Name,
			URL:            canary.URL,
			ExpectedStatus: expected,
			Healthy:        false,
			DurationMillis: time.Since(start).Milliseconds(),
			Error:          err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == expected

	return CanaryResult{
		Name:           canary.Name,
		URL:            canary.URL,
		ExpectedStatus: expected,
		StatusCode:     resp.StatusCode,
		Healthy:        healthy,
		DurationMillis: time.Since(start).Milliseconds(),
	}, nil
}
