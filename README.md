# Agent Align

![Agent Align](icon-resized.png)

agent-align is a Go-based utility that keeps MCP configuration files aligned
across coding agents such as Copilot, VSCode, Codex, Claude Code, Gemini, and
others. Give it a single template file, and it will convert that configuration
into the formats required by each tool while treating one format as the source
of truth. Detailed documentation is hosted on GitHub Pages at
<https://timbuchinger.github.io/agent-align/>.

## Repository layout

- `go.mod` pins the project to Go 1.25.4.
- `cmd/agent-align` contains the CLI entrypoint that reads a template file, chooses
  a source-of-truth agent, and prints the converted configs for the supported
  agents.
- `internal/syncer` implements the conversion logic, template loader, and
  accompanying unit tests.

## Getting started

1. Download the latest binary from the [releases page](https://github.com/timbuchinger/agent-align/releases/latest).
2. Create a config; for example, save this to `agent-align.yml` next to the binary:

   ```yaml
   source: codex
   targets:
     - copilot
     - vscode
     - claudecode
     - gemini
   ```

3. Run the app with your config file:

   ```bash
   ./agent-align -config agent-align.yml
   ```

## CLI Options

Option | Description
------ | -----------
`-source` | Source-of-truth agent name
`-agents` | Comma-separated list of agents to sync (defaults to copilot,vscode,codex,claudecode,gemini)
`-config` | Path to YAML configuration file
`-dry-run` | Only show what would be changed without applying changes
`-confirm` | Skip user confirmation prompt (useful for cron jobs)

Use `-source` and `-agents` together to run without a config file. Omit both
flags (or only set `-config`) to pull values from the YAML config; the CLI
rejects mixes of these options.

### Dry Run Mode

Use `-dry-run` to preview the changes without modifying any files:

```bash
./agent-align -config agent-align.yml -dry-run
```

This displays the converted configurations for each agent along with the target
file paths, but does not write any files.

### Non-Interactive Mode

Use `-confirm` to skip the confirmation prompt:

```bash
./agent-align -config agent-align.yml -confirm
```

This is useful when running agent-align from cron or other automated systems.
For example, this cron entry runs the sync every hour. Append
`>/tmp/agent-align.log 2>&1` if you want to capture logs:

```cron
0 * * * * agent-align -confirm
```

## Development commands

### Build

```bash
go build ./cmd/agent-align
```

This compiles the CLI into the current directory so you can run it repeatedly
without `go run`.

### Run

```bash
go run ./cmd/agent-align -config agent-align.yml
```

Use `-source` or `-agents` if you need to override values in the config for a
single run; the template is inferred from the selected source agent’s config.

## Documentation linting

When editing markdown, run the lint fixer to download the tool and apply all
reported fixes:

```bash
npx markdownlint-cli2 --fix '**/*.md'
```

## Configuration file

`agent-align` looks for a YAML configuration at one of the platform-specific locations:

- Linux: `/etc/agent-align.yml`
- macOS: `/usr/local/etc/agent-align.yml`
- Windows: `C:\ProgramData\agent-align\config.yml`

You can override this path with `-config <path>`. The file should describe the
`source` agent and the list of `targets`; see `CONFIGURATION.md` for the schema
and a sample layout. Config values are used unless you explicitly set both
`-source` and `-agents`. The CLI reads the actual configuration file for the
selected source agent (for example, `~/.codex/config.toml` when `source: codex`)
and uses it as the template automatically. If you provide `-source` and
`-agents`, the config file is ignored entirely and the CLI runs in a flag-only
mode. These flags cannot be combined with `-config`. If no config file is found
and you omit the CLI overrides, the CLI still defaults to `copilot`, `vscode`,
`codex`, `claudecode`, and `gemini`.

The tool will display the converted configurations for each agent and prompt
for confirmation before writing the changes (unless `-confirm` is specified).

## Supported Agents

Agent | Config File | Format | Node Name
----- | ----------- | ------ | ---------
copilot | `~/.copilot/mcp-config.json` | JSON | `mcpServers`
vscode | `~/.config/Code/User/mcp.json` | JSON | `servers`
codex | `~/.codex/config.toml` | TOML | N/A
claudecode | `~/.claude.json` | JSON | `mcpServers`
gemini | `~/.gemini/settings.json` | JSON | `mcpServers`

## Testing

Run:

```bash
go test ./...
```

The unit tests cover template loading and the syncer's validation/conversion logic.

## CI and releases

- **Tests** – `go test ./...` runs on every push and pull request so the core
  package stays green.
- **Commit message format** – a workflow enforces Conventional Commit-style
  messages so releases can be calculated automatically.
- **Release** – a manual workflow dispatch now runs Go tests first, pauses for a
  required approval in the `release-approval` environment, and then builds the
  release artifacts before `semantic-release` publishes the tag/release. During
  the approval step you can choose `auto`, `patch`, `minor`, `major`, or `none`
  for the release type so a reviewer can override the computed bump or skip a
  publication entirely.

### Manual release verification

Start the **Release** workflow from the *Actions* tab and pick a value for the
`release_type` input. The workflow runs the full test suite, waits for approval
in the `release-approval` environment, and then publishes via `semantic-release`
if the selected type is not `none`. Set the environment to require reviewers so
the workflow pauses until someone signs off. Choosing `auto` lets commit history
pick the bump, while the other values force the release type that reaches
semantic-release.
