# Configuration format

`agent-align` reads a single YAML configuration that tells it which agent
format is the source of truth and which agents should be generated from it.

```yaml
source: codex
targets:
  - gemini
  - copilot
  - vscode
  - claudecode
```

## Fields

- `source` (string, required) – the agent whose configuration acts as the
  authoritative template. Allowed values are `codex`, `gemini`, `copilot`,
  `vscode`, and `claudecode`.
- `targets` (sequence, required, non-empty) – a list of one or more agents to
  update from the source. Each target value must also be one of `codex`,
  `gemini`, `copilot`, `vscode`, or `claudecode`. Targets cannot repeat the
  value used for `source`.

The CLI infers the template file from the `source` agent (for example, `codex`
maps to `~/.codex/config.toml`). It reads that file at runtime, so there is no
separate `template` field in the config.

## Supported Agents

Agent | Config File | Format | Description
----- | ----------- | ------ | -----------
copilot | `~/.copilot/mcp-config.json` | JSON | GitHub Copilot configuration
vscode | `~/.config/Code/User/mcp.json` | JSON | VS Code MCP configuration
codex | `~/.codex/config.toml` | TOML | Codex CLI configuration
claudecode | `~/.claude.json` | JSON | Claude Code configuration
gemini | `~/.gemini/settings.json` | JSON | Gemini configuration

These five agent names are supported by the current implementation.
