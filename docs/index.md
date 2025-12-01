# Agent Align

![Agent Align](icon-resized.png)

Agent Align is a Go-based utility that keeps MCP configuration files aligned
across coding agents such as Copilot, Codex, Claude Code, Gemini, Kilocode, and
others. Define your MCP servers once in a neutral YAML file and Agent Align will
convert it into the formats required by each tool while applying
destination-specific tweaks automatically.

## Getting Started

1. Install Go 1.22 or newer (or download the release binary).
2. Create `agent-align-mcp.yml` with your servers using the structure shown in
   `config-mcp.example.yml` (this is the single source of truth).
3. Configure destinations by either copying `config.example.yml` or running the
   init wizard:

   ```bash
   go run ./cmd/agent-align init -config ./agent-align.yml
   ```

4. Run the CLI with both files:

   ```bash
   go run ./cmd/agent-align -config ./agent-align.yml -mcp-config ./agent-align-mcp.yml
   ```

Agent Align prints the generated configs for every agent plus any additional
JSON/file/directory targets so you can review the plan. Accept the prompt (or
pass `-confirm`) to write the changes. Use `-dry-run` to exit after the preview.

## Configuration

Agent Align looks for the target config at these platform-specific locations:

- **Linux**: `/etc/agent-align.yml`
- **macOS**: `/usr/local/etc/agent-align.yml`
- **Windows**: `C:\ProgramData\agent-align\config.yml`

Override the path with `-config <path>`. Within that file, define an
`mcpServers` block with an optional `configPath` (defaults to
`agent-align-mcp.yml` next to the config) and a `targets` block that lists the
agents to update. Each agent entry can optionally set `path` to override the
default location for that tool, and you can repeat an agent with different
paths to write the same format to multiple destinations. Add entries under
`targets.additionalTargets.json` to mirror the MCP payload into other JSON files
(each entry specifies `filePath` and the `jsonPath` where the servers belong).
See the
[Configuration Guide](configuration.md) for the schema and examples. The MCP
servers themselves live in a separate YAML file, and the CLI applies
agent-specific transformations when writing each target.

An optional top-level `extraTargets` block copies files or directories alongside
the MCP sync. Use `extraTargets.files` to mirror a single file into multiple
destinations (for example, `AGENTS.md` into multiple worktrees). Use
`extraTargets.directories` to copy a folder to one or more other locations
(`destinations` is a list of objects with `path` and optional `flatten`) so you
can decide which destinations keep their directory structure.

## Testing

Run the following command to execute all tests:

```bash
GOCACHE=/tmp/agent-align-go-cache go test -coverprofile=coverage.out ./...
```

## More Information

- [Configuration Guide](configuration.md) - Detailed configuration options
