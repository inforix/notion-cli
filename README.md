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
```

### Pages

```bash
notion pages list --query "project" --all
notion pages get <page_id>
notion pages create --body @page.json
notion pages update <page_id> --body @update.json
notion pages archive <page_id>
notion pages export <page_id> --assets=download -o page.md
```

### Databases

```bash
notion databases get <db_id>
notion databases query <db_id> --body @query.json
notion databases create --body @database.json
notion databases update <db_id> --body @update.json
```

### Blocks

```bash
notion blocks get <block_id>
notion blocks update <block_id> --body @block.json
notion blocks children append <block_id> --body @children.json
```

### Users

```bash
notion users list
notion users get <user_id>
```

### Comments

```bash
notion comments list --page-id <page_id>
notion comments create --body @comment.json
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

### Output & Pagination

```bash
notion search --body @query.json --all --page-size 100
notion users list --format '{{json .results}}'
```

## Development

- Source code lives in `./src/`
- Build output goes to `./bin/` or `./dist/` (ignored by git)

## Releases

This project uses GoReleaser. To validate config locally:

```bash
goreleaser check
```
