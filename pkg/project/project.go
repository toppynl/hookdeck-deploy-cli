package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// ProjectConfig is the root configuration parsed from hookdeck.project.jsonc.
type ProjectConfig struct {
	Version string                `json:"version"`
	Env     map[string]*EnvConfig `json:"env,omitempty"`
}

// EnvConfig holds per-environment settings within a project config.
type EnvConfig struct {
	Profile string `json:"profile,omitempty"`
}

// Project is a fully loaded project including its config, resource registry, and root directory.
type Project struct {
	Config   *ProjectConfig
	Registry *Registry
	RootDir  string
}

// LoadProjectConfig reads and parses a hookdeck.project.jsonc file.
func LoadProjectConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading project config: %w", err)
	}

	standardized, err := hujson.Standardize(data)
	if err != nil {
		return nil, fmt.Errorf("parsing JSONC: %w", err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(standardized, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling project config: %w", err)
	}

	return &cfg, nil
}

// DiscoverManifests recursively walks a directory tree and returns the paths of
// all files named hookdeck.jsonc or hookdeck.json.
func DiscoverManifests(root string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if base == "hookdeck.jsonc" || base == "hookdeck.json" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discovering manifests: %w", err)
	}
	return paths, nil
}

// LoadProject loads the project config from projectPath, discovers all manifests
// in the same directory tree, loads each manifest, registers resources, validates
// references, and returns the fully loaded Project or an error.
func LoadProject(projectPath string) (*Project, error) {
	cfg, err := LoadProjectConfig(projectPath)
	if err != nil {
		return nil, err
	}

	rootDir := filepath.Dir(projectPath)

	manifestPaths, err := DiscoverManifests(rootDir)
	if err != nil {
		return nil, err
	}

	registry := NewRegistry()

	var loadErrors []string
	for _, mp := range manifestPaths {
		m, err := manifest.LoadFile(mp)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", mp, err))
			continue
		}
		registry.AddManifest(mp, m)
	}

	if len(loadErrors) > 0 {
		return nil, fmt.Errorf("failed to load manifests:\n  %s", strings.Join(loadErrors, "\n  "))
	}

	if errs := registry.Validate(); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("validation errors:\n  %s", strings.Join(msgs, "\n  "))
	}

	return &Project{
		Config:   cfg,
		Registry: registry,
		RootDir:  rootDir,
	}, nil
}
