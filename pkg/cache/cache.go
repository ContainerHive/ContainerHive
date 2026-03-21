package cache

// BuildkitCache abstracts a BuildKit cache backend for import and export.
type BuildkitCache interface {
	Name() string
	ToAttributes() map[string]string
}
