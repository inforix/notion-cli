package export

import (
	"fmt"
	"strings"

	"github.com/inforix/notion-cli/src/internal/assets"
	"github.com/inforix/notion-cli/src/internal/notion"
)

type renderContext struct {
	assetHelper *assets.Helper
}

func renderBlocks(blocks []notion.Block, ctx renderContext) string {
	var b strings.Builder
	prevListType := ""

	for i, block := range blocks {
		isList := isListItem(block.Type)
		if isList {
			if prevListType != block.Type && b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(renderListItem(block, 0, ctx))
			if i < len(blocks)-1 {
				b.WriteString("\n")
			}
			prevListType = block.Type
			continue
		}

		if prevListType != "" && b.Len() > 0 {
			b.WriteString("\n")
			prevListType = ""
		}

		b.WriteString(renderBlock(block, 0, ctx))
		if i < len(blocks)-1 {
			b.WriteString("\n\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderBlock(block notion.Block, indent int, ctx renderContext) string {
	switch block.Type {
	case "paragraph":
		return indentLines(renderParagraph(block), indent)
	case "heading_1":
		return indentLines("# "+renderRichTextField(block, "heading_1"), indent)
	case "heading_2":
		return indentLines("## "+renderRichTextField(block, "heading_2"), indent)
	case "heading_3":
		return indentLines("### "+renderRichTextField(block, "heading_3"), indent)
	case "to_do":
		return indentLines(renderTodo(block, indent, ctx), 0)
	case "quote":
		return indentLines(renderQuote(block), indent)
	case "divider":
		return indentLines("---", indent)
	case "code":
		return indentLines(renderCode(block), indent)
	case "callout":
		return indentLines(renderCallout(block), indent)
	case "toggle":
		return indentLines(renderToggle(block, indent, ctx), 0)
	case "image":
		return indentLines(renderImage(block, ctx), indent)
	case "file", "pdf", "video":
		return indentLines(renderFile(block, ctx), indent)
	case "table":
		return indentLines(renderTable(block), indent)
	case "child_page":
		return indentLines(renderChild(block, "child_page"), indent)
	case "child_database":
		return indentLines(renderChild(block, "child_database"), indent)
	default:
		return indentLines(fmt.Sprintf("<!-- unsupported block: %s -->", block.Type), indent)
	}
}

func renderListItem(block notion.Block, indent int, ctx renderContext) string {
	prefix := strings.Repeat("  ", indent)
	content := renderListContent(block)

	var line string
	switch block.Type {
	case "bulleted_list_item":
		line = fmt.Sprintf("%s- %s", prefix, content)
	case "numbered_list_item":
		line = fmt.Sprintf("%s1. %s", prefix, content)
	case "to_do":
		line = fmt.Sprintf("%s%s", prefix, renderTodoLine(block, content))
	default:
		line = fmt.Sprintf("%s- %s", prefix, content)
	}

	if len(block.Children) == 0 {
		return line
	}

	children := renderNestedBlocks(block.Children, indent+1, ctx)
	return line + "\n" + children
}

func renderNestedBlocks(blocks []notion.Block, indent int, ctx renderContext) string {
	var b strings.Builder
	prevListType := ""

	for i, block := range blocks {
		isList := isListItem(block.Type)
		if isList {
			if prevListType != block.Type && b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(renderListItem(block, indent, ctx))
			if i < len(blocks)-1 {
				b.WriteString("\n")
			}
			prevListType = block.Type
			continue
		}

		if prevListType != "" && b.Len() > 0 {
			b.WriteString("\n")
			prevListType = ""
		}

		b.WriteString(renderBlock(block, indent, ctx))
		if i < len(blocks)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String()
}

func renderParagraph(block notion.Block) string {
	return renderRichTextField(block, "paragraph")
}

func renderQuote(block notion.Block) string {
	text := renderRichTextField(block, "quote")
	if text == "" {
		return ">"
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

func renderCode(block notion.Block) string {
	data, ok := block.Raw["code"].(map[string]any)
	if !ok {
		return ""
	}
	language, _ := data["language"].(string)
	items, _ := data["rich_text"].([]any)
	content := renderRichTextItems(items)
	var b strings.Builder
	b.WriteString("```")
	b.WriteString(language)
	b.WriteString("\n")
	b.WriteString(content)
	b.WriteString("\n```")
	return b.String()
}

func renderCallout(block notion.Block) string {
	data, ok := block.Raw["callout"].(map[string]any)
	if !ok {
		return ""
	}
	items, _ := data["rich_text"].([]any)
	text := renderRichTextItems(items)
	icon := renderIcon(data["icon"])
	if icon != "" {
		text = icon + " " + text
	}
	if text == "" {
		return ">"
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

func renderToggle(block notion.Block, indent int, ctx renderContext) string {
	data, ok := block.Raw["toggle"].(map[string]any)
	if !ok {
		return ""
	}
	items, _ := data["rich_text"].([]any)
	summary := renderRichTextItems(items)
	var b strings.Builder
	b.WriteString("<details>\n")
	b.WriteString("<summary>")
	b.WriteString(summary)
	b.WriteString("</summary>\n\n")
	if len(block.Children) > 0 {
		b.WriteString(renderNestedBlocks(block.Children, indent+1, ctx))
		b.WriteString("\n")
	}
	b.WriteString("</details>")
	return b.String()
}

func renderImage(block notion.Block, ctx renderContext) string {
	data, ok := block.Raw["image"].(map[string]any)
	if !ok {
		return ""
	}
	url := extractFileURL(data)
	captionItems, _ := data["caption"].([]any)
	caption := renderRichTextItems(captionItems)
	resolved, err := ctx.assetHelper.Resolve(url, "", true)
	if err != nil {
		resolved = url
	}
	alt := caption
	if alt == "" {
		alt = "image"
	}
	return fmt.Sprintf("![%s](%s)", alt, resolved)
}

func renderFile(block notion.Block, ctx renderContext) string {
	data, ok := block.Raw[block.Type].(map[string]any)
	if !ok {
		return ""
	}
	url := extractFileURL(data)
	name := extractFileName(data)
	resolved, err := ctx.assetHelper.Resolve(url, name, false)
	if err != nil {
		resolved = url
	}
	label := name
	if label == "" {
		label = resolved
	}
	return fmt.Sprintf("[%s](%s)", label, resolved)
}

func renderTable(block notion.Block) string {
	data, ok := block.Raw["table"].(map[string]any)
	if !ok {
		return ""
	}
	rows := make([][]string, 0)
	for _, row := range block.Children {
		rowData, ok := row.Raw["table_row"].(map[string]any)
		if !ok {
			continue
		}
		cells, ok := rowData["cells"].([]any)
		if !ok {
			continue
		}
		rowCells := make([]string, 0, len(cells))
		for _, cell := range cells {
			items, ok := cell.([]any)
			if !ok {
				rowCells = append(rowCells, "")
				continue
			}
			rowCells = append(rowCells, renderRichTextItems(items))
		}
		rows = append(rows, rowCells)
	}

	colCount := 0
	if width, ok := data["table_width"].(float64); ok {
		colCount = int(width)
	}
	if colCount == 0 {
		for _, row := range rows {
			if len(row) > colCount {
				colCount = len(row)
			}
		}
	}
	if colCount == 0 {
		return ""
	}

	hasHeader := false
	if header, ok := data["has_column_header"].(bool); ok {
		hasHeader = header
	}

	headers := make([]string, colCount)
	body := rows
	if hasHeader && len(rows) > 0 {
		headers = padRow(rows[0], colCount)
		body = rows[1:]
	} else {
		for i := range headers {
			headers[i] = ""
		}
	}

	var b strings.Builder
	b.WriteString("|")
	for _, h := range headers {
		b.WriteString(" ")
		b.WriteString(escapeTable(h))
		b.WriteString(" |")
	}
	b.WriteString("\n|")
	for range headers {
		b.WriteString(" --- |")
	}
	b.WriteString("\n")
	for _, row := range body {
		row = padRow(row, colCount)
		b.WriteString("|")
		for _, cell := range row {
			b.WriteString(" ")
			b.WriteString(escapeTable(cell))
			b.WriteString(" |")
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderChild(block notion.Block, field string) string {
	data, ok := block.Raw[field].(map[string]any)
	if !ok {
		return ""
	}
	title, _ := data["title"].(string)
	if title == "" {
		title = block.ID
	}
	url := notionURLFromID(block.ID)
	return fmt.Sprintf("[%s](%s)", title, url)
}

func renderListContent(block notion.Block) string {
	switch block.Type {
	case "bulleted_list_item":
		return renderRichTextField(block, "bulleted_list_item")
	case "numbered_list_item":
		return renderRichTextField(block, "numbered_list_item")
	case "to_do":
		return renderRichTextField(block, "to_do")
	default:
		return ""
	}
}

func renderTodo(block notion.Block, indent int, ctx renderContext) string {
	content := renderListContent(block)
	line := renderTodoLine(block, content)
	if len(block.Children) == 0 {
		return line
	}
	children := renderNestedBlocks(block.Children, indent+1, ctx)
	return line + "\n" + children
}

func renderTodoLine(block notion.Block, content string) string {
	data, ok := block.Raw["to_do"].(map[string]any)
	if !ok {
		return "- [ ] " + content
	}
	checked, _ := data["checked"].(bool)
	box := "[ ]"
	if checked {
		box = "[x]"
	}
	return fmt.Sprintf("- %s %s", box, content)
}

func renderRichTextField(block notion.Block, field string) string {
	data, ok := block.Raw[field].(map[string]any)
	if !ok {
		return ""
	}
	items, ok := data["rich_text"].([]any)
	if !ok {
		return ""
	}
	return renderRichTextItems(items)
}

func renderRichTextItems(items []any) string {
	var b strings.Builder
	for _, item := range items {
		rt, ok := item.(map[string]any)
		if !ok {
			continue
		}
		b.WriteString(renderRichText(rt))
	}
	return b.String()
}

func renderRichText(item map[string]any) string {
	text, _ := item["plain_text"].(string)
	if t, ok := item["type"].(string); ok && t == "equation" {
		eq, _ := item["equation"].(map[string]any)
		if expr, ok := eq["expression"].(string); ok {
			text = "$" + expr + "$"
		}
	}

	annotations, _ := item["annotations"].(map[string]any)
	text = applyAnnotations(text, annotations)

	if href, ok := item["href"].(string); ok && href != "" {
		text = fmt.Sprintf("[%s](%s)", text, href)
	}

	return text
}

func applyAnnotations(text string, annotations map[string]any) string {
	if annotations == nil {
		return text
	}
	if code, _ := annotations["code"].(bool); code {
		text = "`" + text + "`"
	}
	if bold, _ := annotations["bold"].(bool); bold {
		text = "**" + text + "**"
	}
	if italic, _ := annotations["italic"].(bool); italic {
		text = "*" + text + "*"
	}
	if strike, _ := annotations["strikethrough"].(bool); strike {
		text = "~~" + text + "~~"
	}
	if underline, _ := annotations["underline"].(bool); underline {
		text = "<u>" + text + "</u>"
	}
	return text
}

func renderIcon(icon any) string {
	data, ok := icon.(map[string]any)
	if !ok {
		return ""
	}
	if t, ok := data["type"].(string); ok && t == "emoji" {
		if emoji, ok := data["emoji"].(string); ok {
			return emoji
		}
	}
	return ""
}

func extractFileURL(data map[string]any) string {
	if file, ok := data["file"].(map[string]any); ok {
		if url, ok := file["url"].(string); ok {
			return url
		}
	}
	if external, ok := data["external"].(map[string]any); ok {
		if url, ok := external["url"].(string); ok {
			return url
		}
	}
	return ""
}

func extractFileName(data map[string]any) string {
	if file, ok := data["file"].(map[string]any); ok {
		if name, ok := file["name"].(string); ok {
			return name
		}
	}
	if name, ok := data["name"].(string); ok {
		return name
	}
	return ""
}

func escapeTable(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func padRow(row []string, length int) []string {
	if len(row) >= length {
		return row
	}
	out := make([]string, length)
	copy(out, row)
	for i := len(row); i < length; i++ {
		out[i] = ""
	}
	return out
}

func indentLines(text string, indent int) string {
	if indent == 0 {
		return text
	}
	prefix := strings.Repeat("  ", indent)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = line
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func isListItem(t string) bool {
	return t == "bulleted_list_item" || t == "numbered_list_item" || t == "to_do"
}

func notionURLFromID(id string) string {
	compact := strings.ReplaceAll(id, "-", "")
	return fmt.Sprintf("https://www.notion.so/%s", compact)
}
