package report

import "encoding/json"

type ProjectReport struct {
	GeneratedAt  string        `json:"generatedAt"`
	Source       string        `json:"source"`
	RegistryAddr string        `json:"registryAddr,omitempty"`
	Images       []ImageReport `json:"images"`
}

type ImageReport struct {
	Name     string            `json:"name"`
	Icon     string            `json:"icon,omitempty"`
	Versions map[string]string `json:"versions,omitempty"`
	Aliases  []string          `json:"aliases,omitempty"`
	Tags     []TagReport       `json:"tags"`
	Variants []VariantReport   `json:"variants,omitempty"`
}

type VariantReport struct {
	Name      string      `json:"name"`
	TagSuffix string      `json:"tagSuffix"`
	Tags      []TagReport `json:"tags"`
}

type TagReport struct {
	Name      string            `json:"name"`
	Platforms []PlatformReport  `json:"platforms"`
	BuildArgs map[string]string `json:"buildArgs,omitempty"`
}

type PlatformReport struct {
	Platform  string            `json:"platform"`
	Digest    string            `json:"digest,omitempty"`
	Size      int64             `json:"size"`
	HasSBOM   bool              `json:"hasSbom"`
	SBOM      []SBOMPackage     `json:"sbom,omitempty"`
	BuildArgs map[string]string `json:"buildArgs,omitempty"`
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
