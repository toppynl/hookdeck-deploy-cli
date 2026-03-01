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

	// 1. Load and resolve manifest
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
	var sources []*manifest.SourceConfig
	for i := range m.Sources {
		sources = append(sources, manifest.ResolveSourceEnv(&m.Sources[i], flagEnv))
	}

	var destinations []*manifest.DestinationConfig
	for i := range m.Destinations {
		destinations = append(destinations, manifest.ResolveDestinationEnv(&m.Destinations[i], flagEnv))
	}

	var transformations []*manifest.TransformationConfig
	for i := range m.Transformations {
		transformations = append(transformations, manifest.ResolveTransformationEnv(&m.Transformations[i], flagEnv))
	}

	var connections []*manifest.ConnectionConfig
	for i := range m.Connections {
		conn := m.Connections[i]
		connections = append(connections, &conn)
	}

	// 3. Interpolate env vars — rebuild a manifest for interpolation
	resolvedManifest := &manifest.Manifest{}
	for _, src := range sources {
		resolvedManifest.Sources = append(resolvedManifest.Sources, *src)
	}
	for _, dst := range destinations {
		resolvedManifest.Destinations = append(resolvedManifest.Destinations, *dst)
	}
	for _, tr := range transformations {
		resolvedManifest.Transformations = append(resolvedManifest.Transformations, *tr)
	}
	for _, conn := range connections {
		resolvedManifest.Connections = append(resolvedManifest.Connections, *conn)
	}

	if err := manifest.InterpolateEnvVars(resolvedManifest); err != nil {
		return fmt.Errorf("interpolating env vars: %w", err)
	}

	// Re-extract pointers after interpolation
	sources = nil
	for i := range resolvedManifest.Sources {
		sources = append(sources, &resolvedManifest.Sources[i])
	}
	destinations = nil
	for i := range resolvedManifest.Destinations {
		destinations = append(destinations, &resolvedManifest.Destinations[i])
	}
	transformations = nil
	for i := range resolvedManifest.Transformations {
		transformations = append(transformations, &resolvedManifest.Transformations[i])
	}
	connections = nil
	for i := range resolvedManifest.Connections {
		connections = append(connections, &resolvedManifest.Connections[i])
	}

	// 4. Resolve credentials
	profileName := flagProfile
	creds, err := credentials.Resolve(profileName)
	if err != nil {
		return fmt.Errorf("resolving credentials: %w", err)
	}

	client := hookdeck.NewClient(creds.APIKey, creds.ProjectID)

	// 5. Fetch remote state and detect drift for each resource
	fmt.Fprintln(os.Stderr, "Fetching remote state...")
	remote, err := fetchRemoteState(ctx, client, sources, destinations, transformations, connections)
	if err != nil {
		return fmt.Errorf("fetching remote state: %w", err)
	}

	// 6. Detect drift
	diffs := drift.Detect(sources, destinations, transformations, connections, remote)

	// 7. Print results
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

func fetchRemoteState(
	ctx context.Context,
	client *hookdeck.Client,
	sources []*manifest.SourceConfig,
	destinations []*manifest.DestinationConfig,
	transformations []*manifest.TransformationConfig,
	connections []*manifest.ConnectionConfig,
) (*drift.RemoteState, error) {
	remote := &drift.RemoteState{}

	for _, src := range sources {
		detail, err := client.GetSourceByName(ctx, src.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching source %q: %w", src.Name, err)
		}
		remote.Sources = append(remote.Sources, detail)
	}

	for _, dst := range destinations {
		detail, err := client.GetDestinationByName(ctx, dst.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching destination %q: %w", dst.Name, err)
		}
		remote.Destinations = append(remote.Destinations, detail)
	}

	for _, conn := range connections {
		detail, err := client.GetConnectionByFullName(ctx, conn.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching connection %q: %w", conn.Name, err)
		}
		remote.Connections = append(remote.Connections, detail)
	}

	for _, tr := range transformations {
		detail, err := client.GetTransformationByName(ctx, tr.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching transformation %q: %w", tr.Name, err)
		}
		remote.Transformations = append(remote.Transformations, detail)
	}

	return remote, nil
}
