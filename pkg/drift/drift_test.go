package drift

import (
	"testing"

	"github.com/toppynl/hookdeck-deploy-cli/pkg/hookdeck"
	"github.com/toppynl/hookdeck-deploy-cli/pkg/manifest"
)

func TestDetect_SourceMissing(t *testing.T) {
	sources := []*manifest.SourceConfig{{Name: "my-source"}}
	remote := &RemoteState{
		Sources: []*hookdeck.SourceDetail{nil},
	}

	diffs := Detect(sources, nil, nil, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Kind != "source" || diffs[0].Status != Missing {
		t.Errorf("expected source missing, got %v", diffs[0])
	}
}

func TestDetect_SourceDescriptionDrift(t *testing.T) {
	sources := []*manifest.SourceConfig{{
		Name:        "my-source",
		Description: "new description",
	}}
	remote := &RemoteState{
		Sources: []*hookdeck.SourceDetail{{
			ID:          "src_123",
			Name:        "my-source",
			Description: "old description",
		}},
	}

	diffs := Detect(sources, nil, nil, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != Drifted {
		t.Errorf("expected drifted, got %v", diffs[0].Status)
	}
	if len(diffs[0].Fields) != 1 || diffs[0].Fields[0].Field != "description" {
		t.Errorf("expected description field diff, got %v", diffs[0].Fields)
	}
}

func TestDetect_DestinationMissing(t *testing.T) {
	destinations := []*manifest.DestinationConfig{{
		Name: "my-dest",
		URL:  "https://example.com",
	}}
	remote := &RemoteState{
		Destinations: []*hookdeck.DestinationDetail{nil},
	}

	diffs := Detect(nil, destinations, nil, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Kind != "destination" || diffs[0].Status != Missing {
		t.Errorf("expected destination missing, got %v", diffs[0])
	}
}

func TestDetect_DestinationURLDrift(t *testing.T) {
	destinations := []*manifest.DestinationConfig{{
		Name: "my-dest",
		URL:  "https://new-url.example.com",
	}}
	remote := &RemoteState{
		Destinations: []*hookdeck.DestinationDetail{{
			ID:   "dst_123",
			Name: "my-dest",
			Config: hookdeck.DestinationConfigDetail{
				URL: "https://old-url.example.com",
			},
		}},
	}

	diffs := Detect(nil, destinations, nil, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != Drifted {
		t.Errorf("expected drifted, got %v", diffs[0].Status)
	}
}

func TestDetect_DestinationRateLimitDrift(t *testing.T) {
	destinations := []*manifest.DestinationConfig{{
		Name:            "my-dest",
		URL:             "https://example.com",
		RateLimit:       100,
		RateLimitPeriod: "second",
	}}
	remote := &RemoteState{
		Destinations: []*hookdeck.DestinationDetail{{
			ID:   "dst_123",
			Name: "my-dest",
			Config: hookdeck.DestinationConfigDetail{
				URL:             "https://example.com",
				RateLimit:       50,
				RateLimitPeriod: "second",
			},
		}},
	}

	diffs := Detect(nil, destinations, nil, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != Drifted {
		t.Errorf("expected drifted, got %v", diffs[0].Status)
	}
	if len(diffs[0].Fields) != 1 || diffs[0].Fields[0].Field != "rate_limit" {
		t.Errorf("expected rate_limit field diff, got %v", diffs[0].Fields)
	}
}

func TestDetect_ConnectionMissing(t *testing.T) {
	connections := []*manifest.ConnectionConfig{{Name: "my-conn"}}
	remote := &RemoteState{
		Connections: []*hookdeck.ConnectionDetail{nil},
	}

	diffs := Detect(nil, nil, nil, connections, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Kind != "connection" || diffs[0].Status != Missing {
		t.Errorf("expected connection missing, got %v", diffs[0])
	}
}

func TestDetect_TransformationMissing(t *testing.T) {
	transformations := []*manifest.TransformationConfig{{Name: "my-transform"}}
	remote := &RemoteState{
		Transformations: []*hookdeck.TransformationDetail{nil},
	}

	diffs := Detect(nil, nil, transformations, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Kind != "transformation" || diffs[0].Status != Missing {
		t.Errorf("expected transformation missing, got %v", diffs[0])
	}
}

func TestDetect_TransformationEnvDrift(t *testing.T) {
	transformations := []*manifest.TransformationConfig{{
		Name: "my-transform",
		Env:  map[string]string{"KEY": "new-value"},
	}}
	remote := &RemoteState{
		Transformations: []*hookdeck.TransformationDetail{{
			ID:   "tr_123",
			Name: "my-transform",
			Env:  map[string]string{"KEY": "old-value"},
		}},
	}

	diffs := Detect(nil, nil, transformations, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != Drifted {
		t.Errorf("expected drifted, got %v", diffs[0].Status)
	}
	if len(diffs[0].Fields) != 1 || diffs[0].Fields[0].Field != "env.KEY" {
		t.Errorf("expected env.KEY field diff, got %v", diffs[0].Fields)
	}
}

func TestDetect_TransformationEnvMissing(t *testing.T) {
	transformations := []*manifest.TransformationConfig{{
		Name: "my-transform",
		Env:  map[string]string{"KEY": "value"},
	}}
	remote := &RemoteState{
		Transformations: []*hookdeck.TransformationDetail{{
			ID:   "tr_123",
			Name: "my-transform",
			Env:  map[string]string{},
		}},
	}

	diffs := Detect(nil, nil, transformations, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != Drifted {
		t.Errorf("expected drifted, got %v", diffs[0].Status)
	}
}

func TestDetect_NoDrift(t *testing.T) {
	sources := []*manifest.SourceConfig{{Name: "my-source"}}
	remote := &RemoteState{
		Sources: []*hookdeck.SourceDetail{{
			ID:   "src_123",
			Name: "my-source",
		}},
	}

	diffs := Detect(sources, nil, nil, nil, remote)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs, got %d: %v", len(diffs), diffs)
	}
}

func TestDetect_AllInSync(t *testing.T) {
	sources := []*manifest.SourceConfig{{Name: "my-source"}}
	destinations := []*manifest.DestinationConfig{{
		Name: "my-dest",
		URL:  "https://example.com",
	}}
	connections := []*manifest.ConnectionConfig{{Name: "my-conn"}}
	transformations := []*manifest.TransformationConfig{{Name: "my-transform"}}

	remote := &RemoteState{
		Sources: []*hookdeck.SourceDetail{{ID: "src_123", Name: "my-source"}},
		Destinations: []*hookdeck.DestinationDetail{{
			ID:   "dst_123",
			Name: "my-dest",
			Config: hookdeck.DestinationConfigDetail{
				URL: "https://example.com",
			},
		}},
		Connections:     []*hookdeck.ConnectionDetail{{ID: "conn_123", Name: "my-conn"}},
		Transformations: []*hookdeck.TransformationDetail{{ID: "tr_123", Name: "my-transform"}},
	}

	diffs := Detect(sources, destinations, transformations, connections, remote)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs, got %d: %v", len(diffs), diffs)
	}
}

func TestDetect_MultipleDrifts(t *testing.T) {
	sources := []*manifest.SourceConfig{{Name: "my-source"}}
	destinations := []*manifest.DestinationConfig{{Name: "my-dest", URL: "https://new.example.com"}}

	remote := &RemoteState{
		// Source missing (nil entry)
		Sources: []*hookdeck.SourceDetail{nil},
		Destinations: []*hookdeck.DestinationDetail{{
			ID:   "dst_123",
			Name: "my-dest",
			Config: hookdeck.DestinationConfigDetail{
				URL: "https://old.example.com",
			},
		}},
	}

	diffs := Detect(sources, destinations, nil, nil, remote)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d: %v", len(diffs), diffs)
	}

	// First diff should be source missing
	if diffs[0].Kind != "source" || diffs[0].Status != Missing {
		t.Errorf("expected source missing, got %v", diffs[0])
	}
	// Second diff should be destination drifted
	if diffs[1].Kind != "destination" || diffs[1].Status != Drifted {
		t.Errorf("expected destination drifted, got %v", diffs[1])
	}
}

func TestDetect_EmptyManifest(t *testing.T) {
	remote := &RemoteState{
		Sources: []*hookdeck.SourceDetail{{ID: "src_123", Name: "orphan-source"}},
	}

	diffs := Detect(nil, nil, nil, nil, remote)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs for empty manifest, got %d: %v", len(diffs), diffs)
	}
}

func TestDetect_DestinationMultipleFieldDrifts(t *testing.T) {
	destinations := []*manifest.DestinationConfig{{
		Name:            "my-dest",
		URL:             "https://new.example.com",
		AuthType:        "bearer_token",
		RateLimit:       200,
		RateLimitPeriod: "minute",
	}}
	remote := &RemoteState{
		Destinations: []*hookdeck.DestinationDetail{{
			ID:   "dst_123",
			Name: "my-dest",
			Config: hookdeck.DestinationConfigDetail{
				URL:             "https://old.example.com",
				AuthType:        "basic_auth",
				RateLimit:       100,
				RateLimitPeriod: "second",
			},
		}},
	}

	diffs := Detect(nil, destinations, nil, nil, remote)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != Drifted {
		t.Errorf("expected drifted, got %v", diffs[0].Status)
	}
	if len(diffs[0].Fields) != 4 {
		t.Errorf("expected 4 field diffs, got %d: %v", len(diffs[0].Fields), diffs[0].Fields)
	}
}
