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
	slog.Info("Generating chisel schema...")
	schema, err := jsonschema.For[model.ChiselCutSpec](&jsonschema.ForOptions{})
	if err != nil {
		slog.Error("Failed to generate schema", "error", err)
		os.Exit(1)
	}

	schema.ID = "https://schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-chisel/latest"
	schema.Title = "Chisel cut spec"
	schema.Description = "chisel.yaml cut spec schema for ContainerHive."

	relaxChiselRelease(schema)

	slog.Info("Writing schema to file...")
	indented, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal indented schema", "error", err)
		os.Exit(1)
	}

	err = os.WriteFile("schemas/chisel.schema.json", indented, 0644)
	if err != nil {
		slog.Error("Failed to write schema file", "error", err)
		os.Exit(1)
	}
}

// relaxChiselRelease widens the release subschema (identified by its
// "git" property) to also accept the scalar branch name form
// ChiselReleaseRef.UnmarshalYAML supports. Struct reflection only sees the
// mapping form.
func relaxChiselRelease(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	for _, child := range s.Properties {
		if child.Properties == nil {
			continue
		}
		if _, hasGit := child.Properties["git"]; !hasGit {
			continue
		}
		if _, hasRef := child.Properties["ref"]; !hasRef {
			continue
		}
		delete(child.Properties, "name")
		branchForm := child.Properties["git"]
		if child.Required != nil {
			for _, prop := range child.Required {
				if _, found := child.Properties[prop]; !found {
					branchForm = nil
					break
				}
			}
		}
		objectForm := &jsonschema.Schema{Type: "object"}
		if branchForm != nil {
			objectForm.Properties = child.Properties
			objectForm.Required = child.Required
			objectForm.AdditionalProperties = child.AdditionalProperties
		}
		child.AnyOf = []*jsonschema.Schema{
			{Type: "string", Description: "Chisel-releases branch name (e.g. ubuntu-24.04)"},
			objectForm,
		}
		child.Type = ""
		child.Properties = nil
		child.Required = nil
		child.AdditionalProperties = nil
	}
}
