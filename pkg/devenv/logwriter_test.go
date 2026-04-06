package devenv

import (
	"bytes"
	"strings"
	"testing"
)

func testColors() LogColors {
	return LogColors{
		Debug:   "\x1b[2m",
		Info:    "\x1b[34;1m",
		Warning: "\x1b[33m",
		Error:   "\x1b[31m",
		Message: "\x1b[37;1m",
		Gray:    "\x1b[2m",
		Reset:   "\x1b[0m",
	}
}

func TestLogWriter_InfoLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())
	line := `time="2024-01-15T10:30:45Z" level=info msg="starting buildkitd" version=v0.12.0` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	if !strings.Contains(out, testColors().Info) {
		t.Errorf("expected info color for level, got: %q", out)
	}
	if !strings.Contains(out, testColors().Message) {
		t.Errorf("expected message color (white) for msg value, got: %q", out)
	}
	if !strings.Contains(out, "starting buildkitd") {
		t.Errorf("expected message text in output, got: %q", out)
	}
	if !strings.Contains(out, testColors().Gray) {
		t.Errorf("expected gray color for timestamp/fields, got: %q", out)
	}
}

func TestLogWriter_WarningLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())
	line := `time="2024-01-15T10:30:45Z" level=warning msg="deprecated option"` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	if !strings.Contains(out, testColors().Warning) {
		t.Errorf("expected warning color for level, got: %q", out)
	}
}

func TestLogWriter_ErrorLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())
	line := `time="2024-01-15T10:30:45Z" level=error msg="failed to start"` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	if !strings.Contains(out, testColors().Error) {
		t.Errorf("expected error color for level, got: %q", out)
	}
}

func TestLogWriter_DebugLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())
	line := `time="2024-01-15T10:30:45Z" level=debug msg="trace info"` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	if !strings.Contains(out, testColors().Debug) {
		t.Errorf("expected debug color for level, got: %q", out)
	}
}

func TestLogWriter_MessageIsWhite(t *testing.T) {
	var buf bytes.Buffer
	colors := testColors()
	w := NewLogWriter(&buf, colors)
	line := `time="2024-01-15T10:30:45Z" level=info msg="hello world"` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	// The msg value should be wrapped in the Message color (bright white).
	expected := colors.Message + `"hello world"` + colors.Reset
	if !strings.Contains(out, expected) {
		t.Errorf("expected msg value in white (%q), got: %q", expected, out)
	}
}

func TestLogWriter_KeysAreGray(t *testing.T) {
	var buf bytes.Buffer
	colors := testColors()
	w := NewLogWriter(&buf, colors)
	line := `time="2024-01-15T10:30:45Z" level=info msg="hello" worker=abc` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	// The "msg=" key and trailing " worker=abc" should both use gray.
	if !strings.Contains(out, colors.Gray+"msg="+colors.Reset) {
		t.Errorf("expected msg= key in gray, got: %q", out)
	}
	if !strings.Contains(out, colors.Gray+" worker=abc"+colors.Reset) {
		t.Errorf("expected trailing fields in gray, got: %q", out)
	}
}

func TestLogWriter_TimestampIsGray(t *testing.T) {
	var buf bytes.Buffer
	colors := testColors()
	w := NewLogWriter(&buf, colors)
	line := `time="2024-01-15T10:30:45Z" level=info msg="hi"` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	if !strings.HasPrefix(out, colors.Gray) {
		t.Errorf("expected output to start with gray for timestamp, got: %q", out)
	}
}

func TestLogWriter_NoColor(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, LogColors{})
	line := `time="2024-01-15T10:30:45Z" level=info msg="hello"` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Errorf("expected no ANSI codes with empty colors, got: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected message preserved, got: %q", out)
	}
}

func TestLogWriter_NonLogrusLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())
	line := "some plain text without logrus format\n"
	w.Write([]byte(line))

	out := buf.String()
	if !strings.Contains(out, "some plain text without logrus format") {
		t.Errorf("expected plain line passed through, got: %q", out)
	}
}

func TestLogWriter_PartialLines(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())

	// Write in two chunks.
	w.Write([]byte(`time="2024-01-15T10:30:45Z" level=info msg="`))
	if buf.Len() != 0 {
		t.Errorf("expected no output for partial line, got: %q", buf.String())
	}

	w.Write([]byte("hello world\"\n"))
	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected reassembled line, got: %q", out)
	}
}

func TestLogWriter_MultipleLines(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())

	input := `time="2024-01-15T10:30:45Z" level=info msg="first"
time="2024-01-15T10:30:46Z" level=warning msg="second"
`
	w.Write([]byte(input))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "first") {
		t.Errorf("expected first message, got: %q", lines[0])
	}
	if !strings.Contains(lines[1], "second") {
		t.Errorf("expected second message, got: %q", lines[1])
	}
}

func TestLogWriter_Flush(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors()).(*logWriter)

	w.Write([]byte(`time="2024-01-15T10:30:45Z" level=info msg="no newline"`))
	if buf.Len() != 0 {
		t.Errorf("expected no output before flush, got: %q", buf.String())
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if !strings.Contains(buf.String(), "no newline") {
		t.Errorf("expected flushed content, got: %q", buf.String())
	}
}

func TestLogWriter_EscapedQuotesInMsg(t *testing.T) {
	var buf bytes.Buffer
	w := NewLogWriter(&buf, testColors())
	line := `time="2024-01-15T10:30:45Z" level=info msg="found worker \"abc\""` + "\n"
	w.Write([]byte(line))

	out := buf.String()
	if !strings.Contains(out, `found worker \"abc\"`) {
		t.Errorf("expected escaped quote content preserved, got: %q", out)
	}
}

func TestExtractField(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		prefix string
		value  string
		found  bool
	}{
		{
			name:   "unquoted value",
			input:  "level=info msg=hello",
			prefix: "level=",
			value:  "info",
			found:  true,
		},
		{
			name:   "quoted value",
			input:  `msg="hello world"`,
			prefix: "msg=",
			value:  "hello world",
			found:  true,
		},
		{
			name:   "escaped quotes",
			input:  `msg="found \"worker\""`,
			prefix: "msg=",
			value:  `found \"worker\"`,
			found:  true,
		},
		{
			name:   "not found",
			input:  "level=info",
			prefix: "msg=",
			value:  "",
			found:  false,
		},
		{
			name:   "value at end",
			input:  "level=info",
			prefix: "level=",
			value:  "info",
			found:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, start, _ := extractField(tc.input, tc.prefix)
			if tc.found && start < 0 {
				t.Fatalf("expected field to be found")
			}
			if !tc.found && start >= 0 {
				t.Fatalf("expected field not to be found")
			}
			if tc.found && val != tc.value {
				t.Errorf("extractField(%q, %q) value = %q, want %q", tc.input, tc.prefix, val, tc.value)
			}
		})
	}
}
