package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListFileUploadsBuildsQuery(t *testing.T) {
	var gotPath string
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list"})
	}))
	defer server.Close()

	client := &Client{
		BaseURL:       server.URL,
		Token:         "token",
		NotionVersion: DefaultNotionVersion,
		HTTPClient:    server.Client(),
		MaxRetries:    0,
	}

	_, _, err := client.ListFileUploads(context.Background(), 42, "cursor-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/v1/file_uploads" {
		t.Fatalf("expected path %q, got %q", "/v1/file_uploads", gotPath)
	}
	if gotQuery == "" {
		t.Fatalf("expected query params, got empty")
	}
}

func TestListFileUploadsNoParams(t *testing.T) {
	var gotPath string
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list"})
	}))
	defer server.Close()

	client := &Client{
		BaseURL:       server.URL,
		Token:         "token",
		NotionVersion: DefaultNotionVersion,
		HTTPClient:    server.Client(),
		MaxRetries:    0,
	}

	_, _, err := client.ListFileUploads(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/v1/file_uploads" {
		t.Fatalf("expected path %q, got %q", "/v1/file_uploads", gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("expected empty query, got %q", gotQuery)
	}
}
