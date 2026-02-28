package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_EnvVarTakesPrecedence(t *testing.T) {
	t.Setenv("HOOKDECK_API_KEY", "env-key-123")

	creds, err := Resolve("some-profile")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if creds.APIKey != "env-key-123" {
		t.Errorf("expected env key, got '%s'", creds.APIKey)
	}
}

func TestResolve_ErrorWhenNoCredentials(t *testing.T) {
	t.Setenv("HOOKDECK_API_KEY", "")
	// Point HOME to a temp dir so no real config is found
	t.Setenv("HOME", t.TempDir())
	// Change to a temp dir so local .hookdeck/config.toml isn't found
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(t.TempDir())

	_, err := Resolve("")
	if err == nil {
		t.Fatal("expected error when no credentials available")
	}
}

func TestResolve_LoadsFromTOMLProfile(t *testing.T) {
	t.Setenv("HOOKDECK_API_KEY", "")

	// Create a temp config file with a named profile
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "hookdeck")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.toml")
	os.WriteFile(configPath, []byte(`
profile = "staging"

[staging]
api_key = "toml-staging-key"
project_id = "proj-123"

[default]
api_key = "toml-default-key"
`), 0o644)

	t.Setenv("HOME", tmpDir)
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(t.TempDir())

	// Explicit profile name takes precedence
	creds, err := Resolve("staging")
	if err != nil {
		t.Fatalf("Resolve with named profile failed: %v", err)
	}
	if creds.APIKey != "toml-staging-key" {
		t.Errorf("expected 'toml-staging-key', got '%s'", creds.APIKey)
	}
	if creds.ProjectID != "proj-123" {
		t.Errorf("expected 'proj-123', got '%s'", creds.ProjectID)
	}
}

func TestResolve_DefaultProfileFromTOML(t *testing.T) {
	t.Setenv("HOOKDECK_API_KEY", "")

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "hookdeck")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.toml")
	os.WriteFile(configPath, []byte(`
profile = "production"

[production]
api_key = "prod-key-456"

[default]
api_key = "default-key-789"
`), 0o644)

	t.Setenv("HOME", tmpDir)
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(t.TempDir())

	// Empty profile name falls back to the "profile" key in TOML
	creds, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve with default profile failed: %v", err)
	}
	if creds.APIKey != "prod-key-456" {
		t.Errorf("expected 'prod-key-456', got '%s'", creds.APIKey)
	}
}

func TestResolve_EnvOverridesToml(t *testing.T) {
	// Even with a valid TOML config, env var takes precedence
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "hookdeck")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.toml")
	os.WriteFile(configPath, []byte(`
[default]
api_key = "toml-key"
`), 0o644)

	t.Setenv("HOME", tmpDir)
	t.Setenv("HOOKDECK_API_KEY", "env-wins")

	creds, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if creds.APIKey != "env-wins" {
		t.Errorf("expected 'env-wins', got '%s'", creds.APIKey)
	}
}

func TestResolve_LocalConfigTakesPrecedence(t *testing.T) {
	t.Setenv("HOOKDECK_API_KEY", "")

	// Create global config
	tmpHome := t.TempDir()
	globalDir := filepath.Join(tmpHome, ".config", "hookdeck")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte(`
[default]
api_key = "global-key"
`), 0o644)

	// Create local config in a working directory
	tmpWork := t.TempDir()
	localDir := filepath.Join(tmpWork, ".hookdeck")
	os.MkdirAll(localDir, 0o755)
	os.WriteFile(filepath.Join(localDir, "config.toml"), []byte(`
[default]
api_key = "local-key"
`), 0o644)

	t.Setenv("HOME", tmpHome)
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmpWork)

	creds, err := Resolve("default")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if creds.APIKey != "local-key" {
		t.Errorf("expected 'local-key', got '%s'", creds.APIKey)
	}
}
