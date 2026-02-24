package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFileURLAtExternal(t *testing.T) {
	data := map[string]any{
		"type": "external",
		"external": map[string]any{
			"url": "https://example.com/file.png",
		},
	}

	gotURL, gotType, err := extractFileURLAt(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotURL != "https://example.com/file.png" {
		t.Fatalf("expected url %q, got %q", "https://example.com/file.png", gotURL)
	}
	if gotType != "external" {
		t.Fatalf("expected type %q, got %q", "external", gotType)
	}
}

func TestExtractFileURLAtFile(t *testing.T) {
	data := map[string]any{
		"type": "file",
		"file": map[string]any{
			"url": "https://example.com/file.bin",
		},
	}

	gotURL, gotType, err := extractFileURLAt(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotURL != "https://example.com/file.bin" {
		t.Fatalf("expected url %q, got %q", "https://example.com/file.bin", gotURL)
	}
	if gotType != "file" {
		t.Fatalf("expected type %q, got %q", "file", gotType)
	}
}

func TestExtractFileURLAtFilesIndex(t *testing.T) {
	data := map[string]any{
		"type": "files",
		"files": []any{
			map[string]any{
				"type": "external",
				"external": map[string]any{
					"url": "https://example.com/first.png",
				},
			},
			map[string]any{
				"type": "file",
				"file": map[string]any{
					"url": "https://example.com/second.bin",
				},
			},
		},
	}

	gotURL, gotType, err := extractFileURLAt(data, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotURL != "https://example.com/second.bin" {
		t.Fatalf("expected url %q, got %q", "https://example.com/second.bin", gotURL)
	}
	if gotType != "file" {
		t.Fatalf("expected type %q, got %q", "file", gotType)
	}

	if _, _, err := extractFileURLAt(data, 2); err == nil {
		t.Fatalf("expected out of range error")
	}
}

func TestExtractFileURLAtFileUploadError(t *testing.T) {
	data := map[string]any{
		"type": "file_upload",
	}

	if _, _, err := extractFileURLAt(data, 0); err == nil {
		t.Fatalf("expected error for file_upload object")
	}
}

func TestExtractFileURLAtObjectFileUploadError(t *testing.T) {
	data := map[string]any{
		"object": "file_upload",
	}

	if _, _, err := extractFileURLAt(data, 0); err == nil {
		t.Fatalf("expected error for file_upload object")
	}
}

func TestExtractFileURLAtFallbackExternal(t *testing.T) {
	data := map[string]any{
		"external": map[string]any{
			"url": "https://example.com/fallback.png",
		},
	}

	gotURL, gotType, err := extractFileURLAt(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotURL != "https://example.com/fallback.png" {
		t.Fatalf("expected url %q, got %q", "https://example.com/fallback.png", gotURL)
	}
	if gotType != "external" {
		t.Fatalf("expected type %q, got %q", "external", gotType)
	}
}

func TestFilenameFromContentDisposition(t *testing.T) {
	name := filenameFromContentDisposition(`attachment; filename="report.pdf"`)
	if name != "report.pdf" {
		t.Fatalf("expected filename %q, got %q", "report.pdf", name)
	}

	utfName := filenameFromContentDisposition(`attachment; filename*=UTF-8''test%20file.txt`)
	if utfName != "test file.txt" {
		t.Fatalf("expected filename %q, got %q", "test file.txt", utfName)
	}
}

func TestFilenameFromURL(t *testing.T) {
	name := filenameFromURL("https://example.com/path/to/data.bin")
	if name != "data.bin" {
		t.Fatalf("expected filename %q, got %q", "data.bin", name)
	}
}

func TestSanitizeFilename(t *testing.T) {
	if got := sanitizeFilename("file.txt"); got != "file.txt" {
		t.Fatalf("expected %q, got %q", "file.txt", got)
	}
	if got := sanitizeFilename("foo/bar"); got != "foo-bar" {
		t.Fatalf("expected %q, got %q", "foo-bar", got)
	}
	if got := sanitizeFilename(".."); got != "" {
		t.Fatalf("expected empty for \"..\", got %q", got)
	}
}

func TestParseJSONMap(t *testing.T) {
	data, err := parseJSONMap([]byte(`{"name":"value"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["name"] != "value" {
		t.Fatalf("expected value, got %v", data["name"])
	}
}

func TestDownloadToFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("binary-data"))
	}))
	defer server.Close()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "out.bin")

	if err := downloadTo(outPath, server.URL); err != nil {
		t.Fatalf("download failed: %v", err)
	}

	contents, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(contents) != "binary-data" {
		t.Fatalf("unexpected contents: %s", string(contents))
	}
}

func TestDownloadToDirectoryUsesContentDisposition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="payload.bin"`)
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "downloads")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	if err := downloadTo(outDir, server.URL); err != nil {
		t.Fatalf("download failed: %v", err)
	}

	target := filepath.Join(outDir, "payload.bin")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected file at %s: %v", target, err)
	}
}

func TestDownloadToStdout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("stdout-data"))
	}))
	defer server.Close()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w

	downloadErr := downloadTo("-", server.URL)

	_ = w.Close()
	os.Stdout = origStdout

	if downloadErr != nil {
		t.Fatalf("download failed: %v", downloadErr)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout failed: %v", err)
	}

	if strings.TrimSpace(buf.String()) != "stdout-data" {
		t.Fatalf("unexpected stdout contents: %q", buf.String())
	}
}
