# Configuration Guide

Server Syncer uses a YAML configuration file to define the source agent and target agents for synchronization.

## Configuration File Locations

The configuration file is searched in the following locations:

| Platform | Path |
|----------|------|
| Linux | `/etc/server-syncer.yml` |
| macOS | `/usr/local/etc/server-syncer.yml` |
| Windows | `C:\ProgramData\server-syncer\config.yml` |

You can override the default location with the `-config` flag:

```bash
go run ./cmd/server-syncer -config /path/to/config.yml
```

## Configuration Schema

The configuration file supports the following options:

```yaml
# The source agent to use as the source of truth
source: codex

# List of target agents to synchronize configurations to
targets:
  - copilot
  - claudecode
  - gemini
```

## Supported Agents

Server Syncer currently supports the following agents:

- **Copilot** - GitHub Copilot
- **Codex** - OpenAI Codex
- **ClaudeCode** - Anthropic Claude Code
- **Gemini** - Google Gemini

## Command Line Options

When a config file is present, its values are used unless you explicitly set these flags:

- `-source` - Override the source agent
- `-agents` - Override the target agents

If no config file is found and you omit `-agents`, the CLI defaults to `Copilot`, `Codex`, `ClaudeCode`, and `Gemini`.
