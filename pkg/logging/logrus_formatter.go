package logging

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
)

// tint-matching ANSI codes
const (
	tintReset        = "\x1b[0m"
	tintFaint        = "\x1b[2m"
	tintBrightGreen  = "\x1b[92m"
	tintBrightYellow = "\x1b[93m"
	tintBrightRed    = "\x1b[91m"
)

// TintFormatter is a logrus Formatter that produces output identical to the
// tint slog handler, so logrus-based libraries blend in with slog output.
type TintFormatter struct {
	// TimeFormat controls the timestamp layout (default: time.DateTime).
	TimeFormat string
}

func (f *TintFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	noColor := os.Getenv("NO_COLOR") != ""
	tf := f.TimeFormat
	if tf == "" {
		tf = time.DateTime
	}

	var buf []byte

	// timestamp — faint
	if !noColor {
		buf = append(buf, tintFaint...)
	}
	buf = entry.Time.Round(0).AppendFormat(buf, tf)
	if !noColor {
		buf = append(buf, tintReset...)
	}
	buf = append(buf, ' ')

	// level — colored label
	label, color := tintLevel(entry.Level)
	if !noColor && color != "" {
		buf = append(buf, color...)
	}
	buf = append(buf, label...)
	if !noColor && color != "" {
		buf = append(buf, tintReset...)
	}
	buf = append(buf, ' ')

	// message — trim trailing newlines that some libraries embed
	buf = append(buf, strings.TrimRight(entry.Message, "\n")...)
	buf = append(buf, ' ')

	// fields — sorted, faint key= then plain value
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := entry.Data[k]
		if !noColor {
			buf = append(buf, tintFaint...)
		}
		buf = append(buf, k...)
		buf = append(buf, '=')
		if !noColor {
			buf = append(buf, tintReset...)
		}
		buf = appendFormattedValue(buf, v)
		buf = append(buf, ' ')
	}

	// replace trailing space with newline
	if len(buf) > 0 {
		buf[len(buf)-1] = '\n'
	}

	return buf, nil
}

func tintLevel(level logrus.Level) (label, color string) {
	switch level {
	case logrus.TraceLevel, logrus.DebugLevel:
		return "DBG", ""
	case logrus.InfoLevel:
		return "INF", tintBrightGreen
	case logrus.WarnLevel:
		return "WRN", tintBrightYellow
	default:
		return "ERR", tintBrightRed
	}
}

func appendFormattedValue(buf []byte, v interface{}) []byte {
	s := fmt.Sprint(v)
	if tintNeedsQuoting(s) {
		return strconv.AppendQuote(buf, s)
	}
	return append(buf, s...)
}

func tintNeedsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if b == ' ' || b == '=' || b == '"' || b == '\\' || b < 0x20 || b == 0x7f {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}
