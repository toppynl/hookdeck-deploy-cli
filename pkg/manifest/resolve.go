package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// ResolveEnv applies the environment overlay to the base manifest.
// If envName is empty, returns the base manifest unchanged (minus env map).
func ResolveEnv(m *Manifest, envName string) (*Manifest, error) {
	if envName == "" {
		result := *m
		result.Env = nil
		return &result, nil
	}

	if m.Env == nil {
		return nil, fmt.Errorf("environment '%s' not found (no env block defined)", envName)
	}

	overlay, ok := m.Env[envName]
	if !ok {
		return nil, fmt.Errorf("environment '%s' not found in env block", envName)
	}

	result := *m
	if overlay.Profile != "" {
		result.Profile = overlay.Profile
	}
	result.Source = mergeSource(m.Source, overlay.Source)
	result.Destination = mergeDestination(m.Destination, overlay.Destination)
	result.Connection = mergeConnection(m.Connection, overlay.Connection)
	result.Transformation = mergeTransformation(m.Transformation, overlay.Transformation)
	result.Env = nil

	return &result, nil
}

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// InterpolateEnvVars replaces ${ENV_VAR} patterns in all string fields.
// Errors if a referenced variable is not set.
func InterpolateEnvVars(m *Manifest) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	var missing []string
	result := envVarPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		varName := envVarPattern.FindSubmatch(match)[1]
		val, ok := os.LookupEnv(string(varName))
		if !ok {
			missing = append(missing, string(varName))
			return match
		}
		escaped, _ := json.Marshal(val)
		// Strip the surrounding quotes from the JSON-encoded string
		return escaped[1 : len(escaped)-1]
	})

	if len(missing) > 0 {
		return fmt.Errorf("undefined environment variables: %v", missing)
	}

	return json.Unmarshal(result, m)
}
