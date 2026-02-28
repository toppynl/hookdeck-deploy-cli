package manifest

// mergeManifests deep-merges parent into child. Child values override parent values.
// The Extends field is cleared from the result.
func mergeManifests(parent, child *Manifest) *Manifest {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	result := &Manifest{
		Schema:         parent.Schema,
		Version:        parent.Version,
		Profile:        parent.Profile,
		Source:         parent.Source,
		Destination:    parent.Destination,
		Connection:     parent.Connection,
		Transformation: parent.Transformation,
		Env:            parent.Env,
	}

	// Override with non-zero child fields
	if child.Schema != "" {
		result.Schema = child.Schema
	}
	if child.Version != "" {
		result.Version = child.Version
	}
	if child.Profile != "" {
		result.Profile = child.Profile
	}

	result.Source = mergeSource(parent.Source, child.Source)
	result.Destination = mergeDestination(parent.Destination, child.Destination)
	result.Connection = mergeConnection(parent.Connection, child.Connection)
	result.Transformation = mergeTransformation(parent.Transformation, child.Transformation)
	result.Env = mergeEnvMap(parent.Env, child.Env)

	// Clear Extends from merged result
	result.Extends = ""

	return result
}

// mergeSource deep-merges two SourceConfig values. Child wins for non-zero fields.
// Config maps are shallow-merged (child keys override parent keys).
func mergeSource(parent, child *SourceConfig) *SourceConfig {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}

	result := &SourceConfig{
		Name:        parent.Name,
		Type:        parent.Type,
		Description: parent.Description,
		Config:      parent.Config,
	}

	if child.Name != "" {
		result.Name = child.Name
	}
	if child.Type != "" {
		result.Type = child.Type
	}
	if child.Description != "" {
		result.Description = child.Description
	}
	if child.Config != nil {
		result.Config = mergeMapStringInterface(parent.Config, child.Config)
	}

	return result
}

// mergeDestination deep-merges two DestinationConfig values. Child wins for non-zero fields.
// Config maps are shallow-merged (child keys override parent keys).
func mergeDestination(parent, child *DestinationConfig) *DestinationConfig {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}

	result := &DestinationConfig{
		Name:            parent.Name,
		URL:             parent.URL,
		Type:            parent.Type,
		Description:     parent.Description,
		AuthType:        parent.AuthType,
		Auth:            parent.Auth,
		Config:          parent.Config,
		RateLimit:       parent.RateLimit,
		RateLimitPeriod: parent.RateLimitPeriod,
	}

	if child.Name != "" {
		result.Name = child.Name
	}
	if child.URL != "" {
		result.URL = child.URL
	}
	if child.Type != "" {
		result.Type = child.Type
	}
	if child.Description != "" {
		result.Description = child.Description
	}
	if child.AuthType != "" {
		result.AuthType = child.AuthType
	}
	if child.Auth != nil {
		result.Auth = mergeMapStringInterface(result.Auth, child.Auth)
	}
	if child.Config != nil {
		result.Config = mergeMapStringInterface(parent.Config, child.Config)
	}
	if child.RateLimit != 0 {
		result.RateLimit = child.RateLimit
	}
	if child.RateLimitPeriod != "" {
		result.RateLimitPeriod = child.RateLimitPeriod
	}

	return result
}

// mergeConnection deep-merges two ConnectionConfig values. Child wins for non-zero fields.
// Rules: if child has Rules, use child's Rules entirely (replace, don't merge arrays).
func mergeConnection(parent, child *ConnectionConfig) *ConnectionConfig {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}

	result := &ConnectionConfig{
		Name:            parent.Name,
		Source:          parent.Source,
		Destination:     parent.Destination,
		Rules:           parent.Rules,
		Filter:          parent.Filter,
		Transformations: parent.Transformations,
	}

	if child.Name != "" {
		result.Name = child.Name
	}
	if child.Source != "" {
		result.Source = child.Source
	}
	if child.Destination != "" {
		result.Destination = child.Destination
	}
	if child.Rules != nil {
		result.Rules = child.Rules
	}
	if child.Filter != nil {
		result.Filter = child.Filter
	}
	if len(child.Transformations) > 0 {
		result.Transformations = child.Transformations
	}

	return result
}

// mergeTransformation deep-merges two TransformationConfig values. Child wins for non-zero fields.
// Env maps are shallow-merged (child keys override parent keys).
func mergeTransformation(parent, child *TransformationConfig) *TransformationConfig {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}

	result := &TransformationConfig{
		Name:        parent.Name,
		Description: parent.Description,
		CodeFile:    parent.CodeFile,
		Env:         parent.Env,
	}

	if child.Name != "" {
		result.Name = child.Name
	}
	if child.Description != "" {
		result.Description = child.Description
	}
	if child.CodeFile != "" {
		result.CodeFile = child.CodeFile
	}
	if child.Env != nil {
		// Merge env: parent provides defaults, child overrides
		merged := make(map[string]string)
		for k, v := range parent.Env {
			merged[k] = v
		}
		for k, v := range child.Env {
			merged[k] = v
		}
		result.Env = merged
	}

	return result
}

// mergeEnvMap deep-merges two env override maps. Child entries override parent entries
// for the same key; parent-only keys are preserved.
func mergeEnvMap(parent, child map[string]*EnvOverride) map[string]*EnvOverride {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}

	result := make(map[string]*EnvOverride)
	for k, v := range parent {
		result[k] = v
	}
	for k, v := range child {
		if existing, ok := result[k]; ok {
			result[k] = mergeEnvOverride(existing, v)
		} else {
			result[k] = v
		}
	}

	return result
}

// mergeEnvOverride deep-merges two EnvOverride values. Child wins for non-zero fields.
func mergeEnvOverride(parent, child *EnvOverride) *EnvOverride {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}

	result := &EnvOverride{
		Profile:        parent.Profile,
		Source:         parent.Source,
		Destination:    parent.Destination,
		Connection:     parent.Connection,
		Transformation: parent.Transformation,
	}

	if child.Profile != "" {
		result.Profile = child.Profile
	}

	result.Source = mergeSource(parent.Source, child.Source)
	result.Destination = mergeDestination(parent.Destination, child.Destination)
	result.Connection = mergeConnection(parent.Connection, child.Connection)
	result.Transformation = mergeTransformation(parent.Transformation, child.Transformation)

	return result
}

// mergeMapStringInterface shallow-merges two map[string]interface{} values.
// Child keys override parent keys.
func mergeMapStringInterface(parent, child map[string]interface{}) map[string]interface{} {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	merged := make(map[string]interface{})
	for k, v := range parent {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}
