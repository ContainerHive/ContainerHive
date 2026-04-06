package logging

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestSetup(t *testing.T) {
	var buf bytes.Buffer

	t.Run("default level", func(t *testing.T) {
		buf.Reset()
		Setup(&buf, "info")
		slog.Info("test info message")
		slog.Debug("test debug message")

		output := buf.String()
		if !contains(output, "test info message") {
			t.Errorf("expected output to contain 'test info message', got %q", output)
		}
		if contains(output, "test debug message") {
			t.Errorf("expected output to NOT contain 'test debug message', got %q", output)
		}
	})

	t.Run("debug level", func(t *testing.T) {
		buf.Reset()
		Setup(&buf, "debug")
		slog.Debug("test debug message")

		output := buf.String()
		if !contains(output, "test debug message") {
			t.Errorf("expected output to contain 'test debug message', got %q", output)
		}
	})

	t.Run("no color", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		buf.Reset()
		Setup(&buf, "info")
		slog.Info("test message")

		output := buf.String()
		// tint uses ANSI escape codes for color, e.g., \033[
		if contains(output, "\033[") {
			t.Errorf("expected output to NOT contain ANSI escape codes (no color), got %q", output)
		}
	})

	t.Run("color enabled", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		buf.Reset()
		Setup(&buf, "info")
		slog.Info("test message")

		output := buf.String()
		// We expect colors since we didn't set NO_COLOR
		if !contains(output, "\033[") {
			t.Errorf("expected output to contain ANSI escape codes (color), got %q", output)
		}
	})
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
