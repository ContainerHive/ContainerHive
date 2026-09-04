package sbom

import (
	"encoding/json"
	"fmt"
	"strings"
)

// stripCycloneDXProperties removes the "properties" field from every
// component (and the metadata component, if present) in a CycloneDX JSON
// document. syft attaches its own package/location/metadata bookkeeping as
// properties on every component, which is not required by the CycloneDX
// spec and is not read by ContainerHive's report generator or by GitLab's
// dependency/container scanning, but can dominate the file size.
func stripCycloneDXProperties(data []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse cyclonedx json: %w", err)
	}

	if components, ok := doc["components"].([]any); ok {
		for _, c := range components {
			if component, ok := c.(map[string]any); ok {
				delete(component, "properties")
			}
		}
	}

	if metadata, ok := doc["metadata"].(map[string]any); ok {
		if component, ok := metadata["component"].(map[string]any); ok {
			delete(component, "properties")
		}
	}

	return json.Marshal(doc)
}

// isCycloneDXFormat reports whether outputFormat refers to a CycloneDX
// encoding (e.g. "cyclonedx-json").
func isCycloneDXFormat(outputFormat string) bool {
	return strings.HasPrefix(outputFormat, "cyclonedx")
}
