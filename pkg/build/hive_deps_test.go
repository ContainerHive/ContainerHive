package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		tempFile.Close()

		// Add cleanup function
		d.cleanups = append(d.cleanups, func() {
			os.Remove(tempFile.Name())
		})

		// Verify file exists before cleanup
		_, err = os.Stat(tempFile.Name())
		if err != nil {
			t.Fatalf("temp file should exist before cleanup: %v", err)
		}

		// Run cleanup
		d.Cleanup()

		// Verify file is removed
		_, err = os.Stat(tempFile.Name())
		if !os.IsNotExist(err) {
			t.Errorf("expected file to be removed after cleanup, got err: %v", err)
		}
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
		if err := os.WriteFile(dockerfile, []byte(dockerfileContent), 0644); err != nil {
			t.Fatalf("failed to write Dockerfile: %v", err)
		}

		// Call ResolveHiveDeps
		result, err := ResolveHiveDeps(HiveDepsOpts{
			DockerfilePath: dockerfile,
			DistPath:       tempDir,
			PlatformStr:    "linux/amd64",
		})

		// Should return nil (no error) and nil result when no hive deps
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
	})

	t.Run("hive dependencies not built yet", func(t *testing.T) {
		// Create a temporary directory and Dockerfile with hive refs
		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		dockerfileContent := `FROM __hive__/base:latest
COPY --from=__hive__/util:1.0 /app /app
`
		if err := os.WriteFile(dockerfile, []byte(dockerfileContent), 0644); err != nil {
			t.Fatalf("failed to write Dockerfile: %v", err)
		}

		// Call ResolveHiveDeps - should fail because dependencies don't exist and no registry
		result, err := ResolveHiveDeps(HiveDepsOpts{
			DockerfilePath: dockerfile,
			DistPath:       tempDir,
			PlatformStr:    "linux/amd64",
		})

		// Should return error because dependencies aren't built
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
		if !strings.Contains(err.Error(), "not built yet") {
			t.Errorf("expected error to contain 'not built yet', got: %v", err)
		}
	})

	t.Run("hive dependencies fall back to registry", func(t *testing.T) {
		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		dockerfileContent := `FROM __hive__/base:latest
RUN echo hello
`
		if err := os.WriteFile(dockerfile, []byte(dockerfileContent), 0644); err != nil {
			t.Fatalf("failed to write Dockerfile: %v", err)
		}

		// No local tars, but registry is configured
		result, err := ResolveHiveDeps(HiveDepsOpts{
			DockerfilePath:  dockerfile,
			DistPath:        tempDir,
			PlatformStr:     "linux/amd64",
			RegistryAddress: "ghcr.io/myorg",
			BuildID:         "42",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		contextVal, ok := result.NamedContexts["context:hive-dep/base:latest"]
		if !ok {
			t.Error("expected NamedContexts to contain key 'context:hive-dep/base:latest'")
		}
		if contextVal != "docker-image://ghcr.io/myorg/base:latest-build.42" {
			t.Errorf("expected context value 'docker-image://ghcr.io/myorg/base:latest-build.42', got %q", contextVal)
		}
		if len(result.OCIStores) != 0 {
			t.Errorf("expected empty OCIStores, got %d entries", len(result.OCIStores))
		}
		result.Cleanup()
	})

	t.Run("hive dependencies fall back to registry without build ID", func(t *testing.T) {
		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		dockerfileContent := `FROM __hive__/base:v1
RUN echo hello
`
		if err := os.WriteFile(dockerfile, []byte(dockerfileContent), 0644); err != nil {
			t.Fatalf("failed to write Dockerfile: %v", err)
		}

		result, err := ResolveHiveDeps(HiveDepsOpts{
			DockerfilePath:  dockerfile,
			DistPath:        tempDir,
			PlatformStr:     "linux/amd64",
			RegistryAddress: "ghcr.io/myorg",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.NamedContexts["context:hive-dep/base:v1"] != "docker-image://ghcr.io/myorg/base:v1" {
			t.Errorf("unexpected context value: %q", result.NamedContexts["context:hive-dep/base:v1"])
		}
		result.Cleanup()
	})

	t.Run("dependency files exist but OCI loading fails", func(t *testing.T) {
		// This test verifies that the function finds dependency files correctly
		// but fails at OCI loading (which is expected for empty tar files)

		tempDir := t.TempDir()
		dockerfile := filepath.Join(tempDir, "Dockerfile")
		dockerfileContent := `FROM __hive__/base:latest
COPY --from=__hive__/util:1.0 /app /app
`
		if err := os.WriteFile(dockerfile, []byte(dockerfileContent), 0644); err != nil {
			t.Fatalf("failed to write Dockerfile: %v", err)
		}

		// Create mock dependency tar files in the correct structure
		// platform.Sanitize("linux/amd64") returns "linux-amd64"
		baseDir := filepath.Join(tempDir, "base", "latest", "linux-amd64")
		utilDir := filepath.Join(tempDir, "util", "1.0", "linux-amd64")
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			t.Fatalf("failed to create base dir: %v", err)
		}
		if err := os.MkdirAll(utilDir, 0755); err != nil {
			t.Fatalf("failed to create util dir: %v", err)
		}

		baseTar := filepath.Join(baseDir, "image.tar")
		utilTar := filepath.Join(utilDir, "image.tar")

		// Create empty tar files
		if _, err := os.Create(baseTar); err != nil {
			t.Fatalf("failed to create base tar: %v", err)
		}
		if _, err := os.Create(utilTar); err != nil {
			t.Fatalf("failed to create util tar: %v", err)
		}

		// Verify files exist
		if !fileExists(baseTar) {
			t.Error("expected base tar to exist")
		}
		if !fileExists(utilTar) {
			t.Error("expected util tar to exist")
		}

		// Call ResolveHiveDeps - will fail at OCI loading (expected)
		result, err := ResolveHiveDeps(HiveDepsOpts{
			DockerfilePath: dockerfile,
			DistPath:       tempDir,
			PlatformStr:    "linux/amd64",
		})

		// Should fail at OCI loading
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "loading OCI layout") {
			t.Errorf("expected error to contain 'loading OCI layout', got: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
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
		if err := os.WriteFile(dockerfile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("failed to write Dockerfile: %v", err)
		}

		// Call rewriteDockerfile
		rewritten, err := rewriteDockerfile(dockerfile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify rewritten file was created
		if !strings.HasSuffix(rewritten, ".hive") {
			t.Errorf("expected rewritten path to end with .hive, got %q", rewritten)
		}

		// Verify content
		content, err := os.ReadFile(rewritten)
		if err != nil {
			t.Fatalf("failed to read rewritten file: %v", err)
		}

		// Should replace __hive__/ in FROM and COPY but not __hive__ without slash
		if !strings.Contains(string(content), "hive-dep/base:latest") {
			t.Error("expected rewritten content to contain 'hive-dep/base:latest'")
		}
		if !strings.Contains(string(content), "hive-dep/util:1.0") {
			t.Error("expected rewritten content to contain 'hive-dep/util:1.0'")
		}
		if !strings.Contains(string(content), "__hive__should-not-be-replaced") {
			t.Error("expected rewritten content to preserve '__hive__should-not-be-replaced'")
		}

		// Cleanup
		os.Remove(rewritten)
	})
}
