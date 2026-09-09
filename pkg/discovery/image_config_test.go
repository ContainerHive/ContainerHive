package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func setupMinimalImageDir(t *testing.T, imageYML string) (projectRoot, configFilePath string) {
	t.Helper()
	dir := t.TempDir()
	configFilePath = filepath.Join(dir, "image.yml")
	if err := os.WriteFile(configFilePath, []byte(imageYML), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return dir, configFilePath
}

func TestProcessImageConfig_Description(t *testing.T) {
	tests := []struct {
		name            string
		imageYML        string
		wantDescription string
	}{
		{
			name: "with description",
			imageYML: `description: "My test image"
tags:
  - name: 1.0.0
`,
			wantDescription: "My test image",
		},
		{
			name: `without description`,
			imageYML: `tags:
  - name: 1.0.0
`,
			wantDescription: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projectRoot, configFilePath := setupMinimalImageDir(t, tc.imageYML)
			img, err := processImageConfig(projectRoot, configFilePath)
			if err != nil {
				t.Fatalf("processImageConfig() error = %v", err)
			}
			if img.Description != tc.wantDescription {
				t.Errorf("Description = %q, want %q", img.Description, tc.wantDescription)
			}
		})
	}
}

func TestProcessImageConfig_TagRanges(t *testing.T) {
	imageYML := `tags:
  - name: 1.0.0
tag_ranges:
  - tag_name: "{{.major}}.{{.minor}}"
    source:
      type: json
      url: "https://example.com/versions.json"
      transform: "$"
    select:
      major: { last: 3 }
      minor: { last: 3 }
      patch: latest
    filter:
      exclude_prerelease: true
      max_major_increment: 5
    versions:
      nodejs: "{{.full}}"
`
	projectRoot, configFilePath := setupMinimalImageDir(t, imageYML)
	img, err := processImageConfig(projectRoot, configFilePath)
	if err != nil {
		t.Fatalf("processImageConfig() error = %v", err)
	}
	if len(img.TagRanges) != 1 {
		t.Fatalf("expected 1 tag_range, got %d", len(img.TagRanges))
	}
	tr := img.TagRanges[0]
	if tr.TagName != "{{.major}}.{{.minor}}" {
		t.Errorf("TagName = %q, want the raw template", tr.TagName)
	}
	if tr.Source == nil || tr.Source.Type != "json" {
		t.Fatalf("Source = %+v, want type json", tr.Source)
	}
	if tr.Select == nil || tr.Select.Major == nil || tr.Select.Major.Last != 3 {
		t.Errorf("Select.Major = %+v, want Last=3", tr.Select.Major)
	}
	if tr.Select.Patch == nil || tr.Select.Patch.Last != 1 {
		t.Errorf("Select.Patch = %+v, want Last=1 (from \"latest\")", tr.Select.Patch)
	}
	if tr.Filter == nil || tr.Filter.ExcludePrerelease == nil || !*tr.Filter.ExcludePrerelease {
		t.Errorf("Filter.ExcludePrerelease = %+v, want true", tr.Filter)
	}
	if tr.Filter.MaxMajorIncrement != 5 {
		t.Errorf("Filter.MaxMajorIncrement = %d, want 5", tr.Filter.MaxMajorIncrement)
	}
}

// TestProcessImageConfig_TagRanges_StrictUnknownKey guards against the
// custom LevelSelect.UnmarshalYAML silently accepting an unknown mapping key
// even though the outer decoder's KnownFields(true) does not propagate into
// custom unmarshalers.
func TestProcessImageConfig_TagRanges_StrictUnknownKey(t *testing.T) {
	imageYML := `tags:
  - name: 1.0.0
tag_ranges:
  - tag_name: "{{.major}}"
    select:
      major: { lst: 3 }
`
	projectRoot, configFilePath := setupMinimalImageDir(t, imageYML)
	if _, err := processImageConfig(projectRoot, configFilePath); err == nil {
		t.Fatal("expected an error for the unknown select field \"lst\", got nil")
	}
}

// TestProcessImageConfig_TagRanges_StrictUnknownTopLevelKey guards the
// existing d.KnownFields(true) strictness, extended to the new tag_ranges
// struct: a typo at the tag_range level (not inside select) must also fail.
func TestProcessImageConfig_TagRanges_StrictUnknownTopLevelKey(t *testing.T) {
	imageYML := `tags:
  - name: 1.0.0
tag_ranges:
  - tag_nmae: "{{.major}}"
`
	projectRoot, configFilePath := setupMinimalImageDir(t, imageYML)
	if _, err := processImageConfig(projectRoot, configFilePath); err == nil {
		t.Fatal("expected an error for the unknown field \"tag_nmae\", got nil")
	}
}
