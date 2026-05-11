package lint

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ladicle/tabwriter"
	"golang.org/x/term"
)

// ANSI codes matching the palette already used by pkg/logging and
// pkg/progress so lint findings blend in with the rest of ch's output.
const (
	ansiReset        = "\x1b[0m"
	ansiBold         = "\x1b[1m"
	ansiFaint        = "\x1b[2m"
	ansiBrightRed    = "\x1b[91m"
	ansiBrightYellow = "\x1b[93m"
	ansiBrightCyan   = "\x1b[96m"
)

// hadolintWikiBase is the documentation root for hadolint rule codes. Each
// finding's check_name (e.g. DL3006) appends as-is.
const hadolintWikiBase = "https://github.com/hadolint/hadolint/wiki/"

// RenderFindings writes findings to w in the text/key-value layout used by
// gitlab-ci-verify (each finding is a small block with bold labels and a
// blank line between entries).
func RenderFindings(w io.Writer, findings []Finding, color bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.TabIndent)
	for i, r := range findings {
		f := r.Finding
		entries := [][2]string{
			{"Code", f.Code},
			{"Severity", FormatLevel(f.Level, color)},
			{"Location", fmt.Sprintf("%s:%d:%d", r.FullPath, f.Line, f.Column)},
			{"Link", hadolintWikiBase + f.Code},
			{"Description", f.Message},
		}
		for _, e := range entries {
			if _, err := fmt.Fprintf(tw, "%s\t%s\n", boldLabel(e[0], color), e[1]); err != nil {
				return err
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		if i < len(findings)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	return nil
}

// boldLabel wraps a label in the ANSI bold escape when color is enabled.
func boldLabel(label string, color bool) string {
	if !color {
		return label
	}
	return ansiBold + label + ansiReset
}

// FormatLevel renders an uppercased hadolint severity, optionally wrapped in
// ANSI color escapes that match the rest of ch's output palette.
func FormatLevel(level string, color bool) string {
	upper := strings.ToUpper(level)
	if !color {
		return upper
	}
	var code string
	switch level {
	case "error":
		code = ansiBrightRed
	case "warning":
		code = ansiBrightYellow
	case "info":
		code = ansiBrightCyan
	case "style":
		code = ansiFaint
	default:
		return upper
	}
	return code + upper + ansiReset
}

// StdoutSupportsColor reports whether ANSI color escapes are appropriate on
// the current stdout: stdout must be a terminal and the NO_COLOR convention
// (https://no-color.org) must not be set.
func StdoutSupportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// relativeReportPath returns path relative to projectRoot so the report entry
// matches what a GitLab runner expects (repo-relative paths). Falls back to
// the absolute path if it can't be made relative.
func relativeReportPath(projectRoot, path string) string {
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(abs, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
