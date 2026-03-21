package containerhive_test

import (
	"testing"
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
