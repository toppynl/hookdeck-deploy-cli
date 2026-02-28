package wrangler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncSourceURL_WritesURL(t *testing.T) {
	dir := t.TempDir()
	wranglerPath := filepath.Join(dir, "wrangler.jsonc")
	os.WriteFile(wranglerPath, []byte(`{
	"name": "my-worker",
	"env": {
		"staging": {
			"vars": {
				"LOG_LEVEL": "debug"
			}
		}
	}
}`), 0644)

	updated, err := SyncSourceURL(wranglerPath, "staging", "https://hkdk.events/abc123")
	if err != nil {
		t.Fatalf("SyncSourceURL failed: %v", err)
	}
	if !updated {
		t.Error("expected updated=true")
	}

	data, _ := os.ReadFile(wranglerPath)
	content := string(data)
	if !strings.Contains(content, "HOOKDECK_SOURCE_URL") {
		t.Error("expected HOOKDECK_SOURCE_URL in output")
	}
	if !strings.Contains(content, "https://hkdk.events/abc123") {
		t.Error("expected URL in output")
	}
}

func TestSyncSourceURL_NoOpWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	wranglerPath := filepath.Join(dir, "wrangler.jsonc")
	os.WriteFile(wranglerPath, []byte(`{
	"env": {
		"staging": {
			"vars": {
				"HOOKDECK_SOURCE_URL": "https://hkdk.events/abc123"
			}
		}
	}
}`), 0644)

	updated, err := SyncSourceURL(wranglerPath, "staging", "https://hkdk.events/abc123")
	if err != nil {
		t.Fatalf("SyncSourceURL failed: %v", err)
	}
	if updated {
		t.Error("expected updated=false when URL unchanged")
	}
}

func TestSyncSourceURL_CreatesNestedStructure(t *testing.T) {
	dir := t.TempDir()
	wranglerPath := filepath.Join(dir, "wrangler.jsonc")
	os.WriteFile(wranglerPath, []byte(`{
	"name": "my-worker"
}`), 0644)

	updated, err := SyncSourceURL(wranglerPath, "staging", "https://hkdk.events/new123")
	if err != nil {
		t.Fatalf("SyncSourceURL failed: %v", err)
	}
	if !updated {
		t.Error("expected updated=true")
	}

	data, _ := os.ReadFile(wranglerPath)
	content := string(data)
	if !strings.Contains(content, "HOOKDECK_SOURCE_URL") {
		t.Error("expected HOOKDECK_SOURCE_URL in output")
	}
}
