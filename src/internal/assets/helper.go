package assets

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/inforix/notion-cli/src/internal/notion"
)

type Mode string

const (
	ModeLink     Mode = "link"
	ModeDownload Mode = "download"
	ModeInline   Mode = "inline"
)

type Helper struct {
	mode   Mode
	root   string
	client *notion.Client
}

func NewHelper(client *notion.Client, mode string, root string) *Helper {
	m := Mode(strings.ToLower(mode))
	if m != ModeDownload && m != ModeInline {
		m = ModeLink
	}
	return &Helper{mode: m, root: root, client: client}
}

func (h *Helper) Resolve(urlStr string, filename string, isImage bool) (string, error) {
	if urlStr == "" {
		return "", nil
	}

	switch h.mode {
	case ModeDownload:
		return h.download(urlStr, filename)
	case ModeInline:
		if isImage {
			return h.inline(urlStr)
		}
		return urlStr, nil
	default:
		return urlStr, nil
	}
}

func (h *Helper) download(urlStr string, filename string) (string, error) {
	if err := os.MkdirAll(h.root, 0o755); err != nil {
		return "", err
	}

	name := sanitizeFilename(filename)
	if name == "" {
		name = filenameFromURL(urlStr)
	}
	if name == "" {
		hash := sha1.Sum([]byte(urlStr))
		name = fmt.Sprintf("asset-%x", hash[:6])
	}

	path := filepath.Join(h.root, name)
	if _, err := os.Stat(path); err == nil {
		return filepath.ToSlash(path), nil
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}

	return filepath.ToSlash(path), nil
}

func (h *Helper) inline(urlStr string) (string, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	encoded := base64.StdEncoding.EncodeToString(bytes)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

func filenameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	name := filepath.Base(parsed.Path)
	return sanitizeFilename(name)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	return name
}
