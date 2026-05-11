package cache

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// sanitizeTagSuffix prepares a scope string for use as an OCI tag suffix.
// It replaces invalid characters with underscores and ensures the result
// conforms to [a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}.
func sanitizeTagSuffix(s string) string {
	sanitized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '_'
	}, s)
	sanitized = strings.TrimLeft(sanitized, ".-")
	sanitized = strings.TrimRight(sanitized, ".-")
	if len(sanitized) > 120 {
		sanitized = sanitized[:120]
	}
	if sanitized == "" {
		return "unnamed"
	}
	return sanitized
}

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
