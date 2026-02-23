package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/inforix/notion-cli/src/internal/config"
	"github.com/inforix/notion-cli/src/internal/export"
	"github.com/inforix/notion-cli/src/internal/notion"
	"github.com/inforix/notion-cli/src/internal/output"
	"github.com/inforix/notion-cli/src/internal/pagination"
	"github.com/spf13/cobra"
)

const envNotionToken = "NOTION_TOKEN"

var version = "dev"

func main() {
	rootOpts := &rootOptions{}

	root := &cobra.Command{
		Use:   "notion",
		Short: fmt.Sprintf("Notion CLI (%s)", version),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			showVersion, _ := cmd.Flags().GetBool("version")
			if showVersion {
				fmt.Fprintln(cmd.OutOrStdout(), version)
				os.Exit(0)
			}
			return rootOpts.loadConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			showVersion, _ := cmd.Flags().GetBool("version")
			if showVersion {
				fmt.Fprintln(cmd.OutOrStdout(), version)
				return nil
			}
			return cmd.Help()
		},
	}

	root.PersistentFlags().StringVar(&rootOpts.token, "token", "", "Notion integration token")
	root.PersistentFlags().StringVar(&rootOpts.notionVersion, "notion-version", "", "Notion API version")
	root.PersistentFlags().BoolVar(&rootOpts.pretty, "pretty", false, "Pretty-print JSON output")
	root.PersistentFlags().StringVar(&rootOpts.format, "format", "", "Go template string or @file for output formatting")
	root.PersistentFlags().IntVar(&rootOpts.pageSize, "page-size", 0, "Page size for list endpoints")
	root.PersistentFlags().StringVar(&rootOpts.cursor, "cursor", "", "Start cursor for list endpoints")
	root.PersistentFlags().BoolVar(&rootOpts.all, "all", false, "Auto-paginate list endpoints")
	root.PersistentFlags().BoolP("version", "V", false, "Print version and exit")

	root.AddCommand(newAuthCommand(rootOpts))
	root.AddCommand(newPagesCommand(rootOpts))
	root.AddCommand(newDatabasesCommand(rootOpts))
	root.AddCommand(newBlocksCommand(rootOpts))
	root.AddCommand(newUsersCommand(rootOpts))
	root.AddCommand(newCommentsCommand(rootOpts))
	root.AddCommand(newSearchCommand(rootOpts))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

type rootOptions struct {
	token         string
	notionVersion string
	pretty        bool
	format        string
	pageSize      int
	cursor        string
	all           bool
	cfg           *config.Config
	cfgPath       string
}

func (o *rootOptions) loadConfig() error {
	if o.cfg != nil {
		return nil
	}
	cfg, path, err := config.Load()
	if err != nil {
		return err
	}
	o.cfg = cfg
	o.cfgPath = path
	return nil
}

func (o *rootOptions) resolvedToken() (string, error) {
	if strings.TrimSpace(o.token) != "" {
		return o.token, nil
	}
	if token := strings.TrimSpace(os.Getenv(envNotionToken)); token != "" {
		return token, nil
	}
	if o.cfg != nil && strings.TrimSpace(o.cfg.Token) != "" {
		return o.cfg.Token, nil
	}
	return "", errors.New("missing Notion token: use --token, set NOTION_TOKEN, or run notion auth set")
}

func (o *rootOptions) resolvedNotionVersion() string {
	if strings.TrimSpace(o.notionVersion) != "" {
		return o.notionVersion
	}
	if o.cfg != nil && strings.TrimSpace(o.cfg.NotionVersion) != "" {
		return o.cfg.NotionVersion
	}
	return notion.DefaultNotionVersion
}

func (o *rootOptions) newClient() (*notion.Client, error) {
	token, err := o.resolvedToken()
	if err != nil {
		return nil, err
	}
	client := notion.NewClient(token, o.resolvedNotionVersion())
	return client, nil
}

func newAuthCommand(root *rootOptions) *cobra.Command {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Manage auth tokens",
	}

	set := &cobra.Command{
		Use:   "set",
		Short: "Store a Notion token in config",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString("token")
			if strings.TrimSpace(token) == "" {
				token = os.Getenv(envNotionToken)
			}
			if strings.TrimSpace(token) == "" {
				return errors.New("token required (use --token or set NOTION_TOKEN)")
			}
			cfg := &config.Config{Token: token, NotionVersion: root.resolvedNotionVersion()}
			path, err := config.Save(cfg)
			if err != nil {
				return err
			}
			fmt.Printf("saved token to %s\n", path)
			return nil
		},
	}
	set.Flags().String("token", "", "Notion integration token")

	status := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgToken := ""
			if root.cfg != nil {
				cfgToken = root.cfg.Token
			}
			fmt.Printf("config: %s\n", mask(cfgToken))
			fmt.Printf("env %s: %s\n", envNotionToken, mask(os.Getenv(envNotionToken)))
			return nil
		},
	}

	clear := &cobra.Command{
		Use:   "clear",
		Short: "Remove stored token",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.Clear()
			if err != nil {
				return err
			}
			fmt.Printf("removed %s\n", path)
			return nil
		},
	}

	auth.AddCommand(set, status, clear)
	return auth
}

func newPagesCommand(root *rootOptions) *cobra.Command {
	pages := &cobra.Command{
		Use:   "pages",
		Short: "Operate on pages",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pages (search wrapper)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			query, _ := cmd.Flags().GetString("query")
			baseBody, err := readBodyMap(bodyArg)
			if err != nil {
				return err
			}
			baseBody = ensurePageSearchFilter(baseBody)
			if strings.TrimSpace(query) != "" {
				baseBody["query"] = query
			}

			fetch := func(cursor string) (map[string]any, *http.Response, error) {
				payload, err := bodyWithPagination(baseBody, root.pageSize, cursor)
				if err != nil {
					return nil, nil, err
				}
				return client.Search(context.Background(), payload)
			}

			data, resp, err := fetchList(root, fetch)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			if strings.TrimSpace(query) != "" {
				data = filterResultsByTitle(data, query)
			}
			return renderData(root, data)
		},
	}
	listCmd.Flags().String("query", "", "Search query text")
	listCmd.Flags().String("body", "", "JSON body, @file, or @- for stdin")

	getCmd := &cobra.Command{
		Use:   "get <page_id>",
		Short: "Retrieve a page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			data, resp, err := client.GetPage(context.Background(), args[0])
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a page",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}
			data, resp, err := client.CreatePage(context.Background(), payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	createCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	updateCmd := &cobra.Command{
		Use:   "update <page_id>",
		Short: "Update a page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}
			data, resp, err := client.UpdatePage(context.Background(), args[0], payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	updateCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	archiveCmd := &cobra.Command{
		Use:   "archive <page_id>",
		Short: "Archive a page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			payload := []byte(`{"archived": true}`)
			data, resp, err := client.UpdatePage(context.Background(), args[0], payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}

	exportCmd := &cobra.Command{
		Use:   "export <page_id>",
		Short: "Export a page to Markdown",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			mode, _ := cmd.Flags().GetString("assets")
			outputPath, _ := cmd.Flags().GetString("output")
			includeFrontmatter, _ := cmd.Flags().GetBool("frontmatter")
			includeTitle, _ := cmd.Flags().GetBool("title")

			md, err := export.ExportPageToMarkdown(context.Background(), client, args[0], export.Options{
				IncludeFrontmatter: includeFrontmatter,
				IncludeTitle:       includeTitle,
				AssetMode:          export.AssetMode(mode),
				OutputPath:         outputPath,
			})
			if err != nil {
				return err
			}

			if outputPath == "" {
				fmt.Println(md)
				return nil
			}
			return writeFile(outputPath, md)
		},
	}

	exportCmd.Flags().String("assets", "link", "Asset handling: link|download|inline")
	exportCmd.Flags().StringP("output", "o", "", "Write output to file")
	exportCmd.Flags().Bool("frontmatter", true, "Include YAML frontmatter")
	exportCmd.Flags().Bool("title", true, "Include page title as H1")

	pages.AddCommand(listCmd, getCmd, createCmd, updateCmd, archiveCmd, exportCmd)
	return pages
}

func newDatabasesCommand(root *rootOptions) *cobra.Command {
	databases := &cobra.Command{
		Use:   "databases",
		Short: "Operate on databases",
	}

	getCmd := &cobra.Command{
		Use:   "get <database_id>",
		Short: "Retrieve a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			data, resp, err := client.GetDatabase(context.Background(), args[0])
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}

	queryCmd := &cobra.Command{
		Use:   "query <database_id>",
		Short: "Query a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			baseBody, err := readBodyMap(bodyArg)
			if err != nil {
				return err
			}

			fetch := func(cursor string) (map[string]any, *http.Response, error) {
				payload, err := bodyWithPagination(baseBody, root.pageSize, cursor)
				if err != nil {
					return nil, nil, err
				}
				return client.QueryDatabase(context.Background(), args[0], payload)
			}

			data, resp, err := fetchList(root, fetch)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	queryCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}
			data, resp, err := client.CreateDatabase(context.Background(), payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	createCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	updateCmd := &cobra.Command{
		Use:   "update <database_id>",
		Short: "Update a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}
			data, resp, err := client.UpdateDatabase(context.Background(), args[0], payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	updateCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	databases.AddCommand(getCmd, queryCmd, createCmd, updateCmd)
	return databases
}

func newBlocksCommand(root *rootOptions) *cobra.Command {
	blocks := &cobra.Command{
		Use:   "blocks",
		Short: "Operate on blocks",
	}

	getCmd := &cobra.Command{
		Use:   "get <block_id>",
		Short: "Retrieve a block",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			data, resp, err := client.GetBlock(context.Background(), args[0])
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update <block_id>",
		Short: "Update a block",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}
			data, resp, err := client.UpdateBlock(context.Background(), args[0], payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	updateCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	childrenCmd := &cobra.Command{
		Use:   "children",
		Short: "Operate on block children",
	}

	appendCmd := &cobra.Command{
		Use:   "append <block_id>",
		Short: "Append children to a block",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}
			data, resp, err := client.AppendBlockChildren(context.Background(), args[0], payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	appendCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	childrenCmd.AddCommand(appendCmd)
	blocks.AddCommand(getCmd, updateCmd, childrenCmd)
	return blocks
}

func newUsersCommand(root *rootOptions) *cobra.Command {
	users := &cobra.Command{
		Use:   "users",
		Short: "Operate on users",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}

			fetch := func(cursor string) (map[string]any, *http.Response, error) {
				return client.ListUsers(context.Background(), cursor, root.pageSize)
			}

			data, resp, err := fetchList(root, fetch)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <user_id>",
		Short: "Retrieve a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			data, resp, err := client.GetUser(context.Background(), args[0])
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}

	users.AddCommand(listCmd, getCmd)
	return users
}

func newCommentsCommand(root *rootOptions) *cobra.Command {
	comments := &cobra.Command{
		Use:   "comments",
		Short: "Operate on comments",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List comments",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			blockID, _ := cmd.Flags().GetString("block-id")
			pageID, _ := cmd.Flags().GetString("page-id")
			blockID = strings.TrimSpace(blockID)
			pageID = strings.TrimSpace(pageID)
			if (blockID == "" && pageID == "") || (blockID != "" && pageID != "") {
				return errors.New("specify exactly one of --block-id or --page-id")
			}

			fetch := func(cursor string) (map[string]any, *http.Response, error) {
				return client.ListComments(context.Background(), blockID, pageID, cursor, root.pageSize)
			}

			data, resp, err := fetchList(root, fetch)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	listCmd.Flags().String("block-id", "", "Filter comments by block ID")
	listCmd.Flags().String("page-id", "", "Filter comments by page ID")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a comment",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}
			data, resp, err := client.CreateComment(context.Background(), payload)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}
	createCmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	comments.AddCommand(listCmd, createCmd)
	return comments
}

func newSearchCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search for pages or databases",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			bodyArg, _ := cmd.Flags().GetString("body")
			baseBody, err := readBodyMap(bodyArg)
			if err != nil {
				return err
			}

			fetch := func(cursor string) (map[string]any, *http.Response, error) {
				payload, err := bodyWithPagination(baseBody, root.pageSize, cursor)
				if err != nil {
					return nil, nil, err
				}
				return client.Search(context.Background(), payload)
			}

			data, resp, err := fetchList(root, fetch)
			if err != nil {
				return err
			}
			if err := ensureSuccess(resp, data); err != nil {
				return err
			}
			return renderData(root, data)
		},
	}

	cmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")
	return cmd
}

type listFetcher func(cursor string) (map[string]any, *http.Response, error)

func fetchList(root *rootOptions, fetch listFetcher) (map[string]any, *http.Response, error) {
	if !root.all {
		return fetch(root.cursor)
	}

	var (
		allResults []any
		cursor     = root.cursor
		lastResp   *http.Response
	)

	for {
		data, resp, err := fetch(cursor)
		if err != nil {
			return nil, resp, err
		}
		if err := ensureSuccess(resp, data); err != nil {
			return data, resp, err
		}
		list, err := pagination.Parse(data)
		if err != nil {
			return data, resp, err
		}
		allResults = append(allResults, list.Results...)
		lastResp = resp
		if !list.HasMore || list.NextCursor == "" {
			break
		}
		cursor = list.NextCursor
	}

	return pagination.Combine(allResults), lastResp, nil
}

func readBodyArg(arg string) ([]byte, error) {
	if strings.HasPrefix(arg, "@") {
		path := strings.TrimPrefix(arg, "@")
		if path == "-" {
			return io.ReadAll(os.Stdin)
		}
		return os.ReadFile(path)
	}
	return []byte(arg), nil
}

func readBodyMap(arg string) (map[string]any, error) {
	body, err := readBodyArg(arg)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return map[string]any{}, nil
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func bodyWithPagination(base map[string]any, pageSize int, cursor string) ([]byte, error) {
	payload := copyMap(base)
	if pageSize > 0 {
		payload["page_size"] = pageSize
	}
	if strings.TrimSpace(cursor) != "" {
		payload["start_cursor"] = cursor
	}
	return json.Marshal(payload)
}

func copyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func ensurePageSearchFilter(body map[string]any) map[string]any {
	if _, ok := body["filter"]; ok {
		return body
	}
	body["filter"] = map[string]any{
		"property": "object",
		"value":    "page",
	}
	return body
}

func filterResultsByTitle(data map[string]any, query string) map[string]any {
	results, ok := data["results"].([]any)
	if !ok {
		return data
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return data
	}
	queryLower := strings.ToLower(query)
	filtered := make([]any, 0, len(results))
	for _, item := range results {
		page, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := pageTitle(page)
		if title == "" {
			continue
		}
		titleLower := strings.ToLower(title)
		if strings.Contains(title, query) || strings.Contains(titleLower, queryLower) {
			filtered = append(filtered, item)
		}
	}
	data["results"] = filtered
	data["has_more"] = false
	data["next_cursor"] = nil
	return data
}

func pageTitle(page map[string]any) string {
	props, ok := page["properties"].(map[string]any)
	if !ok {
		return ""
	}
	for _, prop := range props {
		m, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] != "title" {
			continue
		}
		items, ok := m["title"].([]any)
		if !ok {
			return ""
		}
		var b strings.Builder
		for _, item := range items {
			t, ok := item.(map[string]any)
			if !ok {
				continue
			}
			text, _ := t["plain_text"].(string)
			b.WriteString(text)
		}
		return b.String()
	}
	return ""
}

func renderData(root *rootOptions, data map[string]any) error {
	text, err := output.Render(data, root.pretty, root.format)
	if err != nil {
		return err
	}
	fmt.Println(text)
	return nil
}

func ensureSuccess(resp *http.Response, data map[string]any) error {
	if resp == nil {
		return errors.New("no response from Notion API")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("notion API error (%d): %s", resp.StatusCode, output.ErrorMessage(data))
	}
	return nil
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func mask(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "<empty>"
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
