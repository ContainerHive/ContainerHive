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
