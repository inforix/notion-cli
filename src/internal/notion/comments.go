package notion

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) ListComments(ctx context.Context, blockID string, pageID string, startCursor string, pageSize int) (map[string]any, *http.Response, error) {
	path := "/v1/comments"
	params := url.Values{}
	if blockID != "" {
		params.Set("block_id", blockID)
	}
	if pageID != "" {
		params.Set("page_id", pageID)
	}
	if startCursor != "" {
		params.Set("start_cursor", startCursor)
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(pageSize))
	}
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}
	return c.GetJSON(ctx, path)
}

func (c *Client) CreateComment(ctx context.Context, payload []byte) (map[string]any, *http.Response, error) {
	return c.PostJSON(ctx, "/v1/comments", payload)
}
