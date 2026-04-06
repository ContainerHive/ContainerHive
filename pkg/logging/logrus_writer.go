package logging

import (
	"bytes"
	"io"
	"os"
	"strings"
)

// LogColors holds ANSI escape sequences used to colorize logrus-formatted log output.
type LogColors struct {
	Debug   string // debug level
	Info    string // info level
	Warning string // warning level
	Error   string // error level
	Message string // msg value — bright/white
	Gray    string // timestamps, keys, and field values
	Reset   string // reset all attributes
}

// DefaultLogColors returns the default color scheme for logrus-formatted log output,
// consistent with the progress and slog color palettes.
func DefaultLogColors() LogColors {
	if os.Getenv("NO_COLOR") != "" {
		return LogColors{}
	}
	return LogColors{
		Debug:   "\x1b[2m",    // dim
		Info:    "\x1b[34;1m", // bright blue
		Warning: "\x1b[33m",   // yellow
		Error:   "\x1b[31m",   // red
		Message: "\x1b[37;1m", // bright white
		Gray:    "\x1b[2m",    // dim (gray)
		Reset:   "\x1b[0m",
	}
}

// NewLogWriter wraps w with a writer that colorizes logrus-formatted lines.
// Pass a zero LogColors to disable colorization.
func NewLogWriter(w io.Writer, colors LogColors) io.Writer {
	return &logWriter{
		w:      w,
		colors: colors,
		buf:    bytes.Buffer{},
	}
}

type logWriter struct {
	w      io.Writer
	colors LogColors
	buf    bytes.Buffer
}

func (lw *logWriter) Write(p []byte) (int, error) {
	n := len(p)
	lw.buf.Write(p)

	for {
		idx := bytes.IndexByte(lw.buf.Bytes(), '\n')
		if idx < 0 {
			break
		}
		line := make([]byte, idx)
		lw.buf.Read(line)
		lw.buf.ReadByte() // consume '\n'

		formatted := lw.formatLine(string(line))
		if _, err := io.WriteString(lw.w, formatted+"\n"); err != nil {
			return n, err
		}
	}
	return n, nil
}

// Flush writes any remaining buffered bytes that were not terminated with a newline.
func (lw *logWriter) Flush() error {
	if lw.buf.Len() == 0 {
		return nil
	}
	formatted := lw.formatLine(lw.buf.String())
	lw.buf.Reset()
	_, err := io.WriteString(lw.w, formatted+"\n")
	return err
}

func (lw *logWriter) formatLine(line string) string {
	if lw.colors.Reset == "" {
		return line
	}

	level, levelIdx, levelEnd := extractField(line, "level=")
	if levelIdx < 0 {
		return line
	}

	levelColor := lw.levelColor(level)

	var b strings.Builder
	b.Grow(len(line) + 40)

	// Everything before "level=" is the timestamp portion — gray.
	lw.writeColored(&b, lw.colors.Gray, line[:levelIdx])

	// The level=... field — colored by severity.
	lw.writeColored(&b, levelColor, line[levelIdx:levelEnd])

	rest := line[levelEnd:]

	// Extract msg="..." — white, everything else gray.
	_, msgIdx, msgEnd := extractField(rest, "msg=")
	if msgIdx < 0 {
		lw.writeColored(&b, lw.colors.Gray, rest)
		return b.String()
	}

	// Space between level and msg — gray.
	lw.writeColored(&b, lw.colors.Gray, rest[:msgIdx])
	// "msg=" key gray, value white.
	lw.writeColored(&b, lw.colors.Gray, "msg=")
	lw.writeColored(&b, lw.colors.Message, rest[msgIdx+len("msg="):msgEnd])
	// Trailing fields — gray.
	if msgEnd < len(rest) {
		lw.writeColored(&b, lw.colors.Gray, rest[msgEnd:])
	}

	return b.String()
}

func (lw *logWriter) writeColored(b *strings.Builder, color, text string) {
	if text == "" {
		return
	}
	if color == "" {
		b.WriteString(text)
		return
	}
	b.WriteString(color)
	b.WriteString(text)
	b.WriteString(lw.colors.Reset)
}

func (lw *logWriter) levelColor(level string) string {
	switch level {
	case "debug", "trace":
		return lw.colors.Debug
	case "info":
		return lw.colors.Info
	case "warning", "warn":
		return lw.colors.Warning
	case "error", "fatal", "panic":
		return lw.colors.Error
	default:
		return ""
	}
}

// extractField finds a logrus-style field like key=value or key="quoted value"
// and returns the value (unquoted), the start index of the field, and the end index.
// Returns ("", -1, -1) if the field is not found.
func extractField(s, prefix string) (value string, start, end int) {
	idx := strings.Index(s, prefix)
	if idx < 0 {
		return "", -1, -1
	}
	valStart := idx + len(prefix)
	if valStart >= len(s) {
		return "", idx, valStart
	}

	if s[valStart] == '"' {
		// Quoted value — scan for unescaped closing quote.
		i := valStart + 1
		for i < len(s) {
			if s[i] == '\\' && i+1 < len(s) {
				i += 2
				continue
			}
			if s[i] == '"' {
				return s[valStart+1 : i], idx, i + 1
			}
			i++
		}
		// Unterminated quote — take to end.
		return s[valStart+1:], idx, len(s)
	}

	// Unquoted value — ends at next space.
	spaceIdx := strings.IndexByte(s[valStart:], ' ')
	if spaceIdx < 0 {
		return s[valStart:], idx, len(s)
	}
	return s[valStart : valStart+spaceIdx], idx, valStart + spaceIdx
}
