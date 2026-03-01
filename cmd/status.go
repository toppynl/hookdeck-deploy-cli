package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/credentials"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/hookdeck"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of Hookdeck resources defined in a manifest",
	Long: `Status checks whether each resource declared in a manifest file exists on
Hookdeck. For each resource it prints the name, ID, and URL (for sources).`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Find and load manifest (same resolution as deploy)
	manifestPath, err := resolveManifestPath()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Loading manifest: %s\n", manifestPath)

	// 2. Load manifest
	m, err := manifest.LoadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// 3. Resolve environment overrides per resource and rebuild manifest for interpolation
	resolvedManifest := &manifest.Manifest{}
	for i := range m.Sources {
		resolved := manifest.ResolveSourceEnv(&m.Sources[i], flagEnv)
		resolvedManifest.Sources = append(resolvedManifest.Sources, *resolved)
	}
	for i := range m.Destinations {
		resolved := manifest.ResolveDestinationEnv(&m.Destinations[i], flagEnv)
		resolvedManifest.Destinations = append(resolvedManifest.Destinations, *resolved)
	}
	for i := range m.Transformations {
		resolved := manifest.ResolveTransformationEnv(&m.Transformations[i], flagEnv)
		resolvedManifest.Transformations = append(resolvedManifest.Transformations, *resolved)
	}
	resolvedManifest.Connections = m.Connections

	// 4. Interpolate env vars (needed to resolve names that use ${VAR})
	if err := manifest.InterpolateEnvVars(resolvedManifest); err != nil {
		return fmt.Errorf("interpolating env vars: %w", err)
	}

	// 5. Resolve credentials
	profileName := flagProfile

	creds, err := credentials.Resolve(profileName)
	if err != nil {
		return fmt.Errorf("resolving credentials: %w", err)
	}

	client := hookdeck.NewClient(creds.APIKey, creds.ProjectID)

	// 6. Check each resource
	fmt.Fprintln(os.Stderr)

	hasResources := false

	if len(resolvedManifest.Sources) > 0 {
		hasResources = true
		printStatusHeader("Sources")
		for _, src := range resolvedManifest.Sources {
			info, err := client.FindSourceByName(ctx, src.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", src.Name, err)
			} else if info == nil {
				fmt.Fprintf(os.Stderr, "  %-30s not found\n", src.Name)
			} else {
				line := fmt.Sprintf("  %-30s id: %s", info.Name, info.ID)
				if info.URL != "" {
					line += fmt.Sprintf("  url: %s", info.URL)
				}
				fmt.Fprintln(os.Stderr, line)
			}
		}
	}

	if len(resolvedManifest.Transformations) > 0 {
		hasResources = true
		printStatusHeader("Transformations")
		for _, tr := range resolvedManifest.Transformations {
			info, err := client.FindTransformationByName(ctx, tr.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", tr.Name, err)
			} else if info == nil {
				fmt.Fprintf(os.Stderr, "  %-30s not found\n", tr.Name)
			} else {
				fmt.Fprintf(os.Stderr, "  %-30s id: %s\n", info.Name, info.ID)
			}
		}
	}

	if len(resolvedManifest.Destinations) > 0 {
		hasResources = true
		printStatusHeader("Destinations")
		for _, dst := range resolvedManifest.Destinations {
			info, err := client.FindDestinationByName(ctx, dst.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", dst.Name, err)
			} else if info == nil {
				fmt.Fprintf(os.Stderr, "  %-30s not found\n", dst.Name)
			} else {
				fmt.Fprintf(os.Stderr, "  %-30s id: %s\n", info.Name, info.ID)
			}
		}
	}

	if len(resolvedManifest.Connections) > 0 {
		hasResources = true
		printStatusHeader("Connections")
		for _, conn := range resolvedManifest.Connections {
			info, err := client.FindConnectionByFullName(ctx, conn.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", conn.Name, err)
			} else if info == nil {
				fmt.Fprintf(os.Stderr, "  %-30s not found\n", conn.Name)
			} else {
				fmt.Fprintf(os.Stderr, "  %-30s id: %s\n", info.Name, info.ID)
			}
		}
	}

	if !hasResources {
		fmt.Fprintln(os.Stderr, "No resources defined in manifest.")
	}

	fmt.Fprintln(os.Stderr)

	return nil
}

// printStatusHeader prints a section header for resource status output.
func printStatusHeader(kind string) {
	fmt.Fprintf(os.Stderr, "%s:\n", kind)
}
