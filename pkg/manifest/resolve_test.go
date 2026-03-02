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

func TestResolveConnectionEnv_FilterOverride(t *testing.T) {
	conn := ConnectionConfig{
		Name:        "c1",
		Source:      "src",
		Destination: "dst",
		Filter:      map[string]interface{}{"headers": map[string]interface{}{"x-env": "default"}},
		Env: map[string]*ConnectionOverride{
			"staging": {Filter: map[string]interface{}{"headers": map[string]interface{}{"x-env": "staging"}}},
		},
	}
	resolved := ResolveConnectionEnv(&conn, "staging")
	if resolved.Name != "c1" {
		t.Errorf("expected name 'c1', got '%s'", resolved.Name)
	}
	if resolved.Filter["headers"].(map[string]interface{})["x-env"] != "staging" {
		t.Errorf("expected staging filter, got %v", resolved.Filter)
	}
}

func TestResolveConnectionEnv_TransformationsOverride(t *testing.T) {
	conn := ConnectionConfig{
		Name:            "c1",
		Source:          "src",
		Destination:     "dst",
		Transformations: []string{"anonymizer"},
		Env: map[string]*ConnectionOverride{
			"production": {Transformations: []string{}},
		},
	}

	// Staging has no override — should keep base transformations
	resolved := ResolveConnectionEnv(&conn, "staging")
	if len(resolved.Transformations) != 1 || resolved.Transformations[0] != "anonymizer" {
		t.Errorf("staging: expected [anonymizer], got %v", resolved.Transformations)
	}

	// Production override with empty slice — should clear transformations
	resolved = ResolveConnectionEnv(&conn, "production")
	if len(resolved.Transformations) != 0 {
		t.Errorf("production: expected empty transformations, got %v", resolved.Transformations)
	}
}

func TestResolveConnectionEnv_NoOverride(t *testing.T) {
	conn := ConnectionConfig{
		Name:   "c1",
		Source: "src",
		Filter: map[string]interface{}{"body": "test"},
	}
	resolved := ResolveConnectionEnv(&conn, "production")
	if resolved.Source != "src" {
		t.Errorf("expected source 'src', got '%s'", resolved.Source)
	}
	if resolved.Filter["body"] != "test" {
		t.Errorf("expected base filter preserved, got %v", resolved.Filter)
	}
}

func TestResolveConnectionEnv_EmptyEnvName(t *testing.T) {
	conn := ConnectionConfig{
		Name:   "c1",
		Source: "src",
		Env: map[string]*ConnectionOverride{
			"staging": {Source: "other-src"},
		},
	}
	resolved := ResolveConnectionEnv(&conn, "")
	if resolved.Source != "src" {
		t.Errorf("expected source 'src' for empty envName, got '%s'", resolved.Source)
	}
}

func TestResolveConnectionEnv_UnknownEnv(t *testing.T) {
	conn := ConnectionConfig{
		Name:   "c1",
		Source: "src",
		Env: map[string]*ConnectionOverride{
			"staging": {Source: "staging-src"},
		},
	}
	resolved := ResolveConnectionEnv(&conn, "dev")
	if resolved.Source != "src" {
		t.Errorf("expected source 'src' for unknown env, got '%s'", resolved.Source)
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
