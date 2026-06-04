package lint

import (
	"bytes"
	"strings"
	"testing"

	gohadolint "github.com/timo-reymann/go-hadolint"
)

func TestGitHubActionsFormat_Render(t *testing.T) {
	cols := []Finding{
		{
			Path:     "images/test/Dockerfile",
			FullPath: "/abs/repo/images/test/Dockerfile",
			Finding: gohadolint.Finding{
				Code:    "DL4000",
				Level:   "error",
				Line:    2,
				Column:  1,
				Message: "MAINTAINER is deprecated",
				File:    "Dockerfile",
			},
		},
		{
			Path:     "images/test/Dockerfile",
			FullPath: "/abs/repo/images/test/Dockerfile",
			Finding: gohadolint.Finding{
				Code:    "DL3006",
				Level:   "warning",
				Line:    1,
				Column:  1,
				Message: "Always tag the version of an image explicitly",
				File:    "Dockerfile",
			},
		},
		{
			Path:     "images/other/Dockerfile",
			FullPath: "/abs/repo/images/other/Dockerfile",
			Finding: gohadolint.Finding{
				Code:    "DL3008",
				Level:   "info",
				Line:    5,
				Column:  3,
				Message: "Pin versions in apt get install",
				File:    "Dockerfile",
			},
		},
		{
			Path:     "images/other/Dockerfile",
			FullPath: "/abs/repo/images/other/Dockerfile",
			Finding: gohadolint.Finding{
				Code:    "SC1000",
				Level:   "style",
				Line:    10,
				Column:  1,
				Message: "Style finding",
				File:    "Dockerfile",
			},
		},
	}

	t.Run("severity mapping", func(t *testing.T) {
		f := &GitHubActionsFormat{}
		var buf bytes.Buffer
		if err := f.Render(&buf, cols); err != nil {
			t.Fatalf("render: %v", err)
		}
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines, got %d", len(lines))
		}

		if !strings.HasPrefix(lines[0], "::error ") {
			t.Errorf("DL4000: expected ::error, got %s", lines[0])
		}
		if !strings.HasPrefix(lines[1], "::warning ") {
			t.Errorf("DL3006: expected ::warning, got %s", lines[1])
		}
		if !strings.HasPrefix(lines[2], "::notice ") {
			t.Errorf("DL3008: expected ::notice, got %s", lines[2])
		}
		if !strings.HasPrefix(lines[3], "::notice ") {
			t.Errorf("SC1000: expected ::notice, got %s", lines[3])
		}
	})

	t.Run("fields", func(t *testing.T) {
		f := &GitHubActionsFormat{}
		var buf bytes.Buffer
		if err := f.Render(&buf, cols[:1]); err != nil {
			t.Fatalf("render: %v", err)
		}
		line := strings.TrimRight(buf.String(), "\n")
		if !strings.Contains(line, "file=images/test/Dockerfile") {
			t.Errorf("missing file=, got %s", line)
		}
		if !strings.Contains(line, "line=2") {
			t.Errorf("missing line=2, got %s", line)
		}
		if !strings.Contains(line, "col=1") {
			t.Errorf("missing col=1, got %s", line)
		}
		if !strings.Contains(line, "title=DL4000") {
			t.Errorf("missing title=DL4000, got %s", line)
		}
		if !strings.Contains(line, "::MAINTAINER is deprecated") {
			t.Errorf("missing message, got %s", line)
		}
	})
}

func TestGitHubActionsFormat_SanitizeMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{name: "plain", msg: "simple message", want: "simple message"},
		{name: "newline", msg: "line1\nline2", want: "line1%0Aline2"},
		{name: "carriage return", msg: "line1\rline2", want: "line1line2"},
		{name: "crlf", msg: "line1\r\nline2", want: "line1%0Aline2"},
		{name: "double colon", msg: "error::something", want: "error%3A%3Asomething"},
		{name: "combined", msg: "line1\n::error", want: "line1%0A%3A%3Aerror"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeMessage(tc.msg)
			if got != tc.want {
				t.Errorf("sanitizeMessage(%q) = %q, want %q", tc.msg, got, tc.want)
			}
		})
	}
}

func TestGitHubActionsFormat_EmptyFindings(t *testing.T) {
	f := &GitHubActionsFormat{}
	var buf bytes.Buffer
	if err := f.Render(&buf, nil); err != nil {
		t.Fatalf("render: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty findings, got %q", buf.String())
	}
}

func TestGitHubActionsFormat_SingleFinding(t *testing.T) {
	f := &GitHubActionsFormat{}
	finding := Finding{
		Path:     "Dockerfile",
		FullPath: "/abs/Dockerfile",
		Finding: gohadolint.Finding{
			Code:    "DL3006",
			Level:   "warning",
			Line:    1,
			Column:  1,
			Message: "Always tag the version",
			File:    "Dockerfile",
		},
	}
	var buf bytes.Buffer
	if err := f.Render(&buf, []Finding{finding}); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := strings.TrimRight(buf.String(), "\n")
	want := "::warning file=Dockerfile,line=1,col=1,title=DL3006::Always tag the version"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}
