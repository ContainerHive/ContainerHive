package discovery

import (
	"testing"
)

func TestParseHiveConfigFile(t *testing.T) {
	config, err := parseHiveConfigFile("../testdata/config-project/hive.yml")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	t.Run("buildkit config", func(t *testing.T) {
		if config.BuildKit == nil {
			t.Fatal("expected buildkit config to be set")
		}
		if config.BuildKit.Address != "tcp://127.0.0.1:8502" {
			t.Errorf("expected address tcp://127.0.0.1:8502, got %q", config.BuildKit.Address)
		}
	})

	t.Run("cache config", func(t *testing.T) {
		if config.Cache == nil {
			t.Fatal("expected cache config to be set")
		}
		if config.Cache.Type != "s3" {
			t.Errorf("expected type s3, got %q", config.Cache.Type)
		}
		if config.Cache.Endpoint != "http://localhost:39505" {
			t.Errorf("expected endpoint http://localhost:39505, got %q", config.Cache.Endpoint)
		}
		if config.Cache.Bucket != "buildkit-cache" {
			t.Errorf("expected bucket buildkit-cache, got %q", config.Cache.Bucket)
		}
		if config.Cache.Region != "garage" {
			t.Errorf("expected region garage, got %q", config.Cache.Region)
		}
		if config.Cache.AccessKeyId != "AKTEST" {
			t.Errorf("expected access_key_id AKTEST, got %q", config.Cache.AccessKeyId)
		}
		if config.Cache.SecretAccessKey != "secrettest" {
			t.Errorf("expected secret_access_key secrettest, got %q", config.Cache.SecretAccessKey)
		}
		if !config.Cache.UsePathStyle {
			t.Error("expected use_path_style to be true")
		}
	})

	t.Run("registry config", func(t *testing.T) {
		if config.Registry == nil {
			t.Fatal("expected registry config to be set")
		}
		if config.Registry.Address != "localhost:5000" {
			t.Errorf("expected address localhost:5000, got %q", config.Registry.Address)
		}
	})

	t.Run("template_options config", func(t *testing.T) {
		if len(config.TemplateOptions) != 2 {
			t.Fatalf("expected 2 template options, got %d", len(config.TemplateOptions))
		}
		if config.TemplateOptions["ci_buildkit_image"] != "registry.io/buildkit" {
			t.Errorf("expected ci_buildkit_image 'registry.io/buildkit', got %q", config.TemplateOptions["ci_buildkit_image"])
		}
		if config.TemplateOptions["foo_bar"] != "stum" {
			t.Errorf("expected foo_bar 'stum', got %q", config.TemplateOptions["foo_bar"])
		}
	})
}

func TestParseHiveConfigFile_Empty(t *testing.T) {
	config, err := parseHiveConfigFile("../testdata/minimal-project/hive.yml")
	if err != nil {
		t.Fatalf("failed to parse empty config: %v", err)
	}

	if config.BuildKit != nil {
		t.Error("expected buildkit config to be nil for empty file")
	}
	if config.Cache != nil {
		t.Error("expected cache config to be nil for empty file")
	}
	if config.Registry != nil {
		t.Error("expected registry config to be nil for empty file")
	}
}

func TestDiscoverProject_WithConfig(t *testing.T) {
	project, err := DiscoverProject(t.Context(), "../testdata/config-project")
	if err != nil {
		t.Fatalf("failed to discover project: %v", err)
	}

	if project.Config.BuildKit == nil {
		t.Fatal("expected project config to have buildkit")
	}
	if project.Config.BuildKit.Address != "tcp://127.0.0.1:8502" {
		t.Errorf("expected buildkit address tcp://127.0.0.1:8502, got %q", project.Config.BuildKit.Address)
	}
}
