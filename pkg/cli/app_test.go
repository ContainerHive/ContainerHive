package cli

import (
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestValidatePlatforms(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {
				Identifier: "app",
				Name:       "app",
				Platforms:  []string{"linux/amd64", "linux/arm/v7"},
				Variants: map[string]*model.ImageVariant{
					"riscv": {Name: "riscv", TagSuffix: "-riscv", Platforms: []string{"linux/riscv64"}},
				},
			},
		},
		Config: model.HiveProjectConfig{Platforms: []string{"linux/amd64", "linux/arm64"}},
	}

	tests := []struct {
		name      string
		requested []string
		wantErr   string
	}{
		{name: "no flag", requested: nil},
		{name: "project platform", requested: []string{"arm64"}},
		{name: "image platform", requested: []string{"linux/arm/v7"}},
		{name: "variant platform", requested: []string{"riscv64"}},
		{name: "typo", requested: []string{"arm"}, wantErr: "[arm]"},
		{name: "partially unsupported", requested: []string{"amd64", "s390x"}, wantErr: "[s390x]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlatforms(project, tt.requested)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
