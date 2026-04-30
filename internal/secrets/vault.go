package secrets

import (
	"fmt"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/vault"
)

const vaultResolver = "vault"

// VaultSecretResolver resolves secrets from HashiCorp Vault using the "vault://<path>#<field>" URI scheme.
type VaultSecretResolver struct {
}

func (v VaultSecretResolver) Resolve(value string) (resolvedValue string, err error) {
	if !strings.HasPrefix(value, "vault://") {
		return "", nil
	}

	// Parse the vault secret specification
	// Expected format: vault://<path>#<field>
	spec := value[8:] // Remove "vault://" prefix
	if spec == "" {
		return "", fmt.Errorf("malformed vault secret spec '%s', missing path and field", value)
	}

	specParts := strings.SplitN(spec, "#", 2)
	if len(specParts) != 2 {
		return "", fmt.Errorf("malformed vault secret spec '%s', should be in format 'vault://<path>#<field>'", value)
	}

	path := strings.TrimSpace(specParts[0])
	field := strings.TrimSpace(specParts[1])

	if path == "" {
		return "", fmt.Errorf("malformed vault secret spec '%s', path cannot be empty", value)
	}

	if field == "" {
		return "", fmt.Errorf("malformed vault secret spec '%s', field cannot be empty", value)
	}

	// Use the vault client to get the secret
	return vault.GetSecretWithDefaultConfiguration(path, field)
}
