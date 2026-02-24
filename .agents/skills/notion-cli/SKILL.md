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
notion pages export <page_id>
notion search --body @query.json
notion files create --body @file-upload.json
notion files read <file_upload_id>
notion files list --page-size 100
notion files read --body @file.json --output ./downloads/
```

## Notes

- The CLI uses the Notion API and includes a `Notion-Version` header by default.
- Use `--notion-version` to override when testing a specific API version.
