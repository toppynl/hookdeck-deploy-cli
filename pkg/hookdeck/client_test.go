package hookdeck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSourceByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sources" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "my-source" {
			t.Errorf("unexpected name query: %s", r.URL.Query().Get("name"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"id":          "src_123",
					"name":        "my-source",
					"url":         "https://hk-src_123.hookdeck.com",
					"description": "A test source",
				},
			},
			"count": 1,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetSourceByName(context.Background(), "my-source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "src_123" {
		t.Errorf("expected id src_123, got %s", result.ID)
	}
	if result.Name != "my-source" {
		t.Errorf("expected name my-source, got %s", result.Name)
	}
	if result.URL != "https://hk-src_123.hookdeck.com" {
		t.Errorf("expected url https://hk-src_123.hookdeck.com, got %s", result.URL)
	}
	if result.Description != "A test source" {
		t.Errorf("expected description 'A test source', got %s", result.Description)
	}
}

func TestGetSourceByName_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
			"count":  0,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetSourceByName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for not-found source, got %+v", result)
	}
}

func TestGetDestinationByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/destinations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "my-dest" {
			t.Errorf("unexpected name query: %s", r.URL.Query().Get("name"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"id":          "des_456",
					"name":        "my-dest",
					"description": "A test destination",
					"type":        "HTTP",
					"config": map[string]interface{}{
						"url":              "https://example.com/webhook",
						"auth_type":        "BASIC_AUTH",
						"auth":             map[string]interface{}{"username": "user", "password": "pass"},
						"rate_limit":       100,
						"rate_limit_period": "second",
					},
				},
			},
			"count": 1,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetDestinationByName(context.Background(), "my-dest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "des_456" {
		t.Errorf("expected id des_456, got %s", result.ID)
	}
	if result.Name != "my-dest" {
		t.Errorf("expected name my-dest, got %s", result.Name)
	}
	if result.Config.URL != "https://example.com/webhook" {
		t.Errorf("expected url https://example.com/webhook, got %s", result.Config.URL)
	}
	if result.Description != "A test destination" {
		t.Errorf("expected description 'A test destination', got %s", result.Description)
	}
	if result.Config.AuthType != "BASIC_AUTH" {
		t.Errorf("expected auth_type BASIC_AUTH, got %s", result.Config.AuthType)
	}
	if result.Config.Auth == nil {
		t.Fatal("expected non-nil auth")
	}
	if result.Config.Auth["username"] != "user" {
		t.Errorf("expected auth username 'user', got %v", result.Config.Auth["username"])
	}
	if result.Config.RateLimit != 100 {
		t.Errorf("expected rate_limit 100, got %d", result.Config.RateLimit)
	}
	if result.Config.RateLimitPeriod != "second" {
		t.Errorf("expected rate_limit_period 'second', got %s", result.Config.RateLimitPeriod)
	}
}

func TestGetDestinationByName_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
			"count":  0,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetDestinationByName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for not-found destination, got %+v", result)
	}
}

func TestGetConnectionByFullName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connections" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("full_name") != "my-source->my-dest" {
			t.Errorf("unexpected full_name query: %s", r.URL.Query().Get("full_name"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"id":        "con_789",
					"name":      "my-conn",
					"full_name": "my-source->my-dest",
					"source": map[string]interface{}{
						"id":   "src_123",
						"name": "my-source",
						"url":  "https://hk-src_123.hookdeck.com",
					},
					"destination": map[string]interface{}{
						"id":   "des_456",
						"name": "my-dest",
						"url":  "https://example.com/webhook",
					},
					"rules": []map[string]interface{}{
						{
							"type": "filter",
							"body": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "order.placed",
								},
							},
						},
					},
				},
			},
			"count": 1,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetConnectionByFullName(context.Background(), "my-source->my-dest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "con_789" {
		t.Errorf("expected id con_789, got %s", result.ID)
	}
	if result.Name != "my-conn" {
		t.Errorf("expected name my-conn, got %s", result.Name)
	}
	if result.FullName != "my-source->my-dest" {
		t.Errorf("expected full_name 'my-source->my-dest', got %s", result.FullName)
	}
	if result.Source == nil {
		t.Fatal("expected non-nil source")
	}
	if result.Source.Name != "my-source" {
		t.Errorf("expected source name my-source, got %s", result.Source.Name)
	}
	if result.Destination == nil {
		t.Fatal("expected non-nil destination")
	}
	if result.Destination.Name != "my-dest" {
		t.Errorf("expected destination name my-dest, got %s", result.Destination.Name)
	}
	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}
	if result.Rules[0]["type"] != "filter" {
		t.Errorf("expected rule type 'filter', got %v", result.Rules[0]["type"])
	}
}

func TestGetConnectionByFullName_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
			"count":  0,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetConnectionByFullName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for not-found connection, got %+v", result)
	}
}

func TestGetTransformationByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transformations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "my-transform" {
			t.Errorf("unexpected name query: %s", r.URL.Query().Get("name"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"id":   "trs_abc",
					"name": "my-transform",
					"code": "function handler(request, context) { return request; }",
					"env": map[string]interface{}{
						"API_KEY": "secret",
					},
				},
			},
			"count": 1,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetTransformationByName(context.Background(), "my-transform")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "trs_abc" {
		t.Errorf("expected id trs_abc, got %s", result.ID)
	}
	if result.Name != "my-transform" {
		t.Errorf("expected name my-transform, got %s", result.Name)
	}
	if result.Code != "function handler(request, context) { return request; }" {
		t.Errorf("unexpected code: %s", result.Code)
	}
	if result.Env == nil {
		t.Fatal("expected non-nil env")
	}
	if result.Env["API_KEY"] != "secret" {
		t.Errorf("expected env API_KEY='secret', got %s", result.Env["API_KEY"])
	}
}

func TestGetTransformationByName_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
			"count":  0,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "", WithBaseURL(srv.URL))
	result, err := client.GetTransformationByName(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for not-found transformation, got %+v", result)
	}
}

func TestGetSourceByName_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "unauthorized",
		})
	}))
	defer srv.Close()

	client := NewClient("bad-key", "", WithBaseURL(srv.URL))
	_, err := client.GetSourceByName(context.Background(), "my-source")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestGetSourceByName_SetsAuthHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Basic Auth header is set
		user, _, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth header")
		}
		if user != "test-key" {
			t.Errorf("expected basic auth user 'test-key', got %s", user)
		}
		// Verify project ID header
		if r.Header.Get("X-Project-ID") != "proj_123" {
			t.Errorf("expected X-Project-ID 'proj_123', got %s", r.Header.Get("X-Project-ID"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
			"count":  0,
		})
	}))
	defer srv.Close()

	client := NewClient("test-key", "proj_123", WithBaseURL(srv.URL))
	_, err := client.GetSourceByName(context.Background(), "my-source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
