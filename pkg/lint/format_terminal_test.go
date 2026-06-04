package lint

import (
	"bytes"
	"strings"
	"testing"
)

func TestTerminalFormat_Render(t *testing.T) {
	cols := []Finding{
		{
			Path:     "images/test/Dockerfile",
			FullPath: "/abs/repo/images/test/Dockerfile",
			Finding:  gohadolintFinding("DL4000", "error", 2, 1, "MAINTAINER is deprecated"),
		},
		{
			Path:     "images/test/Dockerfile",
			FullPath: "/abs/repo/images/test/Dockerfile",
			Finding:  gohadolintFinding("DL3006", "warning", 1, 1, "Always tag the version of an image explicitly"),
		},
	}

	t.Run("plain", func(t *testing.T) {
		f := &TerminalFormat{Color: false}
		var buf bytes.Buffer
		if err := f.Render(&buf, cols); err != nil {
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
		f := &TerminalFormat{Color: true}
		var buf bytes.Buffer
		if err := f.Render(&buf, cols); err != nil {
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
