package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ContainerHive/ContainerHive/internal/buildinfo"
)

// httpClient is shared across all sources, mirroring the single
// package-level client already used by pkg/mcp/search.go.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// maxResponseBytes caps how much of a response body is read, so a
// misbehaving or malicious endpoint can't exhaust memory during discovery.
const maxResponseBytes = 32 << 20 // 32 MiB

const maxRetries = 3

// getJSON issues an authenticated, retried GET and decodes the JSON body
// into out. headers are expanded with os.ExpandEnv so values like
// "Bearer $GITHUB_TOKEN" work without embedding the secret in image.yml.
func getJSON(ctx context.Context, url string, headers map[string]string, out any) error {
	body, err := getBody(ctx, url, headers)
	if err != nil {
		return err
	}
	defer body.Close()
	if err := json.NewDecoder(io.LimitReader(body, maxResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return nil
}

// getBody issues an authenticated, retried GET and returns the response
// body for the caller to read and close.
func getBody(ctx context.Context, url string, headers map[string]string) (io.ReadCloser, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepBackoff(ctx, attempt, lastErr); err != nil {
				return nil, err
			}
		}

		resp, err := doGet(ctx, url, headers)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}

		lastErr = classifyStatus(url, resp)
		resp.Body.Close()
		if !isRetryable(resp.StatusCode) {
			return nil, lastErr
		}
	}
	return nil, fmt.Errorf("giving up after %d attempts: %w", maxRetries, lastErr)
}

func doGet(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ContainerHive/"+buildinfo.Version)
	for k, v := range headers {
		req.Header.Set(k, os.ExpandEnv(v))
	}
	return httpClient.Do(req)
}

func isRetryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// retryAfterError carries a server-requested retry delay (from a
// Retry-After header) so sleepBackoff can honor it instead of guessing.
type retryAfterError struct {
	msg   string
	delay time.Duration
}

func (e *retryAfterError) Error() string { return e.msg }

func classifyStatus(url string, resp *http.Response) error {
	limited := io.LimitReader(resp.Body, 2048)
	snippet, _ := io.ReadAll(limited)
	msg := fmt.Sprintf("%s: unexpected status %d: %s", url, resp.StatusCode, snippet)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		msg = fmt.Sprintf("%s: %d from %s (check credentials/token_env): %s", url, resp.StatusCode, resp.Request.Host, snippet)
	}
	if delay, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
		return &retryAfterError{msg: msg, delay: delay}
	}
	return errors.New(msg)
}

// sleepBackoff waits before a retry, honoring a Retry-After header carried
// on lastErr when present, otherwise falling back to jittered exponential
// backoff. Returns ctx.Err() if the context is done first, so a canceled
// discovery run doesn't hang in a retry sleep.
func sleepBackoff(ctx context.Context, attempt int, lastErr error) error {
	delay := backoffDelay(attempt)
	var retryAfter *retryAfterError
	if errors.As(lastErr, &retryAfter) {
		delay = retryAfter.delay
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func backoffDelay(attempt int) time.Duration {
	base := 500 * time.Millisecond * time.Duration(1<<uint(attempt-1))
	if base > 5*time.Second {
		base = 5 * time.Second
	}
	jitter := time.Duration(rand.Int64N(int64(base) / 2))
	return base + jitter
}

// parseRetryAfter parses an HTTP Retry-After header value (seconds).
func parseRetryAfter(v string) (time.Duration, bool) {
	seconds, err := strconv.Atoi(v)
	if err != nil || seconds < 0 {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}
