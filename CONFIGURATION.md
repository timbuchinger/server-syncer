# Configuration format

`agent-align` reads a single YAML configuration that describes the source
agent whose template drives synchronization and the destinations to update.

```yaml
mcpServers:
  sourceAgent: codex
  targets:
    agents:
      - copilot
      - vscode
      - claudecode
      - gemini
    additionalTargets:
      json:
        - filePath: path/to/additional_targets.json
          jsonPath: .mcpServers
extraTargets:
  files:
    - source: path/to/AGENTS.md
      destinations:
        - path/to/other/AGENTS.md
  directories:
    - source: path/to/prompts
      destinations:
        - path: path/to/other/prompts
          flatten: true
        - path: path/to/another/prompts
```

## Fields

- `mcpServers` (mapping, required) – nests the MCP sync configuration.
  - `sourceAgent` (string, required) – the agent whose configuration acts as the
    authoritative template. Acceptable values are `codex`, `gemini`, `copilot`,
    `vscode`, and `claudecode`. For backward compatibility, the legacy `source`
    field is still allowed, but new configs should reference `sourceAgent`.
  - `targets` (mapping, required) – groups the destinations you want to update.
  Use `targets.agents` to list the supported agents. Each name must be one of the
  five supported agent names and cannot match `sourceAgent`. Add an optional
  `targets.additionalTargets.json` list to mirror the MCP payload into other JSON
  files; every entry should specify a `filePath` and may set `jsonPath` to the
  dot-separated node where the servers belong (omit `jsonPath` to replace the
  entire file).
- `extraTargets` (mapping, optional) – copies additional content alongside the
  MCP sync.
  - `files` (sequence) – mirror a single source file to multiple destinations.
    Each entry must specify `source` and at least one `destinations` value.
  - `directories` (sequence) – copy every file within `source` to each entry in
    `destinations`. Every destination entry must specify a `path` and may set
    `flatten: true` to drop the source directory structure while copying.

The CLI infers the template file from the `sourceAgent` (for example,
`~/.codex/config.toml` when `sourceAgent: codex`). It reads that file at runtime,
so there is no separate `template` field in the config.

## Supported Agents

Agent | Config File | Format | Description
----- | ----------- | ------ | -----------
copilot | `~/.copilot/mcp-config.json` | JSON | GitHub Copilot configuration
vscode | `~/.config/Code/User/mcp.json` | JSON | VS Code MCP configuration
codex | `~/.codex/config.toml` | TOML | Codex CLI configuration
claudecode | `~/.claude.json` | JSON | Claude Code configuration
gemini | `~/.gemini/settings.json` | JSON | Gemini configuration

These five agent names are supported by the current implementation.
