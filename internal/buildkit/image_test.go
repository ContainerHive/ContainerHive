package buildkit

import "testing"

func TestImageRef(t *testing.T) {
	cases := []struct {
		name string
		opts map[string]string
		want string
	}{
		{
			name: "defaults when opts is nil",
			opts: nil,
			want: DefaultImage + ":" + DefaultTag,
		},
		{
			name: "defaults when opts is empty",
			opts: map[string]string{},
			want: DefaultImage + ":" + DefaultTag,
		},
		{
			name: "custom image",
			opts: map[string]string{"ci_buildkit_image": "registry.io/buildkit"},
			want: "registry.io/buildkit:" + Version,
		},
		{
			name: "custom version",
			opts: map[string]string{"ci_buildkit_version": "v0.99.0"},
			want: DefaultImage + ":v0.99.0",
		},
		{
			name: "custom image and version",
			opts: map[string]string{
				"ci_buildkit_image":   "registry.io/buildkit",
				"ci_buildkit_version": "v0.99.0",
			},
			want: "registry.io/buildkit:v0.99.0",
		},
		{
			name: "empty string values use defaults",
			opts: map[string]string{
				"ci_buildkit_image":   "",
				"ci_buildkit_version": "",
			},
			want: DefaultImage + ":" + DefaultTag,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ImageRef(tc.opts)
			if got != tc.want {
				t.Errorf("ImageRef(%v) = %q, want %q", tc.opts, got, tc.want)
			}
		})
	}
}

func TestReleaseSeries(t *testing.T) {
	cases := map[string]string{
		"v0.34.0": "v0.34",
		"v1.2.3":  "v1.2",
		"v0.34":   "v0.34",
		"latest":  "latest",
	}
	for version, want := range cases {
		t.Run(version, func(t *testing.T) {
			if got := releaseSeries(version); got != want {
				t.Errorf("releaseSeries(%q) = %q, want %q", version, got, want)
			}
		})
	}
}

func TestTagFor(t *testing.T) {
	if got := TagFor(DefaultImage); got != DefaultTag {
		t.Errorf("TagFor(DefaultImage) = %q, want %q", got, DefaultTag)
	}
	if got := TagFor("moby/buildkit"); got != Version {
		t.Errorf("TagFor(moby/buildkit) = %q, want %q", got, Version)
	}
}
