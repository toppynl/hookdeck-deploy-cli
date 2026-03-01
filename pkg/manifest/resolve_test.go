package manifest

import (
	"os"
	"testing"
)

func TestResolveSourceEnv_WithOverride(t *testing.T) {
	src := SourceConfig{
		Name: "s1",
		Type: "Stripe",
		Env: map[string]*SourceOverride{
			"production": {Type: "HMAC"},
		},
	}
	resolved := ResolveSourceEnv(&src, "production")
	if resolved.Type != "HMAC" {
		t.Errorf("expected type 'HMAC', got '%s'", resolved.Type)
	}
	if resolved.Name != "s1" {
		t.Errorf("expected name 's1', got '%s'", resolved.Name)
	}
}

func TestResolveSourceEnv_NoOverride(t *testing.T) {
	src := SourceConfig{Name: "s1", Type: "Stripe"}
	resolved := ResolveSourceEnv(&src, "production")
	if resolved.Type != "Stripe" {
		t.Errorf("expected type 'Stripe', got '%s'", resolved.Type)
	}
}

func TestResolveSourceEnv_EmptyEnvName(t *testing.T) {
	src := SourceConfig{Name: "s1", Type: "Stripe"}
	resolved := ResolveSourceEnv(&src, "")
	if resolved.Type != "Stripe" {
		t.Errorf("expected type 'Stripe', got '%s'", resolved.Type)
	}
}

func TestResolveDestinationEnv_WithOverride(t *testing.T) {
	dst := DestinationConfig{
		Name: "d1",
		URL:  "https://dev.example.com",
		Env: map[string]*DestinationOverride{
			"production": {URL: "https://api.example.com", RateLimit: 50},
		},
	}
	resolved := ResolveDestinationEnv(&dst, "production")
	if resolved.URL != "https://api.example.com" {
		t.Errorf("expected production URL, got '%s'", resolved.URL)
	}
	if resolved.RateLimit != 50 {
		t.Errorf("expected rate_limit 50, got %d", resolved.RateLimit)
	}
}

func TestResolveTransformationEnv_WithOverride(t *testing.T) {
	tr := TransformationConfig{
		Name:     "t1",
		CodeFile: "handler.js",
		Env:      map[string]string{"API_URL": "https://dev.example.com"},
		EnvOverrides: map[string]*TransformationOverride{
			"production": {Env: map[string]string{"API_URL": "https://api.example.com"}},
		},
	}
	resolved := ResolveTransformationEnv(&tr, "production")
	if resolved.Env["API_URL"] != "https://api.example.com" {
		t.Errorf("expected production API_URL, got '%s'", resolved.Env["API_URL"])
	}
}

func TestInterpolateManifestEnvVars(t *testing.T) {
	os.Setenv("TEST_URL", "https://example.com")
	defer os.Unsetenv("TEST_URL")

	m := &Manifest{
		Destinations: []DestinationConfig{
			{Name: "d1", URL: "${TEST_URL}/webhooks"},
		},
	}
	if err := InterpolateEnvVars(m); err != nil {
		t.Fatalf("InterpolateEnvVars failed: %v", err)
	}
	if m.Destinations[0].URL != "https://example.com/webhooks" {
		t.Errorf("expected interpolated URL, got '%s'", m.Destinations[0].URL)
	}
}

func TestInterpolateManifestEnvVars_MissingVar(t *testing.T) {
	m := &Manifest{
		Sources: []SourceConfig{
			{Name: "${MISSING_VAR}"},
		},
	}
	err := InterpolateEnvVars(m)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}
