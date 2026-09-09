package source

import (
	"context"
	"errors"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestRegistrySource_Fetch(t *testing.T) {
	src := RegistrySource{listTags: func(ctx context.Context, image string) ([]string, error) {
		if image != "library/node" {
			t.Errorf("unexpected image %q", image)
		}
		return []string{"20.20.0", "22.22.0", "latest"}, nil
	}}

	versions, err := src.Fetch(context.Background(), &model.SourceConfig{Type: "registry", Image: "library/node"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 raw tags (filtering happens later), got %d", len(versions))
	}
}

func TestRegistrySource_Fetch_PropagatesError(t *testing.T) {
	src := RegistrySource{listTags: func(ctx context.Context, image string) ([]string, error) {
		return nil, errors.New("boom")
	}}
	if _, err := src.Fetch(context.Background(), &model.SourceConfig{Type: "registry", Image: "library/node"}); err == nil {
		t.Error("expected the underlying error to propagate")
	}
}

func TestRegistrySource_Fetch_TooManyTags(t *testing.T) {
	many := make([]string, maxRegistryTags+1)
	for i := range many {
		many[i] = "tag"
	}
	src := RegistrySource{listTags: func(ctx context.Context, image string) ([]string, error) {
		return many, nil
	}}
	if _, err := src.Fetch(context.Background(), &model.SourceConfig{Type: "registry", Image: "library/node"}); err == nil {
		t.Error("expected an error when tag count exceeds the limit")
	}
}

func TestRegistrySource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *model.SourceConfig
		wantErr bool
	}{
		{"valid", &model.SourceConfig{Type: "registry", Image: "library/node"}, false},
		{"missing image", &model.SourceConfig{Type: "registry"}, true},
		{"stray url field", &model.SourceConfig{Type: "registry", Image: "library/node", URL: "https://x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegistrySource{}.Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultRegistry_KnowsAllSources(t *testing.T) {
	reg := DefaultRegistry()
	for _, sourceType := range []string{"json", "registry", "dockerhub", "github"} {
		if _, err := reg.Get(sourceType); err != nil {
			t.Errorf("DefaultRegistry should know source type %q: %v", sourceType, err)
		}
	}
	if _, err := reg.Get("unknown"); err == nil {
		t.Error("expected an error for an unknown source type")
	}
}

func TestDefaultRegistry_DockerHubAliasesRegistry(t *testing.T) {
	reg := DefaultRegistry()
	dockerhub, err := reg.Get("dockerhub")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := dockerhub.(dockerHubAlias); !ok {
		t.Errorf("expected \"dockerhub\" to resolve to the registry source, got %T", dockerhub)
	}
}
