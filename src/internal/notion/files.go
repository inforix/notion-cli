package notion

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) CreateFileUpload(ctx context.Context, payload []byte) (map[string]any, *http.Response, error) {
	return c.PostJSON(ctx, "/v1/file_uploads", payload)
}

func (c *Client) GetFileUpload(ctx context.Context, fileUploadID string) (map[string]any, *http.Response, error) {
	return c.GetJSON(ctx, "/v1/file_uploads/"+fileUploadID)
}

func (c *Client) ListFileUploads(ctx context.Context, pageSize int, startCursor string) (map[string]any, *http.Response, error) {
	path := "/v1/file_uploads"
	params := url.Values{}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(pageSize))
	}
	if startCursor != "" {
		params.Set("start_cursor", startCursor)
	}
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}
	return c.GetJSON(ctx, path)
}
