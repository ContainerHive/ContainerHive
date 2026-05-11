package hadolint

import (
	"encoding/json"
	"testing"

	gohadolint "github.com/timo-reymann/go-hadolint"
)

func TestMapSeverity(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"error", "blocker"},
		{"warning", "major"},
		{"info", "minor"},
		{"style", "info"},
		{"", "info"},
		{"unknown", "info"},
	}
	for _, tc := range cases {
		if got := mapSeverity(tc.in); got != tc.want {
			t.Errorf("mapSeverity(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestToCodeQuality(t *testing.T) {
	f := gohadolint.Finding{
		File:    "/abs/Dockerfile",
		Line:    7,
		Column:  3,
		Level:   "warning",
		Code:    "DL3006",
		Message: "Always tag the version of an image explicitly",
	}
	entry := ToCodeQuality(f, "images/test/Dockerfile")

	if entry.Description != f.Message {
		t.Errorf("Description = %q, want %q", entry.Description, f.Message)
	}
	if entry.CheckName != "DL3006" {
		t.Errorf("CheckName = %q, want DL3006", entry.CheckName)
	}
	if entry.Severity != "major" {
		t.Errorf("Severity = %q, want major", entry.Severity)
	}
	if entry.Location.Path != "images/test/Dockerfile" {
		t.Errorf("Location.Path = %q, want overridden repo-relative path", entry.Location.Path)
	}
	if entry.Location.Lines.Begin != 7 {
		t.Errorf("Location.Lines.Begin = %d, want 7", entry.Location.Lines.Begin)
	}
	if len(entry.Fingerprint) != 64 { // sha256 hex
		t.Errorf("Fingerprint length = %d, want 64", len(entry.Fingerprint))
	}
}

func TestFingerprintStability(t *testing.T) {
	f := gohadolint.Finding{File: "x", Line: 1, Code: "DL3006", Message: "m"}
	a := ToCodeQuality(f, "p")
	b := ToCodeQuality(f, "p")
	if a.Fingerprint != b.Fingerprint {
		t.Errorf("fingerprint must be stable for identical input: %q vs %q", a.Fingerprint, b.Fingerprint)
	}

	// Different line → different fingerprint.
	f2 := f
	f2.Line = 2
	if c := ToCodeQuality(f2, "p"); c.Fingerprint == a.Fingerprint {
		t.Error("fingerprint must differ when line differs")
	}
}

func TestMarshalCodeQuality_EmptyIsArray(t *testing.T) {
	data, err := MarshalCodeQuality(nil)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out []CodeQualityEntry
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(data) == "null" {
		t.Errorf("empty report must serialize as [], got null")
	}
}

func TestMarshalCodeQuality_RoundTrip(t *testing.T) {
	entries := []CodeQualityEntry{
		ToCodeQuality(gohadolint.Finding{
			File: "Dockerfile", Line: 1, Code: "DL4000", Level: "error", Message: "bad",
		}, "Dockerfile"),
	}
	data, err := MarshalCodeQuality(entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out []CodeQualityEntry
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 || out[0].CheckName != "DL4000" {
		t.Errorf("round trip mismatch: %+v", out)
	}
}
