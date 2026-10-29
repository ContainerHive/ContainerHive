package utils

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// ExtractTar extracts a tar archive to the destination directory.
// The extraction is confined to destDir: entry names that escape it are
// rejected, and the rooted filesystem operations also block symlink-based
// escapes.
func ExtractTar(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return errors.Join(errors.New("failed to open tar"), err)
	}
	defer f.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	root, err := os.OpenRoot(destDir)
	if err != nil {
		return errors.Join(errors.New("failed to open destination root"), err)
	}
	defer root.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := filepath.Clean(hdr.Name)
		if !filepath.IsLocal(name) {
			return errors.New("tar entry escapes destination: " + hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(name, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := root.MkdirAll(filepath.Dir(name), 0o755); err != nil {
				return err
			}
			out, err := root.Create(name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}
