package registry

import "testing"

func TestIsDockerHubAddress(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"docker.io", true},
		{"docker.io/timoreymann", true},
		{"index.docker.io", true},
		{"index.docker.io/library", true},
		{"registry-1.docker.io", true},
		{"registry-1.docker.io/timoreymann/ci-go", true},
		{"ghcr.io", false},
		{"ghcr.io/timo-reymann", false},
		{"localhost:5000", false},
		{"127.0.0.1:5000", false},
		{"my-docker.io.example.com", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			if got := IsDockerHubAddress(tt.addr); got != tt.want {
				t.Errorf("IsDockerHubAddress(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestResolveDockerMediaTypes(t *testing.T) {
	tr, fa := true, false
	tests := []struct {
		name     string
		address  string
		override *bool
		want     bool
	}{
		{"auto: docker hub", "docker.io/foo", nil, true},
		{"auto: ghcr", "ghcr.io/foo", nil, false},
		{"override true beats non-hub", "ghcr.io/foo", &tr, true},
		{"override false beats hub", "docker.io/foo", &fa, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveDockerMediaTypes(tt.address, tt.override); got != tt.want {
				t.Errorf("resolveDockerMediaTypes(%q, %v) = %v, want %v", tt.address, tt.override, got, tt.want)
			}
		})
	}
}
