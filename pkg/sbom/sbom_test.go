package sbom

import (
	"context"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	gen, err := NewGenerator()
	if err != nil {
		t.Fatal(err)
	}
	if gen == nil {
		t.Fatal("expected non-nil generator")
	}
}

func TestGenerate_InvalidPath(t *testing.T) {
	gen, err := NewGenerator()
	if err != nil {
		t.Fatal(err)
	}

	_, err = gen.Generate(context.Background(), "/nonexistent/image.tar", "spdx-json")
	if err == nil {
		t.Error("expected error for nonexistent tar file")
	}
}
