# Configuration format

`agent-align` reads two YAML files:

1. **MCP definitions** – the source of truth for your servers (default
   `agent-align-mcp.yml` next to the target config). This file is required.
2. **Target config** – describes which agents to update, optional path overrides,
   and any extra copy tasks (default `/etc/agent-align.yml`).

## MCP definitions file (agent-align-mcp.yml)

The MCP file lists every server in a neutral JSON-style shape:

```yaml
servers:
  github:
    type: streamable-http
    url: https://api.example.com/mcp/
    headers:
      Authorization: "Bearer REPLACE_WITH_GITHUB_TOKEN"
    tools: []
  claude-cli:
    command: npx
    args:
      - '@example/mcp-server@latest'
    env:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
  prompts:
    command: ./scripts/run-prompts.sh
    args:
      - --watch
```

You can also use the legacy `mcpServers` key instead of `servers`. Each server
entry is a mapping; the keys match the fields you would normally place in the
agent-specific files (for example, `command`, `args`, `env`, `headers`,
`alwaysAllow`, `autoApprove`, `disabled`, `tools`, `type`, and `url`).

## Target config file (agent-align.yml)

The target config points to the MCP file (optional if you accept the default
path) and lists the destinations to update:

```yaml
mcpServers:
  configPath: agent-align-mcp.yml
  targets:
    agents:
      - name: copilot
      - name: vscode
      - name: codex
        path: /custom/.codex/config.toml  # optional override
      - claudecode
      - gemini
      - kilocode
    additionalTargets:
      json:
        - filePath: /path/to/additional_targets.json
          jsonPath: .mcpServers
extraTargets:
  files:
    - source: /path/to/AGENTS.md
      destinations:
        - /path/to/other/AGENTS.md
  directories:
    - source: /path/to/prompts
      destinations:
        - path: /path/to/another/prompts
          flatten: true
```

### Fields

- `mcpServers` (mapping, required) – nests MCP sync settings.
  - `configPath` (string, optional) – path to the MCP definitions file. Defaults
    to `agent-align-mcp.yml` next to the target config when omitted.
  - `targets` (mapping, required) – agents to write plus optional extras.
    - `agents` (sequence, required) – list of agent names or objects with `name`
      and optional `path` override for the destination file. Repeat an agent
      with different `path` values to write the same format to multiple
      destinations. Exact duplicate `name + path` combinations and blank entries
      are ignored.
    - `additionalTargets.json` (sequence, optional) – mirror the MCP payload
      into other JSON files. Each entry must specify `filePath` and may set
      `jsonPath` (dot-separated) where the servers should be placed; omit
      `jsonPath` to replace the entire file.
- `extraTargets` (mapping, optional) – copies additional content alongside the
  MCP sync.
  - `files` (sequence) – mirror a single source file to multiple destinations.
    Each entry must specify `source` and at least one `destinations` value.
  - `directories` (sequence) – copy every file within `source` to each entry in
    `destinations`. Every destination entry must specify a `path` and may set
    `flatten: true` to drop the source directory structure while copying.

## Supported Agents and defaults

Agent | Config File | Format | Root
----- | ----------- | ------ | ----
copilot | `~/.copilot/mcp-config.json` | JSON | `mcpServers`
vscode | `~/.config/Code/User/mcp.json` | JSON | `servers`
codex | `~/.codex/config.toml` | TOML | `mcp_servers`
claudecode | `~/.claude.json` | JSON | `mcpServers`
gemini | `~/.gemini/settings.json` | JSON | `mcpServers`
kilocode | Platform-dependent (see note below) | JSON | `mcpServers`

Every agent accepts a `path` override in `targets.agents` if your installation
lives elsewhere.

Note: Kilocode config paths

- Windows: `~/AppData/Roaming/Code/user/mcp.json`
- Linux: `~/.config/Code/User/globalStorage/kilocode.kilo-code/settings/mcp_settings.json`

## CLI flags and init command

- `-config` – Path to the target config. Defaults to the platform-specific
  location listed above.
- `-mcp-config` – Path to the MCP definitions file. Defaults to
  `agent-align-mcp.yml` next to the selected config.
- `-agents` – Override the target agents defined in the config. Overrides still
  honor per-agent `path` entries if they exist in the file.
- `-dry-run` – Preview changes without writing.
- `-confirm` – Skip the confirmation prompt when applying writes.

Run `agent-align init -config ./agent-align.yml` to generate a starter config via
prompts if you prefer not to edit YAML manually. The wizard collects the agent
list plus optional additional JSON destinations and writes the final file for you.
