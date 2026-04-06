package logging

import (
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func newEntry(level logrus.Level, msg string, fields logrus.Fields) *logrus.Entry {
	entry := logrus.NewEntry(logrus.StandardLogger())
	entry.Level = level
	entry.Message = msg
	entry.Time = time.Date(2026, 4, 6, 21, 36, 47, 0, time.UTC)
	entry.Data = fields
	return entry
}

func TestTintFormatter_MatchesSlogFormat(t *testing.T) {
	f := &TintFormatter{TimeFormat: time.DateTime}
	out, err := f.Format(newEntry(logrus.InfoLevel, "PASS", logrus.Fields{"image": "base-ci"}))
	if err != nil {
		t.Fatal(err)
	}

	s := string(out)
	if !strings.Contains(s, "2026-04-06 21:36:47") {
		t.Errorf("expected datetime formatted timestamp, got: %q", s)
	}
	if !strings.Contains(s, "INF") {
		t.Errorf("expected INF level label, got: %q", s)
	}
	if !strings.Contains(s, "PASS") {
		t.Errorf("expected message, got: %q", s)
	}
	if !strings.Contains(s, "image=") || !strings.Contains(s, "base-ci") {
		t.Errorf("expected field image=base-ci, got: %q", s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Errorf("expected trailing newline, got: %q", s)
	}
	if strings.HasSuffix(s, "\n\n") {
		t.Errorf("expected single trailing newline, got: %q", s)
	}
}

func TestTintFormatter_Levels(t *testing.T) {
	f := &TintFormatter{}
	cases := []struct {
		level logrus.Level
		label string
	}{
		{logrus.DebugLevel, "DBG"},
		{logrus.InfoLevel, "INF"},
		{logrus.WarnLevel, "WRN"},
		{logrus.ErrorLevel, "ERR"},
		{logrus.FatalLevel, "ERR"},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			out, err := f.Format(newEntry(tc.level, "test", nil))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), tc.label) {
				t.Errorf("expected %q in output, got: %q", tc.label, string(out))
			}
		})
	}
}

func TestTintFormatter_TrimsTrailingNewline(t *testing.T) {
	f := &TintFormatter{}
	out, err := f.Format(newEntry(logrus.InfoLevel, "stdout: git version 2.43.0\n", nil))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "\n\n") {
		t.Errorf("expected no double newline, got: %q", s)
	}
	if !strings.Contains(s, "git version 2.43.0") {
		t.Errorf("expected message content, got: %q", s)
	}
}

func TestTintFormatter_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	f := &TintFormatter{}
	out, err := f.Format(newEntry(logrus.InfoLevel, "test", nil))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "\x1b[") {
		t.Errorf("expected no ANSI codes with NO_COLOR, got: %q", string(out))
	}
}

func TestTintFormatter_QuotesValues(t *testing.T) {
	f := &TintFormatter{}
	out, err := f.Format(newEntry(logrus.InfoLevel, "test", logrus.Fields{"path": "/some dir/file"}))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"/some dir/file"`) {
		t.Errorf("expected quoted value with space, got: %q", s)
	}
}
