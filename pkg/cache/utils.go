package cache

import (
	"fmt"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// BuildCacheFromConfig creates a BuildkitCache from the hive.yml cache config,
// using the type field to discriminate between s3 and registry backends.
func BuildCacheFromConfig(cfg *model.CacheConfig, cacheKey string) (BuildkitCache, error) {
	if cfg == nil {
		return nil, nil
	}

	switch cfg.Type {
	case "s3":
		return &S3BuildKitCache{
			EndpointUrl:     cfg.Endpoint,
			Bucket:          cfg.Bucket,
			Region:          cfg.Region,
			AccessKeyId:     cfg.AccessKeyId,
			SecretAccessKey: cfg.SecretAccessKey,
			UsePathStyle:    cfg.UsePathStyle,
			CacheKey:        cacheKey,
		}, nil
	case "registry":
		return &RegistryCache{
			CacheRef: cfg.Ref,
			Insecure: cfg.Insecure,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported cache type %q (supported: s3, registry)", cfg.Type)
	}
}
