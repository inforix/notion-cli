package notion

import (
	"context"
	"net/http"
)

func (c *Client) CreatePage(ctx context.Context, payload []byte) (map[string]any, *http.Response, error) {
	return c.PostJSON(ctx, "/v1/pages", payload)
}

func (c *Client) UpdatePage(ctx context.Context, pageID string, payload []byte) (map[string]any, *http.Response, error) {
	return c.PatchJSON(ctx, "/v1/pages/"+pageID, payload)
}
