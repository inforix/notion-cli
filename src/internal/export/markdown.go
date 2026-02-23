package export

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/inforix/notion-cli/src/internal/assets"
	"github.com/inforix/notion-cli/src/internal/notion"
)

type AssetMode string

const (
	AssetLink     AssetMode = "link"
	AssetDownload AssetMode = "download"
	AssetInline   AssetMode = "inline"
)

type Options struct {
	IncludeFrontmatter bool
	IncludeTitle       bool
	AssetMode          AssetMode
	OutputPath         string
}

func ExportPageToMarkdown(ctx context.Context, client *notion.Client, pageID string, opts Options) (string, error) {
	page, resp, err := client.GetPage(ctx, pageID)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("notion API error (%d): %s", resp.StatusCode, stringifyJSON(page))
	}

	blocks, err := client.GetBlockTree(ctx, pageID)
	if err != nil {
		return "", err
	}

	title := extractTitle(page)
	assetRoot := defaultAssetRoot(opts.OutputPath, pageID)
	assetHelper := assets.NewHelper(client, string(opts.AssetMode), assetRoot)

	content := renderBlocks(blocks, renderContext{
		assetHelper: assetHelper,
	})

	var b strings.Builder
	if opts.IncludeFrontmatter {
		b.WriteString(renderFrontmatter(page, title))
		b.WriteString("\n")
	}
	if opts.IncludeTitle && title != "" {
		b.WriteString("# ")
		b.WriteString(title)
		b.WriteString("\n\n")
	}

	b.WriteString(content)

	return b.String(), nil
}

func defaultAssetRoot(outputPath, pageID string) string {
	base := "."
	if outputPath != "" {
		base = filepath.Dir(outputPath)
	}
	return filepath.Join(base, "assets", pageID)
}

func renderFrontmatter(page map[string]any, title string) string {
	id := stringValue(page, "id")
	url := stringValue(page, "url")
	created := stringValue(page, "created_time")
	edited := stringValue(page, "last_edited_time")

	properties, _ := json.MarshalIndent(page["properties"], "", "  ")

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("id: ")
	b.WriteString(id)
	b.WriteString("\n")
	b.WriteString("url: ")
	b.WriteString(url)
	b.WriteString("\n")
	if title != "" {
		b.WriteString("title: ")
		b.WriteString(escapeYAML(title))
		b.WriteString("\n")
	}
	if created != "" {
		b.WriteString("created_time: ")
		b.WriteString(created)
		b.WriteString("\n")
	}
	if edited != "" {
		b.WriteString("last_edited_time: ")
		b.WriteString(edited)
		b.WriteString("\n")
	}
	b.WriteString("properties_json: |\n")
	for _, line := range strings.Split(string(properties), "\n") {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("---\n")

	return b.String()
}

func extractTitle(page map[string]any) string {
	props, ok := page["properties"].(map[string]any)
	if !ok {
		return ""
	}
	for _, prop := range props {
		m, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		typeName, _ := m["type"].(string)
		if typeName != "title" {
			continue
		}
		items, ok := m["title"].([]any)
		if !ok {
			return ""
		}
		return renderRichTextItems(items)
	}
	return ""
}

func escapeYAML(value string) string {
	if strings.ContainsAny(value, ":\n") {
		return fmt.Sprintf("%q", value)
	}
	return value
}

func stringifyJSON(data any) string {
	b, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(b)
}

func stringValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}
