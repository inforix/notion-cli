---
name: notion-cli
description: Use this skill to operate Notion via the `notion` CLI built from this repo, including install steps and auth setup.
---

# Notion CLI Skill

Use the `notion` binary from this repo to operate Notion. Prefer the local build when developing.

## Install

- Homebrew: `brew install inforix/tap/notion-cli`
- Release binary: download from GitHub releases and place on PATH
- Local build: `go build -o ./bin/notion ./src/cmd/notion`

## Auth

Set a Notion integration token in the environment:

```
export NOTION_TOKEN=YOUR_TOKEN
```

Optional: store the token with the CLI (if supported):

```
notion auth set --token YOUR_TOKEN
```

## Examples

```
notion --version
notion pages list --query "Project"
notion pages export <page_id>
notion pages get <page_id>
notion pages create --body @page.json
notion pages update <page_id> --body @update.json
notion pages archive <page_id>
notion databases query <db_id> --body @query.json
notion blocks children append <block_id> --body @children.json
notion users list --all
notion comments list --page-id <page_id>
notion search --body @query.json
```

## Notes

- The CLI uses the Notion API and includes a `Notion-Version` header by default.
- Use `--notion-version` to override when testing a specific API version.
- Use `--page-size`, `--cursor`, and `--all` for list pagination.
- Use `--format` to render Go templates from JSON output.
