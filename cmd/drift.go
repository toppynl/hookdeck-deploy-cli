package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/credentials"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/drift"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/hookdeck"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between manifest and live Hookdeck resources",
	Long: `Drift compares the resources declared in a manifest file against their
current state on Hookdeck. Reports resources that are missing, drifted
(field values differ), or in sync.`,
	RunE: runDrift,
}

func init() {
	rootCmd.AddCommand(driftCmd)
}

func runDrift(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Load and resolve manifest (same as deploy)
	manifestPath, err := resolveManifestPath()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Loading manifest: %s\n", manifestPath)

	m, err := manifest.LoadWithInheritance(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	m, err = manifest.ResolveEnv(m, flagEnv)
	if err != nil {
		return fmt.Errorf("resolving environment: %w", err)
	}

	if err := manifest.InterpolateEnvVars(m); err != nil {
		return fmt.Errorf("interpolating env vars: %w", err)
	}

	// 2. Resolve credentials
	profileName := flagProfile
	if profileName == "" {
		profileName = m.Profile
	}
	creds, err := credentials.Resolve(profileName)
	if err != nil {
		return fmt.Errorf("resolving credentials: %w", err)
	}

	client := hookdeck.NewClient(creds.APIKey, creds.ProjectID)

	// 3. Fetch remote state
	fmt.Fprintln(os.Stderr, "Fetching remote state...")
	remote, err := fetchRemoteState(ctx, client, m)
	if err != nil {
		return fmt.Errorf("fetching remote state: %w", err)
	}

	// 4. Detect drift
	diffs := drift.Detect(m, remote)

	// 5. Print results
	if len(diffs) == 0 {
		fmt.Fprintln(os.Stderr, "\nAll resources in sync.")
		return nil
	}

	fmt.Fprintln(os.Stderr)
	for _, d := range diffs {
		switch d.Status {
		case drift.Missing:
			fmt.Fprintf(os.Stderr, "  %-16s %-30s MISSING (not found on Hookdeck)\n", d.Kind, d.Name)
		case drift.Drifted:
			fmt.Fprintf(os.Stderr, "  %-16s %-30s DRIFTED\n", d.Kind, d.Name)
			for _, f := range d.Fields {
				fmt.Fprintf(os.Stderr, "    %-20s local: %s\n", f.Field, f.Local)
				fmt.Fprintf(os.Stderr, "    %-20s remote: %s\n", "", f.Remote)
			}
		}
	}
	fmt.Fprintln(os.Stderr)

	return fmt.Errorf("drift detected: %d resource(s) out of sync", len(diffs))
}

func fetchRemoteState(ctx context.Context, client *hookdeck.Client, m *manifest.Manifest) (*drift.RemoteState, error) {
	remote := &drift.RemoteState{}

	if m.Source != nil {
		src, err := client.GetSourceByName(ctx, m.Source.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching source %q: %w", m.Source.Name, err)
		}
		remote.Source = src
	}

	if m.Destination != nil {
		dst, err := client.GetDestinationByName(ctx, m.Destination.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching destination %q: %w", m.Destination.Name, err)
		}
		remote.Destination = dst
	}

	if m.Connection != nil {
		conn, err := client.GetConnectionByFullName(ctx, m.Connection.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching connection %q: %w", m.Connection.Name, err)
		}
		remote.Connection = conn
	}

	if m.Transformation != nil {
		tr, err := client.GetTransformationByName(ctx, m.Transformation.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching transformation %q: %w", m.Transformation.Name, err)
		}
		remote.Transformation = tr
	}

	return remote, nil
}
