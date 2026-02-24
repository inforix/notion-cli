# notion-cli

A CLI for Notion, built in Go.

## Install

### Homebrew

```bash
brew install inforix/tap/notion-cli
```

### Release binary

Download from the GitHub releases page and place the binary on your PATH.

### From source

```bash
go build -o ./bin/notion ./src/cmd/notion
```

## Auth

Set a Notion integration token:

```bash
export NOTION_TOKEN=YOUR_TOKEN
```

## Usage

```bash
notion --help
notion pages export <page_id>
notion files --help
```

### Export to Markdown

```bash
notion pages export <page_id> --assets=link
notion pages export <page_id> --assets=download -o page.md
```

### Auth

```bash
notion auth status
notion auth set --token YOUR_TOKEN
```

### Search

```bash
notion search --body @query.json
```

### File uploads

```bash
# Create a file upload
notion files create --body @file-upload.json

# Retrieve a file upload
notion files read <file_upload_id>

# List file uploads
notion files list --page-size 100

# Download a file from a file object (external or file URL)
notion files read --body @file.json --output ./downloads/
```

## Development

- Source code lives in `./src/`
- Build output goes to `./bin/` or `./dist/` (ignored by git)

## Releases

This project uses GoReleaser. To validate config locally:

```bash
goreleaser check
```
