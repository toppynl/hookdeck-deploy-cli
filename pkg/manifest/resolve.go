package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ResolveSourceEnv applies environment-specific overrides to a source.
func ResolveSourceEnv(src *SourceConfig, envName string) *SourceConfig {
	result := &SourceConfig{
		Name:        src.Name,
		Type:        src.Type,
		Description: src.Description,
		Config:      src.Config,
	}
	if envName == "" || src.Env == nil {
		return result
	}
	override, ok := src.Env[envName]
	if !ok {
		return result
	}
	if override.Type != "" {
		result.Type = override.Type
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.Config != nil {
		result.Config = override.Config
	}
	return result
}

// ResolveDestinationEnv applies environment-specific overrides to a destination.
func ResolveDestinationEnv(dst *DestinationConfig, envName string) *DestinationConfig {
	result := &DestinationConfig{
		Name:            dst.Name,
		URL:             dst.URL,
		Type:            dst.Type,
		Description:     dst.Description,
		AuthType:        dst.AuthType,
		Auth:            dst.Auth,
		Config:          dst.Config,
		RateLimit:       dst.RateLimit,
		RateLimitPeriod: dst.RateLimitPeriod,
	}
	if envName == "" || dst.Env == nil {
		return result
	}
	override, ok := dst.Env[envName]
	if !ok {
		return result
	}
	if override.URL != "" {
		result.URL = override.URL
	}
	if override.Type != "" {
		result.Type = override.Type
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.AuthType != "" {
		result.AuthType = override.AuthType
	}
	if override.Auth != nil {
		result.Auth = override.Auth
	}
	if override.Config != nil {
		result.Config = override.Config
	}
	if override.RateLimit != 0 {
		result.RateLimit = override.RateLimit
	}
	if override.RateLimitPeriod != "" {
		result.RateLimitPeriod = override.RateLimitPeriod
	}
	return result
}

// ResolveTransformationEnv applies environment-specific overrides to a transformation.
func ResolveTransformationEnv(tr *TransformationConfig, envName string) *TransformationConfig {
	result := &TransformationConfig{
		Name:        tr.Name,
		Description: tr.Description,
		CodeFile:    tr.CodeFile,
	}
	if tr.Env != nil {
		result.Env = make(map[string]string)
		for k, v := range tr.Env {
			result.Env[k] = v
		}
	}
	if envName == "" || tr.EnvOverrides == nil {
		return result
	}
	override, ok := tr.EnvOverrides[envName]
	if !ok {
		return result
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.CodeFile != "" {
		result.CodeFile = override.CodeFile
	}
	if override.Env != nil {
		if result.Env == nil {
			result.Env = make(map[string]string)
		}
		for k, v := range override.Env {
			result.Env[k] = v
		}
	}
	return result
}

// InterpolateEnvVars replaces ${ENV_VAR} patterns in all string fields of a Manifest.
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
		return escaped[1 : len(escaped)-1]
	})

	if len(missing) > 0 {
		return fmt.Errorf("undefined environment variables: %v", missing)
	}

	return json.Unmarshal(result, m)
}
