package notion

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) ListUsers(ctx context.Context, startCursor string, pageSize int) (map[string]any, *http.Response, error) {
	path := "/v1/users"
	params := url.Values{}
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

func (c *Client) GetUser(ctx context.Context, userID string) (map[string]any, *http.Response, error) {
	return c.GetJSON(ctx, "/v1/users/"+userID)
}
