package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileExists is a helper function to check if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestHiveDepsCleanup(t *testing.T) {
	t.Run("Cleanup removes temporary files", func(t *testing.T) {
		// Create a temporary HiveDeps instance
		d := &HiveDeps{}

		// Create a temporary file to track cleanup
		tempFile, err := os.CreateTemp("", "test-cleanup-*")
		require.NoError(t, err)
		tempFile.Close()

		// Add cleanup function
		d.cleanups = append(d.cleanups, func() {
			os.Remove(tempFile.Name())
		})

		// Verify file exists before cleanup
		_, err = os.Stat(tempFile.Name())
		require.NoError(t, err)

		// Run cleanup
		d.Cleanup()

		// Verify file is removed
		_, err = os.Stat(tempFile.Name())
		assert.True(t, os.IsNotExist(err))
	})
}

func TestResolveHiveDeps(t *testing.T) {
	t.Run("no hive dependencies", func(t *testing.T) {
		// Create a temporary directory and Dockerfile without hive refs
		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		dockerfileContent := `FROM ubuntu:24.04
RUN apt-get update
`
		require.NoError(t, os.WriteFile(dockerfile, []byte(dockerfileContent), 0644))

		// Call ResolveHiveDeps
		result, err := ResolveHiveDeps(dockerfile, tempDir, "linux/amd64")

		// Should return nil (no error) and nil result when no hive deps
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("hive dependencies not built yet", func(t *testing.T) {
		// Create a temporary directory and Dockerfile with hive refs
		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		dockerfileContent := `FROM __hive__/base:latest
COPY --from=__hive__/util:1.0 /app /app
`
		require.NoError(t, os.WriteFile(dockerfile, []byte(dockerfileContent), 0644))

		// Call ResolveHiveDeps - should fail because dependencies don't exist
		result, err := ResolveHiveDeps(dockerfile, tempDir, "linux/amd64")

		// Should return error because dependencies aren't built
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not built yet")
	})

	t.Run("dependency files exist but OCI loading fails", func(t *testing.T) {
		// This test verifies that the function finds dependency files correctly
		// but fails at OCI loading (which is expected for empty tar files)

		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		dockerfileContent := `FROM __hive__/base:latest
COPY --from=__hive__/util:1.0 /app /app
`
		require.NoError(t, os.WriteFile(dockerfile, []byte(dockerfileContent), 0644))

		// Create mock dependency tar files in the correct structure
		// platform.Sanitize("linux/amd64") returns "linux-amd64"
		baseDir := filepath.Join(tempDir, "base", "latest", "linux-amd64")
		utilDir := filepath.Join(tempDir, "util", "1.0", "linux-amd64")
		require.NoError(t, os.MkdirAll(baseDir, 0755))
		require.NoError(t, os.MkdirAll(utilDir, 0755))

		baseTar := filepath.Join(baseDir, "image.tar")
		utilTar := filepath.Join(utilDir, "image.tar")

		// Create empty tar files
		_, err := os.Create(baseTar)
		require.NoError(t, err)
		_, err = os.Create(utilTar)
		require.NoError(t, err)

		// Verify files exist
		assert.True(t, fileExists(baseTar))
		assert.True(t, fileExists(utilTar))

		// Call ResolveHiveDeps - will fail at OCI loading (expected)
		result, err := ResolveHiveDeps(dockerfile, tempDir, "linux/amd64")

		// Should fail at OCI loading
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading OCI layout")
		assert.Nil(t, result)
	})
}

func TestRewriteDockerfile(t *testing.T) {
	t.Run("rewrites hive prefixes correctly", func(t *testing.T) {
		// Create a temporary Dockerfile
		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		originalContent := `FROM __hive__/base:latest
COPY --from=__hive__/util:1.0 /app /app
RUN echo "__hive__should-not-be-replaced"
`
		require.NoError(t, os.WriteFile(dockerfile, []byte(originalContent), 0644))

		// Call rewriteDockerfile
		rewritten, err := rewriteDockerfile(dockerfile)
		require.NoError(t, err)

		// Verify rewritten file was created
		assert.True(t, strings.HasSuffix(rewritten, ".hive"))

		// Verify content
		content, err := os.ReadFile(rewritten)
		require.NoError(t, err)

		// Should replace __hive__/ in FROM and COPY but not __hive__ without slash
		assert.Contains(t, string(content), "hive-dep/base:latest")
		assert.Contains(t, string(content), "hive-dep/util:1.0")
		assert.Contains(t, string(content), "__hive__should-not-be-replaced")

		// Cleanup
		os.Remove(rewritten)
	})
}
