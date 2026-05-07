package cache

import (
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestSanitizeTagSuffix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple alphanumeric",
			input: "ubuntu",
			want:  "ubuntu",
		},
		{
			name:  "dots preserved",
			input: "22.04",
			want:  "22.04",
		},
		{
			name:  "slashes become underscores",
			input: "linux/amd64",
			want:  "linux_amd64",
		},
		{
			name:  "mixed valid chars",
			input: "python.3.11-slim",
			want:  "python.3.11-slim",
		},
		{
			name:  "full scope with platform",
			input: "ubuntu.22.04.linux/amd64",
			want:  "ubuntu.22.04.linux_amd64",
		},
		{
			name:  "multiple slashes",
			input: "ubuntu.22.04.linux/arm/v7",
			want:  "ubuntu.22.04.linux_arm_v7",
		},
		{
			name:  "leading dot trimmed",
			input: ".ubuntu",
			want:  "ubuntu",
		},
		{
			name:  "trailing dot trimmed",
			input: "ubuntu.",
			want:  "ubuntu",
		},
		{
			name:  "leading hyphen trimmed",
			input: "-ubuntu",
			want:  "ubuntu",
		},
		{
			name:  "only invalid chars become underscores",
			input: "!!!@@@",
			want:  "______",
		},
		{
			name:  "empty string returns unnamed",
			input: "",
			want:  "unnamed",
		},
		{
			name:  "slashes and dots and hyphens mixed",
			input: "alpine.3.20/aarch64-edge",
			want:  "alpine.3.20_aarch64-edge",
		},
		{
			name:  "spaces replaced with underscores",
			input: "my image",
			want:  "my_image",
		},
		{
			name:  "special chars replaced",
			input: "foo@bar#baz",
			want:  "foo_bar_baz",
		},
		{
			name:  "long input truncated",
			input: "aaaaaaaaaa.bbbbbbbbbb.cccccccccc.dddddddddd.eeeeeeeeee.ffffffffff.gggggggggg.hhhhhhhhhh.iiiiiiiiii.jjjjjjjjjj.kkkkkkkkkk.llllllllll",
			want:  "aaaaaaaaaa.bbbbbbbbbb.cccccccccc.dddddddddd.eeeeeeeeee.ffffffffff.gggggggggg.hhhhhhhhhh.iiiiiiiiii.jjjjjjjjjj.kkkkkkkkkk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeTagSuffix(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeTagSuffix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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
