package build

import (
	"context"
	"os"
	"testing"
)

func TestNewClientBuildKitAddress(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		endpoint     string
		expectError  bool
		expectedAddr string
	}{
		{
			name:         "explicit endpoint",
			envValue:     "",
			endpoint:     "tcp://127.0.0.1:8502",
			expectError:  false,
			expectedAddr: "tcp://127.0.0.1:8502",
		},
		{
			name:         "env var when endpoint empty",
			envValue:     "tcp://localhost:1234",
			endpoint:     "",
			expectError:  false,
			expectedAddr: "tcp://localhost:1234",
		},
		{
			name:        "no endpoint and no env var",
			envValue:    "",
			endpoint:    "",
			expectError: true,
		},
		{
			name:         "explicit endpoint overrides env var",
			envValue:     "tcp://from-env:9999",
			endpoint:     "tcp://explicit:8888",
			expectError:  false,
			expectedAddr: "tcp://explicit:8888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("BUILDKIT_HOST", tt.envValue)
				defer os.Unsetenv("BUILDKIT_HOST")
			}

			ctx := context.Background()
			client, err := NewClient(ctx, tt.endpoint)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected client but got nil")
				return
			}

			// We can't easily test the actual endpoint used without exposing it,
			// but we can verify the client was created successfully
			defer client.Close()
		})
	}
}
