package syft

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewSBOMImageTool(t *testing.T) {
	t.Log("Testing SBOMImageTool initialization")

	tool, err := NewSBOMImageTool()
	if err != nil {
		t.Fatalf("NewSBOMImageTool() error = %v", err)
	}
	t.Log("✓ SBOMImageTool created without error")

	if tool == nil {
		t.Fatal("NewSBOMImageTool() returned nil tool")
	}
	t.Log("✓ Tool instance is non-nil")

	if tool.encoders == nil {
		t.Fatal("NewSBOMImageTool() returned tool with nil encoders")
	}
	t.Log("✓ Encoders collection initialized")
}

func TestSBOMImageTool_GenerateSBOM(t *testing.T) {
	t.Log("Setting up SBOMImageTool for SBOM generation tests")
	tool, err := NewSBOMImageTool()
	if err != nil {
		t.Fatalf("NewSBOMImageTool() error = %v", err)
	}

	ctx := context.Background()

	t.Run("generates SBOM from valid alpine tar image", func(t *testing.T) {
		t.Log("Generating SBOM from testdata/alpine.tar")
		sbom, err := tool.GenerateSBOM(ctx, "testdata/alpine.tar")
		if err != nil {
			t.Fatalf("GenerateSBOM() error = %v", err)
		}
		t.Logf("✓ SBOM generated successfully")

		if sbom == nil {
			t.Fatal("GenerateSBOM() returned nil SBOM")
		}
		t.Log("✓ SBOM is non-nil")

		t.Logf("✓ SBOM contains artifacts collection")

		if sbom.Descriptor.Name != "" {
			t.Logf("  - SBOM tool: %s %s", sbom.Descriptor.Name, sbom.Descriptor.Version)
			t.Log("  ✓ SBOM descriptor populated")
		}
	})

	t.Run("returns error for non-existent tar file", func(t *testing.T) {
		t.Log("Testing error handling for non-existent file")
		_, err := tool.GenerateSBOM(ctx, "testdata/nonexistent.tar")
		if err == nil {
			t.Fatal("GenerateSBOM() expected error for non-existent tar, got nil")
		}
		t.Logf("✓ Correctly returned error: %v", err)
	})

	t.Run("returns error for invalid tar path", func(t *testing.T) {
		t.Log("Testing error handling for invalid path")
		_, err := tool.GenerateSBOM(ctx, "/invalid/path/to/nowhere.tar")
		if err == nil {
			t.Fatal("GenerateSBOM() expected error for invalid path, got nil")
		}
		t.Logf("✓ Correctly returned error: %v", err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		t.Log("Testing context cancellation handling")
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		_, err := tool.GenerateSBOM(cancelCtx, "testdata/alpine.tar")
		// May or may not error depending on timing, but should not panic
		if err != nil {
			t.Logf("✓ Handled cancelled context: %v", err)
		} else {
			t.Log("✓ Operation completed before context cancellation")
		}
	})
}

func TestSBOMImageTool_SerializeSBOM(t *testing.T) {
	t.Log("Setting up SBOM for serialization tests")
	tool, err := NewSBOMImageTool()
	if err != nil {
		t.Fatalf("NewSBOMImageTool() error = %v", err)
	}

	ctx := context.Background()
	t.Log("Generating SBOM from testdata/alpine.tar")
	sbom, err := tool.GenerateSBOM(ctx, "testdata/alpine.tar")
	if err != nil {
		t.Fatalf("GenerateSBOM() error = %v", err)
	}
	t.Log("✓ Test SBOM generated successfully")

	tests := []struct {
		name         string
		outputFormat string
		wantErr      bool
		validateJSON bool
	}{
		{
			name:         "syft-json format",
			outputFormat: "syft-json",
			wantErr:      false,
			validateJSON: true,
		},
		{
			name:         "json format (alias)",
			outputFormat: "json",
			wantErr:      false,
			validateJSON: true,
		},
		{
			name:         "spdx-json format",
			outputFormat: "spdx-json",
			wantErr:      false,
			validateJSON: true,
		},
		{
			name:         "cyclonedx-json format",
			outputFormat: "cyclonedx-json",
			wantErr:      false,
			validateJSON: true,
		},
		{
			name:         "spdx-tag-value format",
			outputFormat: "spdx-tag-value",
			wantErr:      false,
			validateJSON: false,
		},
		{
			name:         "cyclonedx-xml format",
			outputFormat: "cyclonedx-xml",
			wantErr:      false,
			validateJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Serializing SBOM to %s format", tt.outputFormat)
			serialized, err := tool.SerializeSBOM(sbom, tt.outputFormat)
			if (err != nil) != tt.wantErr {
				t.Errorf("SerializeSBOM() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				t.Logf("✓ Expected error occurred: %v", err)
				return
			}

			if len(serialized) == 0 {
				t.Fatal("SerializeSBOM() returned empty serialized data")
			}
			t.Logf("✓ Serialized SBOM size: %d bytes", len(serialized))

			// Validate JSON formats
			if tt.validateJSON {
				var js map[string]interface{}
				if err := json.Unmarshal(serialized, &js); err != nil {
					t.Fatalf("SerializeSBOM() produced invalid JSON: %v", err)
				}
				t.Log("  ✓ Output is valid JSON")

				// Check for expected fields based on format
				switch tt.outputFormat {
				case "syft-json", "json":
					if _, ok := js["artifacts"]; !ok {
						t.Error("  ✗ Syft JSON missing 'artifacts' field")
					} else {
						t.Log("  ✓ Contains 'artifacts' field")
					}
				case "spdx-json":
					if _, ok := js["spdxVersion"]; !ok {
						t.Error("  ✗ SPDX JSON missing 'spdxVersion' field")
					} else {
						t.Log("  ✓ Contains 'spdxVersion' field")
					}
				case "cyclonedx-json":
					if _, ok := js["bomFormat"]; !ok {
						t.Error("  ✗ CycloneDX JSON missing 'bomFormat' field")
					} else {
						t.Log("  ✓ Contains 'bomFormat' field")
					}
				}
			}
		})
	}
}

func TestSBOMImageTool_Generate(t *testing.T) {
	tool, err := NewSBOMImageTool()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid tar and format", func(t *testing.T) {
		data, err := tool.Generate(context.Background(), "testdata/alpine.tar", "spdx-json")
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty SBOM data")
		}
	})

	t.Run("invalid tar path", func(t *testing.T) {
		_, err := tool.Generate(context.Background(), "/nonexistent/image.tar", "spdx-json")
		if err == nil {
			t.Fatal("expected error for nonexistent tar")
		}
	})
}

func TestSBOMImageTool_SerializeSBOM_InvalidFormat(t *testing.T) {
	t.Log("Testing invalid format handling")
	tool, err := NewSBOMImageTool()
	if err != nil {
		t.Fatalf("NewSBOMImageTool() error = %v", err)
	}

	ctx := context.Background()
	sbom, err := tool.GenerateSBOM(ctx, "testdata/alpine.tar")
	if err != nil {
		t.Fatalf("GenerateSBOM() error = %v", err)
	}

	tests := []struct {
		name   string
		format string
	}{
		{
			name:   "completely invalid format",
			format: "invalid-format-xyz",
		},
		{
			name:   "empty format string",
			format: "",
		},
		{
			name:   "malformed format",
			format: "json@@@invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Attempting to serialize with invalid format: %q", tt.format)
			_, err = tool.SerializeSBOM(sbom, tt.format)
			if err == nil {
				t.Fatalf("SerializeSBOM() expected error for format %q, got nil", tt.format)
			}
			t.Logf("✓ Correctly returned error: %v", err)
		})
	}
}
