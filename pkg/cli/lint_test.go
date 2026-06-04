package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("pipe: %v", pipeErr)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

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
	writeProject(t, root, "Dockerfile", "FROM ubuntu\n", "")

	if err, _ := runLint(t, root); err != nil {
		t.Fatalf("expected default threshold to ignore warning findings, got %v", err)
	}

	err, stdout := runLint(t, root, "--failure-threshold", "warning")
	if err == nil {
		t.Fatalf("expected failure with --failure-threshold warning, got nil; stdout=%s", stdout)
	}
}

func TestLintCmd_CodeClimateReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	writeProject(t, root, "Dockerfile", "FROM nginx:1.27\nMAINTAINER me@example.com\n", "")

		reportPath := filepath.Join(root, "gl-code-quality-report.json")
	err, _ := runLint(t, root, "--format", "codeclimate="+reportPath)
	if err == nil {
		t.Fatalf("expected lint failure on DL4000")
	}

	data, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("expected report at %s, got %v", reportPath, readErr)
	}

	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, data)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least one entry in report")
	}

	got := entries[0]
	if got["check_name"] != "DL4000" {
		t.Errorf("check_name = %v, want DL4000", got["check_name"])
	}
	if got["severity"] != "blocker" {
		t.Errorf("severity = %v, want blocker (error→blocker mapping)", got["severity"])
	}
	loc, ok := got["location"].(map[string]any)
	if !ok {
		t.Fatalf("location missing: %v", got)
	}
	path, _ := loc["path"].(string)
	if filepath.IsAbs(path) {
		t.Errorf("report path is absolute (%s); expected repo-relative", path)
	}
	if !strings.HasSuffix(path, "Dockerfile") {
		t.Errorf("report path does not end in Dockerfile: %s", path)
	}
}

func TestLintCmd_MultipleFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	writeProject(t, root, "Dockerfile", "FROM nginx:1.27\nMAINTAINER me@example.com\n", "")

	reportPath := filepath.Join(root, "gl-code-quality-report.json")
	err, stdout := runLint(t, root, "--format", "terminal", "--format", "codeclimate="+reportPath, "--format", "github-actions")
	if err == nil {
		t.Fatalf("expected lint failure on DL4000")
	}

	if !strings.Contains(stdout, "DL4000") {
		t.Errorf("terminal output missing DL4000:\n%s", stdout)
	}
	if !strings.Contains(stdout, "::error ") {
		t.Errorf("GitHub Actions annotations missing:\n%s", stdout)
	}

	data, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("expected report at %s, got %v", reportPath, readErr)
	}
	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, data)
	}
	if len(entries) == 0 {
		t.Fatalf("expected entries in code quality report")
	}
}

func TestLintCmd_FormatDefaultsToTerminal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	writeProject(t, root, "Dockerfile", "FROM nginx:1.27\nMAINTAINER me@example.com\n", "")

	err, stdout := runLint(t, root)
	if err == nil {
		t.Fatalf("expected lint failure on DL4000")
	}
	if !strings.Contains(stdout, "DL4000") {
		t.Errorf("expected terminal output with DL4000, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "::error ") {
		t.Errorf("expected no GitHub Actions annotations by default, got:\n%s", stdout)
	}
}

func TestLintCmd_HiveParentSubstituted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	// Variant Dockerfile that uses the placeholder. Without substitution
	// hadolint would flag DL3006 (untagged FROM); with it, the FROM line
	// is the synthetic __hive__/test:1 reference that passes the rule.
	writeProject(t, root, "Dockerfile", "FROM __hive_parent__\nRUN echo ok\n", "")

	err, stdout := runLint(t, root)
	if err != nil {
		t.Fatalf("expected lint to pass after __hive_parent__ substitution, got %v\n%s", err, stdout)
	}
	if strings.Contains(stdout, "DL3006") {
		t.Errorf("DL3006 must not fire for substituted __hive_parent__; output:\n%s", stdout)
	}
}
