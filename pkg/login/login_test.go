package login

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogin_BasicCredentials(t *testing.T) {
	configDir := t.TempDir()

	configPath, err := Login(Options{
		ServerAddress: "registry.example.com",
		Username:      "testuser",
		Password:      "testpass",
		ConfigDir:     configDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	if configPath != filepath.Join(configDir, "config.json") {
		t.Errorf("expected config path %s, got %s", filepath.Join(configDir, "config.json"), configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "registry.example.com") {
		t.Errorf("config should contain registry address, got: %s", content)
	}
}

func TestLogin_PasswordStdin(t *testing.T) {
	configDir := t.TempDir()

	configPath, err := Login(Options{
		ServerAddress: "registry.example.com",
		Username:      "testuser",
		PasswordStdin: strings.NewReader("stdinpass\n"),
		ConfigDir:     configDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "registry.example.com") {
		t.Errorf("config should contain registry address, got: %s", content)
	}
}

func TestLogin_PasswordStdinTrimsCarriageReturn(t *testing.T) {
	configDir := t.TempDir()

	_, err := Login(Options{
		ServerAddress: "registry.example.com",
		Username:      "testuser",
		PasswordStdin: strings.NewReader("stdinpass\r\n"),
		ConfigDir:     configDir,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLogin_MissingUsernameAndPassword(t *testing.T) {
	configDir := t.TempDir()

	_, err := Login(Options{
		ServerAddress: "registry.example.com",
		ConfigDir:     configDir,
	})
	if err == nil {
		t.Fatal("expected error for missing username and password")
	}
	if !strings.Contains(err.Error(), "username and password required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLogin_InvalidRegistry(t *testing.T) {
	configDir := t.TempDir()

	_, err := Login(Options{
		ServerAddress: "://invalid",
		Username:      "user",
		Password:      "pass",
		ConfigDir:     configDir,
	})
	if err == nil {
		t.Fatal("expected error for invalid registry")
	}
}

func TestLogin_DefaultRegistry(t *testing.T) {
	configDir := t.TempDir()

	_, err := Login(Options{
		ServerAddress: "index.docker.io",
		Username:      "testuser",
		Password:      "testpass",
		ConfigDir:     configDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	// Should be stored under the Docker Hub auth key
	if !strings.Contains(content, "https://index.docker.io/v1/") {
		t.Errorf("Docker Hub credentials should use default auth key, got: %s", content)
	}
}

func TestLogin_PasswordStdinOverridesPassword(t *testing.T) {
	configDir := t.TempDir()

	_, err := Login(Options{
		ServerAddress: "registry.example.com",
		Username:      "testuser",
		Password:      "flagpass",
		PasswordStdin: strings.NewReader("stdinpass\n"),
		ConfigDir:     configDir,
	})
	if err != nil {
		t.Fatal(err)
	}
}
