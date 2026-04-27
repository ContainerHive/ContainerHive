//go:build ignore

package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func main() {
	slog.Info("Generating project schema...")
	schema, err := jsonschema.For[model.HiveProjectConfig](&jsonschema.ForOptions{})
	if err != nil {
		slog.Error("Failed to generate schema", "error", err)
		os.Exit(1)
	}

	schema.ID = "https://schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-project/latest"
	schema.Title = "Project configuration"
	schema.Description = "Project-level configuration schema for ContainerHive."

	slog.Info("Writing schema to file...")
	indented, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal indented schema", "error", err)
		os.Exit(1)
	}

	err = os.WriteFile("schemas/project.schema.json", indented, 0644)
	if err != nil {
		slog.Error("Failed to write schema file", "error", err)
		os.Exit(1)
	}
}
