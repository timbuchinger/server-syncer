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
(for example, `~/.codex/config.toml` when `source: codex`) and uses that file as
the template automatically.
