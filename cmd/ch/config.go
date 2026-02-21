package main

import (
	"github.com/timo-reymann/ContainerHive/pkg/cache"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// buildCacheFromConfig creates a BuildkitCache from the hive.yml cache config.
func buildCacheFromConfig(cfg *model.CacheConfig, cacheKey string) cache.BuildkitCache {
	if cfg == nil {
		return nil
	}
	return &cache.S3BuildKitCache{
		EndpointUrl:    cfg.Endpoint,
		Bucket:         cfg.Bucket,
		Region:         cfg.Region,
		AccessKeyId:    cfg.AccessKeyId,
		SecretAccessKey: cfg.SecretAccessKey,
		UsePathStyle:   cfg.UsePathStyle,
		CacheKey:       cacheKey,
	}
}
