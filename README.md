# Agent Align

![Agent Align](docs/icon-resized.png)

agent-align is a Go-based utility that keeps MCP configuration files aligned
across coding agents such as Copilot, VSCode, Codex, Claude Code, Gemini, and
others. Define your MCP servers once in `agent-align-mcp.yml` and agent-align
converts that configuration into the formats required by each tool while
applying agent-specific tweaks automatically. Detailed documentation is hosted
on GitHub Pages at <https://timbuchinger.github.io/agent-align/>.

## Repository layout

- `go.mod` pins the project to Go 1.25.4.
- `cmd/agent-align` contains the CLI entrypoint that reads the MCP definitions
  file plus the target config and prints the converted configs for the supported
  agents.
- `internal/syncer` implements the conversion logic, transformation layer, and
  accompanying unit tests.

## Getting started

### Installation

You can install agent-align using one of the following methods:

**Homebrew (macOS and Linux):**

```bash
brew install timbuchinger/agent-align/agent-align
```

**Manual download:**

1. Download the latest binary from the
   [releases page](https://github.com/timbuchinger/agent-align/releases/latest).

**Build from source:**

```bash
go build ./cmd/agent-align
```

### Configuration

1. Create an MCP definitions file to act as the source of truth; for example,
   save this to `agent-align-mcp.yml` next to the binary:

   ```yaml
   servers:
     github:
       type: streamable-http
       url: https://api.example.com/mcp/
       headers:
         Authorization: "Bearer ${GITHUB_TOKEN}"
   ```

   The MCP definitions file supports environment variable expansion using
   `${VAR}` or `$VAR` syntax, allowing you to securely reference secrets from
   your environment. Default values can be specified with `${VAR:-default}`
   syntax. See `config-mcp.example.yml` for a more complete template with
   command-based servers and environment variables.

2. Create a target config; for example, save this to `agent-align.yml`:

   ```yaml
   mcpServers:
     targets:
       agents:
         - copilot
         - vscode
         - codex
         - kilocode
         - claudecode
         - gemini
   ```

   You can also run `agent-align init -config /path/to/agent-align.yml` to walk
   through the same settings interactively.

3. Run the app with your config files:

   ```bash
   ./agent-align -config agent-align.yml -mcp-config agent-align-mcp.yml
   ```

## CLI Options

Option | Description
------ | -----------
`-agents` | Comma-separated list of agents to sync
`-config` | Path to YAML configuration file for targets and extra copy rules
`-mcp-config` | Path to the base MCP YAML file
`-dry-run` | Only show what would be changed without applying changes
`-confirm` | Skip user confirmation prompt (useful for cron jobs)

Defaults:

- Agents: `copilot,vscode,codex,claudecode,gemini,kilocode`
- MCP config path: `agent-align-mcp.yml` in the same directory as the target config

Use `-agents` to override the target list from the config file. If you omit
`-agents`, the CLI requires a config file so it can pick up the targets and any
path overrides. The MCP definitions are always read from the YAML file provided
via `-mcp-config` (or the default path).

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
go run ./cmd/agent-align -config agent-align.yml -mcp-config agent-align-mcp.yml
```

Use `-agents` if you need to override values in the config for a single run.

## Documentation linting

When editing markdown, run the lint fixer to download the tool and apply all
reported fixes:

```bash
npx markdownlint-cli2 --fix '**/*.md'
```

Note: Internal design notes and maintenance docs (for example, formatter
design and test-coverage TODOs) have been moved to `internal/docs/` so the
`docs/` folder contains only user-facing documentation used by the published
site.

## Configuration file

`agent-align` relies on two YAML files:

1. **MCP definitions (`agent-align-mcp.yml`)** – lists the complete server map
   using the neutral structure shown in the getting started section (see
   `config-mcp.example.yml`).
2. **Target config (`agent-align.yml`)** – describes where to load the MCP
   definitions and which agents/extra files should be updated (see
   `config.example.yml`).

The target config is searched at platform-specific paths (`/etc/agent-align.yml`
on Linux, `/usr/local/etc/agent-align.yml` on macOS, and
`C:\ProgramData\agent-align\config.yml` on Windows). Override the path with
`-config <path>`. Within that file, set `mcpServers.configPath` to point to the
MCP definitions (defaults to `agent-align-mcp.yml` next to the config) and list
the agents under `mcpServers.targets.agents`. Each entry can be either a
string (agent name) or a mapping with a `name` plus optional destination `path`.
Repeat an agent entry with different `path` values if you want the same format
written to multiple destinations (for example, two Gemini installs).
Add entries under `targets.additionalTargets.json` to mirror the MCP payload
into other JSON files (each entry specifies `filePath` and the `jsonPath` where
the servers belong). See `CONFIGURATION.md` for the full schema and additional
examples.

Add the optional top-level `extraTargets` block to copy files or directories
alongside the MCP sync. Use `extraTargets.files` to mirror a single file into
multiple destinations (for example, `AGENTS.md` into multiple worktrees). Use
`extraTargets.directories` to copy a folder to one or more other locations
(`destinations` is a list of objects with `path` and optional `flatten`) so you
can decide which destinations keep their directory structure.

Every run prints the generated configurations and extra copy destinations so
you can review the plan. Pass `-dry-run` to exit after the preview or `-confirm`
to skip the interactive prompt when applying the changes.

## Supported Agents

Agent | Config File | Format | Node Name
----- | ----------- | ------ | ---------
copilot | `~/.copilot/mcp-config.json` | JSON | `mcpServers`
vscode | `~/.config/Code/User/mcp.json` | JSON | `servers`
codex | `~/.codex/config.toml` | TOML | `mcp_servers`
claudecode | `~/.claude.json` | JSON | `mcpServers`
gemini | `~/.gemini/settings.json` | JSON | `mcpServers`
kilocode | Platform-dependent (see note below) | JSON | `mcpServers`

## Testing

Note: Kilocode config paths

- Windows: `~/AppData/Roaming/Code/user/mcp.json`
- Linux: `~/.config/Code/User/globalStorage/kilocode.kilo-code/settings/mcp_settings.json`

Run:

```bash
GOCACHE=/tmp/agent-align-go-cache go test -coverprofile=coverage.out ./...
```

The unit tests cover template loading and the syncer's validation/conversion
logic.

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
