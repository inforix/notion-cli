package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
