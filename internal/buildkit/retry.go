package buildkit

// Builds retry when the BuildKit connection breaks. Transient network
// failures (broken pipe, connection reset, worker listing timeouts) abort a
// solve even though a retry succeeds on a fresh connection. Retries are
// limited and additive: the build cache makes repeated solves cheap, and
// genuine build failures are never retried.

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/moby/buildkit/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// buildMaxRetries is the number of retries after the first attempt. A build
// therefore runs at most buildMaxRetries+1 times.
const buildMaxRetries = 3

// buildRetryDelays waits between attempts. The last entry repeats for any
// further retry. Tests override it with zero delays.
var buildRetryDelays = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}

// transientErrorSubstrings match connection-level failures that gRPC or the
// http transport wrap into plain errors.
var transientErrorSubstrings = []string{
	"broken pipe",
	"connection reset by peer",
	"connection refused",
	"connection closed",
	"transport is closing",
	"use of closed network connection",
	"unexpected eof",
	"failed to list workers",
	"failed to dial",
	"no route to host",
	"i/o timeout",
}

// retryTransient runs build until it succeeds, fails with a non-transient
// error, or exhausts the retries. reset runs before every retry so a broken
// connection is replaced. A cancelled context stops the loop immediately.
func retryTransient(ctx context.Context, build func() error, reset func()) error {
	var lastErr error
	for attempt := 0; attempt <= buildMaxRetries; attempt++ {
		if attempt > 0 {
			delay := buildRetryDelay(attempt)
			slog.Warn("BuildKit connection failed, retrying build",
				"attempt", attempt+1, "maxAttempts", buildMaxRetries+1, "retryIn", delay, "error", lastErr)
			select {
			case <-ctx.Done():
				return errors.Join(lastErr, ctx.Err())
			case <-time.After(delay):
			}
			if reset != nil {
				reset()
			}
		}

		lastErr = build()
		if lastErr == nil {
			return nil
		}
		if !isTransientBuildkitError(lastErr) || ctx.Err() != nil {
			return lastErr
		}
	}
	return lastErr
}

// buildRetryDelay returns the wait before the given retry attempt.
func buildRetryDelay(attempt int) time.Duration {
	index := attempt - 1
	if index >= len(buildRetryDelays) {
		index = len(buildRetryDelays) - 1
	}
	return buildRetryDelays[index]
}

// isTransientBuildkitError reports whether err comes from a broken or
// unavailable BuildKit connection rather than from the build itself.
func isTransientBuildkitError(err error) bool {
	if err == nil {
		return false
	}
	// A cancelled or timed out build is a caller decision, not a flake.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// gRPC reports broken connections as Unavailable, including errors wrapped
	// by the BuildKit client (for example worker listing).
	if status.Code(err) == codes.Unavailable {
		return true
	}

	message := strings.ToLower(err.Error())
	for _, transient := range transientErrorSubstrings {
		if strings.Contains(message, transient) {
			return true
		}
	}
	return false
}

// resetConnection replaces the gRPC connection after a transient failure. A
// client without an endpoint (in-memory connections) is left untouched.
func (c *Client) resetConnection(ctx context.Context) {
	if c.endpoint == "" {
		return
	}
	if err := c.buildkit.Close(); err != nil {
		slog.Debug("Closing broken BuildKit connection failed", "error", err)
	}
	fresh, err := client.New(ctx, c.endpoint)
	if err != nil {
		slog.Warn("Failed to reconnect to BuildKit", "endpoint", c.endpoint, "error", err)
		return
	}
	c.buildkit = fresh
}
