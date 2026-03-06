package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timo-reymann/ContainerHive/pkg/build"
)

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []build.Filter
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: nil,
		},
		{
			name:  "single image name",
			input: []string{"ubuntu"},
			expected: []build.Filter{
				{ImageName: "ubuntu", TagName: ""},
			},
		},
		{
			name:  "single image with tag",
			input: []string{"ubuntu:24.04"},
			expected: []build.Filter{
				{ImageName: "ubuntu", TagName: "24.04"},
			},
		},
		{
			name:  "multiple images",
			input: []string{"ubuntu", "alpine:3.18"},
			expected: []build.Filter{
				{ImageName: "ubuntu", TagName: ""},
				{ImageName: "alpine", TagName: "3.18"},
			},
		},
		{
			name:  "complex image names with tags",
			input: []string{"my-registry.example.com/ubuntu:24.04", "gcr.io/project/alpine"},
			expected: []build.Filter{
				{ImageName: "my-registry.example.com/ubuntu", TagName: "24.04"},
				{ImageName: "gcr.io/project/alpine", TagName: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ParseFilters(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		filters  []build.Filter
		image    string
		tag      string
		expected bool
	}{
		{
			name:     "empty filters match everything",
			filters:  []build.Filter{},
			image:    "ubuntu",
			tag:      "24.04",
			expected: true,
		},
		{
			name: "exact match",
			filters: []build.Filter{
				{ImageName: "ubuntu", TagName: "24.04"},
			},
			image:    "ubuntu",
			tag:      "24.04",
			expected: true,
		},
		{
			name: "image name match, any tag",
			filters: []build.Filter{
				{ImageName: "ubuntu", TagName: ""},
			},
			image:    "ubuntu",
			tag:      "24.04",
			expected: true,
		},
		{
			name: "no match - different image",
			filters: []build.Filter{
				{ImageName: "alpine", TagName: ""},
			},
			image:    "ubuntu",
			tag:      "24.04",
			expected: false,
		},
		{
			name: "no match - different tag",
			filters: []build.Filter{
				{ImageName: "ubuntu", TagName: "22.04"},
			},
			image:    "ubuntu",
			tag:      "24.04",
			expected: false,
		},
		{
			name: "multiple filters - first matches",
			filters: []build.Filter{
				{ImageName: "alpine", TagName: ""},
				{ImageName: "ubuntu", TagName: "24.04"},
			},
			image:    "ubuntu",
			tag:      "24.04",
			expected: true,
		},
		{
			name: "multiple filters - second matches",
			filters: []build.Filter{
				{ImageName: "alpine", TagName: ""},
				{ImageName: "ubuntu", TagName: ""},
			},
			image:    "ubuntu",
			tag:      "24.04",
			expected: true,
		},
		{
			name: "multiple filters - no match",
			filters: []build.Filter{
				{ImageName: "alpine", TagName: ""},
				{ImageName: "debian", TagName: ""},
			},
			image:    "ubuntu",
			tag:      "24.04",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := MatchesFilter(tt.filters, tt.image, tt.tag)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
