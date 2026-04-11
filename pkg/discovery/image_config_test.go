package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func setupMinimalImageDir(t *testing.T, imageYML string) (projectRoot, configFilePath string) {
	t.Helper()
	dir := t.TempDir()
	configFilePath = filepath.Join(dir, "image.yml")
	if err := os.WriteFile(configFilePath, []byte(imageYML), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return dir, configFilePath
}

func TestProcessImageConfig_Description(t *testing.T) {
	tests := []struct {
		name            string
		imageYML        string
		wantDescription string
	}{
		{
			name: "with description",
			imageYML: `description: "My test image"
tags:
  - name: 1.0.0
`,
			wantDescription: "My test image",
		},
		{
			name: `without description`,
			imageYML: `tags:
  - name: 1.0.0
`,
			wantDescription: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projectRoot, configFilePath := setupMinimalImageDir(t, tc.imageYML)
			img, err := processImageConfig(projectRoot, configFilePath)
			if err != nil {
				t.Fatalf("processImageConfig() error = %v", err)
			}
			if img.Description != tc.wantDescription {
				t.Errorf("Description = %q, want %q", img.Description, tc.wantDescription)
			}
		})
	}
}
