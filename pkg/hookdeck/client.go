// Package hookdeck provides an HTTP client adapter that satisfies the
// deploy.Client interface for the Hookdeck REST API.
package hookdeck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/deploy"
)

const defaultBaseURL = "https://api.hookdeck.com/2025-07-01"

// Client is a concrete HTTP client for the Hookdeck API.
type Client struct {
	baseURL    string
	apiKey     string
	projectID  string
	httpClient *http.Client
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL overrides the default API base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient overrides the default http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient creates a Hookdeck API client. The apiKey is required.
// The projectID is optional (omit if the API key is scoped to one project).
func NewClient(apiKey, projectID string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
		projectID:  projectID,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ---------------------------------------------------------------------------
// deploy.Client interface implementation
// ---------------------------------------------------------------------------

// UpsertSource creates or updates a source by name (PUT /sources).
func (c *Client) UpsertSource(ctx context.Context, req *deploy.UpsertSourceRequest) (*deploy.UpsertSourceResult, error) {
	var result deploy.UpsertSourceResult
	if err := c.put(ctx, "/sources", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpsertDestination creates or updates a destination by name (PUT /destinations).
func (c *Client) UpsertDestination(ctx context.Context, req *deploy.UpsertDestinationRequest) (*deploy.UpsertDestinationResult, error) {
	var result deploy.UpsertDestinationResult
	if err := c.put(ctx, "/destinations", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpsertConnection creates or updates a connection (PUT /connections).
func (c *Client) UpsertConnection(ctx context.Context, req *deploy.UpsertConnectionRequest) (*deploy.UpsertConnectionResult, error) {
	var result deploy.UpsertConnectionResult
	if err := c.put(ctx, "/connections", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpsertTransformation creates or updates a transformation by name (PUT /transformations).
func (c *Client) UpsertTransformation(ctx context.Context, req *deploy.UpsertTransformationRequest) (*deploy.UpsertTransformationResult, error) {
	var result deploy.UpsertTransformationResult
	if err := c.put(ctx, "/transformations", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Query helpers (used by the status command)
// ---------------------------------------------------------------------------

// ResourceInfo is a lightweight representation of a Hookdeck resource
// returned by list endpoints.
type ResourceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// listResponse is the generic envelope returned by Hookdeck list endpoints.
type listResponse struct {
	Models []json.RawMessage `json:"models"`
	Count  int               `json:"count"`
}

// sourceModel is the subset of fields we care about from the source response.
type sourceModel struct {
	ID  string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// genericModel is the subset of fields we care about from most responses.
type genericModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// connectionModel has full_name instead of name.
type connectionModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// FindSourceByName queries GET /sources?name=<name> and returns the first match.
func (c *Client) FindSourceByName(ctx context.Context, name string) (*ResourceInfo, error) {
	params := url.Values{"name": {name}}
	body, err := c.get(ctx, "/sources", params)
	if err != nil {
		return nil, err
	}

	var list listResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding source list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}

	var src sourceModel
	if err := json.Unmarshal(list.Models[0], &src); err != nil {
		return nil, fmt.Errorf("decoding source model: %w", err)
	}
	return &ResourceInfo{ID: src.ID, Name: src.Name, URL: src.URL}, nil
}

// FindDestinationByName queries GET /destinations?name=<name> and returns the first match.
func (c *Client) FindDestinationByName(ctx context.Context, name string) (*ResourceInfo, error) {
	params := url.Values{"name": {name}}
	body, err := c.get(ctx, "/destinations", params)
	if err != nil {
		return nil, err
	}

	var list listResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding destination list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}

	var dst genericModel
	if err := json.Unmarshal(list.Models[0], &dst); err != nil {
		return nil, fmt.Errorf("decoding destination model: %w", err)
	}
	return &ResourceInfo{ID: dst.ID, Name: dst.Name}, nil
}

// FindConnectionByFullName queries GET /connections?full_name=<name> and returns the first match.
func (c *Client) FindConnectionByFullName(ctx context.Context, fullName string) (*ResourceInfo, error) {
	params := url.Values{"full_name": {fullName}}
	body, err := c.get(ctx, "/connections", params)
	if err != nil {
		return nil, err
	}

	var list listResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding connection list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}

	var conn connectionModel
	if err := json.Unmarshal(list.Models[0], &conn); err != nil {
		return nil, fmt.Errorf("decoding connection model: %w", err)
	}
	name := conn.FullName
	if name == "" {
		name = conn.Name
	}
	return &ResourceInfo{ID: conn.ID, Name: name}, nil
}

// FindTransformationByName queries GET /transformations?name=<name> and returns the first match.
func (c *Client) FindTransformationByName(ctx context.Context, name string) (*ResourceInfo, error) {
	params := url.Values{"name": {name}}
	body, err := c.get(ctx, "/transformations", params)
	if err != nil {
		return nil, err
	}

	var list listResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding transformation list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}

	var tr genericModel
	if err := json.Unmarshal(list.Models[0], &tr); err != nil {
		return nil, fmt.Errorf("decoding transformation model: %w", err)
	}
	return &ResourceInfo{ID: tr.ID, Name: tr.Name}, nil
}

// ---------------------------------------------------------------------------
// Full resource detail types (used by drift detection)
// ---------------------------------------------------------------------------

// SourceDetail is the full representation of a Hookdeck source.
type SourceDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// DestinationDetail is the full representation of a Hookdeck destination.
// The API returns url, auth_type, auth, rate_limit, rate_limit_period inside a config object.
type DestinationDetail struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      DestinationConfigDetail `json:"config"`
}

// DestinationConfigDetail is the config sub-object of a Hookdeck destination.
type DestinationConfigDetail struct {
	URL             string                 `json:"url"`
	AuthType        string                 `json:"auth_type"`
	Auth            map[string]interface{} `json:"auth"`
	RateLimit       int                    `json:"rate_limit"`
	RateLimitPeriod string                 `json:"rate_limit_period"`
}

// ConnectionDetail is the full representation of a Hookdeck connection.
type ConnectionDetail struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	FullName    string                   `json:"full_name"`
	Source      *SourceDetail            `json:"source"`
	Destination *DestinationDetail       `json:"destination"`
	Rules       []map[string]interface{} `json:"rules"`
}

// TransformationDetail is the full representation of a Hookdeck transformation.
type TransformationDetail struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Code string            `json:"code"`
	Env  map[string]string `json:"env"`
}

// ---------------------------------------------------------------------------
// Full resource fetch methods (used by drift detection)
// ---------------------------------------------------------------------------

// GetSourceByName queries GET /sources?name=<name> and returns full source details.
func (c *Client) GetSourceByName(ctx context.Context, name string) (*SourceDetail, error) {
	params := url.Values{"name": {name}}
	body, err := c.get(ctx, "/sources", params)
	if err != nil {
		return nil, err
	}
	var list struct {
		Models []SourceDetail `json:"models"`
		Count  int            `json:"count"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding source list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}
	return &list.Models[0], nil
}

// GetDestinationByName queries GET /destinations?name=<name> and returns full destination details.
func (c *Client) GetDestinationByName(ctx context.Context, name string) (*DestinationDetail, error) {
	params := url.Values{"name": {name}}
	body, err := c.get(ctx, "/destinations", params)
	if err != nil {
		return nil, err
	}
	var list struct {
		Models []DestinationDetail `json:"models"`
		Count  int                 `json:"count"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding destination list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}
	return &list.Models[0], nil
}

// GetConnectionByFullName queries GET /connections?full_name=<name> and returns full connection details.
func (c *Client) GetConnectionByFullName(ctx context.Context, fullName string) (*ConnectionDetail, error) {
	params := url.Values{"full_name": {fullName}}
	body, err := c.get(ctx, "/connections", params)
	if err != nil {
		return nil, err
	}
	var list struct {
		Models []ConnectionDetail `json:"models"`
		Count  int                `json:"count"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding connection list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}
	return &list.Models[0], nil
}

// GetTransformationByName queries GET /transformations?name=<name> and returns full transformation details.
func (c *Client) GetTransformationByName(ctx context.Context, name string) (*TransformationDetail, error) {
	params := url.Values{"name": {name}}
	body, err := c.get(ctx, "/transformations", params)
	if err != nil {
		return nil, err
	}
	var list struct {
		Models []TransformationDetail `json:"models"`
		Count  int                    `json:"count"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding transformation list: %w", err)
	}
	if list.Count == 0 || len(list.Models) == 0 {
		return nil, nil
	}
	return &list.Models[0], nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

// apiError is the error body returned by the Hookdeck API.
type apiError struct {
	Message string `json:"message"`
}

// put sends a PUT request with a JSON body and decodes the response into out.
func (c *Client) put(ctx context.Context, path string, body interface{}, out interface{}) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Message)
		}
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// get sends a GET request with query parameters and returns the raw body.
func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// setHeaders sets authentication and project headers on the request.
func (c *Client) setHeaders(req *http.Request) {
	// Hookdeck uses HTTP Basic Auth: API key as username, empty password.
	req.SetBasicAuth(c.apiKey, "")

	if c.projectID != "" {
		req.Header.Set("X-Project-ID", c.projectID)
	}
}
