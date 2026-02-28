package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tailscale/hujson"
)

// FindFile locates a manifest file in the given directory.
// Prefers hookdeck.jsonc, falls back to hookdeck.json.
func FindFile(dir string) (string, error) {
	for _, name := range []string{"hookdeck.jsonc", "hookdeck.json"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no hookdeck.jsonc or hookdeck.json found in %s", dir)
}

// LoadFile reads and parses a JSONC manifest file.
func LoadFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	standardized, err := hujson.Standardize(data)
	if err != nil {
		return nil, fmt.Errorf("parsing JSONC: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(standardized, &m); err != nil {
		return nil, fmt.Errorf("unmarshaling manifest: %w", err)
	}

	// Backward compat: merge env_vars into env
	mergeTransformationEnvVars(m.Transformation)
	for _, envOverride := range m.Env {
		if envOverride != nil {
			mergeTransformationEnvVars(envOverride.Transformation)
		}
	}

	return &m, nil
}

// mergeTransformationEnvVars merges the deprecated env_vars field into env.
// Values in env take priority over env_vars when both define the same key.
func mergeTransformationEnvVars(t *TransformationConfig) {
	if t == nil || len(t.EnvVars) == 0 {
		return
	}
	if t.Env == nil {
		t.Env = make(map[string]string)
	}
	for k, v := range t.EnvVars {
		if _, exists := t.Env[k]; !exists {
			t.Env[k] = v
		}
	}
	t.EnvVars = nil // clear after merge
}

// LoadWithInheritance loads a manifest and recursively resolves the extends chain.
func LoadWithInheritance(path string) (*Manifest, error) {
	return loadWithInheritance(path, make(map[string]bool))
}

func loadWithInheritance(path string, seen map[string]bool) (*Manifest, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if seen[absPath] {
		return nil, fmt.Errorf("circular extends detected: %s", absPath)
	}
	seen[absPath] = true

	m, err := LoadFile(path)
	if err != nil {
		return nil, err
	}

	if m.Extends == "" {
		return m, nil
	}

	parentPath := filepath.Join(filepath.Dir(absPath), m.Extends)
	parent, err := loadWithInheritance(parentPath, seen)
	if err != nil {
		return nil, fmt.Errorf("loading parent %s: %w", m.Extends, err)
	}

	return mergeManifests(parent, m), nil
}
