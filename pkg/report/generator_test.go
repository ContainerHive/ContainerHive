package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestReplaceFirstPlaceholder(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		data     string
		expected string
	}{
		{
			name:     "replaces placeholder with data",
			html:     `{"data": /*INJECT_JSON_DATA*/, "other": "value"}`,
			data:     `{"key":"value"}`,
			expected: `{"data": {"key":"value"}, "other": "value"}`,
		},
		{
			name:     "placeholder at start",
			html:     `/*INJECT_JSON_DATA*/ and more content`,
			data:     `first`,
			expected: `first and more content`,
		},
		{
			name:     "no placeholder returns original",
			html:     `no placeholder here`,
			data:     `data`,
			expected: `no placeholder here`,
		},
		{
			name:     "empty data",
			html:     `{"data": /*INJECT_JSON_DATA*/}`,
			data:     ``,
			expected: `{"data": }`,
		},
		{
			name:     "only first occurrence replaced",
			html:     `/*INJECT_JSON_DATA*/ and /*INJECT_JSON_DATA*/ again`,
			data:     `first`,
			expected: `first and /*INJECT_JSON_DATA*/ again`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceFirstPlaceholder(tc.html, "/*INJECT_JSON_DATA*/", tc.data)
			if got != tc.expected {
				t.Errorf("replaceFirstPlaceholder() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestScanImage_Platforms(t *testing.T) {
	img := &model.Image{
		Name:      "test",
		Versions:  map[string]string{"v": "1"},
		Platforms: []string{"linux/amd64", "linux/arm64"},
		Tags:      map[string]*model.Tag{"1.0.0": {Name: "1.0.0"}},
		Variants: map[string]*model.ImageVariant{
			"slim": {
				Name:      "slim",
				TagSuffix: "-slim",
				Platforms: []string{"linux/amd64"},
			},
		},
	}

	report := scanImage("/test", "test", img)

	if len(report.Platforms) != 2 {
		t.Errorf("Platforms len = %d, want 2", len(report.Platforms))
	}
	if len(report.Variants) != 1 {
		t.Fatalf("len(Variants) = %d, want 1", len(report.Variants))
	}
	if len(report.Variants[0].Platforms) != 1 {
		t.Errorf("variant Platforms len = %d, want 1", len(report.Variants[0].Platforms))
	}
}

func TestScanImage(t *testing.T) {
	img := &model.Image{
		Name:     "test",
		Versions: map[string]string{"go": "1.22"},
		Tags: map[string]*model.Tag{
			"1.0.0": {Name: "1.0.0", BuildArgs: model.BuildArgs{"FOO": "bar"}},
		},
		Report: model.ReportModel{Icon: "go-original"},
		Variants: map[string]*model.ImageVariant{
			"slim": {
				Name:      "slim",
				TagSuffix: "-slim",
				Report:    model.ReportModel{Icon: "nodejs-original"},
			},
		},
	}

	report := scanImage("/test", "test", img)

	if report.Name != "test" {
		t.Errorf("Name = %q, want %q", report.Name, "test")
	}
	if report.Report.Icon != "go-original" {
		t.Errorf("Report.Icon = %q, want %q", report.Report.Icon, "go-original")
	}
	if len(report.Tags) != 1 {
		t.Errorf("len(Tags) = %d, want 1", len(report.Tags))
	}
	if len(report.Variants) != 1 {
		t.Errorf("len(Variants) = %d, want 1", len(report.Variants))
	}
	if report.Variants[0].Report.Icon != "nodejs-original" {
		t.Errorf("variant Report.Icon = %q, want %q", report.Variants[0].Report.Icon, "nodejs-original")
	}
}

func TestScanProject(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"img1": {
				{Name: "img1", Versions: map[string]string{"v": "1"}, Tags: map[string]*model.Tag{"1.0.0": {Name: "1.0.0"}}},
			},
			"img2": {
				{Name: "img2", Versions: map[string]string{"v": "2"}, Tags: map[string]*model.Tag{"2.0.0": {Name: "2.0.0"}}},
			},
			"empty": {},
		},
	}

	images := scanProject(project)

	if len(images) != 2 {
		t.Errorf("len(images) = %d, want 2", len(images))
	}
}

func TestScanImage_VariantBuildArgsMerged(t *testing.T) {
	img := &model.Image{
		Name:     "test",
		Versions: map[string]string{"v": "1"},
		Tags: map[string]*model.Tag{
			"1.0.0": {
				Name:      "1.0.0",
				BuildArgs: model.BuildArgs{"BASE": "baseval"},
			},
		},
		Variants: map[string]*model.ImageVariant{
			"slim": {
				Name:      "slim",
				TagSuffix: "-slim",
				BuildArgs: model.BuildArgs{"EXTRA": "extraval"},
			},
		},
	}

	report := scanImage("/test", "test", img)

	if len(report.Variants) != 1 {
		t.Fatalf("len(Variants) = %d, want 1", len(report.Variants))
	}

	vt := report.Variants[0].Tags[0]
	if vt.BuildArgs["BASE"] != "baseval" {
		t.Errorf("variant BuildArgs[BASE] = %q, want inherited %q", vt.BuildArgs["BASE"], "baseval")
	}
	if vt.BuildArgs["EXTRA"] != "extraval" {
		t.Errorf("variant BuildArgs[EXTRA] = %q, want %q", vt.BuildArgs["EXTRA"], "extraval")
	}
}

func TestGenerator_Generate(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"test": {
				{
					Name:     "test",
					Versions: map[string]string{"go": "1.22"},
					Tags:     map[string]*model.Tag{"1.0.0": {Name: "1.0.0"}},
					Report:   model.ReportModel{Icon: "go-original"},
				},
			},
		},
	}

	g := NewGenerator()
	report, err := g.Generate(project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(report.Images) != 1 {
		t.Errorf("len(Images) = %d, want 1", len(report.Images))
	}
	if len(report.Images) > 0 && report.Images[0].Report.Icon != "go-original" {
		t.Errorf("Report.Icon = %q, want %q", report.Images[0].Report.Icon, "go-original")
	}
}

func TestGenerator_GenerateJSON(t *testing.T) {
	g := NewGenerator()

	report := &ProjectReport{
		GeneratedAt: "2024-01-01T00:00:00Z",
		Images: []ImageReport{
			{
				Name:   "test-image",
				Report: Report{Icon: "go-original"},
				Tags: []TagReport{
					{
						Name:      "1.0.0",
						BuildArgs: map[string]string{"FOO": "bar"},
					},
				},
				Variants: []VariantReport{
					{
						Name:      "slim",
						Report:    Report{Icon: "go-original"},
						TagSuffix: "-slim",
						Tags: []TagReport{
							{
								Name:      "1.0.0-slim",
								BuildArgs: map[string]string{"FOO": "bar"},
							},
						},
					},
				},
			},
		},
	}

	got, err := g.GenerateJSON(report)
	if err != nil {
		t.Fatalf("GenerateJSON() error = %v", err)
	}

	if len(got) == 0 {
		t.Error("GenerateJSON() returned empty output")
	}
}

func TestGenerator_Generate_SourceType(t *testing.T) {
	tests := []struct {
		name         string
		setCI        bool
		wantImgCount int
	}{
		{
			name:         "tar source",
			wantImgCount: 1,
		},
		{
			name:         "registry source",
			wantImgCount: 1,
		},
		{
			name:         "auto in non-ci uses tar",
			wantImgCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			oldCI := os.Getenv("CI")
			if tc.setCI {
				os.Setenv("CI", "true")
			} else {
				os.Unsetenv("CI")
			}
			defer os.Setenv("CI", oldCI)

			project := &model.ContainerHiveProject{
				ImagesByName: map[string][]*model.Image{
					"img": {
						{Name: "img", Versions: map[string]string{"v": "1"}, Tags: map[string]*model.Tag{"1.0.0": {Name: "1.0.0"}}},
					},
				},
			}

			g := NewGenerator()
			report, err := g.Generate(project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if len(report.Images) != tc.wantImgCount {
				t.Errorf("len(Images) = %d, want %d", len(report.Images), tc.wantImgCount)
			}
		})
	}
}

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator() returned nil")
	}
}

func TestGenerator_ReadStyleSheet(t *testing.T) {
	tests := []struct {
		name        string
		stylesheet  string
		wantContent string
		wantErr     bool
	}{
		{
			name:        "empty stylesheet path",
			stylesheet:  "",
			wantContent: "",
			wantErr:     false,
		},
		{
			name:        "valid css file",
			stylesheet:  "", // set in body via temp dir
			wantContent: "body { color: red; }",
			wantErr:     false,
		},
		{
			name:        "missing file",
			stylesheet:  "", // set in body to nonexistent path
			wantContent: "",
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGenerator()
			report := &ProjectReport{StyleSheet: tc.stylesheet}

			switch tc.name {
			case "valid css file":
				path := filepath.Join(t.TempDir(), "extra.css")
				if err := os.WriteFile(path, []byte(tc.wantContent), 0644); err != nil {
					t.Fatalf("setup: %v", err)
				}
				report.StyleSheet = path
			case "missing file":
				report.StyleSheet = filepath.Join(t.TempDir(), "nonexistent.css")
			}

			got, err := g.ReadStyleSheet(report)
			if tc.wantErr && err == nil {
				t.Fatal("ReadStyleSheet() expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ReadStyleSheet() unexpected error: %v", err)
			}
			if got != tc.wantContent {
				t.Errorf("ReadStyleSheet() = %q, want %q", got, tc.wantContent)
			}
		})
	}
}

func TestGenerator_GenerateHTMLFromAssets_StyleSheet(t *testing.T) {
	tests := []struct {
		name         string
		stylesheet   string // value for ProjectReport.StyleSheet
		wantContains string // expected content in the output HTML
		wantErr      bool
	}{
		{
			name:         "stylesheet content injected",
			stylesheet:   "", // set in body
			wantContains: ".custom { background: blue; }",
			wantErr:      false,
		},
		{
			name:         "no stylesheet configured",
			stylesheet:   "",
			wantContains: "",
			wantErr:      false,
		},
		{
			name:         "stylesheet file missing",
			stylesheet:   "", // set in body
			wantContains: "",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGenerator()
			report := &ProjectReport{
				GeneratedAt: "2024-01-01T00:00:00Z",
				StyleSheet:  tc.stylesheet,
			}

			switch tc.name {
			case "stylesheet content injected":
				path := filepath.Join(t.TempDir(), "style.css")
				if err := os.WriteFile(path, []byte(tc.wantContains), 0644); err != nil {
					t.Fatalf("setup: %v", err)
				}
				report.StyleSheet = path
			case "stylesheet file missing":
				report.StyleSheet = filepath.Join(t.TempDir(), "missing.css")
			}

			got, err := g.GenerateHTMLFromAssets(report)
			if tc.wantErr {
				if err == nil {
					t.Fatal("GenerateHTMLFromAssets() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GenerateHTMLFromAssets() unexpected error: %v", err)
			}

			html := string(got)

			if tc.wantContains != "" {
				if !strings.Contains(html, tc.wantContains) {
					t.Errorf("output missing expected stylesheet content %q", tc.wantContains)
				}
			}

			if strings.Contains(html, "/*INJECT_STYLE*/") {
				t.Error("output still contains /*INJECT_STYLE*/ placeholder")
			}
			if strings.Contains(html, "/*INJECT_JSON_DATA*/") {
				t.Error("output still contains /*INJECT_JSON_DATA*/ placeholder")
			}
			if strings.Contains(html, "/*INJECT_GENERATED_AT*/") {
				t.Error("output still contains /*INJECT_GENERATED_AT*/ placeholder")
			}
			if !strings.Contains(html, "2024-01-01T00:00:00Z") {
				t.Error("output does not contain injected generatedAt value")
			}
		})
	}
}
