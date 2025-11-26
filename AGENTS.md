# AGENTS

This repository powers **agent-align**, a Go utility for synchronizing MCP (model
configuration profile) configs across coding agents such as Copilot, Codex, Claude
Code, Gemini, and others. It keeps every agent's configuration in lockstep so you
can iterate on a single template and automatically propagate changes to the rest
of the toolchain.

You provide one file as the template, and agent-align converts it into the
formats required by the other agents. One of the agent-specific outputs is chosen
as the source of truth, and the tool uses that to update the remaining files so
all the agents stay in sync.

All commits must follow Conventional Commits so the release workflow can determine
the next semantic version automatically.

## Commit message requirements

Every commit message must follow these rules:

- **Type is required**: Start with a valid type (e.g., `feat:`, `fix:`, `docs:`,
  `chore:`, `refactor:`, `test:`, `ci:`)
- **Subject is required**: Must have a non-empty subject after the type
- **Body line length**: If including a body, each line must not exceed 100
  characters

Example valid commit message:

```text
feat: add VS Code agent support

Add VS Code as a supported agent with config path ~/.config/Code/User/mcp.json
and root element "servers".
```

## Go build cache

Set the Go build cache to a writable directory before running `go build` or
`go test`. All agents must export `GOCACHE=/tmp/agent-align-go-cache` (or a
similar `/tmp` path they control) so the compiler does not attempt to write to
unwritable home directories in sandboxed environments.

## Markdown requirements

All markdown changes must run through `markdownlint-cli2` and have every reported
issue resolved before merging. Run it via `npx markdownlint-cli2 --fix '**/*.md'`
to download the CLI locally and fix issues in every markdown file.
Treat the tool as the single source of truth for markdown style so synchronized
documentation stays consistent.

## Recommended VS Code extensions

When opening this repository in Visual Studio Code, install
`ext:DavidAnson.vscode-markdownlint` so markdownlint warnings surface locally.
