package cli

import (
	"context"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/shard"
	"github.com/urfave/cli/v3"
)

// runResolveShard drives resolveShard through a real cli.Command so flag
// parsing (including env-var coercion) is exercised, not just the function
// body. A malformed env value such as CI_NODE_TOTAL=not-a-number fails
// during urfave/cli's flag parsing, before the Action ever runs - so
// cmd.Run's error is the one callers should assert on, not just resolveErr.
func runResolveShard(t *testing.T, argv []string) (shard.Shard, error) {
	t.Helper()

	var got shard.Shard
	var resolveErr error

	cmd := &cli.Command{
		Flags: shardFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			got, resolveErr = resolveShard(cmd)
			return resolveErr
		},
	}

	if err := cmd.Run(t.Context(), argv); err != nil {
		return got, err
	}
	return got, resolveErr
}

func TestResolveShard_Defaults(t *testing.T) {
	got, err := runResolveShard(t, []string{"ch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := shard.Shard{Current: 1, Max: 1}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got.Enabled() {
		t.Error("default shard should be disabled")
	}
}

func TestResolveShard_ExplicitFlags(t *testing.T) {
	got, err := runResolveShard(t, []string{"ch", "--max-shards", "5", "--current-shard", "2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := shard.Shard{Current: 2, Max: 5}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveShard_EnvOnly(t *testing.T) {
	t.Setenv("CI_NODE_TOTAL", "4")
	t.Setenv("CI_NODE_INDEX", "3")

	got, err := runResolveShard(t, []string{"ch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := shard.Shard{Current: 3, Max: 4}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveShard_FlagOverridesEnv(t *testing.T) {
	t.Setenv("CI_NODE_TOTAL", "4")
	t.Setenv("CI_NODE_INDEX", "3")

	got, err := runResolveShard(t, []string{"ch", "--max-shards", "10", "--current-shard", "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := shard.Shard{Current: 1, Max: 10}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveShard_InvalidEnv(t *testing.T) {
	t.Setenv("CI_NODE_TOTAL", "not-a-number")

	if _, err := runResolveShard(t, []string{"ch"}); err == nil {
		t.Error("expected an error for a non-integer CI_NODE_TOTAL, got nil")
	}
}

func TestResolveShard_CINodeTotalOneDisables(t *testing.T) {
	t.Setenv("CI_NODE_TOTAL", "1")
	t.Setenv("CI_NODE_INDEX", "1")

	got, err := runResolveShard(t, []string{"ch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Enabled() {
		t.Error("CI_NODE_TOTAL=1 should mean sharding is disabled")
	}
}

func TestResolveShard_CurrentGreaterThanMax(t *testing.T) {
	_, err := runResolveShard(t, []string{"ch", "--max-shards", "3", "--current-shard", "4"})
	if err == nil {
		t.Error("expected an error when --current-shard exceeds --max-shards, got nil")
	}
}
