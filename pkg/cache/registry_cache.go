package cache

// RegistryCache implements BuildkitCache using a container registry as the cache backend.
type RegistryCache struct {
	CacheRef string
	Insecure bool
}

func (r RegistryCache) Name() string {
	return "registry"
}

func (r RegistryCache) WithScope(scope string) BuildkitCache {
	return &RegistryCache{
		CacheRef: r.CacheRef + "." + sanitizeTagSuffix(scope),
		Insecure: r.Insecure,
	}
}

func (r RegistryCache) ToAttributes() map[string]string {
	attrs := map[string]string{
		"mode":           "max",
		"ref":            r.CacheRef,
		"image-manifest": "true",
		"oci-mediatypes": "true",
	}
	if r.Insecure {
		attrs["registry.insecure"] = "true"
	}
	return attrs
}
