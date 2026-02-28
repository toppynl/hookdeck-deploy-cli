package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithInheritance_ExtendsParent(t *testing.T) {
	dir := t.TempDir()

	parent := filepath.Join(dir, "hookdeck.jsonc")
	os.WriteFile(parent, []byte(`{
		"profile": "default",
		"env": {
			"staging": { "profile": "staging" },
			"production": { "profile": "production" }
		}
	}`), 0644)

	childDir := filepath.Join(dir, "sources", "my-source")
	os.MkdirAll(childDir, 0755)
	child := filepath.Join(childDir, "hookdeck.jsonc")
	os.WriteFile(child, []byte(`{
		"extends": "../../hookdeck.jsonc",
		"source": { "name": "my-source" }
	}`), 0644)

	m, err := LoadWithInheritance(child)
	if err != nil {
		t.Fatalf("LoadWithInheritance failed: %v", err)
	}
	if m.Source == nil || m.Source.Name != "my-source" {
		t.Error("expected source from child")
	}
	if m.Env == nil || m.Env["staging"] == nil || m.Env["staging"].Profile != "staging" {
		t.Error("expected env.staging.profile from parent")
	}
}

func TestLoadWithInheritance_CircularDetection(t *testing.T) {
	dir := t.TempDir()

	a := filepath.Join(dir, "a.jsonc")
	b := filepath.Join(dir, "b.jsonc")
	os.WriteFile(a, []byte(`{"extends": "b.jsonc"}`), 0644)
	os.WriteFile(b, []byte(`{"extends": "a.jsonc"}`), 0644)

	_, err := LoadWithInheritance(a)
	if err == nil {
		t.Fatal("expected circular inheritance error")
	}
}

func TestLoadWithInheritance_ChildOverridesParent(t *testing.T) {
	dir := t.TempDir()

	parent := filepath.Join(dir, "parent.jsonc")
	os.WriteFile(parent, []byte(`{
		"profile": "default",
		"destination": { "name": "parent-dest", "config": { "url": "https://parent.example.com" } }
	}`), 0644)

	child := filepath.Join(dir, "child.jsonc")
	os.WriteFile(child, []byte(`{
		"extends": "parent.jsonc",
		"profile": "custom",
		"destination": { "name": "child-dest" }
	}`), 0644)

	m, err := LoadWithInheritance(child)
	if err != nil {
		t.Fatalf("LoadWithInheritance failed: %v", err)
	}
	if m.Profile != "custom" {
		t.Errorf("expected profile 'custom', got '%s'", m.Profile)
	}
	if m.Destination.Name != "child-dest" {
		t.Errorf("expected destination name 'child-dest', got '%s'", m.Destination.Name)
	}
	// Child destination has no config, but parent does — deep merge should preserve it
	if m.Destination.Config == nil || m.Destination.Config["url"] != "https://parent.example.com" {
		t.Errorf("expected config url from parent, got '%v'", m.Destination.Config)
	}
}

func TestMergeDestination_URLAndAuth(t *testing.T) {
	parent := &DestinationConfig{
		Name: "dest",
		URL:  "https://parent.example.com",
	}
	child := &DestinationConfig{
		URL:      "https://child.example.com",
		AuthType: "API_KEY",
		Auth:     map[string]interface{}{"key": "Authorization"},
	}
	merged := mergeDestination(parent, child)
	if merged.Name != "dest" {
		t.Errorf("expected name dest, got %s", merged.Name)
	}
	if merged.URL != "https://child.example.com" {
		t.Errorf("expected child URL, got %s", merged.URL)
	}
	if merged.AuthType != "API_KEY" {
		t.Errorf("expected API_KEY, got %s", merged.AuthType)
	}
	if merged.Auth == nil || merged.Auth["key"] != "Authorization" {
		t.Errorf("expected auth key 'Authorization', got %v", merged.Auth)
	}
}

func TestMergeDestination_RateLimitFields(t *testing.T) {
	parent := &DestinationConfig{
		Name:            "dest",
		URL:             "https://example.com",
		RateLimit:       100,
		RateLimitPeriod: "second",
	}
	child := &DestinationConfig{
		RateLimit:       200,
		RateLimitPeriod: "minute",
	}
	merged := mergeDestination(parent, child)
	if merged.Name != "dest" {
		t.Errorf("expected name dest, got %s", merged.Name)
	}
	if merged.URL != "https://example.com" {
		t.Errorf("expected parent URL preserved, got %s", merged.URL)
	}
	if merged.RateLimit != 200 {
		t.Errorf("expected rate_limit 200, got %d", merged.RateLimit)
	}
	if merged.RateLimitPeriod != "minute" {
		t.Errorf("expected rate_limit_period 'minute', got %s", merged.RateLimitPeriod)
	}
}

func TestMergeDestination_AuthDeepMerge(t *testing.T) {
	parent := &DestinationConfig{
		Name:     "dest",
		AuthType: "BASIC_AUTH",
		Auth:     map[string]interface{}{"username": "admin", "password": "secret"},
	}
	child := &DestinationConfig{
		Auth: map[string]interface{}{"password": "new-secret"},
	}
	merged := mergeDestination(parent, child)
	if merged.AuthType != "BASIC_AUTH" {
		t.Errorf("expected parent auth_type preserved, got %s", merged.AuthType)
	}
	if merged.Auth["username"] != "admin" {
		t.Errorf("expected parent username preserved, got %v", merged.Auth["username"])
	}
	if merged.Auth["password"] != "new-secret" {
		t.Errorf("expected child password override, got %v", merged.Auth["password"])
	}
}

func TestMergeDestination_NilParent(t *testing.T) {
	child := &DestinationConfig{
		Name: "child-dest",
		URL:  "https://child.example.com",
	}
	merged := mergeDestination(nil, child)
	if merged.Name != "child-dest" {
		t.Errorf("expected child name, got %s", merged.Name)
	}
	if merged.URL != "https://child.example.com" {
		t.Errorf("expected child URL, got %s", merged.URL)
	}
}

func TestMergeDestination_NilChild(t *testing.T) {
	parent := &DestinationConfig{
		Name: "parent-dest",
		URL:  "https://parent.example.com",
	}
	merged := mergeDestination(parent, nil)
	if merged.Name != "parent-dest" {
		t.Errorf("expected parent name, got %s", merged.Name)
	}
	if merged.URL != "https://parent.example.com" {
		t.Errorf("expected parent URL, got %s", merged.URL)
	}
}

func TestMergeConnection_FilterAndTransformations(t *testing.T) {
	parent := &ConnectionConfig{
		Name:   "conn",
		Source: "src",
	}
	child := &ConnectionConfig{
		Filter:          map[string]interface{}{"data": "value"},
		Transformations: []string{"transform-1"},
	}
	merged := mergeConnection(parent, child)
	if merged.Name != "conn" {
		t.Errorf("expected name 'conn', got %s", merged.Name)
	}
	if merged.Source != "src" {
		t.Errorf("expected source 'src', got %s", merged.Source)
	}
	if merged.Filter == nil {
		t.Error("expected Filter to be non-nil")
	}
	if merged.Filter["data"] != "value" {
		t.Errorf("expected filter data 'value', got %v", merged.Filter["data"])
	}
	if len(merged.Transformations) != 1 {
		t.Errorf("expected 1 transformation, got %d", len(merged.Transformations))
	}
	if merged.Transformations[0] != "transform-1" {
		t.Errorf("expected transformation 'transform-1', got %s", merged.Transformations[0])
	}
}

func TestMergeConnection_ChildOverridesTransformations(t *testing.T) {
	parent := &ConnectionConfig{
		Name:            "conn",
		Source:          "src",
		Destination:     "dest",
		Transformations: []string{"parent-transform"},
	}
	child := &ConnectionConfig{
		Transformations: []string{"child-transform-1", "child-transform-2"},
	}
	merged := mergeConnection(parent, child)
	if merged.Name != "conn" {
		t.Errorf("expected name 'conn', got %s", merged.Name)
	}
	if len(merged.Transformations) != 2 {
		t.Errorf("expected 2 transformations, got %d", len(merged.Transformations))
	}
	if merged.Transformations[0] != "child-transform-1" {
		t.Errorf("expected 'child-transform-1', got %s", merged.Transformations[0])
	}
}

func TestMergeConnection_ChildOverridesFilter(t *testing.T) {
	parent := &ConnectionConfig{
		Name:   "conn",
		Filter: map[string]interface{}{"type": "order"},
	}
	child := &ConnectionConfig{
		Filter: map[string]interface{}{"type": "product", "status": "active"},
	}
	merged := mergeConnection(parent, child)
	// Child filter replaces parent filter entirely (not deep-merged)
	if merged.Filter["type"] != "product" {
		t.Errorf("expected filter type 'product', got %v", merged.Filter["type"])
	}
	if merged.Filter["status"] != "active" {
		t.Errorf("expected filter status 'active', got %v", merged.Filter["status"])
	}
}

func TestMergeConnection_RulesReplacedNotMerged(t *testing.T) {
	parent := &ConnectionConfig{
		Name:  "conn",
		Rules: []map[string]interface{}{{"type": "retry", "count": float64(3)}},
	}
	child := &ConnectionConfig{
		Rules: []map[string]interface{}{{"type": "filter", "body": "value"}},
	}
	merged := mergeConnection(parent, child)
	if len(merged.Rules) != 1 {
		t.Errorf("expected 1 rule from child, got %d", len(merged.Rules))
	}
	if merged.Rules[0]["type"] != "filter" {
		t.Errorf("expected child rule type 'filter', got %v", merged.Rules[0]["type"])
	}
}
