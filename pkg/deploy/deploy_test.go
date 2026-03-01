package deploy

import (
	"context"
	"fmt"
	"testing"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// ---------------------------------------------------------------------------
// Mock client (implements the same Client interface)
// ---------------------------------------------------------------------------

type mockClient struct {
	upsertSourceCalls         int
	upsertDestinationCalls    int
	upsertConnectionCalls     int
	upsertTransformationCalls int

	// Capture last requests for assertions
	lastConnectionReq *UpsertConnectionRequest

	// Allow overriding return values per-name
	sourceResults         map[string]*UpsertSourceResult
	destinationResults    map[string]*UpsertDestinationResult
	connectionResults     map[string]*UpsertConnectionResult
	transformationResults map[string]*UpsertTransformationResult

	// Global error (returned for any call)
	err error
}

func (m *mockClient) UpsertSource(_ context.Context, req *UpsertSourceRequest) (*UpsertSourceResult, error) {
	m.upsertSourceCalls++
	if m.err != nil {
		return nil, m.err
	}
	if m.sourceResults != nil {
		if r, ok := m.sourceResults[req.Name]; ok {
			return r, nil
		}
	}
	return &UpsertSourceResult{ID: "src_" + req.Name, Name: req.Name}, nil
}

func (m *mockClient) UpsertDestination(_ context.Context, req *UpsertDestinationRequest) (*UpsertDestinationResult, error) {
	m.upsertDestinationCalls++
	if m.err != nil {
		return nil, m.err
	}
	if m.destinationResults != nil {
		if r, ok := m.destinationResults[req.Name]; ok {
			return r, nil
		}
	}
	return &UpsertDestinationResult{ID: "des_" + req.Name, Name: req.Name}, nil
}

func (m *mockClient) UpsertConnection(_ context.Context, req *UpsertConnectionRequest) (*UpsertConnectionResult, error) {
	m.upsertConnectionCalls++
	m.lastConnectionReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.connectionResults != nil {
		name := ""
		if req.Name != nil {
			name = *req.Name
		}
		if r, ok := m.connectionResults[name]; ok {
			return r, nil
		}
	}
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	return &UpsertConnectionResult{ID: "con_" + name, Name: name}, nil
}

func (m *mockClient) UpsertTransformation(_ context.Context, req *UpsertTransformationRequest) (*UpsertTransformationResult, error) {
	m.upsertTransformationCalls++
	if m.err != nil {
		return nil, m.err
	}
	if m.transformationResults != nil {
		if r, ok := m.transformationResults[req.Name]; ok {
			return r, nil
		}
	}
	return &UpsertTransformationResult{ID: "trs_" + req.Name, Name: req.Name}, nil
}

// ---------------------------------------------------------------------------
// Dry-run tests
// ---------------------------------------------------------------------------

func TestDeploy_DryRun_MultipleSources(t *testing.T) {
	input := &DeployInput{
		Sources: []*manifest.SourceConfig{
			{Name: "source-a"},
			{Name: "source-b"},
		},
	}

	result, err := Deploy(context.Background(), nil, input, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}
	if len(result.Sources) != 2 {
		t.Fatalf("expected 2 source results, got %d", len(result.Sources))
	}
	for i, name := range []string{"source-a", "source-b"} {
		if result.Sources[i].Name != name {
			t.Errorf("source[%d]: expected name %q, got %q", i, name, result.Sources[i].Name)
		}
		if result.Sources[i].Action != "would upsert" {
			t.Errorf("source[%d]: expected action 'would upsert', got %q", i, result.Sources[i].Action)
		}
	}
}

func TestDeploy_DryRun_AllResourceTypes(t *testing.T) {
	input := &DeployInput{
		Sources:         []*manifest.SourceConfig{{Name: "src-1"}},
		Destinations:    []*manifest.DestinationConfig{{Name: "dst-1"}},
		Transformations: []*manifest.TransformationConfig{{Name: "tr-1", CodeFile: "index.js"}},
		Connections:     []*manifest.ConnectionConfig{{Name: "conn-1", Source: "src-1", Destination: "dst-1"}},
	}

	result, err := Deploy(context.Background(), nil, input, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}
	if len(result.Sources) != 1 {
		t.Fatalf("expected 1 source result, got %d", len(result.Sources))
	}
	if result.Sources[0].Action != "would upsert" {
		t.Errorf("source: expected 'would upsert', got %q", result.Sources[0].Action)
	}
	if len(result.Destinations) != 1 {
		t.Fatalf("expected 1 destination result, got %d", len(result.Destinations))
	}
	if result.Destinations[0].Action != "would upsert" {
		t.Errorf("destination: expected 'would upsert', got %q", result.Destinations[0].Action)
	}
	if len(result.Transformations) != 1 {
		t.Fatalf("expected 1 transformation result, got %d", len(result.Transformations))
	}
	if result.Transformations[0].Action != "would upsert" {
		t.Errorf("transformation: expected 'would upsert', got %q", result.Transformations[0].Action)
	}
	if len(result.Connections) != 1 {
		t.Fatalf("expected 1 connection result, got %d", len(result.Connections))
	}
	if result.Connections[0].Action != "would upsert" {
		t.Errorf("connection: expected 'would upsert', got %q", result.Connections[0].Action)
	}
}

func TestDeploy_DryRun_EmptyInput(t *testing.T) {
	input := &DeployInput{}

	result, err := Deploy(context.Background(), nil, input, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}
	if len(result.Sources) != 0 {
		t.Errorf("expected 0 source results, got %d", len(result.Sources))
	}
	if len(result.Destinations) != 0 {
		t.Errorf("expected 0 destination results, got %d", len(result.Destinations))
	}
	if len(result.Transformations) != 0 {
		t.Errorf("expected 0 transformation results, got %d", len(result.Transformations))
	}
	if len(result.Connections) != 0 {
		t.Errorf("expected 0 connection results, got %d", len(result.Connections))
	}
}

// ---------------------------------------------------------------------------
// Live-mode tests
// ---------------------------------------------------------------------------

func TestDeploy_LiveMode_ResolvesIDsForConnections(t *testing.T) {
	// Override readFile so we don't need real files on disk
	origReadFile := readFile
	readFile = func(path string) ([]byte, error) {
		return []byte("function handler(req, ctx) { return req; }"), nil
	}
	t.Cleanup(func() { readFile = origReadFile })

	mc := &mockClient{
		sourceResults: map[string]*UpsertSourceResult{
			"my-source": {ID: "src_resolved_1", Name: "my-source"},
		},
		destinationResults: map[string]*UpsertDestinationResult{
			"my-dest": {ID: "des_resolved_1", Name: "my-dest"},
		},
		transformationResults: map[string]*UpsertTransformationResult{
			"my-transform": {ID: "trs_resolved_1", Name: "my-transform"},
		},
	}

	input := &DeployInput{
		Sources:         []*manifest.SourceConfig{{Name: "my-source"}},
		Destinations:    []*manifest.DestinationConfig{{Name: "my-dest", URL: "https://example.com"}},
		Transformations: []*manifest.TransformationConfig{{Name: "my-transform", CodeFile: "index.js"}},
		Connections: []*manifest.ConnectionConfig{{
			Name:            "my-conn",
			Source:          "my-source",
			Destination:     "my-dest",
			Transformations: []string{"my-transform"},
		}},
	}

	result, err := Deploy(context.Background(), mc, input, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// All resource types should have been called
	if mc.upsertSourceCalls != 1 {
		t.Errorf("expected 1 UpsertSource call, got %d", mc.upsertSourceCalls)
	}
	if mc.upsertDestinationCalls != 1 {
		t.Errorf("expected 1 UpsertDestination call, got %d", mc.upsertDestinationCalls)
	}
	if mc.upsertTransformationCalls != 1 {
		t.Errorf("expected 1 UpsertTransformation call, got %d", mc.upsertTransformationCalls)
	}
	if mc.upsertConnectionCalls != 1 {
		t.Errorf("expected 1 UpsertConnection call, got %d", mc.upsertConnectionCalls)
	}

	// Verify connection request used resolved IDs
	connReq := mc.lastConnectionReq
	if connReq == nil {
		t.Fatal("expected connection request to be captured")
	}
	if connReq.SourceID == nil || *connReq.SourceID != "src_resolved_1" {
		t.Errorf("expected source_id 'src_resolved_1', got %v", connReq.SourceID)
	}
	if connReq.DestinationID == nil || *connReq.DestinationID != "des_resolved_1" {
		t.Errorf("expected destination_id 'des_resolved_1', got %v", connReq.DestinationID)
	}

	// Verify transformation ID was injected into the transform rule
	if len(connReq.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(connReq.Rules))
	}
	if connReq.Rules[0]["transformation_id"] != "trs_resolved_1" {
		t.Errorf("expected transformation_id 'trs_resolved_1', got %v", connReq.Rules[0]["transformation_id"])
	}

	// Verify results
	if len(result.Sources) != 1 || result.Sources[0].ID != "src_resolved_1" {
		t.Errorf("unexpected source result: %+v", result.Sources)
	}
	if len(result.Destinations) != 1 || result.Destinations[0].ID != "des_resolved_1" {
		t.Errorf("unexpected destination result: %+v", result.Destinations)
	}
	if len(result.Transformations) != 1 || result.Transformations[0].ID != "trs_resolved_1" {
		t.Errorf("unexpected transformation result: %+v", result.Transformations)
	}
	if len(result.Connections) != 1 {
		t.Fatalf("expected 1 connection result, got %d", len(result.Connections))
	}
}

func TestDeploy_LiveMode_NilClientErrors(t *testing.T) {
	input := &DeployInput{
		Sources: []*manifest.SourceConfig{{Name: "test-source"}},
	}

	_, err := Deploy(context.Background(), nil, input, Options{})
	if err == nil {
		t.Fatal("expected error when client is nil in live mode")
	}
}

func TestDeploy_LiveMode_SourceError(t *testing.T) {
	mc := &mockClient{
		err: fmt.Errorf("API error: unauthorized"),
	}
	input := &DeployInput{
		Sources: []*manifest.SourceConfig{{Name: "test-source"}},
	}

	_, err := Deploy(context.Background(), mc, input, Options{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != `upserting source "test-source": API error: unauthorized` {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestDeploy_LiveMode_FilterShorthand(t *testing.T) {
	mc := &mockClient{}
	input := &DeployInput{
		Sources:      []*manifest.SourceConfig{{Name: "my-source"}},
		Destinations: []*manifest.DestinationConfig{{Name: "my-dest", URL: "https://example.com"}},
		Connections: []*manifest.ConnectionConfig{{
			Name:        "my-conn",
			Source:      "my-source",
			Destination: "my-dest",
			Filter: map[string]interface{}{
				"data": map[string]interface{}{
					"type": "order.placed",
				},
			},
		}},
	}

	result, err := Deploy(context.Background(), mc, input, Options{})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if len(result.Connections) != 1 {
		t.Fatalf("expected 1 connection result, got %d", len(result.Connections))
	}
	if result.Connections[0].Action != "upserted" {
		t.Errorf("expected action 'upserted', got %q", result.Connections[0].Action)
	}

	// Verify filter shorthand was converted to a filter rule
	connReq := mc.lastConnectionReq
	if connReq == nil {
		t.Fatal("expected connection request to be captured")
	}
	if len(connReq.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(connReq.Rules))
	}
	rule := connReq.Rules[0]
	if rule["type"] != "filter" {
		t.Errorf("expected rule type 'filter', got %v", rule["type"])
	}
	body, ok := rule["body"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected rule body to be a map, got %T", rule["body"])
	}
	if _, ok := body["data"]; !ok {
		t.Error("expected rule body to contain key 'data'")
	}
}
