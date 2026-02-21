package gcr

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func setupRegistry(t *testing.T) string {
	t.Helper()
	reg := registry.New()
	server := httptest.NewServer(reg)
	t.Cleanup(server.Close)
	return server.Listener.Addr().String()
}

func pushEmptyImage(t *testing.T, ref string) {
	t.Helper()
	tag, err := name.NewTag(ref)
	if err != nil {
		t.Fatalf("failed to parse tag %q: %v", ref, err)
	}

	img, err := mutate.ConfigFile(empty.Image, nil)
	if err != nil {
		t.Fatalf("failed to create empty image: %v", err)
	}

	if err := remote.Write(tag, img); err != nil {
		t.Fatalf("failed to push image %q: %v", ref, err)
	}
}

func TestRetag(t *testing.T) {
	addr := setupRegistry(t)

	sourceRef := fmt.Sprintf("%s/test/image:1.2.3", addr)
	targetRef := fmt.Sprintf("%s/test/image:1.2", addr)

	pushEmptyImage(t, sourceRef)

	if err := Retag(sourceRef, targetRef); err != nil {
		t.Fatalf("Retag failed: %v", err)
	}

	// Verify the target tag exists and has the same digest
	srcTag, _ := name.NewTag(sourceRef)
	dstTag, _ := name.NewTag(targetRef)

	srcDesc, err := remote.Get(srcTag)
	if err != nil {
		t.Fatalf("failed to get source: %v", err)
	}

	dstDesc, err := remote.Get(dstTag)
	if err != nil {
		t.Fatalf("failed to get target: %v", err)
	}

	if srcDesc.Digest != dstDesc.Digest {
		t.Errorf("digest mismatch: source=%v target=%v", srcDesc.Digest, dstDesc.Digest)
	}
}

func TestRetag_InvalidSource(t *testing.T) {
	err := Retag("", "valid:tag")
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
	if !strings.Contains(err.Error(), "invalid source reference") {
		t.Errorf("expected error to contain %q, got: %v", "invalid source reference", err)
	}
}

func TestRetag_InvalidTarget(t *testing.T) {
	err := Retag("valid/image:tag", "")
	if err == nil {
		t.Fatal("expected error for invalid target")
	}
	if !strings.Contains(err.Error(), "invalid target reference") {
		t.Errorf("expected error to contain %q, got: %v", "invalid target reference", err)
	}
}

func TestRetag_SourceNotFound(t *testing.T) {
	addr := setupRegistry(t)
	sourceRef := fmt.Sprintf("%s/test/missing:latest", addr)
	targetRef := fmt.Sprintf("%s/test/missing:alias", addr)

	err := Retag(sourceRef, targetRef)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Errorf("expected error to contain %q, got: %v", "failed to fetch", err)
	}
}
