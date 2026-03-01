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
	"github.com/toppynl/hookdeck-deploy-cli/pkg/project"
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
	// Check if we should use project mode:
	// 1. --project flag was explicitly set, OR
	// 2. no --file flag and a hookdeck.project.jsonc/json exists in CWD
	if flagProject != "" || (flagFile == "" && projectFileExists()) {
		return runProjectDeploy()
	}
	return runSingleFileDeploy()
}

// runSingleFileDeploy handles the single manifest file deploy flow.
func runSingleFileDeploy() error {
	ctx := context.Background()

	// 1. Find and load manifest
	manifestPath, err := resolveManifestPath()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Loading manifest: %s\n", manifestPath)

	m, err := manifest.LoadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// 2. Resolve environment overrides per resource
	input := buildDeployInputFromManifest(m, flagEnv)

	// 3. Interpolate secrets (${ENV_VAR}) — operate on the manifest with resolved resources
	resolvedManifest := deployInputToManifest(input)
	if err := manifest.InterpolateEnvVars(resolvedManifest); err != nil {
		return fmt.Errorf("interpolating env vars: %w", err)
	}
	// Re-extract input after interpolation
	input = manifestToDeployInput(resolvedManifest)

	// 4. Resolve credentials
	profileName := flagProfile

	var client deploy.Client
	if !flagDryRun {
		creds, err := credentials.Resolve(profileName)
		if err != nil {
			return fmt.Errorf("resolving credentials: %w", err)
		}

		// 5. Create HTTP client for Hookdeck API
		client = hookdeck.NewClient(creds.APIKey, creds.ProjectID)
	}

	// 6. Run deploy orchestration
	manifestDir := filepath.Dir(manifestPath)
	opts := deploy.Options{
		DryRun:   flagDryRun,
		CodeRoot: manifestDir,
	}

	if flagDryRun {
		fmt.Fprintln(os.Stderr, "Dry-run mode: no changes will be applied")
	}

	result, err := deploy.Deploy(ctx, client, input, opts)
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	// 7. Print results
	printDeployResult(result)

	// 8. Wrangler sync (if --sync-wrangler and at least one source was deployed)
	if flagSyncWrangler && !flagDryRun && len(result.Sources) > 0 && result.Sources[0].ID != "" {
		if err := syncWrangler(manifestDir, result.Sources[0].ID); err != nil {
			// Wrangler sync is best-effort; warn but don't fail
			fmt.Fprintf(os.Stderr, "Warning: wrangler sync failed: %v\n", err)
		}
	}

	return nil
}

// runProjectDeploy handles the project-wide deploy flow.
func runProjectDeploy() error {
	ctx := context.Background()

	// 1. Resolve project path
	projectPath, err := resolveProjectPath()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Loading project: %s\n", projectPath)

	// 2. Load project (config + discover manifests + registry)
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	// 3. Resolve profile from project config env or --profile flag
	profileName := flagProfile
	if profileName == "" && proj.Config.Env != nil && flagEnv != "" {
		if envCfg, ok := proj.Config.Env[flagEnv]; ok && envCfg.Profile != "" {
			profileName = envCfg.Profile
		}
	}

	// 4. Build DeployInput from registry with env overrides
	input := buildDeployInputFromRegistry(proj.Registry, flagEnv)

	// 5. Interpolate env vars
	resolvedManifest := deployInputToManifest(input)
	if err := manifest.InterpolateEnvVars(resolvedManifest); err != nil {
		return fmt.Errorf("interpolating env vars: %w", err)
	}
	input = manifestToDeployInput(resolvedManifest)

	// 6. Resolve credentials and create client
	var client deploy.Client
	if !flagDryRun {
		creds, err := credentials.Resolve(profileName)
		if err != nil {
			return fmt.Errorf("resolving credentials: %w", err)
		}
		client = hookdeck.NewClient(creds.APIKey, creds.ProjectID)
	}

	// 7. Deploy
	opts := deploy.Options{
		DryRun:   flagDryRun,
		CodeRoot: proj.RootDir,
	}

	if flagDryRun {
		fmt.Fprintln(os.Stderr, "Dry-run mode: no changes will be applied")
	}

	result, err := deploy.Deploy(ctx, client, input, opts)
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	// 8. Print results
	printDeployResult(result)

	return nil
}

// buildDeployInputFromManifest constructs a DeployInput from a loaded manifest,
// applying per-resource environment overrides.
func buildDeployInputFromManifest(m *manifest.Manifest, envName string) *deploy.DeployInput {
	input := &deploy.DeployInput{}

	for i := range m.Sources {
		resolved := manifest.ResolveSourceEnv(&m.Sources[i], envName)
		input.Sources = append(input.Sources, resolved)
	}
	for i := range m.Destinations {
		resolved := manifest.ResolveDestinationEnv(&m.Destinations[i], envName)
		input.Destinations = append(input.Destinations, resolved)
	}
	for i := range m.Transformations {
		resolved := manifest.ResolveTransformationEnv(&m.Transformations[i], envName)
		input.Transformations = append(input.Transformations, resolved)
	}
	for i := range m.Connections {
		conn := m.Connections[i]
		input.Connections = append(input.Connections, &conn)
	}

	return input
}

// buildDeployInputFromRegistry constructs a DeployInput from a project registry,
// applying per-resource environment overrides.
func buildDeployInputFromRegistry(reg *project.Registry, envName string) *deploy.DeployInput {
	input := &deploy.DeployInput{}

	for i := range reg.SourceList {
		resolved := manifest.ResolveSourceEnv(&reg.SourceList[i], envName)
		input.Sources = append(input.Sources, resolved)
	}
	for i := range reg.DestinationList {
		resolved := manifest.ResolveDestinationEnv(&reg.DestinationList[i], envName)
		input.Destinations = append(input.Destinations, resolved)
	}
	for i := range reg.TransformationList {
		resolved := manifest.ResolveTransformationEnv(&reg.TransformationList[i], envName)
		input.Transformations = append(input.Transformations, resolved)
	}
	for i := range reg.ConnectionList {
		conn := reg.ConnectionList[i]
		input.Connections = append(input.Connections, &conn)
	}

	return input
}

// deployInputToManifest converts a DeployInput back to a Manifest for interpolation.
func deployInputToManifest(input *deploy.DeployInput) *manifest.Manifest {
	m := &manifest.Manifest{}
	for _, src := range input.Sources {
		m.Sources = append(m.Sources, *src)
	}
	for _, dst := range input.Destinations {
		m.Destinations = append(m.Destinations, *dst)
	}
	for _, tr := range input.Transformations {
		m.Transformations = append(m.Transformations, *tr)
	}
	for _, conn := range input.Connections {
		m.Connections = append(m.Connections, *conn)
	}
	return m
}

// manifestToDeployInput converts a Manifest to a DeployInput (pointers into the manifest slices).
func manifestToDeployInput(m *manifest.Manifest) *deploy.DeployInput {
	input := &deploy.DeployInput{}
	for i := range m.Sources {
		input.Sources = append(input.Sources, &m.Sources[i])
	}
	for i := range m.Destinations {
		input.Destinations = append(input.Destinations, &m.Destinations[i])
	}
	for i := range m.Transformations {
		input.Transformations = append(input.Transformations, &m.Transformations[i])
	}
	for i := range m.Connections {
		input.Connections = append(input.Connections, &m.Connections[i])
	}
	return input
}

// resolveProjectPath determines which project config file to use.
func resolveProjectPath() (string, error) {
	if flagProject != "" {
		if _, err := os.Stat(flagProject); err != nil {
			return "", fmt.Errorf("project file not found: %s", flagProject)
		}
		return flagProject, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	for _, name := range []string{"hookdeck.project.jsonc", "hookdeck.project.json"} {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no hookdeck.project.jsonc or hookdeck.project.json found in %s", cwd)
}

// projectFileExists checks if a hookdeck.project.jsonc or hookdeck.project.json file
// exists in the current working directory.
func projectFileExists() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	for _, name := range []string{"hookdeck.project.jsonc", "hookdeck.project.json"} {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
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

	for _, name := range []string{"hookdeck.jsonc", "hookdeck.json"} {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no hookdeck.jsonc or hookdeck.json found in %s", cwd)
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
	envName := flagEnv
	if envName == "" {
		envName = "staging" // default environment for wrangler sync
	}

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
	for _, r := range result.Sources {
		printResourceResult("Source", r)
	}
	for _, r := range result.Transformations {
		printResourceResult("Transformation", r)
	}
	for _, r := range result.Destinations {
		printResourceResult("Destination", r)
	}
	for _, r := range result.Connections {
		printResourceResult("Connection", r)
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
