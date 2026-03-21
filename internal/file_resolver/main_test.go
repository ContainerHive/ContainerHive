package file_resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetFileCandidates(t *testing.T) {
	tests := map[string]struct {
		baseName   string
		extensions []string
		expected   []string
	}{
		"without extensions with Dockerfile": {
			baseName:   "Dockerfile",
			extensions: nil,
			expected:   []string{"Dockerfile", "Dockerfile.gotpl"},
		},
		"with yaml and yml extension": {
			baseName:   "test",
			extensions: []string{"yaml", "yml"},
			expected:   []string{"test.yaml", "test.yaml.gotpl", "test.yml", "test.yml.gotpl"},
		},
		"with only yaml extension": {
			baseName:   "config",
			extensions: []string{"yaml"},
			expected:   []string{"config.yaml", "config.yaml.gotpl"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetFileCandidates(tc.baseName, tc.extensions...)

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("getFileCandidates() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestResolveFirstExistingFile(t *testing.T) {
	t.Run("returns first candidate when it exists", func(t *testing.T) {
		root := t.TempDir()
		os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM alpine"), 0644)

		got, err := ResolveFirstExistingFile(root, "Dockerfile", "Dockerfile.gotpl")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(root, "Dockerfile")
		if got != expected {
			t.Errorf("expected %s, got %s", expected, got)
		}
	})

	t.Run("returns second candidate when first does not exist", func(t *testing.T) {
		root := t.TempDir()
		os.WriteFile(filepath.Join(root, "Dockerfile.gotpl"), []byte("FROM {{ .ImageName }}"), 0644)

		got, err := ResolveFirstExistingFile(root, "Dockerfile", "Dockerfile.gotpl")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(root, "Dockerfile.gotpl")
		if got != expected {
			t.Errorf("expected %s, got %s", expected, got)
		}
	})

	t.Run("returns error when no candidates exist", func(t *testing.T) {
		root := t.TempDir()

		_, err := ResolveFirstExistingFile(root, "Dockerfile", "Dockerfile.gotpl")
		if err != NoFileCandidatesErr {
			t.Errorf("expected NoFileCandidatesErr, got %v", err)
		}
	})

	t.Run("skips directories matching candidate name", func(t *testing.T) {
		root := t.TempDir()
		os.Mkdir(filepath.Join(root, "Dockerfile"), 0755)
		os.WriteFile(filepath.Join(root, "Dockerfile.gotpl"), []byte("FROM alpine"), 0644)

		got, err := ResolveFirstExistingFile(root, "Dockerfile", "Dockerfile.gotpl")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(root, "Dockerfile.gotpl")
		if got != expected {
			t.Errorf("expected %s, got %s", expected, got)
		}
	})

	t.Run("returns error with empty candidates", func(t *testing.T) {
		root := t.TempDir()

		_, err := ResolveFirstExistingFile(root)
		if err != NoFileCandidatesErr {
			t.Errorf("expected NoFileCandidatesErr, got %v", err)
		}
	})
}
