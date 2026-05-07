package cache

import "testing"

func TestS3BuildKitCache_Name(t *testing.T) {
	s := &S3BuildKitCache{}
	if s.Name() != "s3" {
		t.Errorf("expected 's3', got %q", s.Name())
	}
}

func TestS3BuildKitCache_WithScope(t *testing.T) {
	s := &S3BuildKitCache{
		EndpointUrl:     "http://localhost:9000",
		Bucket:          "my-bucket",
		Region:          "us-east-1",
		AccessKeyId:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
		CacheKey:        "my-cache",
	}

	scoped := s.WithScope("ubuntu.22.04.linux/amd64")
	s3scoped, ok := scoped.(*S3BuildKitCache)
	if !ok {
		t.Fatalf("expected *S3BuildKitCache, got %T", scoped)
	}

	if s3scoped.CacheKey != "my-cache-ubuntu.22.04.linux/amd64" {
		t.Errorf("CacheKey = %q, want %q", s3scoped.CacheKey, "my-cache-ubuntu.22.04.linux/amd64")
	}
	if s3scoped.EndpointUrl != s.EndpointUrl {
		t.Error("WithScope should preserve EndpointUrl")
	}
	if s3scoped.Bucket != s.Bucket {
		t.Error("WithScope should preserve Bucket")
	}
}

func TestS3BuildKitCache_WithScope_PreservesOriginal(t *testing.T) {
	s := &S3BuildKitCache{
		CacheKey: "my-cache",
	}

	_ = s.WithScope("some.scope")

	if s.CacheKey != "my-cache" {
		t.Error("WithScope should not mutate the original cache")
	}
}

func TestS3BuildKitCache_ToAttributes(t *testing.T) {
	s := &S3BuildKitCache{
		EndpointUrl:     "http://localhost:9000",
		Bucket:          "my-bucket",
		Region:          "us-east-1",
		AccessKeyId:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
		CacheKey:        "my-cache",
	}

	attrs := s.ToAttributes()

	tests := []struct {
		key      string
		expected string
	}{
		{"endpoint_url", "http://localhost:9000"},
		{"bucket", "my-bucket"},
		{"region", "us-east-1"},
		{"access_key_id", "test-key"},
		{"secret_access_key", "test-secret"},
		{"use_path_style", "true"},
		{"mode", "max"},
		{"name", "my-cache"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got, ok := attrs[tt.key]; !ok {
				t.Errorf("missing key %q", tt.key)
			} else if got != tt.expected {
				t.Errorf("for key %q: expected %q, got %q", tt.key, tt.expected, got)
			}
		})
	}
}
