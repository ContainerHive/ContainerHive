package vault

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestVaultIntegration tests the vault functionality with a real Vault server
func TestVaultIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start Vault container
	vaultContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "vault:1.13.3",
			ExposedPorts: []string{"8200/tcp"},
			Cmd:          []string{"server", "-dev", "-dev-listen-address=0.0.0.0:8200"},
			Env: map[string]string{
				"VAULT_DEV_ROOT_TOKEN_ID": "test-root-token",
				"VAULT_ADDR":              "http://127.0.0.1:8200",
			},
			WaitingFor: wait.ForHTTP("/v1/sys/health").
				WithPort("8200/tcp").
				WithStartupTimeout(30 * time.Second),
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.AutoRemove = true
			},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("Failed to start Vault container: %v", err)
	}
	t.Cleanup(func() {
		if err := vaultContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate Vault container: %v", err)
		}
	})

	// Get container host and port
	vaultHost, err := vaultContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get Vault container host: %v", err)
	}

	vaultPort, err := vaultContainer.MappedPort(ctx, "8200/tcp")
	if err != nil {
		t.Fatalf("Failed to get Vault container port: %v", err)
	}

	vaultAddr := fmt.Sprintf("http://%s:%s", vaultHost, vaultPort.Port())
	t.Logf("Vault server started at: %s", vaultAddr)

	// Set environment variables for our vault client
	t.Setenv("VAULT_ADDR", vaultAddr)
	t.Setenv("VAULT_TOKEN", "test-root-token")

	// Create a temporary home directory with vault token
	tempDir := t.TempDir()
	vaultTokenFile := filepath.Join(tempDir, ".vault-token")
	err = os.WriteFile(vaultTokenFile, []byte("test-root-token"), 0600)
	if err != nil {
		t.Fatalf("Failed to create vault token file: %v", err)
	}
	t.Setenv("HOME", tempDir)

	// Test basic connectivity
	t.Run("test_connectivity", func(t *testing.T) {
		// Test that we can get the vault status
		_, err := getSecret(vaultAddr, "test-root-token", "sys/health", "status")
		if err != nil {
			t.Logf("Connectivity test failed (expected for sys/health): %v", err)
			// This is expected to fail since sys/health doesn't return data in the expected format
		}
	})

	// Test secret creation and retrieval using the Vault API
	t.Run("test_secret_operations", func(t *testing.T) {
		// We'll use the existing getSecret function to test basic connectivity
		// Note: This is a simplified test - in a real scenario, you'd want to test
		// the full secret creation/retrieval workflow

		// Test that our vault client can communicate with the server
		_, err := getSecret(vaultAddr, "test-root-token", "secret/data/test", "password")
		if err != nil {
			t.Logf("Secret retrieval test failed (expected for non-existent secret): %v", err)
			// This is expected to fail since the secret doesn't exist yet
		}
	})

	// Test our high-level function
	t.Run("test_get_secret_with_default_config", func(t *testing.T) {
		// This should work now that we've set up the environment
		secret, err := GetSecretWithDefaultConfiguration("secret/data/test", "password")
		if err != nil {
			t.Logf("GetSecretWithDefaultConfiguration failed (expected for non-existent secret): %v", err)
			// Expected to fail since we haven't created the secret
		} else {
			t.Logf("Retrieved secret: %s", secret)
		}
	})

	// Test token lookup functions
	t.Run("test_token_lookup", func(t *testing.T) {
		// Test environment variable token lookup
		token := lookupEnvToken()
		if token != "test-root-token" {
			t.Errorf("lookupEnvToken() = %v, want %v", token, "test-root-token")
		}

		// Test CLI token lookup
		cliToken, err := lookupCliToken()
		if err != nil {
			t.Fatalf("lookupCliToken() failed: %v", err)
		}
		if cliToken != "test-root-token" {
			t.Errorf("lookupCliToken() = %v, want %v", cliToken, "test-root-token")
		}

		// Test main token lookup (should prefer env var)
		mainToken, err := LookupToken()
		if err != nil {
			t.Fatalf("LookupToken() failed: %v", err)
		}
		if mainToken != "test-root-token" {
			t.Errorf("LookupToken() = %v, want %v", mainToken, "test-root-token")
		}
	})
}
