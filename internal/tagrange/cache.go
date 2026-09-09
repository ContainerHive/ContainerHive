package tagrange

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"golang.org/x/sync/singleflight"

	"github.com/ContainerHive/ContainerHive/internal/tagrange/source"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// CacheDirEnvVar overrides the resolved cache directory, taking precedence
// over hive.yml's cache_dir. Named to match the existing
// CONTAINER_HIVE_REGISTRY precedent.
const CacheDirEnvVar = "CONTAINER_HIVE_CACHE_DIR"

// defaultTTL is used when neither the source nor hive.yml specify a TTL.
const defaultTTL = time.Hour

// cacheSchemaVersion is bumped whenever the on-disk entry format changes,
// so old entries are orphaned (treated as a miss) rather than misread.
const cacheSchemaVersion = 1

// ResolveCacheDir determines the directory tag_ranges' version cache lives
// in: envValue (the CONTAINER_HIVE_CACHE_DIR value, or "" if unset) wins,
// then hive.yml's cache_dir (resolved relative to projectRoot), then the
// XDG cache directory. hive.yml is parsed before images are discovered, so
// callers pass the value explicitly rather than this function reading the
// environment or config on its own.
func ResolveCacheDir(envValue, hiveConfiguredDir, projectRoot string) (string, error) {
	if envValue != "" {
		return envValue, nil
	}
	if hiveConfiguredDir != "" {
		if filepath.IsAbs(hiveConfiguredDir) {
			return hiveConfiguredDir, nil
		}
		return filepath.Join(projectRoot, hiveConfiguredDir), nil
	}
	dir, err := xdg.CacheFile("containerhive/versions")
	if err != nil {
		return "", fmt.Errorf("failed to resolve the XDG cache directory: %w", err)
	}
	return filepath.Dir(dir), nil
}

// cacheEntry is the on-disk JSON format for one cached fetch.
type cacheEntry struct {
	SchemaVersion int              `json:"schema_version"`
	Source        string           `json:"source"`
	Descriptor    string           `json:"descriptor"`
	FetchedAt     time.Time        `json:"fetched_at"`
	ExpiresAt     time.Time        `json:"expires_at"`
	Versions      []source.Version `json:"versions"`
}

// Cache is a disk-backed, TTL-based cache of fetched versions, safe for
// concurrent use within one process and across processes sharing the same
// directory. It never serves an expired entry - a cold miss with a failed
// fetch is a hard error, by design (see FetchVersions).
type Cache struct {
	dir string
	now func() time.Time

	group singleflight.Group

	mu     sync.Mutex
	memory map[string]cacheEntry

	// unwritable is set once a write to dir fails, so repeated cache misses
	// don't repeatedly retry a filesystem that has already proven
	// unwritable (e.g. a read-only bind mount in CI). Degrading to
	// in-memory-only must never fail a build.
	unwritable bool
}

// NewCache constructs a Cache rooted at dir. dir is created lazily on first
// write, not here, so a cache that's never used never touches the
// filesystem.
func NewCache(dir string) *Cache {
	return &Cache{dir: dir, now: time.Now, memory: make(map[string]cacheEntry)}
}

// entryPath returns the on-disk path for a cache key.
func (c *Cache) entryPath(sourceType, key string) string {
	return filepath.Join(c.dir, fmt.Sprintf("v%d", cacheSchemaVersion), sourceType, key+".json")
}

// cacheKey derives a stable, secret-free cache key from a source's
// CacheKeyParts plus a token fingerprint, so a credential change (which may
// change which versions are visible) invalidates the entry without the
// token itself ever touching disk.
func cacheKey(parts []string, tokenFingerprint string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	h.Write([]byte(tokenFingerprint))
	return hex.EncodeToString(h.Sum(nil))
}

// TokenFingerprint hashes a token for use in cacheKey, so a credential
// change invalidates cached results without ever writing the token itself
// to disk. Returns "" for an empty (anonymous) token.
func TokenFingerprint(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])[:16]
}

// FetchVersions returns the cached versions for cfg if a valid (unexpired,
// well-formed) entry exists, or fetches via src, caches the result, and
// returns it. A cold miss whose fetch also fails is a hard error - this
// cache never serves a stale entry.
func (c *Cache) FetchVersions(ctx context.Context, src source.VersionSource, cfg *model.SourceConfig, tokenFingerprint string) ([]source.Version, error) {
	key := cacheKey(src.CacheKeyParts(cfg), tokenFingerprint)

	if entry, ok := c.load(src.Name(), key); ok {
		return entry.Versions, nil
	}

	result, err, _ := c.group.Do(src.Name()+"/"+key, func() (any, error) {
		// Re-check under the singleflight key: a concurrent caller may have
		// already populated the entry while this one was waiting to enter
		// Do.
		if entry, ok := c.load(src.Name(), key); ok {
			return entry.Versions, nil
		}

		versions, err := src.Fetch(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf(
				"no usable cache entry for %s (cache dir: %s) and fetch failed: %w\n(override the cache directory with %s or hive.yml's cache_dir)",
				src.Descriptor(cfg), c.dir, err, CacheDirEnvVar,
			)
		}

		ttl := defaultTTL
		if cfg.TTL != "" {
			if parsed, perr := time.ParseDuration(cfg.TTL); perr == nil {
				ttl = parsed
			}
		}
		entry := cacheEntry{
			SchemaVersion: cacheSchemaVersion,
			Source:        src.Name(),
			Descriptor:    src.Descriptor(cfg),
			FetchedAt:     c.now(),
			ExpiresAt:     c.now().Add(ttl),
			Versions:      versions,
		}
		c.store(src.Name(), key, entry)
		return versions, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]source.Version), nil
}

// load returns a valid (unexpired, correctly versioned) cache entry, first
// checking the in-memory layer, then falling back to disk. A miss for any
// reason (missing, corrupt, wrong schema, expired) returns ok=false.
func (c *Cache) load(sourceType, key string) (cacheEntry, bool) {
	c.mu.Lock()
	if entry, ok := c.memory[sourceType+"/"+key]; ok {
		c.mu.Unlock()
		if c.valid(entry) {
			return entry, true
		}
		return cacheEntry{}, false
	}
	c.mu.Unlock()

	data, err := os.ReadFile(c.entryPath(sourceType, key))
	if err != nil {
		return cacheEntry{}, false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return cacheEntry{}, false
	}
	if !c.valid(entry) {
		return cacheEntry{}, false
	}

	c.mu.Lock()
	c.memory[sourceType+"/"+key] = entry
	c.mu.Unlock()
	return entry, true
}

func (c *Cache) valid(entry cacheEntry) bool {
	if entry.SchemaVersion != cacheSchemaVersion {
		return false
	}
	return c.now().Before(entry.ExpiresAt)
}

// store writes an entry to the in-memory layer always, and to disk via a
// temp-file-plus-rename (atomic on the same filesystem, so concurrent
// readers never observe a partial write and concurrent writers can't
// corrupt each other) unless the cache directory has already proven
// unwritable this process.
func (c *Cache) store(sourceType, key string, entry cacheEntry) {
	c.mu.Lock()
	c.memory[sourceType+"/"+key] = entry
	unwritable := c.unwritable
	c.mu.Unlock()
	if unwritable {
		return
	}

	path := c.entryPath(sourceType, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		c.markUnwritable()
		return
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return // never fails a build over a marshal error; entry stays in-memory only
	}

	tmp := path + fmt.Sprintf(".tmp.%d", os.Getpid())
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		c.markUnwritable()
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		c.markUnwritable()
	}
}

func (c *Cache) markUnwritable() {
	c.mu.Lock()
	c.unwritable = true
	c.mu.Unlock()
}
