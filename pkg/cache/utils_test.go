package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func TestBuildCacheFromConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     *model.CacheConfig
		cacheKey   string
		wantErr    bool
		expectType string
	}{
		{
			name:       "nil config",
			config:     nil,
			cacheKey:   "test-key",
			wantErr:    false,
			expectType: "nil",
		},
		{
			name: "s3 cache",
			config: &model.CacheConfig{
				Type:            "s3",
				Endpoint:        "https://s3.example.com",
				Bucket:          "my-bucket",
				Region:          "us-west-2",
				AccessKeyId:     "test-access-key",
				SecretAccessKey: "test-secret-key",
				UsePathStyle:    true,
			},
			cacheKey:   "test-key",
			wantErr:    false,
			expectType: "s3",
		},
		{
			name: "registry cache",
			config: &model.CacheConfig{
				Type:     "registry",
				Ref:      "my-registry.example.com/my-repo:cache",
				Insecure: true,
			},
			cacheKey:   "test-key",
			wantErr:    false,
			expectType: "registry",
		},
		{
			name: "unsupported cache type",
			config: &model.CacheConfig{
				Type: "filesystem",
			},
			cacheKey:   "test-key",
			wantErr:    true,
			expectType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := BuildCacheFromConfig(tt.config, tt.cacheKey)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cache)
				return
			}

			require.NoError(t, err)

			switch tt.expectType {
			case "nil":
				assert.Nil(t, cache)
			case "s3":
				s3Cache, ok := cache.(*S3BuildKitCache)
				require.True(t, ok)
				assert.Equal(t, "https://s3.example.com", s3Cache.EndpointUrl)
				assert.Equal(t, "my-bucket", s3Cache.Bucket)
				assert.Equal(t, "us-west-2", s3Cache.Region)
				assert.Equal(t, "test-access-key", s3Cache.AccessKeyId)
				assert.Equal(t, "test-secret-key", s3Cache.SecretAccessKey)
				assert.True(t, s3Cache.UsePathStyle)
				assert.Equal(t, "test-key", s3Cache.CacheKey)
			case "registry":
				regCache, ok := cache.(*RegistryCache)
				require.True(t, ok)
				assert.Equal(t, "my-registry.example.com/my-repo:cache", regCache.CacheRef)
				assert.True(t, regCache.Insecure)
			}
		})
	}
}
