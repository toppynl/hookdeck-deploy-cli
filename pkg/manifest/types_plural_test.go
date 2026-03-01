package manifest

import (
	"encoding/json"
	"testing"
)

func TestManifest_UnmarshalPluralSources(t *testing.T) {
	input := `{
		"$schema": "https://example.com/schema.json",
		"sources": [
			{"name": "src-one", "type": "HTTP"},
			{"name": "src-two", "description": "second source"}
		]
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if m.Schema != "https://example.com/schema.json" {
		t.Errorf("Schema = %q, want %q", m.Schema, "https://example.com/schema.json")
	}
	if len(m.Sources) != 2 {
		t.Fatalf("len(Sources) = %d, want 2", len(m.Sources))
	}
	if m.Sources[0].Name != "src-one" {
		t.Errorf("Sources[0].Name = %q, want %q", m.Sources[0].Name, "src-one")
	}
	if m.Sources[0].Type != "HTTP" {
		t.Errorf("Sources[0].Type = %q, want %q", m.Sources[0].Type, "HTTP")
	}
	if m.Sources[1].Name != "src-two" {
		t.Errorf("Sources[1].Name = %q, want %q", m.Sources[1].Name, "src-two")
	}
	if m.Sources[1].Description != "second source" {
		t.Errorf("Sources[1].Description = %q, want %q", m.Sources[1].Description, "second source")
	}
}

func TestManifest_UnmarshalAllResourceTypes(t *testing.T) {
	input := `{
		"$schema": "https://example.com/schema.json",
		"sources": [
			{"name": "my-source", "type": "HTTP"}
		],
		"destinations": [
			{"name": "my-dest", "url": "https://example.com/webhook", "type": "HTTP", "rate_limit": 100, "rate_limit_period": "second"}
		],
		"transformations": [
			{"name": "my-transform", "code_file": "transform.js", "env": {"KEY": "val"}}
		],
		"connections": [
			{"name": "my-conn", "source": "my-source", "destination": "my-dest", "transformations": ["my-transform"]}
		]
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Sources
	if len(m.Sources) != 1 {
		t.Fatalf("len(Sources) = %d, want 1", len(m.Sources))
	}
	if m.Sources[0].Name != "my-source" {
		t.Errorf("Sources[0].Name = %q, want %q", m.Sources[0].Name, "my-source")
	}

	// Destinations
	if len(m.Destinations) != 1 {
		t.Fatalf("len(Destinations) = %d, want 1", len(m.Destinations))
	}
	d := m.Destinations[0]
	if d.Name != "my-dest" {
		t.Errorf("Destinations[0].Name = %q, want %q", d.Name, "my-dest")
	}
	if d.URL != "https://example.com/webhook" {
		t.Errorf("Destinations[0].URL = %q, want %q", d.URL, "https://example.com/webhook")
	}
	if d.RateLimit != 100 {
		t.Errorf("Destinations[0].RateLimit = %d, want 100", d.RateLimit)
	}
	if d.RateLimitPeriod != "second" {
		t.Errorf("Destinations[0].RateLimitPeriod = %q, want %q", d.RateLimitPeriod, "second")
	}

	// Transformations
	if len(m.Transformations) != 1 {
		t.Fatalf("len(Transformations) = %d, want 1", len(m.Transformations))
	}
	tr := m.Transformations[0]
	if tr.Name != "my-transform" {
		t.Errorf("Transformations[0].Name = %q, want %q", tr.Name, "my-transform")
	}
	if tr.CodeFile != "transform.js" {
		t.Errorf("Transformations[0].CodeFile = %q, want %q", tr.CodeFile, "transform.js")
	}
	if tr.Env["KEY"] != "val" {
		t.Errorf("Transformations[0].Env[KEY] = %q, want %q", tr.Env["KEY"], "val")
	}

	// Connections
	if len(m.Connections) != 1 {
		t.Fatalf("len(Connections) = %d, want 1", len(m.Connections))
	}
	c := m.Connections[0]
	if c.Name != "my-conn" {
		t.Errorf("Connections[0].Name = %q, want %q", c.Name, "my-conn")
	}
	if c.Source != "my-source" {
		t.Errorf("Connections[0].Source = %q, want %q", c.Source, "my-source")
	}
	if c.Destination != "my-dest" {
		t.Errorf("Connections[0].Destination = %q, want %q", c.Destination, "my-dest")
	}
	if len(c.Transformations) != 1 || c.Transformations[0] != "my-transform" {
		t.Errorf("Connections[0].Transformations = %v, want [my-transform]", c.Transformations)
	}
}

func TestManifest_UnmarshalEmpty(t *testing.T) {
	input := `{}`

	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if m.Schema != "" {
		t.Errorf("Schema = %q, want empty", m.Schema)
	}
	if m.Sources != nil {
		t.Errorf("Sources = %v, want nil", m.Sources)
	}
	if m.Destinations != nil {
		t.Errorf("Destinations = %v, want nil", m.Destinations)
	}
	if m.Transformations != nil {
		t.Errorf("Transformations = %v, want nil", m.Transformations)
	}
	if m.Connections != nil {
		t.Errorf("Connections = %v, want nil", m.Connections)
	}
}

func TestSourceConfig_EnvOverrides(t *testing.T) {
	input := `{
		"sources": [
			{
				"name": "my-source",
				"type": "HTTP",
				"description": "default desc",
				"env": {
					"staging": {
						"type": "HTTP",
						"description": "staging source",
						"config": {"key": "staging-val"}
					},
					"production": {
						"description": "prod source"
					}
				}
			}
		]
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(m.Sources) != 1 {
		t.Fatalf("len(Sources) = %d, want 1", len(m.Sources))
	}
	src := m.Sources[0]
	if src.Name != "my-source" {
		t.Errorf("Name = %q, want %q", src.Name, "my-source")
	}
	if src.Env == nil {
		t.Fatal("Env is nil, want map with 2 entries")
	}
	if len(src.Env) != 2 {
		t.Fatalf("len(Env) = %d, want 2", len(src.Env))
	}

	staging := src.Env["staging"]
	if staging == nil {
		t.Fatal("Env[staging] is nil")
	}
	if staging.Type != "HTTP" {
		t.Errorf("Env[staging].Type = %q, want %q", staging.Type, "HTTP")
	}
	if staging.Description != "staging source" {
		t.Errorf("Env[staging].Description = %q, want %q", staging.Description, "staging source")
	}
	if staging.Config == nil {
		t.Fatal("Env[staging].Config is nil")
	}
	if staging.Config["key"] != "staging-val" {
		t.Errorf("Env[staging].Config[key] = %v, want %q", staging.Config["key"], "staging-val")
	}

	prod := src.Env["production"]
	if prod == nil {
		t.Fatal("Env[production] is nil")
	}
	if prod.Description != "prod source" {
		t.Errorf("Env[production].Description = %q, want %q", prod.Description, "prod source")
	}
}

func TestDestinationConfig_EnvOverrides(t *testing.T) {
	input := `{
		"destinations": [
			{
				"name": "my-dest",
				"url": "https://default.example.com",
				"type": "HTTP",
				"env": {
					"staging": {
						"url": "https://staging.example.com",
						"type": "HTTP",
						"description": "staging dest",
						"auth_type": "bearer",
						"auth": {"token": "staging-token"},
						"config": {"retry": true},
						"rate_limit": 50,
						"rate_limit_period": "minute"
					},
					"production": {
						"url": "https://prod.example.com",
						"rate_limit": 1000
					}
				}
			}
		]
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(m.Destinations) != 1 {
		t.Fatalf("len(Destinations) = %d, want 1", len(m.Destinations))
	}
	dest := m.Destinations[0]
	if dest.Name != "my-dest" {
		t.Errorf("Name = %q, want %q", dest.Name, "my-dest")
	}
	if dest.Env == nil {
		t.Fatal("Env is nil, want map with 2 entries")
	}
	if len(dest.Env) != 2 {
		t.Fatalf("len(Env) = %d, want 2", len(dest.Env))
	}

	staging := dest.Env["staging"]
	if staging == nil {
		t.Fatal("Env[staging] is nil")
	}
	if staging.URL != "https://staging.example.com" {
		t.Errorf("Env[staging].URL = %q, want %q", staging.URL, "https://staging.example.com")
	}
	if staging.Type != "HTTP" {
		t.Errorf("Env[staging].Type = %q, want %q", staging.Type, "HTTP")
	}
	if staging.Description != "staging dest" {
		t.Errorf("Env[staging].Description = %q, want %q", staging.Description, "staging dest")
	}
	if staging.AuthType != "bearer" {
		t.Errorf("Env[staging].AuthType = %q, want %q", staging.AuthType, "bearer")
	}
	if staging.Auth == nil || staging.Auth["token"] != "staging-token" {
		t.Errorf("Env[staging].Auth = %v, want map with token=staging-token", staging.Auth)
	}
	if staging.Config == nil || staging.Config["retry"] != true {
		t.Errorf("Env[staging].Config = %v, want map with retry=true", staging.Config)
	}
	if staging.RateLimit != 50 {
		t.Errorf("Env[staging].RateLimit = %d, want 50", staging.RateLimit)
	}
	if staging.RateLimitPeriod != "minute" {
		t.Errorf("Env[staging].RateLimitPeriod = %q, want %q", staging.RateLimitPeriod, "minute")
	}

	prod := dest.Env["production"]
	if prod == nil {
		t.Fatal("Env[production] is nil")
	}
	if prod.URL != "https://prod.example.com" {
		t.Errorf("Env[production].URL = %q, want %q", prod.URL, "https://prod.example.com")
	}
	if prod.RateLimit != 1000 {
		t.Errorf("Env[production].RateLimit = %d, want 1000", prod.RateLimit)
	}
}
