package cache

type BuildkitCache interface {
	Name() string
	ToAttributes() map[string]string
}
