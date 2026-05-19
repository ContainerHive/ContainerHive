package hadolint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	gohadolint "github.com/timo-reymann/go-hadolint"
)

// CodeQualityEntry is one finding rendered in GitLab's Code Quality format
// (a subset of the Code Climate spec):
// https://docs.gitlab.com/ee/ci/testing/code_quality.html#implement-a-custom-tool
type CodeQualityEntry struct {
	Description string           `json:"description"`
	CheckName   string           `json:"check_name"`
	Fingerprint string           `json:"fingerprint"`
	Severity    string           `json:"severity"`
	Location    CodeQualityLines `json:"location"`
}

// CodeQualityLines is the location block of a CodeQualityEntry.
type CodeQualityLines struct {
	Path  string         `json:"path"`
	Lines CodeQualityRow `json:"lines"`
}

// CodeQualityRow holds the starting line of a finding.
type CodeQualityRow struct {
	Begin int `json:"begin"`
}

// ToCodeQuality renders a hadolint finding as a GitLab Code Quality entry.
// pathOverride lets the caller record the original Dockerfile path (e.g. the
// repository-relative path) instead of whatever hadolint emitted.
func ToCodeQuality(f gohadolint.Finding, pathOverride string) CodeQualityEntry {
	path := f.File
	if pathOverride != "" {
		path = pathOverride
	}
	return CodeQualityEntry{
		Description: f.Message,
		CheckName:   f.Code,
		Fingerprint: fingerprint(path, f),
		Severity:    mapSeverity(f.Level),
		Location: CodeQualityLines{
			Path:  path,
			Lines: CodeQualityRow{Begin: f.Line},
		},
	}
}

// MarshalCodeQuality serializes a slice of entries to JSON suitable for
// GitLab's artifacts:reports:codequality field.
func MarshalCodeQuality(entries []CodeQualityEntry) ([]byte, error) {
	if entries == nil {
		entries = []CodeQualityEntry{}
	}
	return json.MarshalIndent(entries, "", "  ")
}

// mapSeverity maps hadolint severity strings ("error", "warning", "info",
// "style") to Code Climate severities. Unknown values default to "info" so
// the report stays renderable.
func mapSeverity(level string) string {
	switch level {
	case "error":
		return "blocker"
	case "warning":
		return "major"
	case "info":
		return "minor"
	case "style":
		return "info"
	default:
		return "info"
	}
}

// fingerprint is a stable hash over the fields a human would use to identify
// a finding. It's intentionally not the file contents — moving a Dockerfile
// or rewording its body should not destabilize the fingerprint, but a new
// rule code or message line should.
func fingerprint(path string, f gohadolint.Finding) string {
	h := sha256.New()
	h.Write([]byte(path))
	h.Write([]byte{0})
	h.Write([]byte(f.Code))
	h.Write([]byte{0})
	h.Write([]byte(f.Message))
	h.Write([]byte{0})
	// Include line so two findings of the same code in one file produce
	// distinct fingerprints.
	var lineBytes [8]byte
	for i := 0; i < 8; i++ {
		lineBytes[i] = byte(f.Line >> (i * 8))
	}
	h.Write(lineBytes[:])
	return hex.EncodeToString(h.Sum(nil))
}
