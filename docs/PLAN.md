# Repo + Skill Setup Plan (Notion CLI)

## Summary
Create the public GitHub repo `inforix/notion-cli`, initialize it with standard project files, structure source under `./src/`, configure GoReleaser (including Homebrew tap `inforix/homebrew-tap`), and add a repo-local `notion-cli` skill that documents how to install and run the `notion` binary and set `NOTION_TOKEN`.

## Assumptions & Defaults
- Binary name is `notion`.
- Skill location: `./.agents/skills/notion-cli/`.
- Homebrew tap repo: `inforix/homebrew-tap`.
- Build outputs go to `./bin/` or `./dist/` and are ignored in git.

## Plan Steps
1. Verify GitHub auth and org access.
2. Create and initialize the repo with `README.md`, `LICENSE`, `.gitignore`.
3. Place source under `./src/` with a minimal CLI entry point.
4. Add `.goreleaser.yaml` with Homebrew tap configuration.
5. Create the `notion-cli` skill under `./.agents/skills/notion-cli/`.

## Tests/Verification
- `goreleaser check`
- `go build -o ./bin/notion ./src/cmd/notion`
- `./bin/notion --help`
