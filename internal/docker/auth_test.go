package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/registry"
)

// writeDockerConfig points DOCKER_CONFIG at a throwaway directory holding the
// given config.json contents.
func writeDockerConfig(t *testing.T, contents string) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write config.json: %v", err)
	}
	t.Setenv("DOCKER_CONFIG", dir)
}

func TestRegistryAuthForReturnsCredentialsOfMatchingRegistry(t *testing.T) {
	// "AWS:secret" base64-encoded, the shape `ch login` stores for ECR.
	writeDockerConfig(t, `{"auths":{"123.dkr.ecr.eu-west-1.amazonaws.com":{"auth":"QVdTOnNlY3JldA=="}}}`)

	got, err := registryAuthFor("123.dkr.ecr.eu-west-1.amazonaws.com/deepl/devex/ci/ubuntu-base:26.04")
	if err != nil {
		t.Fatalf("registryAuthFor: %v", err)
	}
	if got == "" {
		t.Fatal("expected credentials, got empty auth")
	}

	decoded, err := registry.DecodeAuthConfig(got)
	if err != nil {
		t.Fatalf("decode auth: %v", err)
	}
	if decoded.Username != "AWS" || decoded.Password != "secret" {
		t.Errorf("got %q/%q, want AWS/secret", decoded.Username, decoded.Password)
	}
	if decoded.ServerAddress != "123.dkr.ecr.eu-west-1.amazonaws.com" {
		t.Errorf("got server address %q", decoded.ServerAddress)
	}
}

func TestRegistryAuthForIsEmptyWithoutMatchingCredentials(t *testing.T) {
	writeDockerConfig(t, `{"auths":{"123.dkr.ecr.eu-west-1.amazonaws.com":{"auth":"QVdTOnNlY3JldA=="}}}`)

	// An anonymous pull must stay anonymous rather than leaking another
	// registry's credentials to the daemon.
	got, err := registryAuthFor("ubuntu:24.04")
	if err != nil {
		t.Fatalf("registryAuthFor: %v", err)
	}
	if got != "" {
		decoded, _ := registry.DecodeAuthConfig(got)
		out, _ := json.Marshal(decoded)
		t.Errorf("expected empty auth for unauthenticated registry, got %s", out)
	}
}

func TestRegistryAuthForRejectsInvalidReference(t *testing.T) {
	writeDockerConfig(t, `{"auths":{}}`)

	if _, err := registryAuthFor("not a valid ref"); err == nil {
		t.Fatal("expected an error for an invalid image reference")
	}
}
