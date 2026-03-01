package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tailscale/hujson"
)

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

	return &m, nil
}
