package lint

import (
	"fmt"
	"io"
	"strings"
)

type GitHubActionsFormat struct{}

func (f *GitHubActionsFormat) HasPath() bool { return false }

func (f *GitHubActionsFormat) Name() string {
	return "github-actions"
}

func (f *GitHubActionsFormat) Render(w io.Writer, findings []Finding) error {
	for _, r := range findings {
		cmd := mapSeverityToCommand(r.Finding.Level)
		msg := sanitizeMessage(r.Finding.Message)
		line := fmt.Sprintf("::%s file=%s,line=%d,col=%d,title=%s::%s\n",
			cmd,
			r.Path,
			r.Finding.Line,
			r.Finding.Column,
			r.Finding.Code,
			msg,
		)
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	return nil
}

func mapSeverityToCommand(level string) string {
	switch level {
	case "error":
		return "error"
	case "warning":
		return "warning"
	default:
		return "notice"
	}
}

func sanitizeMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\r\n", "\n")
	msg = strings.ReplaceAll(msg, "\r", "")
	msg = strings.ReplaceAll(msg, "\n", "%0A")
	msg = strings.ReplaceAll(msg, "::", "%3A%3A")
	return msg
}
