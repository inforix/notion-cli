package pagination

import "fmt"

type List struct {
	Results    []any
	HasMore    bool
	NextCursor string
}

func Parse(data map[string]any) (List, error) {
	if data == nil {
		return List{}, fmt.Errorf("missing response data")
	}
	results, ok := data["results"].([]any)
	if !ok {
		return List{}, fmt.Errorf("response missing results array")
	}
	hasMore, _ := data["has_more"].(bool)
	nextCursor := ""
	if cursor, ok := data["next_cursor"].(string); ok {
		nextCursor = cursor
	}
	return List{
		Results:    results,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func Combine(results []any) map[string]any {
	return map[string]any{
		"object":      "list",
		"results":     results,
		"has_more":    false,
		"next_cursor": nil,
	}
}
