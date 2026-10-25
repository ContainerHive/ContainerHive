package buildkit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsTransientBuildkitError(t *testing.T) {
	tests := map[string]struct {
		err  error
		want bool
	}{
		"nil":                    {err: nil},
		"grpc unavailable":       {err: status.Error(codes.Unavailable, "connection closed"), want: true},
		"wrapped unavailable":    {err: fmt.Errorf("listing workers for Build: failed to list workers: %w", status.Error(codes.Unavailable, "write: broken pipe")), want: true},
		"broken pipe":            {err: errors.New("write tcp 172.17.0.2:33652->172.17.0.1:32778: write: broken pipe"), want: true},
		"connection reset":       {err: errors.New("read tcp: connection reset by peer"), want: true},
		"eof":                    {err: io.EOF, want: true},
		"unexpected eof":         {err: fmt.Errorf("reading response: %w", io.ErrUnexpectedEOF), want: true},
		"worker listing timeout": {err: errors.New("listing workers for Build: failed to list workers: context deadline exceeded"), want: true},
		"build failure":          {err: errors.New(`failed to solve: process "/bin/sh -euc chisel cut" did not complete successfully: exit code: 1`), want: false},
		"invalid argument":       {err: status.Error(codes.InvalidArgument, "invalid reference"), want: false},
		"cancelled":              {err: fmt.Errorf("build aborted: %w", context.Canceled), want: false},
		"deadline exceeded":      {err: fmt.Errorf("build aborted: %w", context.DeadlineExceeded), want: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := isTransientBuildkitError(tc.err); got != tc.want {
				t.Errorf("isTransientBuildkitError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// zeroRetryDelays removes the production backoff for tests and restores it
// afterwards.
func zeroRetryDelays(t *testing.T) {
	t.Helper()
	previous := buildRetryDelays
	buildRetryDelays = []time.Duration{0, 0, 0}
	t.Cleanup(func() { buildRetryDelays = previous })
}

func TestRetryTransientSucceedsAfterFailures(t *testing.T) {
	zeroRetryDelays(t)

	attempts := 0
	resets := 0
	build := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("write tcp: broken pipe")
		}
		return nil
	}

	err := retryTransient(context.Background(), build, func() { resets++ })
	if err != nil {
		t.Fatalf("retryTransient() = %v, want nil", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if resets != 2 {
		t.Errorf("resets = %d, want 2", resets)
	}
}

func TestRetryTransientGivesUp(t *testing.T) {
	zeroRetryDelays(t)

	attempts := 0
	build := func() error {
		attempts++
		return status.Error(codes.Unavailable, "connection reset by peer")
	}

	err := retryTransient(context.Background(), build, nil)
	if err == nil {
		t.Fatal("retryTransient() = nil, want the last connection error")
	}
	if want := buildMaxRetries + 1; attempts != want {
		t.Errorf("attempts = %d, want %d", attempts, want)
	}
}

func TestRetryTransientDoesNotRetryBuildFailures(t *testing.T) {
	zeroRetryDelays(t)

	attempts := 0
	buildErr := errors.New("failed to solve: exit code: 1")
	build := func() error {
		attempts++
		return buildErr
	}

	err := retryTransient(context.Background(), build, nil)
	if !errors.Is(err, buildErr) {
		t.Errorf("retryTransient() = %v, want the build error", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryTransientStopsOnCancelledContext(t *testing.T) {
	zeroRetryDelays(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	build := func() error {
		attempts++
		return errors.New("write tcp: broken pipe")
	}

	err := retryTransient(ctx, build, nil)
	if err == nil {
		t.Fatal("retryTransient() = nil, want an error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestBuildRetryDelay(t *testing.T) {
	previous := buildRetryDelays
	buildRetryDelays = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}
	t.Cleanup(func() { buildRetryDelays = previous })

	tests := map[int]time.Duration{
		1: 2 * time.Second,
		2: 5 * time.Second,
		3: 10 * time.Second,
		4: 10 * time.Second,
	}
	for attempt, want := range tests {
		if got := buildRetryDelay(attempt); got != want {
			t.Errorf("buildRetryDelay(%d) = %v, want %v", attempt, got, want)
		}
	}
}
