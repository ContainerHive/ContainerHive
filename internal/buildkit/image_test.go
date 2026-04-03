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
			want: DefaultImage + ":" + Version,
		},
		{
			name: "defaults when opts is empty",
			opts: map[string]string{},
			want: DefaultImage + ":" + Version,
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
			want: DefaultImage + ":" + Version,
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
