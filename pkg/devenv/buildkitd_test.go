package devenv

import (
	"testing"

	"github.com/timo-reymann/ContainerHive/internal/buildkit"
)

func TestResolveImage(t *testing.T) {
	cases := []struct {
		name string
		opts map[string]string
		want string
	}{
		{
			name: "defaults when opts is nil",
			opts: nil,
			want: buildkit.DefaultImage + ":" + buildkit.Version,
		},
		{
			name: "custom image from template opts",
			opts: map[string]string{"ci_buildkit_image": "my-registry.io/buildkit"},
			want: "my-registry.io/buildkit:" + buildkit.Version,
		},
		{
			name: "custom version from template opts",
			opts: map[string]string{"ci_buildkit_version": "v1.0.0"},
			want: buildkit.DefaultImage + ":v1.0.0",
		},
		{
			name: "both custom image and version",
			opts: map[string]string{
				"ci_buildkit_image":   "my-registry.io/buildkit",
				"ci_buildkit_version": "v1.0.0",
			},
			want: "my-registry.io/buildkit:v1.0.0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveImage(tc.opts)
			if got != tc.want {
				t.Errorf("ResolveImage(%v) = %q, want %q", tc.opts, got, tc.want)
			}
		})
	}
}
