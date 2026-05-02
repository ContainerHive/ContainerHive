//go:build ignore

package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/google/jsonschema-go/jsonschema"
)

type StructureTestConfig struct {
	SchemaVersion       string               `json:"schemaVersion"`
	GlobalEnvVars       []EnvVar             `json:"globalEnvVars,omitempty"`
	CommandTests        []CommandTest        `json:"commandTests,omitempty"`
	FileExistenceTests  []FileExistenceTest  `json:"fileExistenceTests,omitempty"`
	FileContentTests    []FileContentTest    `json:"fileContentTests,omitempty"`
	MetadataTest        *MetadataTest        `json:"metadataTest,omitempty"`
	LicenseTests        []LicenseTest        `json:"licenseTests,omitempty"`
	ContainerRunOptions *ContainerRunOptions `json:"containerRunOptions,omitempty"`
}

type EnvVar struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex,omitempty"`
}

type Label struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex,omitempty"`
}

type CommandTest struct {
	Name           string     `json:"name"`
	Setup          [][]string `json:"setup,omitempty"`
	Teardown       [][]string `json:"teardown,omitempty"`
	EnvVars        []EnvVar   `json:"envVars,omitempty"`
	ExitCode       int        `json:"exitCode"`
	Command        string     `json:"command"`
	Args           []string   `json:"args,omitempty"`
	ExpectedOutput []string   `json:"expectedOutput,omitempty"`
	ExcludedOutput []string   `json:"excludedOutput,omitempty"`
	ExpectedError  []string   `json:"expectedError,omitempty"`
	ExcludedError  []string   `json:"excludedError,omitempty"`
}

type FileExistenceTest struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	ShouldExist    bool   `json:"shouldExist"`
	Permissions    string `json:"permissions,omitempty"`
	Uid            int    `json:"uid,omitempty"`
	Gid            int    `json:"gid,omitempty"`
	IsExecutableBy string `json:"isExecutableBy,omitempty"`
}

type FileContentTest struct {
	Name             string   `json:"name"`
	Path             string   `json:"path"`
	ExpectedContents []string `json:"expectedContents,omitempty"`
	ExcludedContents []string `json:"excludedContents,omitempty"`
}

type MetadataTest struct {
	EnvVars          []EnvVar  `json:"envVars,omitempty"`
	UnboundEnvVars   []EnvVar  `json:"unboundEnvVars,omitempty"`
	ExposedPorts     []string  `json:"exposedPorts,omitempty"`
	UnexposedPorts   []string  `json:"unexposedPorts,omitempty"`
	Entrypoint       *[]string `json:"entrypoint,omitempty"`
	Cmd              *[]string `json:"cmd,omitempty"`
	Workdir          string    `json:"workdir,omitempty"`
	Volumes          []string  `json:"volumes,omitempty"`
	UnmountedVolumes []string  `json:"unmountedVolumes,omitempty"`
	Labels           []Label   `json:"labels,omitempty"`
	User             string    `json:"user,omitempty"`
}

type LicenseTest struct {
	Debian bool     `json:"debian,omitempty"`
	Files  []string `json:"files,omitempty"`
}

type ContainerRunOptions struct {
	User         string   `json:"user,omitempty"`
	Privileged   bool     `json:"privileged,omitempty"`
	TTY          bool     `json:"allocateTty,omitempty"`
	EnvVars      []string `json:"envVars,omitempty"`
	EnvFile      string   `json:"envFile,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	BindMounts   []string `json:"bindMounts,omitempty"`
}

func main() {
	slog.Info("Generating structure test schema...")
	schema, err := jsonschema.For[StructureTestConfig](&jsonschema.ForOptions{})
	if err != nil {
		slog.Error("Failed to generate schema", "error", err)
		os.Exit(1)
	}

	schema.ID = "https://schema-nest.timo-reymann.de/api/schema/json-schema/containerhive-structure-test-v2/latest"
	schema.Title = "Container structure test (v2)"
	schema.Description = "Container structure test v2 configuration schema for ContainerHive."

	slog.Info("Writing schema to file...")
	indented, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal indented schema", "error", err)
		os.Exit(1)
	}

	err = os.WriteFile("schemas/structure-test.schema.json", indented, 0644)
	if err != nil {
		slog.Error("Failed to write schema file", "error", err)
		os.Exit(1)
	}
}
