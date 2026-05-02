package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
)

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to read dir %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", dst, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", srcPath, err)
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				t.Fatalf("failed to write %s: %v", dstPath, err)
			}
		}
	}
}

func TestGenerateProject(t *testing.T) {
	tmpDir := t.TempDir()
	copyDir(t, "../testdata/minimal-project", tmpDir)

	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Value: "."},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return generateProject(ctx, cmd)
		},
	}

	if err := cmd.Run(t.Context(), []string{"ch", "--project", tmpDir}); err != nil {
		t.Fatal(err)
	}

	distPath := filepath.Join(tmpDir, "dist")
	tagDir := filepath.Join(distPath, "nginx", "1.27")

	if _, err := os.Stat(distPath); err != nil {
		t.Errorf("dist/ directory not created: %v", err)
	}
	if _, err := os.Stat(tagDir); err != nil {
		t.Errorf("expected tag dir %s to exist: %v", tagDir, err)
	}
	if _, err := os.Stat(filepath.Join(tagDir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not found: %v", err)
	}
}

func TestGenerateFlag_Registered(t *testing.T) {
	app := NewApp()
	var found bool
	for _, flag := range app.Flags {
		for _, name := range flag.Names() {
			if name == "generate" {
				found = true
			}
		}
	}
	if !found {
		t.Error("--generate flag not found on root command")
	}
}
