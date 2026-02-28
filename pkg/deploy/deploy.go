// Package deploy orchestrates upserts of Hookdeck resources (sources,
// transformations, destinations, connections) from a resolved manifest.
package deploy

import (
	"context"
	"fmt"
	"os"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// ---------------------------------------------------------------------------
// API client interface
// ---------------------------------------------------------------------------
// We define a thin interface so the deploy logic is decoupled from any
// concrete HTTP client. An adapter around the official hookdeck-cli Client
// (or a lightweight REST wrapper) can satisfy this.

// Client is the interface the deploy orchestrator needs.
type Client interface {
	UpsertSource(ctx context.Context, req *UpsertSourceRequest) (*UpsertSourceResult, error)
	UpsertDestination(ctx context.Context, req *UpsertDestinationRequest) (*UpsertDestinationResult, error)
	UpsertConnection(ctx context.Context, req *UpsertConnectionRequest) (*UpsertConnectionResult, error)
	UpsertTransformation(ctx context.Context, req *UpsertTransformationRequest) (*UpsertTransformationResult, error)
}

// ---------------------------------------------------------------------------
// Request / response types (transport-agnostic, aligned with API schema)
// ---------------------------------------------------------------------------

// UpsertSourceRequest is the payload for upserting a source.
type UpsertSourceRequest struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type,omitempty"`
	Description *string                `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// UpsertSourceResult is the API response after upserting a source.
type UpsertSourceResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpsertDestinationRequest is the payload for upserting a destination.
// The Hookdeck API expects url, auth_type, auth, rate_limit, rate_limit_period
// inside a "config" object, not as top-level fields.
type UpsertDestinationRequest struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type,omitempty"`
	Description *string                `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// UpsertDestinationResult is the API response after upserting a destination.
type UpsertDestinationResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpsertConnectionRequest is the payload for upserting a connection.
// The Hookdeck API accepts both ID-based references (source_id, destination_id)
// and name-based references (source.name, destination.name).
type UpsertConnectionRequest struct {
	Name          *string                  `json:"name,omitempty"`
	SourceID      *string                  `json:"source_id,omitempty"`
	DestinationID *string                  `json:"destination_id,omitempty"`
	Source        *ConnectionSourceRef     `json:"source,omitempty"`
	Destination   *ConnectionDestRef       `json:"destination,omitempty"`
	Rules         []map[string]interface{} `json:"rules,omitempty"`
}

// ConnectionSourceRef is a name-based source reference for connection upsert.
type ConnectionSourceRef struct {
	Name string `json:"name"`
}

// ConnectionDestRef is a name-based destination reference for connection upsert.
type ConnectionDestRef struct {
	Name string `json:"name"`
}

// UpsertConnectionResult is the API response after upserting a connection.
type UpsertConnectionResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpsertTransformationRequest is the payload for upserting a transformation.
type UpsertTransformationRequest struct {
	Name string            `json:"name"`
	Code string            `json:"code"`
	Env  map[string]string `json:"env,omitempty"`
}

// UpsertTransformationResult is the API response after upserting a transformation.
type UpsertTransformationResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ---------------------------------------------------------------------------
// Result types
// ---------------------------------------------------------------------------

// ResourceResult captures the outcome for a single resource.
type ResourceResult struct {
	Name   string `json:"name"`
	ID     string `json:"id,omitempty"`
	Action string `json:"action"` // "upserted", "would upsert", "skipped"
}

// Result is the aggregate outcome of a deploy run.
type Result struct {
	Source         *ResourceResult `json:"source,omitempty"`
	Transformation *ResourceResult `json:"transformation,omitempty"`
	Destination    *ResourceResult `json:"destination,omitempty"`
	Connection     *ResourceResult `json:"connection,omitempty"`
}

// Options controls deploy behaviour.
type Options struct {
	DryRun   bool
	CodeRoot string // base directory for resolving relative code_file paths
}

// ---------------------------------------------------------------------------
// Deploy orchestrator
// ---------------------------------------------------------------------------

// Deploy upserts resources declared in the manifest in dependency order:
//  1. Source
//  2. Transformation
//  3. Destination
//  4. Connection (references source, destination, and optionally transformation)
//
// In dry-run mode no API calls are made and client may be nil.
func Deploy(ctx context.Context, client Client, m *manifest.Manifest, opts Options) (*Result, error) {
	if !opts.DryRun && client == nil {
		return nil, fmt.Errorf("client must not be nil in live mode")
	}

	result := &Result{}

	// Track IDs resolved from earlier upserts so that the connection step can
	// reference them.
	var sourceID, destinationID, transformationID string

	// 1. Source
	if m.Source != nil {
		if opts.DryRun {
			result.Source = &ResourceResult{Name: m.Source.Name, Action: "would upsert"}
		} else {
			req := buildSourceRequest(m.Source)
			res, err := client.UpsertSource(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("upserting source %q: %w", m.Source.Name, err)
			}
			sourceID = res.ID
			result.Source = &ResourceResult{Name: res.Name, ID: res.ID, Action: "upserted"}
		}
	}

	// 2. Transformation (before connection, because connection rules reference it)
	if m.Transformation != nil {
		if opts.DryRun {
			result.Transformation = &ResourceResult{Name: m.Transformation.Name, Action: "would upsert"}
		} else {
			code, err := resolveCode(m.Transformation, opts.CodeRoot)
			if err != nil {
				return nil, fmt.Errorf("resolving transformation code for %q: %w", m.Transformation.Name, err)
			}
			req := buildTransformationRequest(m.Transformation, code)
			res, err := client.UpsertTransformation(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("upserting transformation %q: %w", m.Transformation.Name, err)
			}
			transformationID = res.ID
			result.Transformation = &ResourceResult{Name: res.Name, ID: res.ID, Action: "upserted"}
		}
	}

	// 3. Destination
	if m.Destination != nil {
		if opts.DryRun {
			result.Destination = &ResourceResult{Name: m.Destination.Name, Action: "would upsert"}
		} else {
			req := buildDestinationRequest(m.Destination)
			res, err := client.UpsertDestination(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("upserting destination %q: %w", m.Destination.Name, err)
			}
			destinationID = res.ID
			result.Destination = &ResourceResult{Name: res.Name, ID: res.ID, Action: "upserted"}
		}
	}

	// 4. Connection
	if m.Connection != nil {
		if opts.DryRun {
			result.Connection = &ResourceResult{Name: m.Connection.Name, Action: "would upsert"}
		} else {
			req := buildConnectionRequest(m.Connection, sourceID, destinationID, transformationID)
			res, err := client.UpsertConnection(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("upserting connection %q: %w", m.Connection.Name, err)
			}
			result.Connection = &ResourceResult{Name: res.Name, ID: res.ID, Action: "upserted"}
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Request builders
// ---------------------------------------------------------------------------

func buildSourceRequest(src *manifest.SourceConfig) *UpsertSourceRequest {
	req := &UpsertSourceRequest{
		Name: src.Name,
	}
	if src.Type != "" {
		req.Type = src.Type
	}
	if src.Description != "" {
		desc := src.Description
		req.Description = &desc
	}
	if src.Config != nil {
		req.Config = src.Config
	}
	return req
}

func buildDestinationRequest(dst *manifest.DestinationConfig) *UpsertDestinationRequest {
	req := &UpsertDestinationRequest{
		Name: dst.Name,
	}
	if dst.Type != "" {
		req.Type = dst.Type
	}
	if dst.Description != "" {
		desc := dst.Description
		req.Description = &desc
	}

	// Build config map: the Hookdeck API expects url, auth_type, auth,
	// rate_limit, rate_limit_period inside config, not as top-level fields.
	config := make(map[string]interface{})

	// Start with any explicit config entries from the manifest
	for k, v := range dst.Config {
		config[k] = v
	}

	// Map top-level manifest fields into config
	if dst.URL != "" {
		config["url"] = dst.URL
	}
	if dst.AuthType != "" {
		config["auth_type"] = dst.AuthType
	}
	if dst.Auth != nil {
		config["auth"] = dst.Auth
	}
	if dst.RateLimit != 0 {
		config["rate_limit"] = dst.RateLimit
	}
	if dst.RateLimitPeriod != "" {
		config["rate_limit_period"] = dst.RateLimitPeriod
	}

	if len(config) > 0 {
		req.Config = config
	}

	return req
}

func buildTransformationRequest(tr *manifest.TransformationConfig, code string) *UpsertTransformationRequest {
	req := &UpsertTransformationRequest{
		Name: tr.Name,
		Code: code,
	}
	if len(tr.Env) > 0 {
		req.Env = tr.Env
	}
	return req
}

func buildConnectionRequest(conn *manifest.ConnectionConfig, sourceID, destinationID, transformationID string) *UpsertConnectionRequest {
	req := &UpsertConnectionRequest{}

	if conn.Name != "" {
		name := conn.Name
		req.Name = &name
	}
	// Prefer resolved IDs from earlier upserts; fall back to name-based references
	if sourceID != "" {
		req.SourceID = &sourceID
	} else if conn.Source != "" {
		req.Source = &ConnectionSourceRef{Name: conn.Source}
	}
	if destinationID != "" {
		req.DestinationID = &destinationID
	} else if conn.Destination != "" {
		req.Destination = &ConnectionDestRef{Name: conn.Destination}
	}

	// Build rules from explicit rules + shorthands
	var rules []map[string]interface{}

	// Start with explicit rules (if any)
	for _, rule := range conn.Rules {
		ruleCopy := make(map[string]interface{})
		for k, v := range rule {
			ruleCopy[k] = v
		}
		// If this is a transform rule and we have a resolved transformation ID,
		// inject it.
		if ruleType, ok := ruleCopy["type"].(string); ok && ruleType == "transform" && transformationID != "" {
			ruleCopy["transformation_id"] = transformationID
		}
		rules = append(rules, ruleCopy)
	}

	// Convert transformations shorthand to transform rules
	for _, name := range conn.Transformations {
		rule := map[string]interface{}{
			"type":                "transform",
			"transformation_name": name,
		}
		if transformationID != "" {
			rule["transformation_id"] = transformationID
		}
		rules = append(rules, rule)
	}

	// Convert filter shorthand to filter rule
	if conn.Filter != nil {
		rules = append(rules, map[string]interface{}{
			"type": "filter",
			"body": conn.Filter,
		})
	}

	if len(rules) > 0 {
		req.Rules = rules
	}

	return req
}

// resolveCode reads the code file for a transformation.
// If CodeFile is empty, returns an empty string (the API may reject it, but
// that is an API-level concern, not ours).
func resolveCode(tr *manifest.TransformationConfig, codeRoot string) (string, error) {
	if tr.CodeFile == "" {
		return "", nil
	}

	path := tr.CodeFile
	if codeRoot != "" {
		path = codeRoot + "/" + tr.CodeFile
	}

	// For now we pass the code_file path as the code value. In the real deploy
	// command, the caller reads the file and passes the content via a pre-
	// processing step, or we read it here. Let's read it here.
	data, err := readFile(path)
	if err != nil {
		return "", fmt.Errorf("reading code file %q: %w", path, err)
	}
	return string(data), nil
}

// readFile is a package-level variable so tests can override it.
var readFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}
