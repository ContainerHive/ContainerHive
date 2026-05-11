package cli

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/urfave/cli/v3"
)

// writeProject lays down a minimal hive project at root containing one image
// named "test" with the supplied Dockerfile contents under the requested
// filename ("Dockerfile" or "Dockerfile.gotpl").
func writeProject(t *testing.T, root, dockerfileName, dockerfileBody, hiveYAML string) {
	t.Helper()
	imageDir := filepath.Join(root, "images", "test")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if hiveYAML == "" {
		hiveYAML = "platforms:\n  - linux/amd64\n"
	}
	if err := os.WriteFile(filepath.Join(root, "hive.yml"), []byte(hiveYAML), 0o644); err != nil {
		t.Fatalf("write hive.yml: %v", err)
	}
	imageYAML := "tags:\n  - name: \"1\"\n"
	if err := os.WriteFile(filepath.Join(imageDir, "image.yml"), []byte(imageYAML), 0o644); err != nil {
		t.Fatalf("write image.yml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, dockerfileName), []byte(dockerfileBody), 0o644); err != nil {
		t.Fatalf("write %s: %v", dockerfileName, err)
	}
}

// runLint executes the lint command against a project root and returns its
// error and the captured stdout.
func runLint(t *testing.T, projectRoot string, flags ...string) (err error, stdout string) {
	t.Helper()

	// Capture stdout so we can assert on printed findings.
	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("pipe: %v", pipeErr)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	// Silence slog during tests to keep output deterministic.
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer slog.SetDefault(prevLogger)

	app := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Value: "."},
		},
		Commands: []*cli.Command{lintCmd()},
	}
	args := []string{"ch", "--project", projectRoot, "lint"}
	args = append(args, flags...)
	err = app.Run(context.Background(), args)

	w.Close()
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("copy stdout: %v", copyErr)
	}
	stdout = buf.String()
	return
}

func TestLintCmd_CleanDockerfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	writeProject(t, root, "Dockerfile", "FROM nginx:1.27\n", "")

	err, _ := runLint(t, root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLintCmd_ViolationFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	// `MAINTAINER` trips DL4000 at error severity, which fails the default
	// "error" failure threshold.
	writeProject(t, root, "Dockerfile", "FROM nginx:1.27\nMAINTAINER me@example.com\n", "")

	err, stdout := runLint(t, root)
	if err == nil {
		t.Fatalf("expected lint to fail, got nil error")
	}
	if !strings.Contains(stdout, "DL4000") {
		t.Errorf("expected DL4000 in findings, got: %s", stdout)
	}
}

func TestLintCmd_TemplatedDockerfileSkipped(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	// Bad content but the .gotpl extension means hadolint never sees it.
	writeProject(t, root, "Dockerfile.gotpl", "FROM ubuntu\n", "")

	err, stdout := runLint(t, root)
	if err != nil {
		t.Fatalf("expected templated Dockerfile to be skipped, got %v", err)
	}
	if strings.Contains(stdout, "DL3006") {
		t.Errorf("templated Dockerfile must not be linted, but findings appeared: %s", stdout)
	}
}

func TestLintCmd_IgnoredRulePasses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	hiveYAML := "platforms:\n  - linux/amd64\nlint:\n  ignored:\n    - DL4000\n"
	writeProject(t, root, "Dockerfile", "FROM nginx:1.27\nMAINTAINER me@example.com\n", hiveYAML)

	err, _ := runLint(t, root)
	if err != nil {
		t.Fatalf("expected lint to pass with DL4000 ignored, got %v", err)
	}
}

func TestLintCmd_FailureThresholdOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	// `FROM ubuntu` (no tag) is DL3006 at warning level. Under the default
	// "error" threshold this passes; lowering the threshold to "warning"
	// flips it to a failure.
	writeProject(t, root, "Dockerfile", "FROM ubuntu\n", "")

	if err, _ := runLint(t, root); err != nil {
		t.Fatalf("expected default threshold to ignore warning findings, got %v", err)
	}

	err, stdout := runLint(t, root, "--failure-threshold", "warning")
	if err == nil {
		t.Fatalf("expected failure with --failure-threshold warning, got nil; stdout=%s", stdout)
	}
}

func TestResolveLintConfig(t *testing.T) {
	ignoredSeed := []string{"DL3000"}
	cases := []struct {
		name       string
		project    *model.LintConfig
		cliThresh  string
		wantThresh string
	}{
		{name: "no config no flag", wantThresh: "error"},
		{name: "cli flag only", cliThresh: "warning", wantThresh: "warning"},
		{name: "project only", project: &model.LintConfig{FailureThreshold: "info"}, wantThresh: "info"},
		{name: "cli overrides project", project: &model.LintConfig{FailureThreshold: "info"}, cliThresh: "warning", wantThresh: "warning"},
		{name: "project no threshold", project: &model.LintConfig{Ignored: ignoredSeed}, wantThresh: "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveLintConfig(tc.project, tc.cliThresh)
			if got == nil {
				t.Fatalf("expected non-nil config")
			}
			if got.FailureThreshold != tc.wantThresh {
				t.Errorf("FailureThreshold = %q, want %q", got.FailureThreshold, tc.wantThresh)
			}
			// resolveLintConfig must not mutate the caller's struct.
			if tc.project != nil && tc.project.FailureThreshold == "info" && got == tc.project {
				t.Errorf("resolveLintConfig returned the same pointer; expected a copy")
			}
		})
	}
}
