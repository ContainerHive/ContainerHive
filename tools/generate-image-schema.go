//go:build ignore

package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/google/jsonschema-go/jsonschema"
)

func main() {
	slog.Info("Generating image schema...")
	schema, err := jsonschema.For[model.ImageDefinitionConfig](&jsonschema.ForOptions{})
	if err != nil {
		slog.Error("Failed to generate schema", "error", err)
		os.Exit(1)
	}

	schema.ID = "https://schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-image/latest"
	schema.Title = "Image definition"
	schema.Description = "Image definition configuration schema for ContainerHive."

	relaxLevelSelect(schema)

	slog.Info("Writing schema to file...")
	indented, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal indented schema", "error", err)
		os.Exit(1)
	}

	err = os.WriteFile("schemas/image.schema.json", indented, 0644)
	if err != nil {
		slog.Error("Failed to write schema file", "error", err)
		os.Exit(1)
	}
}

// relaxLevelSelect widens every reflected model.LevelSelect subschema
// (identified by having exactly the "last"/"all" properties) to also accept
// the scalar forms LevelSelect.UnmarshalYAML supports ("latest", "all", a
// bare integer) — struct reflection only sees the mapping form.
func relaxLevelSelect(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	if isLevelSelectSchema(s) {
		widened := &jsonschema.Schema{
			Description: s.Description,
			AnyOf: []*jsonschema.Schema{
				{Type: "string", Enum: []any{"latest", "all"}},
				{Type: "integer", Minimum: floatPtr(1)},
				{Types: s.Types, Type: s.Type, Properties: s.Properties, AdditionalProperties: s.AdditionalProperties},
			},
		}
		*s = *widened
		return
	}
	for _, child := range s.Properties {
		relaxLevelSelect(child)
	}
	if s.Items != nil {
		relaxLevelSelect(s.Items)
	}
	for _, def := range s.Defs {
		relaxLevelSelect(def)
	}
}

func isLevelSelectSchema(s *jsonschema.Schema) bool {
	if len(s.Properties) != 2 {
		return false
	}
	_, hasLast := s.Properties["last"]
	_, hasAll := s.Properties["all"]
	return hasLast && hasAll
}

func floatPtr(f float64) *float64 { return &f }
