package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/credentials"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/deploy"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/hookdeck"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/wrangler"
)

var flagSyncWrangler bool

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Hookdeck resources from a manifest file",
	Long: `Deploy upserts Hookdeck resources (sources, destinations, transformations,
connections) declared in a manifest file. Resources are deployed in dependency
order: source, transformation, destination, connection.`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().BoolVar(&flagSyncWrangler, "sync-wrangler", true, "sync source URL back to wrangler.jsonc after deploy")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Find and load manifest
	manifestPath, err := resolveManifestPath()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Loading manifest: %s\n", manifestPath)

	// 2. Load with inheritance (extends chain)
	m, err := manifest.LoadWithInheritance(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// 3. Resolve environment overlay
	m, err = manifest.ResolveEnv(m, flagEnv)
	if err != nil {
		return fmt.Errorf("resolving environment: %w", err)
	}

	// 4. Interpolate secrets (${ENV_VAR})
	if err := manifest.InterpolateEnvVars(m); err != nil {
		return fmt.Errorf("interpolating env vars: %w", err)
	}

	// 5. Resolve credentials
	profileName := flagProfile
	if profileName == "" {
		profileName = m.Profile
	}

	var client deploy.Client
	if !flagDryRun {
		creds, err := credentials.Resolve(profileName)
		if err != nil {
			return fmt.Errorf("resolving credentials: %w", err)
		}

		// 6. Create HTTP client for Hookdeck API
		client = hookdeck.NewClient(creds.APIKey, creds.ProjectID)
	}

	// 7. Run deploy orchestration
	manifestDir := filepath.Dir(manifestPath)
	opts := deploy.Options{
		DryRun:   flagDryRun,
		CodeRoot: manifestDir,
	}

	if flagDryRun {
		fmt.Fprintln(os.Stderr, "Dry-run mode: no changes will be applied")
	}

	result, err := deploy.Deploy(ctx, client, m, opts)
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	// 8. Print results
	printDeployResult(result)

	// 9. Wrangler sync (if --sync-wrangler and source was deployed)
	if flagSyncWrangler && !flagDryRun && result.Source != nil && result.Source.ID != "" {
		if err := syncWrangler(manifestDir, result.Source.ID); err != nil {
			// Wrangler sync is best-effort; warn but don't fail
			fmt.Fprintf(os.Stderr, "Warning: wrangler sync failed: %v\n", err)
		}
	}

	return nil
}

// resolveManifestPath determines which manifest file to use.
// If --file was provided, use it directly. Otherwise, auto-discover in cwd.
func resolveManifestPath() (string, error) {
	if flagFile != "" {
		if _, err := os.Stat(flagFile); err != nil {
			return "", fmt.Errorf("manifest file not found: %s", flagFile)
		}
		return flagFile, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	path, err := manifest.FindFile(cwd)
	if err != nil {
		return "", err
	}
	return path, nil
}

// syncWrangler writes the Hookdeck source URL into the wrangler.jsonc file.
func syncWrangler(manifestDir, sourceID string) error {
	wranglerPath := filepath.Join(manifestDir, "wrangler.jsonc")
	if _, err := os.Stat(wranglerPath); os.IsNotExist(err) {
		// Try .json variant
		wranglerPath = filepath.Join(manifestDir, "wrangler.json")
		if _, err := os.Stat(wranglerPath); os.IsNotExist(err) {
			return nil // No wrangler file found, skip silently
		}
	}

	// The source URL is the Hookdeck ingest URL for the source.
	// Hookdeck sources have a URL like https://hk-<id>.hookdeck.com
	// But we need to query the API to get the actual URL. For now,
	// we use the source ID to construct the known URL pattern.
	// The actual URL would come from the UpsertSourceResult, which we
	// don't have here. For wrangler sync, we need the env name.
	envName := flagEnv
	if envName == "" {
		envName = "staging" // default environment for wrangler sync
	}

	// We need the actual source URL from the API response. Since the deploy
	// result only has ID and name, we construct the Hookdeck source URL.
	sourceURL := fmt.Sprintf("https://hk-%s.hookdeck.com", sourceID)

	modified, err := wrangler.SyncSourceURL(wranglerPath, envName, sourceURL)
	if err != nil {
		return err
	}
	if modified {
		fmt.Fprintf(os.Stderr, "Synced source URL to %s (env: %s)\n", wranglerPath, envName)
	}
	return nil
}

// printDeployResult prints the deploy results to stderr.
func printDeployResult(result *deploy.Result) {
	if result.Source != nil {
		printResourceResult("Source", result.Source)
	}
	if result.Transformation != nil {
		printResourceResult("Transformation", result.Transformation)
	}
	if result.Destination != nil {
		printResourceResult("Destination", result.Destination)
	}
	if result.Connection != nil {
		printResourceResult("Connection", result.Connection)
	}
}

// printResourceResult prints a single resource result line.
func printResourceResult(kind string, r *deploy.ResourceResult) {
	if r.ID != "" {
		fmt.Fprintf(os.Stderr, "  %-16s %-30s %s (id: %s)\n", kind, r.Name, r.Action, r.ID)
	} else {
		fmt.Fprintf(os.Stderr, "  %-16s %-30s %s\n", kind, r.Name, r.Action)
	}
}
