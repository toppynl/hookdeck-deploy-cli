package project

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/deploy"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

// buildDeployInput resolves all resources in a registry for the given environment
// and returns a DeployInput ready for deploy.Deploy.
func buildDeployInput(reg *Registry, envName string) *deploy.DeployInput {
	input := &deploy.DeployInput{}
	for i := range reg.SourceList {
		input.Sources = append(input.Sources, manifest.ResolveSourceEnv(&reg.SourceList[i], envName))
	}
	for i := range reg.DestinationList {
		input.Destinations = append(input.Destinations, manifest.ResolveDestinationEnv(&reg.DestinationList[i], envName))
	}
	for i := range reg.TransformationList {
		input.Transformations = append(input.Transformations, manifest.ResolveTransformationEnv(&reg.TransformationList[i], envName))
	}
	for i := range reg.ConnectionList {
		input.Connections = append(input.Connections, manifest.ResolveConnectionEnv(&reg.ConnectionList[i], envName))
	}
	return input
}

func TestIntegration_FullProjectDeploy(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "hookdeck.project.jsonc", `{
		"version": "2",
		"env": {"staging": {"profile": "staging"}}
	}`)

	writeFile(t, dir, "sources/hookdeck.jsonc", `{
		"sources": [{"name": "webhook-src", "type": "Stripe"}]
	}`)

	writeFile(t, dir, "api/hookdeck.jsonc", `{
		"destinations": [{"name": "api-dest", "url": "https://api.example.com"}],
		"connections": [{"name": "webhook-to-api", "source": "webhook-src", "destination": "api-dest"}]
	}`)

	proj, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}

	// Verify project loaded correctly
	if proj.Config.Version != "2" {
		t.Errorf("expected version '2', got %q", proj.Config.Version)
	}
	if len(proj.Registry.SourceList) != 1 {
		t.Fatalf("expected 1 source in registry, got %d", len(proj.Registry.SourceList))
	}
	if len(proj.Registry.DestinationList) != 1 {
		t.Fatalf("expected 1 destination in registry, got %d", len(proj.Registry.DestinationList))
	}
	if len(proj.Registry.ConnectionList) != 1 {
		t.Fatalf("expected 1 connection in registry, got %d", len(proj.Registry.ConnectionList))
	}

	// Resolve env and build deploy input
	input := buildDeployInput(proj.Registry, "staging")

	// Deploy dry-run (no API calls, client can be nil)
	result, err := deploy.Deploy(context.Background(), nil, input, deploy.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}

	// Verify results
	if len(result.Sources) != 1 {
		t.Fatalf("expected 1 source result, got %d", len(result.Sources))
	}
	if result.Sources[0].Name != "webhook-src" {
		t.Errorf("expected source name 'webhook-src', got %q", result.Sources[0].Name)
	}
	if result.Sources[0].Action != "would upsert" {
		t.Errorf("expected action 'would upsert', got %q", result.Sources[0].Action)
	}

	if len(result.Destinations) != 1 {
		t.Fatalf("expected 1 destination result, got %d", len(result.Destinations))
	}
	if result.Destinations[0].Name != "api-dest" {
		t.Errorf("expected destination name 'api-dest', got %q", result.Destinations[0].Name)
	}

	if len(result.Connections) != 1 {
		t.Fatalf("expected 1 connection result, got %d", len(result.Connections))
	}
	if result.Connections[0].Name != "webhook-to-api" {
		t.Errorf("expected connection name 'webhook-to-api', got %q", result.Connections[0].Name)
	}

	if len(result.Transformations) != 0 {
		t.Errorf("expected 0 transformation results, got %d", len(result.Transformations))
	}
}

func TestIntegration_EnvOverrides(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "hookdeck.project.jsonc", `{
		"version": "2",
		"env": {
			"staging":    {"profile": "staging"},
			"production": {"profile": "production"}
		}
	}`)

	// Destination with per-env URL overrides
	writeFile(t, dir, "hookdeck.jsonc", `{
		"sources": [{"name": "ingest"}],
		"destinations": [{
			"name": "backend",
			"url": "https://default.example.com/webhook",
			"env": {
				"staging": {
					"url": "https://staging.example.com/webhook"
				},
				"production": {
					"url": "https://prod.example.com/webhook"
				}
			}
		}],
		"connections": [{"name": "ingest-to-backend", "source": "ingest", "destination": "backend"}]
	}`)

	proj, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}
	if len(proj.Registry.DestinationList) != 1 {
		t.Fatalf("expected 1 destination, got %d", len(proj.Registry.DestinationList))
	}

	// Resolve for staging -- URL should be overridden
	stagingDst := manifest.ResolveDestinationEnv(&proj.Registry.DestinationList[0], "staging")
	if stagingDst.URL != "https://staging.example.com/webhook" {
		t.Errorf("staging URL: expected 'https://staging.example.com/webhook', got %q", stagingDst.URL)
	}

	// Resolve for production -- URL should be overridden differently
	prodDst := manifest.ResolveDestinationEnv(&proj.Registry.DestinationList[0], "production")
	if prodDst.URL != "https://prod.example.com/webhook" {
		t.Errorf("production URL: expected 'https://prod.example.com/webhook', got %q", prodDst.URL)
	}

	// Resolve for an undefined env -- URL should remain default
	devDst := manifest.ResolveDestinationEnv(&proj.Registry.DestinationList[0], "dev")
	if devDst.URL != "https://default.example.com/webhook" {
		t.Errorf("dev URL: expected 'https://default.example.com/webhook', got %q", devDst.URL)
	}

	// Build staging deploy input and run dry-run
	input := buildDeployInput(proj.Registry, "staging")

	result, err := deploy.Deploy(context.Background(), nil, input, deploy.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run failed: %v", err)
	}

	if len(result.Sources) != 1 {
		t.Errorf("expected 1 source result, got %d", len(result.Sources))
	}
	if len(result.Destinations) != 1 {
		t.Errorf("expected 1 destination result, got %d", len(result.Destinations))
	}
	if len(result.Connections) != 1 {
		t.Errorf("expected 1 connection result, got %d", len(result.Connections))
	}

	// Verify the resolved destination in the input has the staging URL
	if input.Destinations[0].URL != "https://staging.example.com/webhook" {
		t.Errorf("deploy input destination URL: expected staging URL, got %q", input.Destinations[0].URL)
	}
}

func TestIntegration_ConnectionEnvOverrides(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "hookdeck.project.jsonc", `{
		"version": "2",
		"env": {
			"staging":    {"profile": "staging"},
			"production": {"profile": "production"}
		}
	}`)

	writeFile(t, dir, "transform.js", `function handler(request, context) { return request; }`)

	writeFile(t, dir, "hookdeck.jsonc", `{
		"sources": [{"name": "ingest"}],
		"destinations": [{"name": "backend", "url": "https://api.example.com"}],
		"transformations": [{"name": "anonymizer", "code_file": "transform.js"}],
		"connections": [{
			"name": "ingest-to-backend",
			"source": "ingest",
			"destination": "backend",
			"filter": {"headers": {"x-env": "default"}},
			"transformations": ["anonymizer"],
			"env": {
				"staging": {
					"filter": {"headers": {"x-env": "staging"}},
					"transformations": ["anonymizer"]
				},
				"production": {
					"filter": {"headers": {"x-env": "production"}},
					"transformations": []
				}
			}
		}]
	}`)

	proj, err := LoadProject(filepath.Join(dir, "hookdeck.project.jsonc"))
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}
	if len(proj.Registry.ConnectionList) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(proj.Registry.ConnectionList))
	}

	// Staging: filter overridden, transformations include anonymizer
	stagingInput := buildDeployInput(proj.Registry, "staging")
	stagingConn := stagingInput.Connections[0]
	if stagingConn.Filter["headers"].(map[string]interface{})["x-env"] != "staging" {
		t.Errorf("staging filter: expected x-env=staging, got %v", stagingConn.Filter)
	}
	if len(stagingConn.Transformations) != 1 || stagingConn.Transformations[0] != "anonymizer" {
		t.Errorf("staging transformations: expected [anonymizer], got %v", stagingConn.Transformations)
	}

	// Production: filter overridden, transformations cleared
	prodInput := buildDeployInput(proj.Registry, "production")
	prodConn := prodInput.Connections[0]
	if prodConn.Filter["headers"].(map[string]interface{})["x-env"] != "production" {
		t.Errorf("production filter: expected x-env=production, got %v", prodConn.Filter)
	}
	if len(prodConn.Transformations) != 0 {
		t.Errorf("production transformations: expected empty, got %v", prodConn.Transformations)
	}

	// Dry-run deploy succeeds for both environments
	for _, env := range []string{"staging", "production"} {
		input := buildDeployInput(proj.Registry, env)
		result, err := deploy.Deploy(context.Background(), nil, input, deploy.Options{DryRun: true})
		if err != nil {
			t.Fatalf("Deploy dry-run (%s) failed: %v", env, err)
		}
		if len(result.Connections) != 1 {
			t.Errorf("%s: expected 1 connection result, got %d", env, len(result.Connections))
		}
		if result.Connections[0].Name != "ingest-to-backend" {
			t.Errorf("%s: expected connection name 'ingest-to-backend', got %q", env, result.Connections[0].Name)
		}
	}
}
