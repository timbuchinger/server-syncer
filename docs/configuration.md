# Configuration Guide

Agent Align uses a YAML configuration file to define the source and target agents
for synchronization.

## Configuration File Locations

The configuration file is searched in the following locations:

| Platform | Path |
| --- | --- |
| Linux | `/etc/agent-align.yml` |
| macOS | `/usr/local/etc/agent-align.yml` |
| Windows | `C:\ProgramData\agent-align\config.yml` |

You can override the default location with the `-config` flag:

```bash
go run ./cmd/agent-align -config /path/to/config.yml
```

## Configuration Schema

The configuration file supports the following structure:

```yaml
mcpServers:
  sourceAgent: codex
  targets:
    agents:
      - copilot
      - claudecode
    additionalTargets:
      json:
        - filePath: path/to/additional.json
          jsonPath: .mcpServers

extraTargets:
  files:
    - source: path/to/AGENTS.md
      destinations:
        - path/to/clone/AGENTS.md
  directories:
    - source: path/to/prompts
      destinations:
        - path: path/to/clone/prompts
          flatten: true
        - path: path/to/more/prompts
```

- `mcpServers.sourceAgent` (string, required) – the agent whose native configuration
  acts as the template. Valid values are `copilot`, `vscode`, `codex`, `claudecode`,
  and `gemini`. The legacy `source` attribute is still accepted when migrating
  old configs.
- `mcpServers.targets.agents` (sequence, required when using agents) – the list
  of agent names to update. Each entry must be one of the supported agents and
  cannot duplicate `sourceAgent`.
- `mcpServers.targets.additionalTargets.json` (sequence, optional) – a list of
  additional JSON files to update with the MCP servers. Each entry must specify
  a `filePath`; set `jsonPath` to the dot-separated node where the servers
  should live (omit it to replace the entire file).
- `extraTargets.files` (sequence, optional) – copies a single file into one or
  more destinations. Each entry must list the `source` and at least one
  `destinations` entry.
- `extraTargets.directories` (sequence, optional) – copies every file within a
  directory from `source` to each listed destination. Each destination entry must
  include a `path` and may set `flatten: true` to drop the source directory
  structure when writing the files.

## Supported Agents

Agent Align currently supports the following agents:

- **Copilot** - GitHub Copilot
- **Codex** - OpenAI Codex
- **ClaudeCode** - Anthropic Claude Code
- **Gemini** - Google Gemini

## Command Line Options

Config file values are used unless you explicitly set these flags:

- `-source` - Override the source agent
- `-agents` - Override the target agents

Agent Align reads the actual configuration file for the selected source agent
(for example, `~/.codex/config.toml` when `sourceAgent: codex`) and uses that file
as the template automatically.
