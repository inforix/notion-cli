package notion

import (
	"context"
	"net/http"
)

func (c *Client) GetDatabase(ctx context.Context, databaseID string) (map[string]any, *http.Response, error) {
	return c.GetJSON(ctx, "/v1/databases/"+databaseID)
}

func (c *Client) QueryDatabase(ctx context.Context, databaseID string, payload []byte) (map[string]any, *http.Response, error) {
	return c.PostJSON(ctx, "/v1/databases/"+databaseID+"/query", payload)
}

func (c *Client) CreateDatabase(ctx context.Context, payload []byte) (map[string]any, *http.Response, error) {
	return c.PostJSON(ctx, "/v1/databases", payload)
}

func (c *Client) UpdateDatabase(ctx context.Context, databaseID string, payload []byte) (map[string]any, *http.Response, error) {
	return c.PatchJSON(ctx, "/v1/databases/"+databaseID, payload)
}
