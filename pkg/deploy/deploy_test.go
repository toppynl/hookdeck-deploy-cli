package deploy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// --- dry-run tests (no API client needed) ---

func TestDeploy_DryRun_SourceOnly(t *testing.T) {
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "test-source"},
	}

	result, err := Deploy(context.Background(), nil, m, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}
	if result.Source == nil {
		t.Fatal("expected source result, got nil")
	}
	if result.Source.Name != "test-source" {
		t.Errorf("expected name 'test-source', got '%s'", result.Source.Name)
	}
	if result.Source.Action != "would upsert" {
		t.Errorf("expected action 'would upsert', got '%s'", result.Source.Action)
	}
}

func TestDeploy_DryRun_AllResources(t *testing.T) {
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "test-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "test-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{Name: "test-conn", Source: "test-source"},
		Transformation: &manifest.TransformationConfig{
			Name:     "test-transform",
			CodeFile: "dist/index.js",
			Env:      map[string]string{"KEY": "value"},
		},
	}

	result, err := Deploy(context.Background(), nil, m, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}

	if result.Source == nil {
		t.Fatal("expected source result, got nil")
	}
	if result.Source.Action != "would upsert" {
		t.Errorf("source: expected 'would upsert', got '%s'", result.Source.Action)
	}

	if result.Transformation == nil {
		t.Fatal("expected transformation result, got nil")
	}
	if result.Transformation.Action != "would upsert" {
		t.Errorf("transformation: expected 'would upsert', got '%s'", result.Transformation.Action)
	}

	if result.Destination == nil {
		t.Fatal("expected destination result, got nil")
	}
	if result.Destination.Action != "would upsert" {
		t.Errorf("destination: expected 'would upsert', got '%s'", result.Destination.Action)
	}

	if result.Connection == nil {
		t.Fatal("expected connection result, got nil")
	}
	if result.Connection.Action != "would upsert" {
		t.Errorf("connection: expected 'would upsert', got '%s'", result.Connection.Action)
	}
}

func TestDeploy_DryRun_EmptyManifest(t *testing.T) {
	m := &manifest.Manifest{}

	result, err := Deploy(context.Background(), nil, m, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}
	if result.Source != nil {
		t.Error("expected nil source result for empty manifest")
	}
	if result.Destination != nil {
		t.Error("expected nil destination result for empty manifest")
	}
	if result.Connection != nil {
		t.Error("expected nil connection result for empty manifest")
	}
	if result.Transformation != nil {
		t.Error("expected nil transformation result for empty manifest")
	}
}

func TestDeploy_DryRun_ConnectionWithRules(t *testing.T) {
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com/webhook"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
			Rules: []map[string]interface{}{
				{"type": "transform"},
			},
		},
		Transformation: &manifest.TransformationConfig{
			Name:     "my-transform",
			CodeFile: "dist/index.js",
		},
	}

	result, err := Deploy(context.Background(), nil, m, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}

	// All four resources should be present
	if result.Source == nil || result.Destination == nil || result.Connection == nil || result.Transformation == nil {
		t.Fatal("expected all four resource results for full manifest")
	}
}

func TestDeploy_MissingCodeFile(t *testing.T) {
	m := &manifest.Manifest{
		Transformation: &manifest.TransformationConfig{
			Name: "no-code-file",
		},
	}

	_, err := Deploy(context.Background(), &mockClient{}, m, Options{})
	if err == nil {
		t.Fatal("expected error when code_file is missing, got nil")
	}
	if !strings.Contains(err.Error(), "code_file is required") {
		t.Errorf("expected error to contain %q, got %q", "code_file is required", err.Error())
	}
}

// --- mock client tests (live mode with mock) ---

type mockClient struct {
	upsertSourceCalls         int
	upsertDestinationCalls    int
	upsertConnectionCalls     int
	upsertTransformationCalls int

	// Capture last requests for assertions
	lastConnectionReq *UpsertConnectionRequest

	// Allow overriding return values
	sourceResult         *UpsertSourceResult
	destinationResult    *UpsertDestinationResult
	connectionResult     *UpsertConnectionResult
	transformationResult *UpsertTransformationResult
	err                  error
}

func (m *mockClient) UpsertSource(_ context.Context, req *UpsertSourceRequest) (*UpsertSourceResult, error) {
	m.upsertSourceCalls++
	if m.err != nil {
		return nil, m.err
	}
	if m.sourceResult != nil {
		return m.sourceResult, nil
	}
	return &UpsertSourceResult{ID: "src_123", Name: req.Name}, nil
}

func (m *mockClient) UpsertDestination(_ context.Context, req *UpsertDestinationRequest) (*UpsertDestinationResult, error) {
	m.upsertDestinationCalls++
	if m.err != nil {
		return nil, m.err
	}
	if m.destinationResult != nil {
		return m.destinationResult, nil
	}
	return &UpsertDestinationResult{ID: "des_456", Name: req.Name}, nil
}

func (m *mockClient) UpsertConnection(_ context.Context, req *UpsertConnectionRequest) (*UpsertConnectionResult, error) {
	m.upsertConnectionCalls++
	m.lastConnectionReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.connectionResult != nil {
		return m.connectionResult, nil
	}
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	return &UpsertConnectionResult{ID: "con_789", Name: name}, nil
}

func (m *mockClient) UpsertTransformation(_ context.Context, req *UpsertTransformationRequest) (*UpsertTransformationResult, error) {
	m.upsertTransformationCalls++
	if m.err != nil {
		return nil, m.err
	}
	if m.transformationResult != nil {
		return m.transformationResult, nil
	}
	return &UpsertTransformationResult{ID: "trs_abc", Name: req.Name}, nil
}

func TestDeploy_LiveMode_SourceOnly(t *testing.T) {
	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "test-source"},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if mc.upsertSourceCalls != 1 {
		t.Errorf("expected 1 UpsertSource call, got %d", mc.upsertSourceCalls)
	}
	if result.Source == nil {
		t.Fatal("expected source result")
	}
	if result.Source.ID != "src_123" {
		t.Errorf("expected ID 'src_123', got '%s'", result.Source.ID)
	}
	if result.Source.Action != "upserted" {
		t.Errorf("expected action 'upserted', got '%s'", result.Source.Action)
	}
}

func TestDeploy_LiveMode_AllResources(t *testing.T) {
	// Override readFile so we don't need real files on disk
	origReadFile := readFile
	readFile = func(path string) ([]byte, error) {
		return []byte("function handler(request, context) { return request; }"), nil
	}
	t.Cleanup(func() { readFile = origReadFile })

	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{Name: "my-conn", Source: "my-source"},
		Transformation: &manifest.TransformationConfig{
			Name:     "my-transform",
			CodeFile: "dist/index.js",
			Env:      map[string]string{"API_KEY": "secret"},
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if mc.upsertSourceCalls != 1 {
		t.Errorf("expected 1 UpsertSource call, got %d", mc.upsertSourceCalls)
	}
	if mc.upsertDestinationCalls != 1 {
		t.Errorf("expected 1 UpsertDestination call, got %d", mc.upsertDestinationCalls)
	}
	if mc.upsertConnectionCalls != 1 {
		t.Errorf("expected 1 UpsertConnection call, got %d", mc.upsertConnectionCalls)
	}
	if mc.upsertTransformationCalls != 1 {
		t.Errorf("expected 1 UpsertTransformation call, got %d", mc.upsertTransformationCalls)
	}

	if result.Source.Action != "upserted" {
		t.Errorf("source action: expected 'upserted', got '%s'", result.Source.Action)
	}
	if result.Destination.Action != "upserted" {
		t.Errorf("destination action: expected 'upserted', got '%s'", result.Destination.Action)
	}
	if result.Connection.Action != "upserted" {
		t.Errorf("connection action: expected 'upserted', got '%s'", result.Connection.Action)
	}
	if result.Transformation.Action != "upserted" {
		t.Errorf("transformation action: expected 'upserted', got '%s'", result.Transformation.Action)
	}
}

func TestDeploy_LiveMode_ConnectionResolvesIDs(t *testing.T) {
	mc := &mockClient{
		sourceResult:      &UpsertSourceResult{ID: "src_resolved", Name: "my-source"},
		destinationResult: &UpsertDestinationResult{ID: "des_resolved", Name: "my-dest"},
	}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Connection should be created
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
	if result.Connection.ID != "con_789" {
		t.Errorf("expected connection ID 'con_789', got '%s'", result.Connection.ID)
	}
}

func TestDeploy_LiveMode_SourceError(t *testing.T) {
	mc := &mockClient{
		err: fmt.Errorf("API error: unauthorized"),
	}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "test-source"},
	}

	_, err := Deploy(context.Background(), mc, m, Options{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeploy_LiveMode_NilClientErrors(t *testing.T) {
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "test-source"},
	}

	_, err := Deploy(context.Background(), nil, m, Options{})
	if err == nil {
		t.Fatal("expected error when client is nil in live mode")
	}
}

func TestDeploy_LiveMode_SourceWithConfig(t *testing.T) {
	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{
			Name: "configured-source",
			Type: "WEBHOOK",
			Config: map[string]interface{}{
				"auth": map[string]interface{}{
					"type":       "API_KEY",
					"header_key": "x-api-key",
					"api_key":    "secret123",
				},
			},
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if mc.upsertSourceCalls != 1 {
		t.Errorf("expected 1 UpsertSource call, got %d", mc.upsertSourceCalls)
	}
	if result.Source.Name != "configured-source" {
		t.Errorf("expected name 'configured-source', got '%s'", result.Source.Name)
	}
}

func TestDeploy_LiveMode_ConnectionWithFilterRule(t *testing.T) {
	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
			Rules: []map[string]interface{}{
				{
					"type": "filter",
					"body": map[string]interface{}{
						"type": "order.placed",
					},
				},
			},
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
	if result.Connection.Action != "upserted" {
		t.Errorf("expected action 'upserted', got '%s'", result.Connection.Action)
	}
}

func TestDeploy_LiveMode_ConnectionWithRetryRule(t *testing.T) {
	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
			Rules: []map[string]interface{}{
				{
					"type":     "retry",
					"strategy": "linear",
					"count":    5,
					"interval": 60,
				},
			},
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
}

func TestDeploy_LiveMode_ConnectionWithTransformRule(t *testing.T) {
	// Override readFile so we don't need real files on disk
	origReadFile := readFile
	readFile = func(path string) ([]byte, error) {
		return []byte("function handler(request, context) { return request; }"), nil
	}
	t.Cleanup(func() { readFile = origReadFile })

	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
			Rules: []map[string]interface{}{
				{"type": "transform"},
			},
		},
		Transformation: &manifest.TransformationConfig{
			Name:     "my-transform",
			CodeFile: "dist/index.js",
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
	// Verify the transform rule got the transformation_id injected
	if mc.lastConnectionReq == nil {
		t.Fatal("expected connection request to be captured")
	}
	if len(mc.lastConnectionReq.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(mc.lastConnectionReq.Rules))
	}
	rule := mc.lastConnectionReq.Rules[0]
	if rule["type"] != "transform" {
		t.Errorf("expected rule type 'transform', got '%v'", rule["type"])
	}
	if rule["transformation_id"] != "trs_abc" {
		t.Errorf("expected transformation_id 'trs_abc', got '%v'", rule["transformation_id"])
	}
}

func TestDeploy_LiveMode_ConnectionWithFilterShorthand(t *testing.T) {
	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
			Filter: map[string]interface{}{
				"data": map[string]interface{}{
					"entity": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
	if mc.lastConnectionReq == nil {
		t.Fatal("expected connection request to be captured")
	}
	if len(mc.lastConnectionReq.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(mc.lastConnectionReq.Rules))
	}
	rule := mc.lastConnectionReq.Rules[0]
	if rule["type"] != "filter" {
		t.Errorf("expected rule type 'filter', got '%v'", rule["type"])
	}
	body, ok := rule["body"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected rule body to be a map, got %T", rule["body"])
	}
	if _, ok := body["data"]; !ok {
		t.Error("expected rule body to contain key 'data'")
	}
}

func TestDeploy_LiveMode_ConnectionWithTransformationsShorthand(t *testing.T) {
	// Override readFile so we don't need real files on disk
	origReadFile := readFile
	readFile = func(path string) ([]byte, error) {
		return []byte("function handler(request, context) { return request; }"), nil
	}
	t.Cleanup(func() { readFile = origReadFile })

	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:            "my-conn",
			Source:          "my-source",
			Transformations: []string{"my-transform"},
		},
		Transformation: &manifest.TransformationConfig{
			Name:     "my-transform",
			CodeFile: "dist/index.js",
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
	if mc.lastConnectionReq == nil {
		t.Fatal("expected connection request to be captured")
	}
	if len(mc.lastConnectionReq.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(mc.lastConnectionReq.Rules))
	}
	rule := mc.lastConnectionReq.Rules[0]
	if rule["type"] != "transform" {
		t.Errorf("expected rule type 'transform', got '%v'", rule["type"])
	}
	if rule["transformation_name"] != "my-transform" {
		t.Errorf("expected transformation_name 'my-transform', got '%v'", rule["transformation_name"])
	}
	// Transformation ID should be injected from the upserted transformation
	if rule["transformation_id"] != "trs_abc" {
		t.Errorf("expected transformation_id 'trs_abc', got '%v'", rule["transformation_id"])
	}
}

func TestDeploy_LiveMode_ConnectionWithBothShorthands(t *testing.T) {
	// Override readFile so we don't need real files on disk
	origReadFile := readFile
	readFile = func(path string) ([]byte, error) {
		return []byte("function handler(request, context) { return request; }"), nil
	}
	t.Cleanup(func() { readFile = origReadFile })

	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
			Filter: map[string]interface{}{
				"data": map[string]interface{}{
					"type": "order.placed",
				},
			},
			Transformations: []string{"my-transform"},
		},
		Transformation: &manifest.TransformationConfig{
			Name:     "my-transform",
			CodeFile: "dist/index.js",
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
	if mc.lastConnectionReq == nil {
		t.Fatal("expected connection request to be captured")
	}
	// Should have 2 rules: transform + filter
	if len(mc.lastConnectionReq.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(mc.lastConnectionReq.Rules))
	}
	// Transform rule should come first (transformations shorthand before filter shorthand)
	transformRule := mc.lastConnectionReq.Rules[0]
	if transformRule["type"] != "transform" {
		t.Errorf("expected first rule type 'transform', got '%v'", transformRule["type"])
	}
	filterRule := mc.lastConnectionReq.Rules[1]
	if filterRule["type"] != "filter" {
		t.Errorf("expected second rule type 'filter', got '%v'", filterRule["type"])
	}
}

func TestDeploy_LiveMode_ConnectionWithExplicitRulesAndShorthands(t *testing.T) {
	mc := &mockClient{}
	m := &manifest.Manifest{
		Source: &manifest.SourceConfig{Name: "my-source"},
		Destination: &manifest.DestinationConfig{
			Name:   "my-dest",
			Config: map[string]interface{}{"url": "https://example.com"},
		},
		Connection: &manifest.ConnectionConfig{
			Name:   "my-conn",
			Source: "my-source",
			Rules: []map[string]interface{}{
				{
					"type":     "retry",
					"strategy": "linear",
					"count":    5,
					"interval": 60,
				},
			},
			Filter: map[string]interface{}{
				"data": map[string]interface{}{
					"type": "order.placed",
				},
			},
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Connection == nil {
		t.Fatal("expected connection result")
	}
	if mc.lastConnectionReq == nil {
		t.Fatal("expected connection request to be captured")
	}
	// Should have 2 rules: explicit retry + filter shorthand
	if len(mc.lastConnectionReq.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(mc.lastConnectionReq.Rules))
	}
	// Explicit rules come first
	if mc.lastConnectionReq.Rules[0]["type"] != "retry" {
		t.Errorf("expected first rule type 'retry', got '%v'", mc.lastConnectionReq.Rules[0]["type"])
	}
	// Filter shorthand comes after
	if mc.lastConnectionReq.Rules[1]["type"] != "filter" {
		t.Errorf("expected second rule type 'filter', got '%v'", mc.lastConnectionReq.Rules[1]["type"])
	}
}

func TestDeploy_LiveMode_DestinationWithRateLimit(t *testing.T) {
	mc := &mockClient{}
	m := &manifest.Manifest{
		Destination: &manifest.DestinationConfig{
			Name:            "rate-limited-dest",
			Config:          map[string]interface{}{"url": "https://example.com"},
			RateLimit:       100,
			RateLimitPeriod: "second",
		},
	}

	result, err := Deploy(context.Background(), mc, m, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if result.Destination == nil {
		t.Fatal("expected destination result")
	}
	if result.Destination.Name != "rate-limited-dest" {
		t.Errorf("expected name 'rate-limited-dest', got '%s'", result.Destination.Name)
	}
}
