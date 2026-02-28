package manifest

// Manifest is the top-level structure of a hookdeck.jsonc file.
type Manifest struct {
	Schema         string                  `json:"$schema,omitempty"`
	Version        string                  `json:"version,omitempty"`
	Extends        string                  `json:"extends,omitempty"`
	Profile        string                  `json:"profile,omitempty"`
	Source         *SourceConfig           `json:"source,omitempty"`
	Destination    *DestinationConfig      `json:"destination,omitempty"`
	Connection     *ConnectionConfig       `json:"connection,omitempty"`
	Transformation *TransformationConfig   `json:"transformation,omitempty"`
	Env            map[string]*EnvOverride `json:"env,omitempty"`
}

// EnvOverride holds per-environment overrides for manifest fields.
type EnvOverride struct {
	Profile        string               `json:"profile,omitempty"`
	Source         *SourceConfig        `json:"source,omitempty"`
	Destination    *DestinationConfig   `json:"destination,omitempty"`
	Connection     *ConnectionConfig    `json:"connection,omitempty"`
	Transformation *TransformationConfig `json:"transformation,omitempty"`
}

// SourceConfig defines a Hookdeck source (aligned with API schema).
type SourceConfig struct {
	Name        string                 `json:"name,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// DestinationConfig defines a Hookdeck destination (aligned with API schema).
type DestinationConfig struct {
	Name            string                 `json:"name,omitempty"`
	URL             string                 `json:"url,omitempty"`
	Type            string                 `json:"type,omitempty"`
	Description     string                 `json:"description,omitempty"`
	AuthType        string                 `json:"auth_type,omitempty"`
	Auth            map[string]interface{} `json:"auth,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	RateLimit       int                    `json:"rate_limit,omitempty"`
	RateLimitPeriod string                 `json:"rate_limit_period,omitempty"`
}

// ConnectionConfig defines a Hookdeck connection between a source and destination (aligned with API schema).
type ConnectionConfig struct {
	Name        string                   `json:"name,omitempty"`
	Source      string                   `json:"source,omitempty"`
	Destination string                   `json:"destination,omitempty"`
	Rules       []map[string]interface{} `json:"rules,omitempty"`
	// Shorthand fields — converted to rules during deploy
	Filter          map[string]interface{} `json:"filter,omitempty"`
	Transformations []string               `json:"transformations,omitempty"`
}

// TransformationConfig defines a Hookdeck transformation.
type TransformationConfig struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	CodeFile    string            `json:"code_file,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"` // backward compat alias for Env
}
