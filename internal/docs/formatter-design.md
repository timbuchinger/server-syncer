# Formatter Design

This document outlines how agent-align parses, transforms, and outputs MCP
configuration files across different coding agents.

## Overview

1. Load MCP server definitions from the YAML file (default
   `agent-align-mcp.yml`) under the `servers` key.
2. Normalize and copy the definitions per target agent, applying
   agent-specific transforms (for example, Copilot transport renames and GitHub
   token handling for Codex).
3. Format and write the result to each agent’s config file, applying any path
   overrides provided in `mcpServers.targets.agents`.

## Supported Agents and Formats

Agent | Format | Root Element | Default Location
----- | ------ | ------------ | ----------------
Copilot | JSON | `mcpServers` | `~/.copilot/mcp-config.json`
VS Code | JSON | `servers` | `~/.config/Code/User/mcp.json`
Codex | TOML | `mcp_servers` | `~/.codex/config.toml`
ClaudeCode | JSON | `mcpServers` | `~/.claude/.claude.json`
Gemini | JSON | `mcpServers` | `~/.gemini/settings.json`
Kilocode | JSON | `mcpServers` | Platform-dependent (see note below)

## Core Types

Note: Kilocode config paths

- Windows: `~/AppData/Roaming/Code/user/mcp.json`
- Linux: `~/.config/Code/User/globalStorage/kilocode.kilo-code/settings/mcp_settings.json`

- `AgentConfig` – holds the agent name, format, root node, and destination path
  (with optional overrides applied).
- `AgentTarget` – represents a requested destination (agent name plus optional
  path override).
- `Syncer` – accepts a slice of `AgentTarget` values and renders the servers
  map into agent-specific outputs.

## Transformation Layer

`internal/transforms` hosts agent-specific rules:

- Copilot: ensures every server has a `tools` array, renames `stdio` → `local`
  and `streamable-http` → `http`, and validates network servers include both
  `type` and `url`.
- Codex: replaces GitHub `Authorization` headers with the static
  `bearer_token_env_var = CODEX_GITHUB_PERSONAL_ACCESS_TOKEN` that the Codex CLI
  expects.
- Other agents currently use the no-op transformer; adding per-server rules is
  centralized here.

## Package Layout

```text
internal/
├── config/       # Target config loading and validation
├── mcpconfig/    # MCP definitions loader
├── syncer/       # Sync logic plus parsing/formatting helpers
└── transforms/   # Agent-specific mutation rules
```
