package manifest

import (
	"testing"
)

func TestResolveEnv_AppliesOverlay(t *testing.T) {
	m := &Manifest{
		Profile: "default",
		Destination: &DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://dev.example.com"},
		},
		Env: map[string]*EnvOverride{
			"staging": {
				Profile: "staging",
				Destination: &DestinationConfig{
					Config: map[string]interface{}{"url": "https://staging.example.com"},
				},
			},
		},
	}

	resolved, err := ResolveEnv(m, "staging")
	if err != nil {
		t.Fatalf("ResolveEnv failed: %v", err)
	}
	if resolved.Profile != "staging" {
		t.Errorf("expected profile 'staging', got '%s'", resolved.Profile)
	}
	if resolved.Destination.Config["url"] != "https://staging.example.com" {
		t.Errorf("expected staging URL, got '%v'", resolved.Destination.Config["url"])
	}
	if resolved.Destination.Name != "my-dest" {
		t.Errorf("expected name preserved, got '%s'", resolved.Destination.Name)
	}
}

func TestResolveEnv_NoEnvUsesBase(t *testing.T) {
	m := &Manifest{
		Profile: "default",
		Source:  &SourceConfig{Name: "my-source"},
	}

	resolved, err := ResolveEnv(m, "")
	if err != nil {
		t.Fatalf("ResolveEnv failed: %v", err)
	}
	if resolved.Source.Name != "my-source" {
		t.Errorf("expected source name preserved")
	}
}

func TestResolveEnv_UnknownEnvErrors(t *testing.T) {
	m := &Manifest{
		Env: map[string]*EnvOverride{
			"staging": {Profile: "staging"},
		},
	}

	_, err := ResolveEnv(m, "production")
	if err == nil {
		t.Fatal("expected error for unknown env")
	}
}

func TestResolveEnv_DestinationURL(t *testing.T) {
	m := &Manifest{
		Destination: &DestinationConfig{
			Name: "dest",
			URL:  "https://dev.example.com",
		},
		Env: map[string]*EnvOverride{
			"staging": {
				Destination: &DestinationConfig{
					URL: "https://staging.example.com",
				},
			},
		},
	}
	resolved, err := ResolveEnv(m, "staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Destination.URL != "https://staging.example.com" {
		t.Errorf("expected staging URL, got %s", resolved.Destination.URL)
	}
	if resolved.Destination.Name != "dest" {
		t.Errorf("expected name preserved from base, got %s", resolved.Destination.Name)
	}
}

func TestResolveEnv_ConnectionFilterOverlay(t *testing.T) {
	m := &Manifest{
		Connection: &ConnectionConfig{
			Name:   "conn",
			Source: "src",
			Filter: map[string]interface{}{"type": "order"},
		},
		Env: map[string]*EnvOverride{
			"production": {
				Connection: &ConnectionConfig{
					Filter: map[string]interface{}{"type": "order", "env": "prod"},
				},
			},
		},
	}
	resolved, err := ResolveEnv(m, "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Connection.Name != "conn" {
		t.Errorf("expected name preserved, got %s", resolved.Connection.Name)
	}
	if resolved.Connection.Source != "src" {
		t.Errorf("expected source preserved, got %s", resolved.Connection.Source)
	}
	if resolved.Connection.Filter["env"] != "prod" {
		t.Errorf("expected production filter overlay, got %v", resolved.Connection.Filter)
	}
}

func TestResolveEnv_DestinationAuthOverlay(t *testing.T) {
	m := &Manifest{
		Destination: &DestinationConfig{
			Name:     "dest",
			URL:      "https://api.example.com",
			AuthType: "API_KEY",
			Auth:     map[string]interface{}{"key": "X-Api-Key", "value": "dev-key"},
		},
		Env: map[string]*EnvOverride{
			"production": {
				Destination: &DestinationConfig{
					Auth: map[string]interface{}{"value": "prod-key"},
				},
			},
		},
	}
	resolved, err := ResolveEnv(m, "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Destination.AuthType != "API_KEY" {
		t.Errorf("expected auth_type preserved, got %s", resolved.Destination.AuthType)
	}
	if resolved.Destination.Auth["key"] != "X-Api-Key" {
		t.Errorf("expected auth key preserved, got %v", resolved.Destination.Auth["key"])
	}
	if resolved.Destination.Auth["value"] != "prod-key" {
		t.Errorf("expected auth value overridden, got %v", resolved.Destination.Auth["value"])
	}
}

func TestInterpolateEnvVars_Replaces(t *testing.T) {
	t.Setenv("TEST_SECRET", "my-secret-value")

	m := &Manifest{
		Source: &SourceConfig{
			Name: "source-${TEST_SECRET}",
		},
	}

	err := InterpolateEnvVars(m)
	if err != nil {
		t.Fatalf("InterpolateEnvVars failed: %v", err)
	}
	if m.Source.Name != "source-my-secret-value" {
		t.Errorf("expected interpolated name, got '%s'", m.Source.Name)
	}
}

func TestInterpolateEnvVars_ErrorOnMissing(t *testing.T) {
	m := &Manifest{
		Source: &SourceConfig{
			Name: "source-${NONEXISTENT_VAR_12345}",
		},
	}

	err := InterpolateEnvVars(m)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}
