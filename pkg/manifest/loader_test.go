package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile_BasicSources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hookdeck.jsonc")
	content := `{
		// A comment to test JSONC support
		"sources": [
			{"name": "src-a", "type": "Stripe"},
			{"name": "src-b"}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if len(m.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(m.Sources))
	}
	if m.Sources[0].Name != "src-a" {
		t.Errorf("expected name 'src-a', got '%s'", m.Sources[0].Name)
	}
}

func TestLoadFile_AllResourceTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hookdeck.jsonc")
	content := `{
		"sources": [{"name": "s1"}],
		"destinations": [{"name": "d1", "url": "https://example.com"}],
		"transformations": [{"name": "t1", "code_file": "handler.js"}],
		"connections": [{"name": "c1", "source": "s1", "destination": "d1"}]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if len(m.Sources) != 1 || len(m.Destinations) != 1 || len(m.Transformations) != 1 || len(m.Connections) != 1 {
		t.Error("expected 1 of each resource type")
	}
}

func TestLoadFile_FileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/hookdeck.jsonc")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hookdeck.jsonc")
	if err := os.WriteFile(path, []byte("{invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
