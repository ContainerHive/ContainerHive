package rendering

import (
	"errors"
	"os"
	"path/filepath"
)

// copyFlakeLock copies flake.lock from the build entrypoint directory into
// the rendered target directory. A missing flake.lock is not an error: the
// flake then resolves its inputs without a lockfile.
func copyFlakeLock(target, entrypointPath string) error {
	lockPath := filepath.Join(filepath.Dir(entrypointPath), "flake.lock")
	stat, err := os.Stat(lockPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errors.Join(errors.New("failed to stat flake.lock"), err)
	}
	if stat.IsDir() {
		return errors.New("flake.lock is a directory")
	}
	return copyFile(lockPath, filepath.Join(target, "flake.lock"))
}
