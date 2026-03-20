package build

import (
	"testing"

	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func TestResolveTagConfig(t *testing.T) {
	image := &model.Image{
		Name:      "test",
		Versions:  model.Versions{"go": "1.21"},
		BuildArgs: model.BuildArgs{"MODE": "prod"},
		Secrets:   model.Secrets{},
	}
	tag := &model.Tag{
		Name:      "latest",
		Versions:  model.Versions{"node": "20"},
		BuildArgs: model.BuildArgs{"DEBUG": "false"},
	}

	config, err := ResolveTagConfig(image, tag)
	if err != nil {
		t.Fatal(err)
	}

	if config.BuildArgs["GO_VERSION"] != "1.21" {
		t.Errorf("expected GO_VERSION=1.21, got %q", config.BuildArgs["GO_VERSION"])
	}
	if config.BuildArgs["NODE_VERSION"] != "20" {
		t.Errorf("expected NODE_VERSION=20, got %q", config.BuildArgs["NODE_VERSION"])
	}
	if config.BuildArgs["MODE"] != "prod" {
		t.Errorf("expected MODE=prod, got %q", config.BuildArgs["MODE"])
	}
	if config.BuildArgs["DEBUG"] != "false" {
		t.Errorf("expected DEBUG=false, got %q", config.BuildArgs["DEBUG"])
	}
}

func TestResolveVariantConfig(t *testing.T) {
	image := &model.Image{
		Name:      "test",
		Versions:  model.Versions{"go": "1.21"},
		BuildArgs: model.BuildArgs{},
		Secrets:   model.Secrets{},
	}
	tag := &model.Tag{
		Name:     "latest",
		Versions: model.Versions{"node": "20"},
	}
	variant := &model.ImageVariant{
		Name:      "slim",
		TagSuffix: "-slim",
		Versions:  model.Versions{"node": "20-slim"},
		BuildArgs: model.BuildArgs{"EXTRA": "yes"},
	}

	config, err := ResolveVariantConfig(image, variant, tag)
	if err != nil {
		t.Fatal(err)
	}

	if config.BuildArgs["NODE_VERSION"] != "20-slim" {
		t.Errorf("expected variant to override NODE_VERSION, got %q", config.BuildArgs["NODE_VERSION"])
	}
	if config.BuildArgs["EXTRA"] != "yes" {
		t.Errorf("expected EXTRA=yes, got %q", config.BuildArgs["EXTRA"])
	}
}

func TestResolveTagConfig_WithPlainSecrets(t *testing.T) {
	image := &model.Image{
		Name:      "test",
		Versions:  model.Versions{},
		BuildArgs: model.BuildArgs{},
		Secrets: model.Secrets{
			"token": {SourceType: "plain", Value: "my-secret-value"},
		},
	}
	tag := &model.Tag{Name: "latest"}

	config, err := ResolveTagConfig(image, tag)
	if err != nil {
		t.Fatal(err)
	}

	if string(config.Secrets["token"]) != "my-secret-value" {
		t.Errorf("expected secret token=my-secret-value, got %q", string(config.Secrets["token"]))
	}
}

func TestResolveTagConfig_WithEnvSecret(t *testing.T) {
	t.Setenv("TEST_SECRET_VALUE", "env-secret")

	image := &model.Image{
		Name:      "test",
		Versions:  model.Versions{},
		BuildArgs: model.BuildArgs{},
		Secrets: model.Secrets{
			"api_key": {SourceType: "env", Value: "${TEST_SECRET_VALUE}"},
		},
	}
	tag := &model.Tag{Name: "latest"}

	config, err := ResolveTagConfig(image, tag)
	if err != nil {
		t.Fatal(err)
	}

	if string(config.Secrets["api_key"]) != "env-secret" {
		t.Errorf("expected secret api_key=env-secret, got %q", string(config.Secrets["api_key"]))
	}
}

func TestResolveTagConfig_NilVersionsAndBuildArgs(t *testing.T) {
	image := &model.Image{
		Name:    "test",
		Secrets: model.Secrets{},
	}
	tag := &model.Tag{Name: "latest"}

	config, err := ResolveTagConfig(image, tag)
	if err != nil {
		t.Fatal(err)
	}

	if config.BuildArgs == nil {
		t.Error("expected non-nil BuildArgs")
	}
	if config.Secrets == nil {
		t.Error("expected non-nil Secrets")
	}
}

func TestResolveVariantConfig_InheritsImageSecrets(t *testing.T) {
	image := &model.Image{
		Name:      "test",
		Versions:  model.Versions{},
		BuildArgs: model.BuildArgs{},
		Secrets: model.Secrets{
			"token": {SourceType: "plain", Value: "base-secret"},
		},
	}
	tag := &model.Tag{Name: "latest"}
	variant := &model.ImageVariant{
		Name:      "slim",
		TagSuffix: "-slim",
	}

	config, err := ResolveVariantConfig(image, variant, tag)
	if err != nil {
		t.Fatal(err)
	}

	if string(config.Secrets["token"]) != "base-secret" {
		t.Errorf("expected variant to inherit image secrets, got %q", string(config.Secrets["token"]))
	}
}
