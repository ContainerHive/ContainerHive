package report

import "encoding/json"

type ProjectReport struct {
	GeneratedAt string        `json:"generatedAt"`
	Images      []ImageReport `json:"images"`
}

type Report struct {
	Icon string `json:"icon,omitempty"`
}

type ImageReport struct {
	Name      string            `json:"name"`
	Report    Report            `json:"report,omitempty"`
	Versions  map[string]string `json:"versions,omitempty"`
	Platforms []string          `json:"platforms,omitempty"`
	Tags      []TagReport       `json:"tags"`
	Variants  []VariantReport   `json:"variants,omitempty"`
	SBOM      []SBOMPackage     `json:"sbom,omitempty"`
}

type VariantReport struct {
	Name      string      `json:"name"`
	Report    Report      `json:"report,omitempty"`
	TagSuffix string      `json:"tagSuffix"`
	Platforms []string    `json:"platforms,omitempty"`
	Tags      []TagReport `json:"tags"`
}

type TagReport struct {
	Name      string            `json:"name"`
	BuildArgs map[string]string `json:"buildArgs,omitempty"`
	Platforms []PlatformReport  `json:"platforms,omitempty"`
}

type PlatformReport struct {
	Platform string        `json:"platform"`
	SBOM     []SBOMPackage `json:"sbom,omitempty"`
}

type SBOMPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (s *SBOMPackage) MarshalJSON() ([]byte, error) {
	type Alias SBOMPackage
	return json.Marshal(&struct {
		Alias
		Version any `json:"version,omitempty"`
	}{
		Alias:   Alias(*s),
		Version: s.Version,
	})
}
