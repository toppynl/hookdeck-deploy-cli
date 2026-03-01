package project

import (
	"fmt"
	"path/filepath"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// fileRef records the file where a named resource was defined.
type fileRef struct {
	FilePath string
}

// Registry accumulates resources from multiple manifest files and detects
// naming collisions and broken references.
type Registry struct {
	Sources      map[string]fileRef
	Destinations map[string]fileRef
	Transformations map[string]fileRef
	Connections  map[string]fileRef

	SourceList         []manifest.SourceConfig
	DestinationList    []manifest.DestinationConfig
	TransformationList []manifest.TransformationConfig
	ConnectionList     []manifest.ConnectionConfig

	// TransformationFiles maps transformation name to the resolved code_file path.
	TransformationFiles map[string]string

	collisionErrors []error
}

// NewRegistry creates an empty Registry ready to receive manifests.
func NewRegistry() *Registry {
	return &Registry{
		Sources:             make(map[string]fileRef),
		Destinations:        make(map[string]fileRef),
		Transformations:     make(map[string]fileRef),
		Connections:         make(map[string]fileRef),
		TransformationFiles: make(map[string]string),
	}
}

// AddManifest registers all resources from a manifest loaded from filePath,
// detecting naming collisions within each resource type.
func (r *Registry) AddManifest(filePath string, m *manifest.Manifest) {
	manifestDir := filepath.Dir(filePath)

	for _, s := range m.Sources {
		if existing, ok := r.Sources[s.Name]; ok {
			r.collisionErrors = append(r.collisionErrors,
				fmt.Errorf("duplicate source %q: defined in %s and %s", s.Name, existing.FilePath, filePath))
		} else {
			r.Sources[s.Name] = fileRef{FilePath: filePath}
		}
		r.SourceList = append(r.SourceList, s)
	}

	for _, d := range m.Destinations {
		if existing, ok := r.Destinations[d.Name]; ok {
			r.collisionErrors = append(r.collisionErrors,
				fmt.Errorf("duplicate destination %q: defined in %s and %s", d.Name, existing.FilePath, filePath))
		} else {
			r.Destinations[d.Name] = fileRef{FilePath: filePath}
		}
		r.DestinationList = append(r.DestinationList, d)
	}

	for _, tr := range m.Transformations {
		if existing, ok := r.Transformations[tr.Name]; ok {
			r.collisionErrors = append(r.collisionErrors,
				fmt.Errorf("duplicate transformation %q: defined in %s and %s", tr.Name, existing.FilePath, filePath))
		} else {
			r.Transformations[tr.Name] = fileRef{FilePath: filePath}
		}
		r.TransformationList = append(r.TransformationList, tr)
		if tr.CodeFile != "" {
			r.TransformationFiles[tr.Name] = filepath.Join(manifestDir, tr.CodeFile)
		}
	}

	for _, c := range m.Connections {
		if existing, ok := r.Connections[c.Name]; ok {
			r.collisionErrors = append(r.collisionErrors,
				fmt.Errorf("duplicate connection %q: defined in %s and %s", c.Name, existing.FilePath, filePath))
		} else {
			r.Connections[c.Name] = fileRef{FilePath: filePath}
		}
		r.ConnectionList = append(r.ConnectionList, c)
	}
}

// Validate returns all accumulated collision errors plus any broken references
// from connections to sources, destinations, or transformations.
func (r *Registry) Validate() []error {
	var errs []error
	errs = append(errs, r.collisionErrors...)

	for _, c := range r.ConnectionList {
		if c.Source != "" {
			if _, ok := r.Sources[c.Source]; !ok {
				errs = append(errs, fmt.Errorf("connection %q references undefined source %q", c.Name, c.Source))
			}
		}
		if c.Destination != "" {
			if _, ok := r.Destinations[c.Destination]; !ok {
				errs = append(errs, fmt.Errorf("connection %q references undefined destination %q", c.Name, c.Destination))
			}
		}
		for _, trName := range c.Transformations {
			if _, ok := r.Transformations[trName]; !ok {
				errs = append(errs, fmt.Errorf("connection %q references undefined transformation %q", c.Name, trName))
			}
		}
	}

	return errs
}
