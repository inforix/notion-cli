package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/inforix/notion-cli/src/internal/config"
	"github.com/inforix/notion-cli/src/internal/export"
	"github.com/inforix/notion-cli/src/internal/notion"
	"github.com/inforix/notion-cli/src/internal/output"
	"github.com/spf13/cobra"
)

const envNotionToken = "NOTION_TOKEN"

func main() {
	rootOpts := &rootOptions{}

	root := &cobra.Command{
		Use:   "notion",
		Short: "Notion CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return rootOpts.loadConfig()
		},
	}

	root.PersistentFlags().StringVar(&rootOpts.token, "token", "", "Notion integration token")
	root.PersistentFlags().StringVar(&rootOpts.notionVersion, "notion-version", "", "Notion API version")
	root.PersistentFlags().BoolVar(&rootOpts.pretty, "pretty", false, "Pretty-print JSON output")

	root.AddCommand(newAuthCommand(rootOpts))
	root.AddCommand(newPagesCommand(rootOpts))
	root.AddCommand(newFilesCommand(rootOpts))
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

	pages.AddCommand(exportCmd)
	return pages
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
			payload, err := readBodyArg(bodyArg)
			if err != nil {
				return err
			}

			data, resp, err := client.Search(context.Background(), payload)
			if err != nil {
				return err
			}
			if resp.StatusCode >= 400 {
				return fmt.Errorf("notion API error (%d): %s", resp.StatusCode, output.ErrorMessage(data))
			}
			text, err := output.JSON(data, root.pretty)
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		},
	}

	cmd.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")
	return cmd
}

func newFilesCommand(root *rootOptions) *cobra.Command {
	files := &cobra.Command{
		Use:   "files",
		Short: "Operate on file uploads",
	}

	create := &cobra.Command{
		Use:   "create",
		Short: "Create a file upload",
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

			data, resp, err := client.CreateFileUpload(context.Background(), payload)
			if err != nil {
				return err
			}
			if resp.StatusCode >= 400 {
				return fmt.Errorf("notion API error (%d): %s", resp.StatusCode, output.ErrorMessage(data))
			}

			text, err := output.JSON(data, root.pretty)
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		},
	}
	create.Flags().String("body", "{}", "JSON body, @file, or @- for stdin")

	read := &cobra.Command{
		Use:     "read [file_upload_id]",
		Aliases: []string{"get"},
		Short:   "Retrieve a file upload or download a file object",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyArg, _ := cmd.Flags().GetString("body")
			outputPath, _ := cmd.Flags().GetString("output")

			if bodyArg != "" && len(args) > 0 {
				return errors.New("use either <file_upload_id> or --body, not both")
			}

			var data map[string]any
			if bodyArg != "" {
				payload, err := readBodyArg(bodyArg)
				if err != nil {
					return err
				}
				data, err = parseJSONMap(payload)
				if err != nil {
					return err
				}
			} else {
				if len(args) == 0 {
					return errors.New("file_upload_id required unless --body is provided")
				}
				client, err := root.newClient()
				if err != nil {
					return err
				}

				respData, resp, err := client.GetFileUpload(context.Background(), args[0])
				if err != nil {
					return err
				}
				if resp.StatusCode >= 400 {
					return fmt.Errorf("notion API error (%d): %s", resp.StatusCode, output.ErrorMessage(respData))
				}
				data = respData
			}

			if outputPath != "" {
				index, _ := cmd.Flags().GetInt("index")
				urlStr, _, err := extractFileURLAt(data, index)
				if err != nil {
					return err
				}
				return downloadTo(outputPath, urlStr)
			}

			text, err := output.JSON(data, root.pretty)
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		},
	}
	read.Flags().String("body", "", "File object JSON, @file, or @- for stdin")
	read.Flags().StringP("output", "o", "", "Write file contents to path")
	read.Flags().Int("index", 0, "Index for files list objects")

	list := &cobra.Command{
		Use:   "list",
		Short: "List file uploads",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := root.newClient()
			if err != nil {
				return err
			}
			pageSize, _ := cmd.Flags().GetInt("page-size")
			startCursor, _ := cmd.Flags().GetString("start-cursor")

			data, resp, err := client.ListFileUploads(context.Background(), pageSize, startCursor)
			if err != nil {
				return err
			}
			if resp.StatusCode >= 400 {
				return fmt.Errorf("notion API error (%d): %s", resp.StatusCode, output.ErrorMessage(data))
			}

			text, err := output.JSON(data, root.pretty)
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		},
	}
	list.Flags().Int("page-size", 0, "Number of results per page")
	list.Flags().String("start-cursor", "", "Pagination cursor")

	files.AddCommand(create, read, list)
	return files
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

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func parseJSONMap(body []byte) (map[string]any, error) {
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func extractFileURLAt(data map[string]any, index int) (string, string, error) {
	if data == nil {
		return "", "", errors.New("empty file object")
	}
	if objectType, ok := data["object"].(string); ok && objectType == "file_upload" {
		return "", "file_upload", errors.New("file upload objects do not include a downloadable URL")
	}

	if t, ok := data["type"].(string); ok {
		switch t {
		case "external":
			if external, ok := data["external"].(map[string]any); ok {
				if urlStr, ok := external["url"].(string); ok && urlStr != "" {
					return urlStr, t, nil
				}
			}
			return "", t, errors.New("external file object missing url")
		case "file":
			if file, ok := data["file"].(map[string]any); ok {
				if urlStr, ok := file["url"].(string); ok && urlStr != "" {
					return urlStr, t, nil
				}
			}
			return "", t, errors.New("file object missing url")
		case "file_upload":
			return "", t, errors.New("file_upload objects do not include a downloadable URL")
		case "files":
			if index < 0 {
				return "", t, errors.New("index must be >= 0")
			}
			list, ok := data["files"].([]any)
			if !ok || len(list) == 0 {
				return "", t, errors.New("files list is empty")
			}
			if index >= len(list) {
				return "", t, fmt.Errorf("files list index %d out of range", index)
			}
			item, ok := list[index].(map[string]any)
			if !ok {
				return "", t, errors.New("files list item is not an object")
			}
			return extractFileURLAt(item, 0)
		}
	}

	if external, ok := data["external"].(map[string]any); ok {
		if urlStr, ok := external["url"].(string); ok && urlStr != "" {
			return urlStr, "external", nil
		}
	}
	if file, ok := data["file"].(map[string]any); ok {
		if urlStr, ok := file["url"].(string); ok && urlStr != "" {
			return urlStr, "file", nil
		}
	}

	return "", "", errors.New("file URL not found in object")
}

func downloadTo(outputPath string, urlStr string) error {
	if strings.TrimSpace(outputPath) == "" {
		return errors.New("output path required")
	}
	if strings.TrimSpace(urlStr) == "" {
		return errors.New("file URL missing")
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	if outputPath == "-" {
		_, err := io.Copy(os.Stdout, resp.Body)
		return err
	}

	finalPath := outputPath
	if isDirPath(outputPath) {
		if err := os.MkdirAll(outputPath, 0o755); err != nil {
			return err
		}
		name := filenameFromContentDisposition(resp.Header.Get("Content-Disposition"))
		if name == "" {
			name = filenameFromURL(urlStr)
		}
		if name == "" {
			name = "file"
		}
		finalPath = filepath.Join(outputPath, name)
	} else {
		if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
			return err
		}
	}

	out, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func isDirPath(value string) bool {
	if strings.HasSuffix(value, string(os.PathSeparator)) {
		return true
	}
	info, err := os.Stat(value)
	return err == nil && info.IsDir()
}

func filenameFromContentDisposition(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}
	if name, ok := params["filename*"]; ok {
		if strings.HasPrefix(strings.ToLower(name), "utf-8''") {
			decoded, err := url.QueryUnescape(name[len("utf-8''"):])
			if err == nil {
				name = decoded
			}
		}
		return sanitizeFilename(name)
	}
	if name, ok := params["filename"]; ok {
		return sanitizeFilename(name)
	}
	return ""
}

func filenameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	name := path.Base(parsed.Path)
	return sanitizeFilename(name)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, string(os.PathSeparator), "-")
	if name == "" || name == "." || name == ".." {
		return ""
	}
	return name
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
