package cache

// BuildkitCache abstracts a BuildKit cache backend for import and export.
type BuildkitCache interface {
	Name() string
	ToAttributes() map[string]string

	// WithScope returns a new BuildkitCache with the cache ref/key scoped
	// to the given scope string (e.g. "<image>.<tag>.<platform>").
	// This allows parallel builds to use isolated cache refs.
	WithScope(scope string) BuildkitCache
}
