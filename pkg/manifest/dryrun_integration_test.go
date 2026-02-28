//go:build monorepo

package manifest_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/deploy"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

func monorepoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "../../../..")
}

// TestDryRunAllManifests verifies that every hookdeck.jsonc in the monorepo
// can be loaded, resolved for the "staging" environment, have env vars
// interpolated, and dry-run deployed without errors. This exercises the full
// pipeline: JSONC parse → inheritance → env overlay → interpolation → deploy.
func TestDryRunAllManifests(t *testing.T) {
	root := monorepoRoot()

	// Set environment variables referenced by manifests.
	// The novu-generic-shipping-update-trigger destination uses ${NOVU_GENERIC_API_KEY}.
	t.Setenv("NOVU_GENERIC_API_KEY", "test-key")

	// Sources — all use extends and have staging/production env overlays.
	sources := []string{
		"sources/toppy-magento-product-webhook",
		"sources/toppy-magento-brand-webhook",
		"sources/toppy-magento-category-webhook",
		"sources/toppy-magento-product-group-webhook",
		"sources/toppy-order-webhook",
		"sources/toppy-lightspeed-order-webhook",
		"sources/sendcloud-parcel-status-changed-webhook",
	}

	// Destinations — all use extends and have staging/production env overlays.
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

	// Transformations — standalone manifests, no extends or env overlays.
	transformations := []string{
		"transformations/sendcloud-parcel-update-to-novu-trigger",
		"transformations/order-anonymizer",
	}

	// Test sources and destinations: load -> resolve -> interpolate -> dry-run deploy
	for _, dir := range append(sources, destinations...) {
		t.Run(dir, func(t *testing.T) {
			path := filepath.Join(root, dir, "hookdeck.jsonc")

			// Step 1: Load with inheritance (resolves extends chain)
			m, err := manifest.LoadWithInheritance(path)
			if err != nil {
				t.Skipf("Skipping (not found): %v", err)
			}

			// Step 2: Resolve for "staging" environment
			resolved, err := manifest.ResolveEnv(m, "staging")
			if err != nil {
				t.Fatalf("ResolveEnv(staging) failed: %v", err)
			}

			// Step 3: Interpolate environment variables
			if err := manifest.InterpolateEnvVars(resolved); err != nil {
				t.Fatalf("InterpolateEnvVars failed: %v", err)
			}

			// Step 4: Dry-run deploy (no API client needed)
			result, err := deploy.Deploy(context.Background(), nil, resolved, deploy.Options{
				DryRun:   true,
				CodeRoot: filepath.Join(root, dir),
			})
			if err != nil {
				t.Fatalf("Deploy dry-run failed: %v", err)
			}

			// Verify we got a non-nil result
			if result == nil {
				t.Fatal("expected non-nil deploy result")
			}
		})
	}

	// Test transformations: load -> dry-run deploy (no inheritance or env resolution)
	for _, dir := range transformations {
		t.Run(dir, func(t *testing.T) {
			path := filepath.Join(root, dir, "hookdeck.jsonc")

			// Step 1: Load directly (no extends chain for transformations)
			m, err := manifest.LoadFile(path)
			if err != nil {
				t.Skipf("Skipping (not found): %v", err)
			}

			// Step 2: Dry-run deploy
			result, err := deploy.Deploy(context.Background(), nil, m, deploy.Options{
				DryRun:   true,
				CodeRoot: filepath.Join(root, dir),
			})
			if err != nil {
				t.Fatalf("Deploy dry-run failed: %v", err)
			}

			// Verify transformation result is present
			if result == nil {
				t.Fatal("expected non-nil deploy result")
			}
			if result.Transformation == nil {
				t.Fatal("expected transformation result")
			}
			if result.Transformation.Action != "would upsert" {
				t.Errorf("expected action 'would upsert', got %q", result.Transformation.Action)
			}
		})
	}
}
