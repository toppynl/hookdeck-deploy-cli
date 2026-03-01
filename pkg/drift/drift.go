// Package drift compares a resolved manifest against live Hookdeck resource
// state and produces a diff report describing what is missing or drifted.
package drift

import (
	"fmt"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/hookdeck"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

// DriftStatus describes the state of a resource compared to its manifest definition.
type DriftStatus string

const (
	// Missing means the resource exists in the manifest but not remotely.
	Missing DriftStatus = "missing"
	// Drifted means the resource exists remotely but one or more fields differ.
	Drifted DriftStatus = "drifted"
	// InSync means the resource matches its manifest definition.
	InSync DriftStatus = "in_sync"
)

// Diff describes the drift status of a single resource.
type Diff struct {
	Kind   string      // "source", "destination", "connection", "transformation"
	Name   string      // resource name
	Status DriftStatus // missing, drifted, or in_sync
	Fields []FieldDiff // populated when Status == Drifted
}

// FieldDiff describes a single field that has drifted.
type FieldDiff struct {
	Field  string // field name (e.g. "url", "env.KEY")
	Local  string // value from the manifest
	Remote string // value from the live resource
}

// RemoteState holds the live Hookdeck resources to compare against a manifest.
// Each slice is positionally aligned with the corresponding manifest resource list.
type RemoteState struct {
	Sources         []*hookdeck.SourceDetail
	Destinations    []*hookdeck.DestinationDetail
	Connections     []*hookdeck.ConnectionDetail
	Transformations []*hookdeck.TransformationDetail
}

// Detect compares resolved manifest resources against remote state and returns a list
// of diffs. Only resources defined in the manifest are checked; extra remote
// resources are ignored. A nil return (or empty slice) means everything is in sync.
//
// The remote slices are expected to be positionally aligned with the local slices
// (i.e., remote.Sources[0] corresponds to sources[0], etc.). A nil entry in a remote
// slice means the resource was not found remotely.
func Detect(
	sources []*manifest.SourceConfig,
	destinations []*manifest.DestinationConfig,
	transformations []*manifest.TransformationConfig,
	connections []*manifest.ConnectionConfig,
	remote *RemoteState,
) []Diff {
	var diffs []Diff

	for i, src := range sources {
		var remoteSrc *hookdeck.SourceDetail
		if i < len(remote.Sources) {
			remoteSrc = remote.Sources[i]
		}
		if d := detectSource(src, remoteSrc); d != nil {
			diffs = append(diffs, *d)
		}
	}

	for i, dst := range destinations {
		var remoteDst *hookdeck.DestinationDetail
		if i < len(remote.Destinations) {
			remoteDst = remote.Destinations[i]
		}
		if d := detectDestination(dst, remoteDst); d != nil {
			diffs = append(diffs, *d)
		}
	}

	for i, conn := range connections {
		var remoteConn *hookdeck.ConnectionDetail
		if i < len(remote.Connections) {
			remoteConn = remote.Connections[i]
		}
		if d := detectConnection(conn, remoteConn); d != nil {
			diffs = append(diffs, *d)
		}
	}

	for i, tr := range transformations {
		var remoteTr *hookdeck.TransformationDetail
		if i < len(remote.Transformations) {
			remoteTr = remote.Transformations[i]
		}
		if d := detectTransformation(tr, remoteTr); d != nil {
			diffs = append(diffs, *d)
		}
	}

	return diffs
}

// detectSource checks a source config against its live state.
func detectSource(local *manifest.SourceConfig, remote *hookdeck.SourceDetail) *Diff {
	if remote == nil {
		return &Diff{Kind: "source", Name: local.Name, Status: Missing}
	}

	var fields []FieldDiff
	if local.Name != remote.Name {
		fields = append(fields, FieldDiff{"name", local.Name, remote.Name})
	}
	if local.Description != "" && local.Description != remote.Description {
		fields = append(fields, FieldDiff{"description", local.Description, remote.Description})
	}

	if len(fields) > 0 {
		return &Diff{Kind: "source", Name: local.Name, Status: Drifted, Fields: fields}
	}
	return nil
}

// detectDestination checks a destination config against its live state.
func detectDestination(local *manifest.DestinationConfig, remote *hookdeck.DestinationDetail) *Diff {
	if remote == nil {
		return &Diff{Kind: "destination", Name: local.Name, Status: Missing}
	}

	// The API returns url, auth_type, rate_limit, rate_limit_period inside config.
	cfg := remote.Config
	var fields []FieldDiff
	if local.URL != "" && local.URL != cfg.URL {
		fields = append(fields, FieldDiff{"url", local.URL, cfg.URL})
	}
	if local.AuthType != "" && local.AuthType != cfg.AuthType {
		fields = append(fields, FieldDiff{"auth_type", local.AuthType, cfg.AuthType})
	}
	if local.RateLimit != 0 && local.RateLimit != cfg.RateLimit {
		fields = append(fields, FieldDiff{"rate_limit", fmt.Sprint(local.RateLimit), fmt.Sprint(cfg.RateLimit)})
	}
	if local.RateLimitPeriod != "" && local.RateLimitPeriod != cfg.RateLimitPeriod {
		fields = append(fields, FieldDiff{"rate_limit_period", local.RateLimitPeriod, cfg.RateLimitPeriod})
	}

	if len(fields) > 0 {
		return &Diff{Kind: "destination", Name: local.Name, Status: Drifted, Fields: fields}
	}
	return nil
}

// detectConnection checks a connection config against its live state.
func detectConnection(local *manifest.ConnectionConfig, remote *hookdeck.ConnectionDetail) *Diff {
	if remote == nil {
		return &Diff{Kind: "connection", Name: local.Name, Status: Missing}
	}

	var fields []FieldDiff
	// Connection drift is currently limited to existence checks.
	// Future: compare rules, filter, transformations.

	if len(fields) > 0 {
		return &Diff{Kind: "connection", Name: local.Name, Status: Drifted, Fields: fields}
	}
	return nil
}

// detectTransformation checks a transformation config against its live state.
func detectTransformation(local *manifest.TransformationConfig, remote *hookdeck.TransformationDetail) *Diff {
	if remote == nil {
		return &Diff{Kind: "transformation", Name: local.Name, Status: Missing}
	}

	var fields []FieldDiff

	// Check env vars — each key defined locally must match the remote value.
	for k, v := range local.Env {
		if remoteVal, ok := remote.Env[k]; !ok || remoteVal != v {
			fields = append(fields, FieldDiff{
				Field:  fmt.Sprintf("env.%s", k),
				Local:  v,
				Remote: remoteVal,
			})
		}
	}

	if len(fields) > 0 {
		return &Diff{Kind: "transformation", Name: local.Name, Status: Drifted, Fields: fields}
	}
	return nil
}
