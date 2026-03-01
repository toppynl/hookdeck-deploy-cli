package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// writeFile is a helper that creates a file with the given content inside dir,
// creating any intermediate directories as needed.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// ---------------------------------------------------------------------------
// LoadProjectConfig tests
// ---------------------------------------------------------------------------

func TestLoadProjectConfig_Basic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.project.jsonc", `{
		// project comment
		"version": "2025-01-01",
		"env": {
			"production": {"profile": "prod"},
			"staging": {"profile": "stg"}
		}
	}`)

	cfg, err := LoadProjectConfig(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}
	if cfg.Version != "2025-01-01" {
		t.Errorf("expected version '2025-01-01', got %q", cfg.Version)
	}
	if len(cfg.Env) != 2 {
		t.Fatalf("expected 2 envs, got %d", len(cfg.Env))
	}
	if cfg.Env["production"].Profile != "prod" {
		t.Errorf("expected profile 'prod', got %q", cfg.Env["production"].Profile)
	}
	if cfg.Env["staging"].Profile != "stg" {
		t.Errorf("expected profile 'stg', got %q", cfg.Env["staging"].Profile)
	}
}

func TestLoadProjectConfig_FileNotFound(t *testing.T) {
	_, err := LoadProjectConfig("/nonexistent/hookdeck.project.jsonc")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadProjectConfig_Minimal(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.project.jsonc", `{"version": "1.0"}`)

	cfg, err := LoadProjectConfig(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}
	if cfg.Version != "1.0" {
		t.Errorf("expected version '1.0', got %q", cfg.Version)
	}
	if cfg.Env != nil && len(cfg.Env) != 0 {
		t.Errorf("expected nil or empty env map, got %d entries", len(cfg.Env))
	}
}

// ---------------------------------------------------------------------------
// DiscoverManifests tests
// ---------------------------------------------------------------------------

func TestDiscoverManifests_Recursive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.jsonc", `{}`)
	writeFile(t, dir, "subdir/hookdeck.jsonc", `{}`)
	writeFile(t, dir, "subdir/deep/hookdeck.jsonc", `{}`)

	paths, err := DiscoverManifests(dir)
	if err != nil {
		t.Fatalf("DiscoverManifests failed: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 manifests, got %d: %v", len(paths), paths)
	}
}

func TestDiscoverManifests_BothExtensions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a/hookdeck.jsonc", `{}`)
	writeFile(t, dir, "b/hookdeck.json", `{}`)

	paths, err := DiscoverManifests(dir)
	if err != nil {
		t.Fatalf("DiscoverManifests failed: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 manifests, got %d: %v", len(paths), paths)
	}

	hasJsonc := false
	hasJson := false
	for _, p := range paths {
		if strings.HasSuffix(p, "hookdeck.jsonc") {
			hasJsonc = true
		}
		if strings.HasSuffix(p, "hookdeck.json") {
			hasJson = true
		}
	}
	if !hasJsonc || !hasJson {
		t.Errorf("expected both extensions, found jsonc=%v json=%v", hasJsonc, hasJson)
	}
}

func TestDiscoverManifests_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	paths, err := DiscoverManifests(dir)
	if err != nil {
		t.Fatalf("DiscoverManifests failed: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 manifests in empty dir, got %d", len(paths))
	}
}

func TestDiscoverManifests_IgnoresOtherFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.jsonc", `{}`)
	writeFile(t, dir, "other.json", `{}`)
	writeFile(t, dir, "hookdeck.yaml", `{}`)

	paths, err := DiscoverManifests(dir)
	if err != nil {
		t.Fatalf("DiscoverManifests failed: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 manifest, got %d: %v", len(paths), paths)
	}
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestRegistry_ValidRefs(t *testing.T) {
	r := NewRegistry()
	r.AddManifest("file1.jsonc", &manifest.Manifest{
		Sources:      []manifest.SourceConfig{{Name: "src-a"}},
		Destinations: []manifest.DestinationConfig{{Name: "dst-a", URL: "https://example.com"}},
		Transformations: []manifest.TransformationConfig{{Name: "tr-a", CodeFile: "handler.js"}},
		Connections: []manifest.ConnectionConfig{{
			Name:            "conn-a",
			Source:          "src-a",
			Destination:     "dst-a",
			Transformations: []string{"tr-a"},
		}},
	})

	errs := r.Validate()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if len(r.SourceList) != 1 {
		t.Errorf("expected 1 source in list, got %d", len(r.SourceList))
	}
	if len(r.DestinationList) != 1 {
		t.Errorf("expected 1 destination in list, got %d", len(r.DestinationList))
	}
	if len(r.TransformationList) != 1 {
		t.Errorf("expected 1 transformation in list, got %d", len(r.TransformationList))
	}
	if len(r.ConnectionList) != 1 {
		t.Errorf("expected 1 connection in list, got %d", len(r.ConnectionList))
	}
}

func TestRegistry_NamingCollision(t *testing.T) {
	r := NewRegistry()
	r.AddManifest("file1.jsonc", &manifest.Manifest{
		Sources: []manifest.SourceConfig{{Name: "src-a"}},
	})
	r.AddManifest("file2.jsonc", &manifest.Manifest{
		Sources: []manifest.SourceConfig{{Name: "src-a"}},
	})

	errs := r.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected 1 collision error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "duplicate source") {
		t.Errorf("expected 'duplicate source' error, got %q", errs[0].Error())
	}
	if !strings.Contains(errs[0].Error(), "src-a") {
		t.Errorf("expected error to mention 'src-a', got %q", errs[0].Error())
	}
}

func TestRegistry_CrossTypeAllowed(t *testing.T) {
	// Same name across different resource types is allowed.
	r := NewRegistry()
	r.AddManifest("file1.jsonc", &manifest.Manifest{
		Sources:      []manifest.SourceConfig{{Name: "shared-name"}},
		Destinations: []manifest.DestinationConfig{{Name: "shared-name", URL: "https://example.com"}},
	})

	errs := r.Validate()
	if len(errs) != 0 {
		t.Errorf("cross-type same name should be allowed, got %v", errs)
	}
}

func TestRegistry_BrokenSourceRef(t *testing.T) {
	r := NewRegistry()
	r.AddManifest("file1.jsonc", &manifest.Manifest{
		Destinations: []manifest.DestinationConfig{{Name: "dst-a", URL: "https://example.com"}},
		Connections: []manifest.ConnectionConfig{{
			Name:        "conn-a",
			Source:      "missing-source",
			Destination: "dst-a",
		}},
	})

	errs := r.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "undefined source") {
		t.Errorf("expected 'undefined source' error, got %q", errs[0].Error())
	}
}

func TestRegistry_BrokenDestinationRef(t *testing.T) {
	r := NewRegistry()
	r.AddManifest("file1.jsonc", &manifest.Manifest{
		Sources: []manifest.SourceConfig{{Name: "src-a"}},
		Connections: []manifest.ConnectionConfig{{
			Name:        "conn-a",
			Source:      "src-a",
			Destination: "missing-dest",
		}},
	})

	errs := r.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "undefined destination") {
		t.Errorf("expected 'undefined destination' error, got %q", errs[0].Error())
	}
}

func TestRegistry_BrokenTransformationRef(t *testing.T) {
	r := NewRegistry()
	r.AddManifest("file1.jsonc", &manifest.Manifest{
		Sources:      []manifest.SourceConfig{{Name: "src-a"}},
		Destinations: []manifest.DestinationConfig{{Name: "dst-a", URL: "https://example.com"}},
		Connections: []manifest.ConnectionConfig{{
			Name:            "conn-a",
			Source:          "src-a",
			Destination:     "dst-a",
			Transformations: []string{"missing-tr"},
		}},
	})

	errs := r.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "undefined transformation") {
		t.Errorf("expected 'undefined transformation' error, got %q", errs[0].Error())
	}
}

func TestRegistry_MultiFileRefs(t *testing.T) {
	r := NewRegistry()
	r.AddManifest("services/a/hookdeck.jsonc", &manifest.Manifest{
		Sources:      []manifest.SourceConfig{{Name: "src-a"}},
		Destinations: []manifest.DestinationConfig{{Name: "dst-a", URL: "https://a.example.com"}},
	})
	r.AddManifest("services/b/hookdeck.jsonc", &manifest.Manifest{
		Sources: []manifest.SourceConfig{{Name: "src-b"}},
		Transformations: []manifest.TransformationConfig{{Name: "tr-b", CodeFile: "transform.js"}},
		Connections: []manifest.ConnectionConfig{
			{
				Name:            "conn-ab",
				Source:          "src-a", // from file a
				Destination:     "dst-a", // from file a
				Transformations: []string{"tr-b"}, // from file b
			},
		},
	})

	errs := r.Validate()
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid cross-file refs, got %v", errs)
	}
	if len(r.SourceList) != 2 {
		t.Errorf("expected 2 sources total, got %d", len(r.SourceList))
	}
	if len(r.TransformationFiles) != 1 {
		t.Errorf("expected 1 transformation file entry, got %d", len(r.TransformationFiles))
	}
	// TransformationFiles should resolve relative to the manifest dir.
	expected := filepath.Join("services", "b", "transform.js")
	if r.TransformationFiles["tr-b"] != expected {
		t.Errorf("expected transformation file %q, got %q", expected, r.TransformationFiles["tr-b"])
	}
}

func TestRegistry_CollisionAllTypes(t *testing.T) {
	r := NewRegistry()
	r.AddManifest("a.jsonc", &manifest.Manifest{
		Sources:         []manifest.SourceConfig{{Name: "dup"}},
		Destinations:    []manifest.DestinationConfig{{Name: "dup", URL: "https://a.example.com"}},
		Transformations: []manifest.TransformationConfig{{Name: "dup"}},
		Connections:     []manifest.ConnectionConfig{{Name: "dup", Source: "dup", Destination: "dup"}},
	})
	r.AddManifest("b.jsonc", &manifest.Manifest{
		Sources:         []manifest.SourceConfig{{Name: "dup"}},
		Destinations:    []manifest.DestinationConfig{{Name: "dup", URL: "https://b.example.com"}},
		Transformations: []manifest.TransformationConfig{{Name: "dup"}},
		Connections:     []manifest.ConnectionConfig{{Name: "dup", Source: "dup", Destination: "dup"}},
	})

	errs := r.Validate()
	// 4 collision errors (one per type)
	collisionCount := 0
	for _, e := range errs {
		if strings.Contains(e.Error(), "duplicate") {
			collisionCount++
		}
	}
	if collisionCount != 4 {
		t.Errorf("expected 4 collision errors, got %d (total errors: %v)", collisionCount, errs)
	}
}

// ---------------------------------------------------------------------------
// LoadProject tests
// ---------------------------------------------------------------------------

func TestLoadProject_FullFlow(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.project.jsonc", `{
		"version": "2025-01-01",
		"env": {
			"production": {"profile": "prod"}
		}
	}`)
	writeFile(t, dir, "hookdeck.jsonc", `{
		"sources": [{"name": "src-a"}],
		"destinations": [{"name": "dst-a", "url": "https://example.com"}],
		"connections": [{"name": "conn-a", "source": "src-a", "destination": "dst-a"}]
	}`)

	proj, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}
	if proj.Config.Version != "2025-01-01" {
		t.Errorf("expected version '2025-01-01', got %q", proj.Config.Version)
	}
	if proj.RootDir != dir {
		t.Errorf("expected RootDir %q, got %q", dir, proj.RootDir)
	}
	if len(proj.Registry.SourceList) != 1 {
		t.Errorf("expected 1 source, got %d", len(proj.Registry.SourceList))
	}
	if len(proj.Registry.ConnectionList) != 1 {
		t.Errorf("expected 1 connection, got %d", len(proj.Registry.ConnectionList))
	}
}

func TestLoadProject_MultipleManifests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.project.jsonc", `{"version": "1.0"}`)
	writeFile(t, dir, "svc-a/hookdeck.jsonc", `{
		"sources": [{"name": "src-a"}],
		"destinations": [{"name": "dst-a", "url": "https://a.example.com"}]
	}`)
	writeFile(t, dir, "svc-b/hookdeck.jsonc", `{
		"sources": [{"name": "src-b"}],
		"connections": [{"name": "conn-ab", "source": "src-a", "destination": "dst-a"}]
	}`)

	proj, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}
	if len(proj.Registry.SourceList) != 2 {
		t.Errorf("expected 2 sources, got %d", len(proj.Registry.SourceList))
	}
	if len(proj.Registry.ConnectionList) != 1 {
		t.Errorf("expected 1 connection, got %d", len(proj.Registry.ConnectionList))
	}
}

func TestLoadProject_ValidationErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.project.jsonc", `{"version": "1.0"}`)
	writeFile(t, dir, "hookdeck.jsonc", `{
		"connections": [{"name": "conn-a", "source": "missing-src", "destination": "missing-dst"}]
	}`)

	_, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "undefined source") {
		t.Errorf("expected error about undefined source, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "undefined destination") {
		t.Errorf("expected error about undefined destination, got %q", err.Error())
	}
}

func TestLoadProject_NoManifests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.project.jsonc", `{"version": "1.0"}`)

	proj, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProject should succeed with no manifests, got: %v", err)
	}
	if len(proj.Registry.SourceList) != 0 {
		t.Errorf("expected 0 sources, got %d", len(proj.Registry.SourceList))
	}
}

func TestLoadProject_MissingProjectConfig(t *testing.T) {
	_, err := LoadProject("/nonexistent/hookdeck.project.jsonc")
	if err == nil {
		t.Fatal("expected error for missing project config")
	}
}

func TestLoadProject_CollisionAcrossManifests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hookdeck.project.jsonc", `{"version": "1.0"}`)
	writeFile(t, dir, "a/hookdeck.jsonc", `{
		"sources": [{"name": "shared-src"}]
	}`)
	writeFile(t, dir, "b/hookdeck.jsonc", `{
		"sources": [{"name": "shared-src"}]
	}`)

	_, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "duplicate source") {
		t.Errorf("expected 'duplicate source' error, got %q", err.Error())
	}
}
