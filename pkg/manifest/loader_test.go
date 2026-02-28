package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_BasicJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hookdeck.jsonc")
	content := []byte(`{
		// A comment
		"source": {
			"name": "test-source",
			"description": "A test source"
		}
	}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if m.Source == nil {
		t.Fatal("expected source to be non-nil")
	}
	if m.Source.Name != "test-source" {
		t.Errorf("expected name 'test-source', got '%s'", m.Source.Name)
	}
}

func TestFindManifestFile_PrefersJSONC(t *testing.T) {
	dir := t.TempDir()
	jsoncPath := filepath.Join(dir, "hookdeck.jsonc")
	jsonPath := filepath.Join(dir, "hookdeck.json")
	os.WriteFile(jsoncPath, []byte(`{}`), 0644)
	os.WriteFile(jsonPath, []byte(`{}`), 0644)

	found, err := FindFile(dir)
	if err != nil {
		t.Fatalf("FindFile failed: %v", err)
	}
	if found != jsoncPath {
		t.Errorf("expected %s, got %s", jsoncPath, found)
	}
}

func TestFindManifestFile_FallsBackToJSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "hookdeck.json")
	os.WriteFile(jsonPath, []byte(`{}`), 0644)

	found, err := FindFile(dir)
	if err != nil {
		t.Fatalf("FindFile failed: %v", err)
	}
	if found != jsonPath {
		t.Errorf("expected %s, got %s", jsonPath, found)
	}
}

func TestFindManifestFile_ErrorWhenNone(t *testing.T) {
	dir := t.TempDir()
	_, err := FindFile(dir)
	if err == nil {
		t.Fatal("expected error when no manifest found")
	}
}

func TestLoadFile_ConnectionFilterAndTransformations(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"connection": {
			"name": "test-conn",
			"source": "my-source",
			"destination": "my-dest",
			"filter": {
				"data": { "entity": { "enabled": true } }
			},
			"transformations": ["my-transform"]
		}
	}`
	path := filepath.Join(dir, "hookdeck.jsonc")
	os.WriteFile(path, []byte(content), 0644)

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Connection == nil {
		t.Fatal("expected Connection to be non-nil")
	}
	if m.Connection.Filter == nil {
		t.Fatal("expected Filter to be non-nil")
	}
	dataVal, ok := m.Connection.Filter["data"]
	if !ok {
		t.Fatal("expected Filter to contain key 'data'")
	}
	dataMap, ok := dataVal.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Filter['data'] to be a map, got %T", dataVal)
	}
	if _, ok := dataMap["entity"]; !ok {
		t.Error("expected Filter['data'] to contain key 'entity'")
	}
	if len(m.Connection.Transformations) != 1 || m.Connection.Transformations[0] != "my-transform" {
		t.Errorf("expected Transformations [my-transform], got %v", m.Connection.Transformations)
	}
}

func TestLoadFile_TransformationEnvVars(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"transformation": {
			"name": "test-transform",
			"env_vars": {
				"WORKFLOW_ID": "my-workflow"
			}
		}
	}`
	path := filepath.Join(dir, "hookdeck.jsonc")
	os.WriteFile(path, []byte(content), 0644)

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Transformation.Env["WORKFLOW_ID"] != "my-workflow" {
		t.Errorf("expected Env[WORKFLOW_ID] = my-workflow, got %v", m.Transformation.Env)
	}
}

func TestLoadFile_TransformationEnvVarsInEnvOverride(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"transformation": {
			"name": "test-transform",
			"env": {
				"BASE_URL": "https://example.com"
			}
		},
		"env": {
			"staging": {
				"transformation": {
					"env_vars": {
						"WORKFLOW_ID": "staging-workflow"
					}
				}
			}
		}
	}`
	path := filepath.Join(dir, "hookdeck.jsonc")
	os.WriteFile(path, []byte(content), 0644)

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Env["staging"].Transformation.Env["WORKFLOW_ID"] != "staging-workflow" {
		t.Errorf("expected staging Env[WORKFLOW_ID] = staging-workflow, got %v", m.Env["staging"].Transformation.Env)
	}
}

func TestLoadFile_TransformationEnvVarsMergedWithEnv(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"transformation": {
			"name": "test-transform",
			"env": {
				"BASE_URL": "https://example.com"
			},
			"env_vars": {
				"WORKFLOW_ID": "my-workflow",
				"BASE_URL": "should-not-override"
			}
		}
	}`
	path := filepath.Join(dir, "hookdeck.jsonc")
	os.WriteFile(path, []byte(content), 0644)

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// env_vars should be merged into env, but env takes priority for conflicts
	if m.Transformation.Env["WORKFLOW_ID"] != "my-workflow" {
		t.Errorf("expected Env[WORKFLOW_ID] = my-workflow, got %v", m.Transformation.Env["WORKFLOW_ID"])
	}
	if m.Transformation.Env["BASE_URL"] != "https://example.com" {
		t.Errorf("expected Env[BASE_URL] = https://example.com (env wins over env_vars), got %v", m.Transformation.Env["BASE_URL"])
	}
}

func TestLoadFile_DestinationURLAndAuth(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"destination": {
			"name": "test-dest",
			"url": "https://example.com/webhook",
			"auth_type": "API_KEY",
			"auth": {
				"key": "Authorization",
				"api_key": "Bearer secret",
				"to": "header"
			},
			"rate_limit": 1,
			"rate_limit_period": "concurrent"
		}
	}`
	path := filepath.Join(dir, "hookdeck.jsonc")
	os.WriteFile(path, []byte(content), 0644)

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Destination.URL != "https://example.com/webhook" {
		t.Errorf("expected URL https://example.com/webhook, got %q", m.Destination.URL)
	}
	if m.Destination.AuthType != "API_KEY" {
		t.Errorf("expected AuthType API_KEY, got %q", m.Destination.AuthType)
	}
	if m.Destination.Auth == nil {
		t.Fatal("expected Auth to be non-nil")
	}
	if m.Destination.Auth["key"] != "Authorization" {
		t.Errorf("expected auth key 'Authorization', got %v", m.Destination.Auth["key"])
	}
}
