package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Block struct {
	ID          string
	Type        string
	HasChildren bool
	Raw         map[string]any
	Children    []Block
}

type listResponse struct {
	Results    []json.RawMessage `json:"results"`
	HasMore    bool              `json:"has_more"`
	NextCursor *string           `json:"next_cursor"`
}

func (c *Client) GetPage(ctx context.Context, pageID string) (map[string]any, *http.Response, error) {
	return c.GetJSON(ctx, "/v1/pages/"+pageID)
}

func (c *Client) Search(ctx context.Context, payload []byte) (map[string]any, *http.Response, error) {
	return c.PostJSON(ctx, "/v1/search", payload)
}

func (c *Client) GetBlockTree(ctx context.Context, blockID string) ([]Block, error) {
	blocks, err := c.getBlockChildrenAll(ctx, blockID)
	if err != nil {
		return nil, err
	}

	for i := range blocks {
		if blocks[i].HasChildren {
			children, err := c.GetBlockTree(ctx, blocks[i].ID)
			if err != nil {
				return nil, err
			}
			blocks[i].Children = children
		}
	}

	return blocks, nil
}

func (c *Client) getBlockChildrenAll(ctx context.Context, blockID string) ([]Block, error) {
	var all []Block
	var cursor *string

	for {
		path := "/v1/blocks/" + blockID + "/children"
		params := url.Values{}
		params.Set("page_size", "100")
		if cursor != nil && *cursor != "" {
			params.Set("start_cursor", *cursor)
		}
		path = path + "?" + params.Encode()

		resp, body, err := c.Do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("notion API error (%d): %s", resp.StatusCode, string(body))
		}

		var list listResponse
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, err
		}
		for _, raw := range list.Results {
			block, err := parseBlock(raw)
			if err != nil {
				return nil, err
			}
			all = append(all, block)
		}

		if !list.HasMore || list.NextCursor == nil || *list.NextCursor == "" {
			break
		}
		cursor = list.NextCursor
	}

	return all, nil
}

func parseBlock(raw json.RawMessage) (Block, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return Block{}, err
	}

	block := Block{Raw: m}
	if id, ok := m["id"].(string); ok {
		block.ID = id
	}
	if t, ok := m["type"].(string); ok {
		block.Type = t
	}
	if hasChildren, ok := m["has_children"].(bool); ok {
		block.HasChildren = hasChildren
	}

	return block, nil
}
