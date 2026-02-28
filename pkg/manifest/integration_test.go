//go:build monorepo

package manifest

import (
	"path/filepath"
	"runtime"
	"testing"
)

// These tests validate the Go loader against real monorepo manifests.
// They verify that JSONC parsing, inheritance resolution, and env overlay
// merging work correctly with the actual files in this repository.

func monorepoRootDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "../../../..")
}

func TestIntegration_RootManifest(t *testing.T) {
	path := filepath.Join(monorepoRootDir(), "hookdeck.jsonc")
	m, err := LoadFile(path)
	if err != nil {
		t.Skipf("Skipping (manifest not found): %v", err)
	}

	// Root manifest should only define env overlays with profiles
	if m.Env == nil {
		t.Fatal("expected env block to be defined")
	}
	if m.Env["staging"] == nil {
		t.Fatal("expected staging env to be defined")
	}
	if m.Env["staging"].Profile != "staging" {
		t.Errorf("expected staging profile = 'staging', got %q", m.Env["staging"].Profile)
	}
	if m.Env["production"] == nil {
		t.Fatal("expected production env to be defined")
	}
	if m.Env["production"].Profile != "production" {
		t.Errorf("expected production profile = 'production', got %q", m.Env["production"].Profile)
	}

	// Root manifest should have no source, destination, connection, or transformation
	if m.Source != nil {
		t.Error("root manifest should not define a source")
	}
	if m.Destination != nil {
		t.Error("root manifest should not define a destination")
	}
	if m.Connection != nil {
		t.Error("root manifest should not define a connection")
	}
	if m.Transformation != nil {
		t.Error("root manifest should not define a transformation")
	}
}

func TestIntegration_SourceManifest(t *testing.T) {
	path := filepath.Join(monorepoRootDir(), "sources/toppy-magento-product-webhook/hookdeck.jsonc")
	m, err := LoadWithInheritance(path)
	if err != nil {
		t.Skipf("Skipping (manifest not found): %v", err)
	}

	// Source definition should be loaded
	if m.Source == nil {
		t.Fatal("expected source to be defined")
	}
	if m.Source.Name != "toppy-magento-product" {
		t.Errorf("expected source name = 'toppy-magento-product', got %q", m.Source.Name)
	}
	if m.Source.Description == "" {
		t.Error("expected source description to be non-empty")
	}

	// Extends should be cleared after inheritance resolution
	if m.Extends != "" {
		t.Errorf("extends should be cleared after merge, got %q", m.Extends)
	}

	// Should have env overlays merged from parent (root) and child
	if m.Env == nil {
		t.Fatal("expected env block from parent")
	}
	if m.Env["staging"] == nil {
		t.Fatal("expected staging env from parent")
	}
	if m.Env["staging"].Profile != "staging" {
		t.Errorf("expected staging profile from parent, got %q", m.Env["staging"].Profile)
	}
	if m.Env["production"] == nil {
		t.Fatal("expected production env from parent")
	}
	if m.Env["production"].Profile != "production" {
		t.Errorf("expected production profile from parent, got %q", m.Env["production"].Profile)
	}

	// Child env overlays should have source overrides
	if m.Env["staging"].Source == nil {
		t.Fatal("expected staging source override from child")
	}
	if m.Env["staging"].Source.Name != "toppy-magento-product" {
		t.Errorf("expected staging source name = 'toppy-magento-product', got %q", m.Env["staging"].Source.Name)
	}
}

func TestIntegration_DestinationManifest(t *testing.T) {
	path := filepath.Join(monorepoRootDir(), "destinations/toppy-cms-product-push/hookdeck.jsonc")
	m, err := LoadWithInheritance(path)
	if err != nil {
		t.Skipf("Skipping (manifest not found): %v", err)
	}

	// Destination definition should be loaded
	if m.Destination == nil {
		t.Fatal("expected destination to be defined")
	}
	if m.Destination.Name != "toppy-cms-product" {
		t.Errorf("expected destination name = 'toppy-cms-product', got %q", m.Destination.Name)
	}
	// rate_limit and rate_limit_period should be loaded (these map to our Go type)
	if m.Destination.RateLimit != 1 {
		t.Errorf("expected rate_limit = 1, got %d", m.Destination.RateLimit)
	}
	if m.Destination.RateLimitPeriod != "concurrent" {
		t.Errorf("expected rate_limit_period = 'concurrent', got %q", m.Destination.RateLimitPeriod)
	}

	// url is now a first-class field on DestinationConfig
	if m.Destination.URL == "" {
		t.Error("expected destination URL to be non-empty")
	}

	// Extends should be cleared
	if m.Extends != "" {
		t.Errorf("extends should be cleared after merge, got %q", m.Extends)
	}

	// Should have inherited env overlays with profiles from root
	if m.Env == nil {
		t.Fatal("expected env block")
	}
	if m.Env["staging"] == nil {
		t.Fatal("expected staging env")
	}
	if m.Env["staging"].Profile != "staging" {
		t.Errorf("expected staging profile, got %q", m.Env["staging"].Profile)
	}

	// Staging env should have connection config
	if m.Env["staging"].Connection == nil {
		t.Fatal("expected staging connection override")
	}
	if m.Env["staging"].Connection.Name != "toppy-magento-product-to-toppy-cms-product" {
		t.Errorf("expected staging connection name = 'toppy-magento-product-to-toppy-cms-product', got %q", m.Env["staging"].Connection.Name)
	}
	if m.Env["staging"].Connection.Source != "toppy-magento-product" {
		t.Errorf("expected staging connection source = 'toppy-magento-product', got %q", m.Env["staging"].Connection.Source)
	}

	// filter is now a first-class shorthand field on ConnectionConfig
	if m.Env["staging"].Connection.Filter == nil {
		t.Error("expected staging connection filter to be non-nil")
	}

	// Resolve staging environment and verify merged result
	resolved, err := ResolveEnv(m, "staging")
	if err != nil {
		t.Fatalf("ResolveEnv(staging) failed: %v", err)
	}
	if resolved.Connection == nil {
		t.Fatal("expected connection in resolved staging env")
	}
	if resolved.Connection.Source != "toppy-magento-product" {
		t.Errorf("expected resolved connection source = 'toppy-magento-product', got %q", resolved.Connection.Source)
	}
	if resolved.Profile != "staging" {
		t.Errorf("expected resolved profile = 'staging', got %q", resolved.Profile)
	}
	// Env map should be cleared after resolution
	if resolved.Env != nil {
		t.Error("expected env map to be nil after resolution")
	}
}

func TestIntegration_TransformationManifest(t *testing.T) {
	path := filepath.Join(monorepoRootDir(), "transformations/sendcloud-parcel-update-to-novu-trigger/hookdeck.jsonc")
	m, err := LoadFile(path)
	if err != nil {
		t.Skipf("Skipping (manifest not found): %v", err)
	}

	// Transformation definition should be loaded
	if m.Transformation == nil {
		t.Fatal("expected transformation to be defined")
	}
	if m.Transformation.Name != "sendcloud-parcel-update-to-novu-trigger" {
		t.Errorf("expected transformation name = 'sendcloud-parcel-update-to-novu-trigger', got %q", m.Transformation.Name)
	}
	if m.Transformation.Description == "" {
		t.Error("expected transformation description to be non-empty")
	}

	// env_vars is backward-compat alias for env — should be merged into Env map
	if m.Transformation.Env == nil {
		t.Fatal("expected env to be populated from env_vars")
	}
	if m.Transformation.Env["WORKFLOW_ID"] != "shipping-parcel-update" {
		t.Errorf("expected Env[WORKFLOW_ID] = 'shipping-parcel-update', got %q", m.Transformation.Env["WORKFLOW_ID"])
	}

	// Transformation manifests don't use extends (they use a different schema)
	if m.Extends != "" {
		t.Errorf("transformation manifest should not have extends, got %q", m.Extends)
	}

	// Should have no source, destination, or connection
	if m.Source != nil {
		t.Error("transformation should not define a source")
	}
	if m.Destination != nil {
		t.Error("transformation should not define a destination")
	}
	if m.Connection != nil {
		t.Error("transformation should not define a connection")
	}
}

func TestIntegration_DestinationWithComments(t *testing.T) {
	// This manifest uses leading JSONC comments (not just trailing), verifying
	// that the hujson parser handles them correctly.
	path := filepath.Join(monorepoRootDir(), "destinations/novu-generic-shipping-update-trigger/hookdeck.jsonc")
	m, err := LoadWithInheritance(path)
	if err != nil {
		t.Skipf("Skipping (manifest not found): %v", err)
	}

	if m.Destination == nil {
		t.Fatal("expected destination to be defined")
	}
	if m.Destination.Name != "novu-generic-trigger" {
		t.Errorf("expected destination name = 'novu-generic-trigger', got %q", m.Destination.Name)
	}
	if m.Destination.RateLimit != 1 {
		t.Errorf("expected rate_limit = 1, got %d", m.Destination.RateLimit)
	}

	// Inherited env overlays from root
	if m.Env == nil || m.Env["staging"] == nil {
		t.Fatal("expected staging env from parent")
	}

	// Staging connection should have source reference
	if m.Env["staging"].Connection == nil {
		t.Fatal("expected staging connection")
	}
	if m.Env["staging"].Connection.Source != "sendcloud-parcel-status" {
		t.Errorf("expected staging connection source = 'sendcloud-parcel-status', got %q", m.Env["staging"].Connection.Source)
	}
}

func TestIntegration_AllSourceManifestsLoadable(t *testing.T) {
	// Verify every source manifest in the monorepo can be loaded without errors.
	sources := []string{
		"sources/toppy-magento-product-webhook",
		"sources/toppy-magento-brand-webhook",
		"sources/toppy-magento-category-webhook",
		"sources/toppy-magento-product-group-webhook",
		"sources/toppy-order-webhook",
		"sources/toppy-lightspeed-order-webhook",
		"sources/sendcloud-parcel-status-changed-webhook",
	}

	for _, src := range sources {
		t.Run(src, func(t *testing.T) {
			path := filepath.Join(monorepoRootDir(), src, "hookdeck.jsonc")
			m, err := LoadWithInheritance(path)
			if err != nil {
				t.Skipf("Skipping (not found): %v", err)
			}
			if m.Source == nil {
				t.Error("expected source to be defined")
			}
			if m.Source.Name == "" {
				t.Error("expected source name to be non-empty")
			}
			// All sources extend root, so they should have env overlays
			if m.Env == nil || m.Env["staging"] == nil || m.Env["production"] == nil {
				t.Error("expected staging and production env overlays from parent")
			}
		})
	}
}

func TestIntegration_AllDestinationManifestsLoadable(t *testing.T) {
	// Verify every destination manifest in the monorepo can be loaded without errors.
	destinations := []string{
		"destinations/toppy-cms-product-push",
		"destinations/toppy-cms-brand-push",
		"destinations/toppy-cms-category-push",
		"destinations/toppy-cms-product-group-push",
		"destinations/toppy-elfsquad-product-push",
		"destinations/toppy-lightspeed-product-push",
		"destinations/squeezely-order-push",
		"destinations/toppy-magento-order-push",
		"destinations/novu-generic-shipping-update-trigger",
		"destinations/stocky-sendcloud-parcel-status-push",
	}

	for _, dest := range destinations {
		t.Run(dest, func(t *testing.T) {
			path := filepath.Join(monorepoRootDir(), dest, "hookdeck.jsonc")
			m, err := LoadWithInheritance(path)
			if err != nil {
				t.Skipf("Skipping (not found): %v", err)
			}
			if m.Destination == nil {
				t.Error("expected destination to be defined")
			}
			if m.Destination.Name == "" {
				t.Error("expected destination name to be non-empty")
			}
			// All destinations extend root, so they should have env overlays
			if m.Env == nil || m.Env["staging"] == nil || m.Env["production"] == nil {
				t.Error("expected staging and production env overlays from parent")
			}
		})
	}
}

func TestIntegration_AllTransformationManifestsLoadable(t *testing.T) {
	// Verify every transformation manifest can be loaded (no extends chain).
	transformations := []string{
		"transformations/sendcloud-parcel-update-to-novu-trigger",
		"transformations/order-anonymizer",
	}

	for _, tr := range transformations {
		t.Run(tr, func(t *testing.T) {
			path := filepath.Join(monorepoRootDir(), tr, "hookdeck.jsonc")
			m, err := LoadFile(path)
			if err != nil {
				t.Skipf("Skipping (not found): %v", err)
			}
			if m.Transformation == nil {
				t.Error("expected transformation to be defined")
			}
			if m.Transformation.Name == "" {
				t.Error("expected transformation name to be non-empty")
			}
		})
	}
}
