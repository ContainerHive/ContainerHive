package rendering

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// copyChiselSlices copies the optional slices/ directory with custom slice
// definitions from the build entrypoint directory into the rendered target
// directory for chisel entrypoints. A missing directory is not an error: the
// build then cuts only the release slices.
func copyChiselSlices(target, entrypointPath string) error {
	if !strings.HasPrefix(filepath.Base(entrypointPath), "chisel.yaml") {
		return nil
	}
	slicesPath := filepath.Join(filepath.Dir(entrypointPath), "slices")
	stat, err := os.Stat(slicesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errors.Join(errors.New("failed to stat slices directory"), err)
	}
	if !stat.IsDir() {
		return errors.New("slices is not a directory")
	}
	return copyDir(slicesPath, filepath.Join(target, "slices"))
}
