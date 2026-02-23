package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL      = "https://api.notion.com"
	DefaultNotionVersion = "2025-09-03"
)

type Client struct {
	BaseURL       string
	Token         string
	NotionVersion string
	HTTPClient    *http.Client
	MaxRetries    int
}

func NewClient(token string, notionVersion string) *Client {
	ver := strings.TrimSpace(notionVersion)
	if ver == "" {
		ver = DefaultNotionVersion
	}

	return &Client{
		BaseURL:       DefaultBaseURL,
		Token:         token,
		NotionVersion: ver,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		MaxRetries: 3,
	}
}

func (c *Client) Do(ctx context.Context, method, path string, body []byte) (*http.Response, []byte, error) {
	url := c.BaseURL + path

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, nil, err
		}
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		req.Header.Set("Notion-Version", c.NotionVersion)
		if method == http.MethodPost || method == http.MethodPatch {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return resp, nil, readErr
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			wait := retryAfterSeconds(resp.Header.Get("Retry-After"))
			if wait > 0 && attempt < c.MaxRetries {
				time.Sleep(wait)
				continue
			}
		}

		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			if attempt < c.MaxRetries {
				backoff := time.Duration(500*(attempt+1)) * time.Millisecond
				time.Sleep(backoff)
				continue
			}
		}

		return resp, respBody, nil
	}

	return nil, nil, fmt.Errorf("request failed after retries: %w", lastErr)
}

func retryAfterSeconds(value string) time.Duration {
	if value == "" {
		return 0
	}
	secs, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return time.Duration(secs) * time.Second
}

func (c *Client) GetJSON(ctx context.Context, path string) (map[string]any, *http.Response, error) {
	resp, body, err := c.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, resp, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

func (c *Client) PostJSON(ctx context.Context, path string, payload []byte) (map[string]any, *http.Response, error) {
	resp, body, err := c.Do(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, resp, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

func (c *Client) PatchJSON(ctx context.Context, path string, payload []byte) (map[string]any, *http.Response, error) {
	resp, body, err := c.Do(ctx, http.MethodPatch, path, payload)
	if err != nil {
		return nil, resp, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}
