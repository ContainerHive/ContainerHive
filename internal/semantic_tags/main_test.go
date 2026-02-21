package semantic_tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSemanticVersion(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    *SemanticTagVersion
		shouldError bool
	}{
		{
			name:  "Standard semantic version",
			input: "1.2.3",
			expected: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
			shouldError: false,
		},
		{
			name:  "Version with 'v' prefix",
			input: "v1.2.3",
			expected: &SemanticTagVersion{
				Prefix: "v",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
			shouldError: false,
		},
		{
			name:  "Version with custom prefix",
			input: "version-1.2.3",
			expected: &SemanticTagVersion{
				Prefix: "version-",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
			shouldError: false,
		},
		{
			name:  "Major only",
			input: "1",
			expected: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "",
				Patch:  "",
				Build:  "",
			},
			shouldError: false,
		},
		{
			name:  "Major.minor only",
			input: "1.2",
			expected: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "",
				Build:  "",
			},
			shouldError: false,
		},
		{
			name:  "Major.minor.patch.build",
			input: "1.2.3.4",
			expected: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "4",
			},
			shouldError: false,
		},
		{
			name:  "Prefixed major.minor.patch.build",
			input: "v1.2.3.4",
			expected: &SemanticTagVersion{
				Prefix: "v",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "4",
			},
			shouldError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid format",
			input:       "abc",
			expected:    nil,
			shouldError: true,
		},
		{
			name:  "Version with suffix",
			input: "1.2.3-alpha.1",
			expected: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Suffix: "alpha.1",
			},
			shouldError: false,
		},
		{
			name:  "Version with suffix and prefix",
			input: "v1.0.0-alpine",
			expected: &SemanticTagVersion{
				Prefix: "v",
				Major:  "1",
				Minor:  "0",
				Patch:  "0",
				Suffix: "alpine",
			},
			shouldError: false,
		},
		{
			name:  "Version with build metadata",
			input: "1.2.3+build123",
			expected: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "build123",
			},
			shouldError: false,
		},
		{
			name:  "Version with suffix and build metadata",
			input: "1.2.3-alpine+build123",
			expected: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "build123",
				Suffix: "alpine",
			},
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NewSemanticVersion(tc.input)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestNewSemanticVersionEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected *SemanticTagVersion
	}{
		{
			name:  "Multiple character prefix",
			input: "rel-v1.2.3",
			expected: &SemanticTagVersion{
				Prefix: "rel-v",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
		},
		{
			name:  "Prefix with dots",
			input: "v.1.2.3",
			expected: &SemanticTagVersion{
				Prefix: "v.",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
		},
		{
			name:  "Whitespace handling",
			input: "  v1.2.3  ",
			expected: &SemanticTagVersion{
				Prefix: "v",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NewSemanticVersion(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetLowerVariants(t *testing.T) {
	testCases := []struct {
		name     string
		version  *SemanticTagVersion
		expected []string
	}{
		{
			name: "Major only",
			version: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "",
				Patch:  "",
				Build:  "",
			},
			expected: []string{},
		},
		{
			name: "Major.minor",
			version: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "",
				Build:  "",
			},
			expected: []string{"1"},
		},
		{
			name: "Major.minor.patch",
			version: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
			expected: []string{"1.2", "1"},
		},
		{
			name: "Major.minor.patch.build",
			version: &SemanticTagVersion{
				Prefix: "",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "4",
			},
			expected: []string{"1.2.3", "1.2", "1"},
		},
		{
			name: "Prefixed major.minor.patch",
			version: &SemanticTagVersion{
				Prefix: "v",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
			expected: []string{"v1.2", "v1"},
		},
		{
			name: "Custom prefix major.minor.patch",
			version: &SemanticTagVersion{
				Prefix: "version-",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Build:  "",
			},
			expected: []string{"version-1.2", "version-1"},
		},
		{
			name: "Major.minor.patch with suffix",
			version: &SemanticTagVersion{
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Suffix: "alpine",
			},
			expected: []string{"1.2-alpine", "1-alpine"},
		},
		{
			name: "Prefixed major.minor.patch with suffix",
			version: &SemanticTagVersion{
				Prefix: "v",
				Major:  "1",
				Minor:  "2",
				Patch:  "3",
				Suffix: "alpine",
			},
			expected: []string{"v1.2-alpine", "v1-alpine"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.version.GetLowerVariants()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSemanticVersionComparison(t *testing.T) {
	testCases := []struct {
		name     string
		version1 *SemanticTagVersion
		version2 *SemanticTagVersion
		expected int // -1, 0, 1 for less, equal, greater
	}{
		{
			name:     "Equal versions",
			version1: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			expected: 0,
		},
		{
			name:     "Major version difference",
			version1: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			version2: &SemanticTagVersion{Major: "2", Minor: "2", Patch: "3"},
			expected: -1,
		},
		{
			name:     "Minor version difference",
			version1: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			version2: &SemanticTagVersion{Major: "1", Minor: "3", Patch: "3"},
			expected: -1,
		},
		{
			name:     "Patch version difference",
			version1: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "4"},
			expected: -1,
		},
		{
			name:     "Build version difference",
			version1: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3", Build: "1"},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3", Build: "2"},
			expected: -1,
		},
		{
			name:     "Missing minor vs present minor",
			version1: &SemanticTagVersion{Major: "1", Minor: "", Patch: ""},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: ""},
			expected: -1,
		},
		{
			name:     "Missing patch vs present patch",
			version1: &SemanticTagVersion{Major: "1", Minor: "2", Patch: ""},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			expected: -1,
		},
		{
			name:     "Greater major version",
			version1: &SemanticTagVersion{Major: "2", Minor: "2", Patch: "3"},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			expected: 1,
		},
		{
			name:     "Greater minor version",
			version1: &SemanticTagVersion{Major: "1", Minor: "3", Patch: "3"},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			expected: 1,
		},
		{
			name:     "Greater patch version",
			version1: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "4"},
			version2: &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.version1.Compare(tc.version2)
			assert.Equal(t, tc.expected, result)

			// Test the convenience methods
			switch tc.expected {
			case -1:
				assert.True(t, tc.version1.Less(tc.version2))
				assert.False(t, tc.version1.Greater(tc.version2))
				assert.False(t, tc.version1.Equal(tc.version2))
			case 0:
				assert.False(t, tc.version1.Less(tc.version2))
				assert.False(t, tc.version1.Greater(tc.version2))
				assert.True(t, tc.version1.Equal(tc.version2))
			case 1:
				assert.False(t, tc.version1.Less(tc.version2))
				assert.True(t, tc.version1.Greater(tc.version2))
				assert.False(t, tc.version1.Equal(tc.version2))
			}
		})
	}
}

func TestSemanticVersionString(t *testing.T) {
	testCases := []struct {
		name     string
		version  *SemanticTagVersion
		expected string
	}{
		{
			name:     "Major only",
			version:  &SemanticTagVersion{Major: "1"},
			expected: "1",
		},
		{
			name:     "Major.minor",
			version:  &SemanticTagVersion{Major: "1", Minor: "2"},
			expected: "1.2",
		},
		{
			name:     "Major.minor.patch",
			version:  &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3"},
			expected: "1.2.3",
		},
		{
			name:     "Major.minor.patch.build",
			version:  &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3", Build: "4"},
			expected: "1.2.3.4",
		},
		{
			name:     "With prefix",
			version:  &SemanticTagVersion{Prefix: "v", Major: "1", Minor: "2", Patch: "3"},
			expected: "v1.2.3",
		},
		{
			name:     "Custom prefix",
			version:  &SemanticTagVersion{Prefix: "version-", Major: "1", Minor: "2", Patch: "3"},
			expected: "version-1.2.3",
		},
		{
			name:     "With suffix",
			version:  &SemanticTagVersion{Major: "1", Minor: "2", Patch: "3", Suffix: "alpine"},
			expected: "1.2.3-alpine",
		},
		{
			name:     "With prefix and suffix",
			version:  &SemanticTagVersion{Prefix: "v", Major: "1", Minor: "0", Patch: "0", Suffix: "slim"},
			expected: "v1.0.0-slim",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.version.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSemanticVersionSorting(t *testing.T) {
	// Test sorting functionality using the comparison methods
	versions := []*SemanticTagVersion{
		{Major: "2", Minor: "0", Patch: "0"},
		{Major: "1", Minor: "5", Patch: "0"},
		{Major: "1", Minor: "2", Patch: "3"},
		{Major: "1", Minor: "2", Patch: "1"},
		{Major: "1", Minor: "0", Patch: "0"},
		{Major: "0", Minor: "9", Patch: "0"},
	}

	// Sort using the Less method
	for i := 0; i < len(versions); i++ {
		for j := i + 1; j < len(versions); j++ {
			if versions[i].Greater(versions[j]) {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}

	// Expected order after sorting
	expectedOrder := []string{
		"0.9.0",
		"1.0.0",
		"1.2.1",
		"1.2.3",
		"1.5.0",
		"2.0.0",
	}

	// Verify the order
	for i, version := range versions {
		assert.Equal(t, expectedOrder[i], version.String())
	}
}
