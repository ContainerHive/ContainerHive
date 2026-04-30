package cache

import (
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
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
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if cache != nil {
					t.Errorf("expected nil cache, got %v", cache)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tt.expectType {
			case "nil":
				if cache != nil {
					t.Errorf("expected nil cache, got %v", cache)
				}
			case "s3":
				s3Cache, ok := cache.(*S3BuildKitCache)
				if !ok {
					t.Fatalf("expected *S3BuildKitCache, got %T", cache)
				}
				if s3Cache.EndpointUrl != "https://s3.example.com" {
					t.Errorf("EndpointUrl = %q, want %q", s3Cache.EndpointUrl, "https://s3.example.com")
				}
				if s3Cache.Bucket != "my-bucket" {
					t.Errorf("Bucket = %q, want %q", s3Cache.Bucket, "my-bucket")
				}
				if s3Cache.Region != "us-west-2" {
					t.Errorf("Region = %q, want %q", s3Cache.Region, "us-west-2")
				}
				if s3Cache.AccessKeyId != "test-access-key" {
					t.Errorf("AccessKeyId = %q, want %q", s3Cache.AccessKeyId, "test-access-key")
				}
				if s3Cache.SecretAccessKey != "test-secret-key" {
					t.Errorf("SecretAccessKey = %q, want %q", s3Cache.SecretAccessKey, "test-secret-key")
				}
				if !s3Cache.UsePathStyle {
					t.Error("expected UsePathStyle to be true")
				}
				if s3Cache.CacheKey != "test-key" {
					t.Errorf("CacheKey = %q, want %q", s3Cache.CacheKey, "test-key")
				}
			case "registry":
				regCache, ok := cache.(*RegistryCache)
				if !ok {
					t.Fatalf("expected *RegistryCache, got %T", cache)
				}
				if regCache.CacheRef != "my-registry.example.com/my-repo:cache" {
					t.Errorf("CacheRef = %q, want %q", regCache.CacheRef, "my-registry.example.com/my-repo:cache")
				}
				if !regCache.Insecure {
					t.Error("expected Insecure to be true")
				}
			}
		})
	}
}
