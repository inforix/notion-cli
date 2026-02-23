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
```

## Development

- Source code lives in `./src/`
- Build output goes to `./bin/` or `./dist/` (ignored by git)

## Releases

This project uses GoReleaser. To validate config locally:

```bash
goreleaser check
```
