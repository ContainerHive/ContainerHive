package tagrange

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ContainerHive/ContainerHive/internal/tagrange/source"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// countingSource wraps a fetch function and counts how many times Fetch is
// actually called, so tests can assert a warm cache performs zero network
// calls.
type countingSource struct {
	fetch func(ctx context.Context) ([]source.Version, error)
	calls atomic.Int64
}

func (*countingSource) Name() string                              { return "test" }
func (*countingSource) Validate(*model.SourceConfig) error         { return nil }
func (*countingSource) Descriptor(*model.SourceConfig) string      { return "test source" }
func (*countingSource) CacheKeyParts(*model.SourceConfig) []string { return []string{"test"} }
func (s *countingSource) Fetch(ctx context.Context, _ *model.SourceConfig) ([]source.Version, error) {
	s.calls.Add(1)
	return s.fetch(ctx)
}

func TestCache_MissThenHit(t *testing.T) {
	c := NewCache(t.TempDir())
	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		return []source.Version{{Raw: "1.0.0"}}, nil
	}}
	cfg := &model.SourceConfig{Type: "test"}

	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error on first fetch: %v", err)
	}
	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error on second fetch: %v", err)
	}
	if got := src.calls.Load(); got != 1 {
		t.Errorf("expected exactly 1 underlying fetch, got %d", got)
	}
}

func TestCache_ExpiredIsAMissNotStale(t *testing.T) {
	c := NewCache(t.TempDir())
	now := time.Now()
	c.now = func() time.Time { return now }

	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		return []source.Version{{Raw: "1.0.0"}}, nil
	}}
	cfg := &model.SourceConfig{Type: "test", TTL: "1h"}

	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Advance past the TTL and make the next fetch fail, to prove the
	// expired entry is never served as a fallback.
	now = now.Add(2 * time.Hour)
	src.fetch = func(ctx context.Context) ([]source.Version, error) {
		return nil, errors.New("network down")
	}
	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err == nil {
		t.Error("expected an error: the cache must not serve an expired entry on a cold-miss fetch failure")
	}
}

func TestCache_ColdMissFetchFailureIsHardError(t *testing.T) {
	c := NewCache(t.TempDir())
	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		return nil, errors.New("dial tcp: no such host")
	}}
	_, err := c.FetchVersions(context.Background(), src, &model.SourceConfig{Type: "test"}, "")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no usable cache entry") {
		t.Errorf("expected an actionable error message, got: %v", err)
	}
}

func TestCache_CorruptFileIsAMiss(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		return []source.Version{{Raw: "1.0.0"}}, nil
	}}
	cfg := &model.SourceConfig{Type: "test"}

	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	key := cacheKey(src.CacheKeyParts(cfg), "")
	path := c.entryPath(src.Name(), key)
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Force a re-read from disk by using a second Cache instance sharing
	// the directory (bypasses the in-memory layer).
	c2 := NewCache(dir)
	if _, err := c2.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := src.calls.Load(); got != 2 {
		t.Errorf("expected the corrupt entry to be treated as a miss (2 total fetches), got %d", got)
	}
}

func TestCache_SchemaVersionMismatchIsAMiss(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		return []source.Version{{Raw: "1.0.0"}}, nil
	}}
	cfg := &model.SourceConfig{Type: "test"}

	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	key := cacheKey(src.CacheKeyParts(cfg), "")
	path := c.entryPath(src.Name(), key)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bumped := []byte(strings.Replace(string(data), `"schema_version": 1`, `"schema_version": 99`, 1))
	if err := os.WriteFile(path, bumped, 0o644); err != nil {
		t.Fatal(err)
	}

	c2 := NewCache(dir)
	if _, err := c2.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := src.calls.Load(); got != 2 {
		t.Errorf("expected the schema-mismatched entry to be treated as a miss, got %d fetches", got)
	}
}

func TestCache_ConcurrentFetchesAreSingleFlighted(t *testing.T) {
	c := NewCache(t.TempDir())
	var callCount atomic.Int64
	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		callCount.Add(1)
		time.Sleep(20 * time.Millisecond) // widen the race window
		return []source.Version{{Raw: "1.0.0"}}, nil
	}}
	cfg := &model.SourceConfig{Type: "test"}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := callCount.Load(); got != 1 {
		t.Errorf("expected exactly 1 underlying fetch across 20 concurrent callers, got %d", got)
	}
}

func TestCache_UnwritableDirDegradesToInMemory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: permission bits don't restrict writes")
	}
	dir := t.TempDir()
	readOnlySub := filepath.Join(dir, "ro")
	if err := os.Mkdir(readOnlySub, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlySub, 0o700) })

	c := NewCache(filepath.Join(readOnlySub, "versions"))
	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		return []source.Version{{Raw: "1.0.0"}}, nil
	}}
	cfg := &model.SourceConfig{Type: "test"}

	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("an unwritable cache dir must not fail the fetch: %v", err)
	}
	// Second call still hits the in-memory layer, not the source again.
	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := src.calls.Load(); got != 1 {
		t.Errorf("expected the in-memory layer to serve the second call, got %d fetches", got)
	}
}

func TestCache_ForceRefresh_IgnoresWarmEntry(t *testing.T) {
	c := NewCache(t.TempDir())
	src := &countingSource{fetch: func(ctx context.Context) ([]source.Version, error) {
		return []source.Version{{Raw: "1.0.0"}}, nil
	}}
	cfg := &model.SourceConfig{Type: "test"}

	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := src.calls.Load(); got != 1 {
		t.Fatalf("expected 1 fetch before forcing refresh, got %d", got)
	}

	c.SetForceRefresh(true)
	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := src.calls.Load(); got != 2 {
		t.Errorf("expected forceRefresh to trigger a second fetch despite a warm entry, got %d calls", got)
	}

	// The refetched result is still recorded, so a subsequent non-forced
	// call is served from the cache again.
	c.SetForceRefresh(false)
	if _, err := c.FetchVersions(context.Background(), src, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := src.calls.Load(); got != 2 {
		t.Errorf("expected the refetched result to warm the cache, got %d calls", got)
	}
}

func TestResolveCacheDir_Precedence(t *testing.T) {
	t.Run("env wins over hive.yml", func(t *testing.T) {
		got, err := ResolveCacheDir("/env/dir", "/hive/dir", "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/env/dir" {
			t.Errorf("got %q, want /env/dir", got)
		}
	})

	t.Run("hive.yml wins over XDG when env unset", func(t *testing.T) {
		got, err := ResolveCacheDir("", "myCacheDir", "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != filepath.Join("/project", "myCacheDir") {
			t.Errorf("got %q, want /project/myCacheDir", got)
		}
	})

	t.Run("hive.yml absolute path is used as-is", func(t *testing.T) {
		got, err := ResolveCacheDir("", "/abs/cache", "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/abs/cache" {
			t.Errorf("got %q, want /abs/cache", got)
		}
	})

	t.Run("falls back to XDG cache dir", func(t *testing.T) {
		got, err := ResolveCacheDir("", "", "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected a non-empty XDG-derived path")
		}
	})
}

func TestTokenFingerprint(t *testing.T) {
	if TokenFingerprint("") != "" {
		t.Error("expected an empty fingerprint for an anonymous (empty) token")
	}
	a := TokenFingerprint("token-a")
	b := TokenFingerprint("token-b")
	if a == b {
		t.Error("expected different tokens to produce different fingerprints")
	}
	if TokenFingerprint("token-a") != a {
		t.Error("expected the same token to produce a stable fingerprint")
	}
}
