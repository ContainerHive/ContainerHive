package sbom

import (
	"encoding/json"
	"testing"
)

func TestStripCycloneDXProperties(t *testing.T) {
	input := `{
		"bomFormat": "CycloneDX",
		"specVersion": "1.5",
		"metadata": {
			"component": {
				"name": "root-image",
				"properties": [{"name": "syft:package:foo", "value": "bar"}]
			}
		},
		"components": [
			{"name": "pkg-a", "version": "1.0", "properties": [{"name": "syft:location:0:path", "value": "/usr/lib/pkg-a"}]},
			{"name": "pkg-b", "version": "2.0"}
		]
	}`

	out, err := stripCycloneDXProperties([]byte(input))
	if err != nil {
		t.Fatalf("stripCycloneDXProperties failed: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("stripped output is not valid JSON: %v", err)
	}

	components := doc["components"].([]any)
	if len(components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(components))
	}
	for _, c := range components {
		component := c.(map[string]any)
		if _, ok := component["properties"]; ok {
			t.Errorf("expected properties to be stripped from component %v", component)
		}
		if _, ok := component["name"]; !ok {
			t.Errorf("expected component name to be preserved: %v", component)
		}
	}

	metadataComponent := doc["metadata"].(map[string]any)["component"].(map[string]any)
	if _, ok := metadataComponent["properties"]; ok {
		t.Errorf("expected properties to be stripped from metadata component")
	}
	if metadataComponent["name"] != "root-image" {
		t.Errorf("expected metadata component name to be preserved")
	}
}

func TestIsCycloneDXFormat(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{"cyclonedx-json", true},
		{"cyclonedx-xml", true},
		{"spdx-json", false},
		{"syft-json", false},
	}
	for _, tt := range tests {
		if got := isCycloneDXFormat(tt.format); got != tt.want {
			t.Errorf("isCycloneDXFormat(%q) = %v, want %v", tt.format, got, tt.want)
		}
	}
}
