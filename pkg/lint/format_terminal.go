package lint

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Ladicle/tabwriter"
	"golang.org/x/term"
)

const (
	ansiReset        = "\x1b[0m"
	ansiBold         = "\x1b[1m"
	ansiFaint        = "\x1b[2m"
	ansiBrightRed    = "\x1b[91m"
	ansiBrightYellow = "\x1b[93m"
	ansiBrightCyan   = "\x1b[96m"
)

const hadolintWikiBase = "https://github.com/hadolint/hadolint/wiki/"

type TerminalFormat struct {
	Color bool
}

func (f *TerminalFormat) Name() string {
	return "terminal"
}

func (f *TerminalFormat) HasPath() bool { return false }

func (f *TerminalFormat) Render(w io.Writer, findings []Finding) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.TabIndent)
	for i, r := range findings {
		finding := r.Finding
		entries := [][2]string{
			{"Code", finding.Code},
			{"Severity", FormatLevel(finding.Level, f.Color)},
			{"Location", fmt.Sprintf("%s:%d:%d", r.FullPath, finding.Line, finding.Column)},
			{"Link", hadolintWikiBase + finding.Code},
			{"Description", finding.Message},
		}
		for _, e := range entries {
			if _, err := fmt.Fprintf(tw, "%s\t%s\n", boldLabel(e[0], f.Color), e[1]); err != nil {
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

func boldLabel(label string, color bool) string {
	if !color {
		return label
	}
	return ansiBold + label + ansiReset
}

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

func StdoutSupportsColor() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}
