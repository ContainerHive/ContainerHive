package buildconfig_resolver

import (
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/google/go-cmp/cmp"
)

func TestForTag(t *testing.T) {
	tests := map[string]struct {
		image    *model.Image
		tag      *model.Tag
		expected *ResolvedBuildValues
	}{
		"empty image and tag": {
			image: &model.Image{},
			tag:   &model.Tag{},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions:  model.Versions{},
				Secrets:   map[string][]byte{},
			},
		},
		"image with versions only": {
			image: &model.Image{
				Versions: model.Versions{
					"python": "3.11",
					"pip":    "23.0",
				},
			},
			tag: &model.Tag{},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions: model.Versions{
					"python": "3.11",
					"pip":    "23.0",
				},
				Secrets: map[string][]byte{},
			},
		},
		"image with build args only": {
			image: &model.Image{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
					"WORKDIR":    "/app",
				},
			},
			tag: &model.Tag{},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
					"WORKDIR":    "/app",
				},
				Versions: model.Versions{},
				Secrets:  map[string][]byte{},
			},
		},
		"tag with versions only": {
			image: &model.Image{},
			tag: &model.Tag{
				Versions: model.Versions{
					"node": "20.0.0",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions: model.Versions{
					"node": "20.0.0",
				},
				Secrets: map[string][]byte{},
			},
		},
		"tag with build args only": {
			image: &model.Image{},
			tag: &model.Tag{
				BuildArgs: model.BuildArgs{
					"BUILD_TYPE": "release",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BUILD_TYPE": "release",
				},
				Versions: model.Versions{},
				Secrets:  map[string][]byte{},
			},
		},
		"tag versions override image versions": {
			image: &model.Image{
				Versions: model.Versions{
					"python": "3.10",
					"pip":    "22.0",
				},
			},
			tag: &model.Tag{
				Versions: model.Versions{
					"python": "3.11",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions: model.Versions{
					"python": "3.11", // overridden by tag
					"pip":    "22.0", // from image
				},
				Secrets: map[string][]byte{},
			},
		},
		"image build args override tag build args": {
			image: &model.Image{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:3.18",
					"EXTRA_ARG":  "value",
				},
			},
			tag: &model.Tag{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:3.18", // image overrides tag
					"EXTRA_ARG":  "value",       // from image
				},
				Versions: model.Versions{},
				Secrets:  map[string][]byte{},
			},
		},
		"complex merge scenario": {
			image: &model.Image{
				Versions: model.Versions{
					"python": "3.10",
					"poetry": "1.5.0",
				},
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:3.18",
					"WORKDIR":    "/app",
				},
			},
			tag: &model.Tag{
				Versions: model.Versions{
					"python": "3.11",
					"pip":    "23.0",
				},
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
					"BUILD_TYPE": "release",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:3.18", // image overrides tag
					"WORKDIR":    "/app",        // from image
					"BUILD_TYPE": "release",     // from tag
				},
				Versions: model.Versions{
					"python": "3.11",  // tag overrides image
					"poetry": "1.5.0", // from image
					"pip":    "23.0",  // from tag
				},
				Secrets: map[string][]byte{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ForTag(tc.image, tc.tag)
			if err != nil {
				t.Fatalf("ForTag() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("ForTag() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestForTagVariant(t *testing.T) {
	tests := map[string]struct {
		image    *model.Image
		variant  *model.ImageVariant
		tag      *model.Tag
		expected *ResolvedBuildValues
	}{
		"empty image, variant, and tag": {
			image:   &model.Image{},
			variant: &model.ImageVariant{},
			tag:     &model.Tag{},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions:  model.Versions{},
				Secrets:   map[string][]byte{},
			},
		},
		"variant with versions only": {
			image: &model.Image{},
			variant: &model.ImageVariant{
				Versions: model.Versions{
					"nodejs": "20.0.0",
				},
			},
			tag: &model.Tag{},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions: model.Versions{
					"nodejs": "20.0.0",
				},
				Secrets: map[string][]byte{},
			},
		},
		"variant with build args only": {
			image: &model.Image{},
			variant: &model.ImageVariant{
				BuildArgs: model.BuildArgs{
					"VARIANT_ARG": "value",
				},
			},
			tag: &model.Tag{},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"VARIANT_ARG": "value",
				},
				Versions: model.Versions{},
				Secrets:  map[string][]byte{},
			},
		},
		"variant versions override tag and image versions": {
			image: &model.Image{
				Versions: model.Versions{
					"python": "3.10",
					"poetry": "1.5.0",
				},
			},
			variant: &model.ImageVariant{
				Versions: model.Versions{
					"python": "3.11",
					"nodejs": "20.0.0",
				},
			},
			tag: &model.Tag{
				Versions: model.Versions{
					"python": "3.9",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions: model.Versions{
					"python": "3.11",   // variant overrides all
					"poetry": "1.5.0",  // from image
					"nodejs": "20.0.0", // from variant
				},
				Secrets: map[string][]byte{},
			},
		},
		"variant build args override image build args": {
			image: &model.Image{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:3.18",
					"WORKDIR":    "/app",
				},
			},
			variant: &model.ImageVariant{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE":  "ubuntu:22.04",
					"VARIANT_ARG": "value",
				},
			},
			tag: &model.Tag{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE":  "ubuntu:22.04", // variant overrides all
					"WORKDIR":     "/app",         // from image
					"VARIANT_ARG": "value",        // from variant
				},
				Versions: model.Versions{},
				Secrets:  map[string][]byte{},
			},
		},
		"complex three-way merge": {
			image: &model.Image{
				Versions: model.Versions{
					"python": "3.10",
					"poetry": "1.5.0",
					"pip":    "22.0",
				},
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:3.18",
					"WORKDIR":    "/app",
					"USER":       "appuser",
				},
			},
			variant: &model.ImageVariant{
				Versions: model.Versions{
					"python": "3.11",
					"nodejs": "20.0.0",
				},
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE":  "ubuntu:22.04",
					"VARIANT_ARG": "variant_value",
				},
			},
			tag: &model.Tag{
				Versions: model.Versions{
					"python": "3.9",
					"go":     "1.21",
				},
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
					"BUILD_TYPE": "release",
				},
			},
			expected: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE":  "ubuntu:22.04",  // variant overrides all
					"WORKDIR":     "/app",          // from image
					"USER":        "appuser",       // from image
					"BUILD_TYPE":  "release",       // from tag
					"VARIANT_ARG": "variant_value", // from variant
				},
				Versions: model.Versions{
					"python": "3.11",   // variant overrides tag and image
					"poetry": "1.5.0",  // from image
					"pip":    "22.0",   // from image
					"go":     "1.21",   // from tag
					"nodejs": "20.0.0", // from variant
				},
				Secrets: map[string][]byte{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ForTagVariant(tc.image, tc.variant, tc.tag)
			if err != nil {
				t.Fatalf("ForTagVariant() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("ForTagVariant() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestToBuildArgs(t *testing.T) {
	tests := map[string]struct {
		resolved *ResolvedBuildValues
		expected model.BuildArgs
	}{
		"empty resolved values": {
			resolved: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions:  model.Versions{},
			},
			expected: model.BuildArgs{},
		},
		"build args only": {
			resolved: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
					"WORKDIR":    "/app",
				},
				Versions: model.Versions{},
			},
			expected: model.BuildArgs{
				"BASE_IMAGE": "alpine:latest",
				"WORKDIR":    "/app",
			},
		},
		"versions only": {
			resolved: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions: model.Versions{
					"python": "3.11",
					"node":   "20.0.0",
				},
			},
			expected: model.BuildArgs{
				"PYTHON_VERSION": "3.11",
				"NODE_VERSION":   "20.0.0",
			},
		},
		"both build args and versions": {
			resolved: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"BASE_IMAGE": "alpine:latest",
					"WORKDIR":    "/app",
				},
				Versions: model.Versions{
					"python": "3.11",
					"node":   "20.0.0",
				},
			},
			expected: model.BuildArgs{
				"BASE_IMAGE":     "alpine:latest",
				"WORKDIR":        "/app",
				"PYTHON_VERSION": "3.11",
				"NODE_VERSION":   "20.0.0",
			},
		},
		"hyphenated keys in versions": {
			resolved: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{},
				Versions: model.Versions{
					"some-package": "1.2.3",
					"another-tool": "4.5.6",
				},
			},
			expected: model.BuildArgs{
				"SOME_PACKAGE_VERSION": "1.2.3",
				"ANOTHER_TOOL_VERSION": "4.5.6",
			},
		},
		"hyphenated keys in build args": {
			resolved: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"some-arg":    "value1",
					"another-arg": "value2",
				},
				Versions: model.Versions{},
			},
			expected: model.BuildArgs{
				"SOME_ARG":    "value1",
				"ANOTHER_ARG": "value2",
			},
		},
		"complex scenario": {
			resolved: &ResolvedBuildValues{
				BuildArgs: model.BuildArgs{
					"base-image": "alpine:latest",
					"work-dir":   "/app",
					"BUILD_TYPE": "release",
				},
				Versions: model.Versions{
					"python":  "3.11",
					"node-js": "20.0.0",
					"go-lang": "1.21",
				},
			},
			expected: model.BuildArgs{
				"BASE_IMAGE":      "alpine:latest",
				"WORK_DIR":        "/app",
				"BUILD_TYPE":      "release",
				"PYTHON_VERSION":  "3.11",
				"NODE_JS_VERSION": "20.0.0",
				"GO_LANG_VERSION": "1.21",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.resolved.ToBuildArgs()

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("ToBuildArgs() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}
