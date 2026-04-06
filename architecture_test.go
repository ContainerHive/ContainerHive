package containerhive_test

import (
	"testing"

	"github.com/kcmvp/archunit"
)

func TestPackageDirections(t *testing.T) {
	checks := []struct {
		err         error
		description string
	}{
		{
			cmdLayer.ShouldNotReferLayers(internalLayer),
			"cmd is not allowed to reference internal",
		},
		{
			pkgLayer.ShouldNotReferLayers(cmdLayer),
			"pkg should not refer to cmd",
		},
	}

	for _, check := range checks {
		if check.err != nil {
			t.Fatal(check.err, check.description)
		}
	}
}

func TestNoDirectLogImport(t *testing.T) {
	layers := []archunit.Layer{cmdLayer, pkgLayer, internalLayer}
	for _, layer := range layers {
		for _, imp := range layer.Imports() {
			if imp == "log" {
				t.Errorf("layer %q imports \"log\" directly — use \"log/slog\" instead", layer.A)
			}
		}
	}
}
