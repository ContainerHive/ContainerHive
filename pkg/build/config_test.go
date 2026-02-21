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
