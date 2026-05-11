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

	"github.com/ContainerHive/ContainerHive/pkg/model"
	gohadolint "github.com/timo-reymann/go-hadolint"
	"github.com/urfave/cli/v3"
)

func gohadolintFinding(code, level string, line, column int, message string) gohadolint.Finding {
	return gohadolint.Finding{
		Code:    code,
		Level:   level,
		Line:    line,
		Column:  column,
		Message: message,
		File:    "Dockerfile",
	}
}

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

func TestLintCmd_CodeClimateReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hadolint integration test in short mode")
	}
	root := t.TempDir()
	writeProject(t, root, "Dockerfile", "FROM nginx:1.27\nMAINTAINER me@example.com\n", "")

	reportPath := filepath.Join(root, "gl-code-quality-report.json")
	err, _ := runLint(t, root, "--codeclimate-report", reportPath)
	// The Dockerfile has a DL4000 error → command should fail, but report
	// must still be written before the action returns the error.
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
	// Path must be repo-relative — not the absolute temp-dir path.
	if filepath.IsAbs(path) {
		t.Errorf("report path is absolute (%s); expected repo-relative", path)
	}
	if !strings.HasSuffix(path, "Dockerfile") {
		t.Errorf("report path does not end in Dockerfile: %s", path)
	}
}

func TestRenderFindingsTable(t *testing.T) {
	rows := []tableFinding{
		{
			path:     "images/test/Dockerfile",
			fullPath: "/abs/repo/images/test/Dockerfile",
			finding:  gohadolintFinding("DL4000", "error", 2, 1, "MAINTAINER is deprecated"),
		},
		{
			path:     "images/test/Dockerfile",
			fullPath: "/abs/repo/images/test/Dockerfile",
			finding:  gohadolintFinding("DL3006", "warning", 1, 1, "Always tag the version of an image explicitly"),
		},
	}

	t.Run("plain", func(t *testing.T) {
		var buf bytes.Buffer
		if err := renderFindingsTable(&buf, rows, false); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := buf.String()
		for _, want := range []string{
			"Code", "Severity", "Location", "Link", "Description",
			"DL4000", "DL3006",
			"ERROR", "WARNING",
			"https://github.com/hadolint/hadolint/wiki/DL4000",
			"/abs/repo/images/test/Dockerfile:2:1",
			"MAINTAINER is deprecated",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q\n%s", want, out)
			}
		}
		if strings.Contains(out, "\x1b[") {
			t.Errorf("output must not contain ANSI escapes when color is disabled:\n%s", out)
		}
	})

	t.Run("colored", func(t *testing.T) {
		var buf bytes.Buffer
		if err := renderFindingsTable(&buf, rows, true); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, ansiBold+"Code"+ansiReset) {
			t.Errorf("label missing bold escape:\n%s", out)
		}
		if !strings.Contains(out, ansiBrightRed+"ERROR"+ansiReset) {
			t.Errorf("ERROR severity missing red escape:\n%s", out)
		}
		if !strings.Contains(out, ansiBrightYellow+"WARNING"+ansiReset) {
			t.Errorf("WARNING severity missing yellow escape:\n%s", out)
		}
	})
}

func TestFormatLevel(t *testing.T) {
	cases := []struct {
		level    string
		color    bool
		want     string
		wantTags bool // whether result must include ANSI escapes
	}{
		{level: "error", color: false, want: "ERROR"},
		{level: "warning", color: false, want: "WARNING"},
		{level: "error", color: true, want: ansiBrightRed + "ERROR" + ansiReset, wantTags: true},
		{level: "warning", color: true, want: ansiBrightYellow + "WARNING" + ansiReset, wantTags: true},
		{level: "info", color: true, want: ansiBrightCyan + "INFO" + ansiReset, wantTags: true},
		{level: "style", color: true, want: ansiFaint + "STYLE" + ansiReset, wantTags: true},
		// Unknown levels stay plain even with color enabled so we don't emit
		// a stray escape sequence for a future hadolint severity.
		{level: "wat", color: true, want: "WAT"},
	}
	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			got := formatLevel(tc.level, tc.color)
			if got != tc.want {
				t.Errorf("formatLevel(%q, %v) = %q, want %q", tc.level, tc.color, got, tc.want)
			}
			hasEsc := strings.Contains(got, "\x1b[")
			if hasEsc != tc.wantTags {
				t.Errorf("formatLevel(%q, %v): escape presence = %v, want %v", tc.level, tc.color, hasEsc, tc.wantTags)
			}
		})
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
