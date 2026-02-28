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

	// 2. Load with inheritance
	m, err := manifest.LoadWithInheritance(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// 3. Resolve environment overlay
	m, err = manifest.ResolveEnv(m, flagEnv)
	if err != nil {
		return fmt.Errorf("resolving environment: %w", err)
	}

	// 4. Interpolate env vars (needed to resolve names that use ${VAR})
	if err := manifest.InterpolateEnvVars(m); err != nil {
		return fmt.Errorf("interpolating env vars: %w", err)
	}

	// 5. Resolve credentials
	profileName := flagProfile
	if profileName == "" {
		profileName = m.Profile
	}

	creds, err := credentials.Resolve(profileName)
	if err != nil {
		return fmt.Errorf("resolving credentials: %w", err)
	}

	client := hookdeck.NewClient(creds.APIKey, creds.ProjectID)

	// 6. Check each resource
	fmt.Fprintln(os.Stderr)

	hasResources := false

	if m.Source != nil {
		hasResources = true
		printStatusHeader("Source")
		info, err := client.FindSourceByName(ctx, m.Source.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", m.Source.Name, err)
		} else if info == nil {
			fmt.Fprintf(os.Stderr, "  %-30s not found\n", m.Source.Name)
		} else {
			line := fmt.Sprintf("  %-30s id: %s", info.Name, info.ID)
			if info.URL != "" {
				line += fmt.Sprintf("  url: %s", info.URL)
			}
			fmt.Fprintln(os.Stderr, line)
		}
	}

	if m.Transformation != nil {
		hasResources = true
		printStatusHeader("Transformation")
		info, err := client.FindTransformationByName(ctx, m.Transformation.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", m.Transformation.Name, err)
		} else if info == nil {
			fmt.Fprintf(os.Stderr, "  %-30s not found\n", m.Transformation.Name)
		} else {
			fmt.Fprintf(os.Stderr, "  %-30s id: %s\n", info.Name, info.ID)
		}
	}

	if m.Destination != nil {
		hasResources = true
		printStatusHeader("Destination")
		info, err := client.FindDestinationByName(ctx, m.Destination.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", m.Destination.Name, err)
		} else if info == nil {
			fmt.Fprintf(os.Stderr, "  %-30s not found\n", m.Destination.Name)
		} else {
			fmt.Fprintf(os.Stderr, "  %-30s id: %s\n", info.Name, info.ID)
		}
	}

	if m.Connection != nil {
		hasResources = true
		printStatusHeader("Connection")
		info, err := client.FindConnectionByFullName(ctx, m.Connection.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", m.Connection.Name, err)
		} else if info == nil {
			fmt.Fprintf(os.Stderr, "  %-30s not found\n", m.Connection.Name)
		} else {
			fmt.Fprintf(os.Stderr, "  %-30s id: %s\n", info.Name, info.ID)
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
