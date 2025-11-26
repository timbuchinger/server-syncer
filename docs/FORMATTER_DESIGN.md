# Formatter Design

This document outlines the architecture for parsing, transforming, and outputting
MCP configuration files across different coding agents.

## Overview

agent-align needs a formatter system that can:

1. Read configuration from any supported agent format
2. Convert to a common intermediary representation
3. Output to one or more target agent formats

## Supported Agents and Formats

| Agent      | Format | Root Element  | Location                   |
|------------|--------|---------------|----------------------------|
| Copilot    | JSON   | `mcpServers`  | `.github/copilot-mcp.json` |
| Codex      | TOML   | `mcp_servers` | `codex.toml`               |
| ClaudeCode | JSON   | `mcpServers`  | `.mcp.json`                |
| Gemini     | JSON   | `mcpServers`  | `.gemini/settings.json`    |

## Recommended Architecture

### Intermediary Format: JSON

We recommend using JSON as the canonical intermediary format because:

- Most agents (Copilot, ClaudeCode, Gemini) already use JSON natively
- JSON has broad Go ecosystem support via `encoding/json`
- Conversion overhead only applies when Codex (TOML) is involved
- The MCP server configuration structure maps naturally to JSON

When TOML input or output is required (Codex), the formatter will perform the
necessary conversion. Otherwise, the intermediary step is effectively a no-op
for JSON-based agents.

### Formatter Interface

```go
package formatter

// MCPServer represents a single MCP server configuration in the intermediary
// format.
type MCPServer struct {
    Command string            `json:"command"`
    Args    []string          `json:"args,omitempty"`
    Env     map[string]string `json:"env,omitempty"`
}

// Config represents the intermediary MCP configuration.
type Config struct {
    Servers map[string]MCPServer `json:"mcpServers"`
}

// Formatter defines the interface for agent-specific formatters.
type Formatter interface {
    // Parse reads agent-specific configuration and returns the intermediary
    // format.
    Parse(data []byte) (Config, error)

    // Format converts the intermediary format to agent-specific output.
    Format(cfg Config) ([]byte, error)

    // Agent returns the name of the agent this formatter handles.
    Agent() string
}
```

### Package Layout

```text
internal/
├── formatter/
│   ├── formatter.go       # Formatter interface and Config types
│   ├── registry.go        # Registry for looking up formatters by agent name
│   ├── json.go            # Shared JSON utilities
│   ├── copilot.go         # Copilot formatter implementation
│   ├── codex.go           # Codex (TOML) formatter implementation
│   ├── claudecode.go      # ClaudeCode formatter implementation
│   ├── gemini.go          # Gemini formatter implementation
│   └── formatter_test.go  # Tests for all formatters
```

### Registry Pattern

```go
package formatter

import "fmt"

var registry = map[string]Formatter{}

// Register adds a formatter to the global registry.
func Register(f Formatter) {
    registry[f.Agent()] = f
}

// Get returns the formatter for the given agent name.
func Get(agent string) (Formatter, error) {
    f, ok := registry[agent]
    if !ok {
        return nil, fmt.Errorf("no formatter registered for agent %q", agent)
    }
    return f, nil
}

func init() {
    Register(&CopilotFormatter{})
    Register(&CodexFormatter{})
    Register(&ClaudeCodeFormatter{})
    Register(&GeminiFormatter{})
}
```

### Example Formatter Implementation

```go
package formatter

import "encoding/json"

type CopilotFormatter struct{}

func (f *CopilotFormatter) Agent() string { return "copilot" }

func (f *CopilotFormatter) Parse(data []byte) (Config, error) {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return Config{}, err
    }
    return cfg, nil
}

func (f *CopilotFormatter) Format(cfg Config) ([]byte, error) {
    return json.MarshalIndent(cfg, "", "  ")
}
```

For Codex (TOML), the formatter would use `github.com/BurntSushi/toml` or
`github.com/pelletier/go-toml/v2` to handle TOML parsing and generation.

## Transformation Pipeline

```text
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Source Agent   │     │  Intermediary   │     │  Target Agent   │
│  (e.g., Codex)  │────▶│  JSON Config    │────▶│  (e.g., Copilot)│
│  TOML file      │     │                 │     │  JSON file      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
      Parse()                                        Format()
```

1. **Parse**: The source formatter reads its native format and produces the
   intermediary `Config` struct.
2. **Transform** (optional): Apply any normalization or validation.
3. **Format**: Each target formatter converts the intermediary to its native
   output format.

## Handling Different Root Elements

Different agents use different root element names (e.g., `mcpServers` vs
`mcp_servers`). Each formatter handles this mapping internally:

- **JSON agents**: Use Go struct tags to map the correct field name.
- **TOML agents**: Use TOML struct tags with the appropriate key name.

```go
// For Codex (TOML)
type CodexConfig struct {
    Servers map[string]MCPServer `toml:"mcp_servers"`
}
```

## Configuration-Driven Workflow

The user's YAML config file (see `CONFIGURATION.md`) drives the pipeline:

```yaml
mcpServers:
  sourceAgent: codex
  targets:
    agents:
      - copilot
      - claudecode
      - gemini
```

The syncer will:

1. Look up the source formatter via `formatter.Get(cfg.SourceAgent)`.
2. Read the source agent's config file.
3. Call `Parse()` to produce the intermediary format.
4. For each target agent, call `Format()` and write the output.

## Error Handling

Each formatter should return descriptive errors that include:

- The agent name
- The operation that failed (parse or format)
- The underlying cause

```go
fmt.Errorf("copilot: failed to parse config: %w", err)
```

## Testing Strategy

- **Unit tests**: Each formatter should have parse/format round-trip tests.
- **Integration tests**: End-to-end tests that verify the full pipeline from
  source to target.
- **Golden files**: Store expected outputs for regression testing.

## Dependencies

| Package                           | Purpose          |
|-----------------------------------|------------------|
| `encoding/json` (stdlib)          | JSON handling    |
| `github.com/pelletier/go-toml/v2` | TOML handling    |
| `gopkg.in/yaml.v3`                | Config file I/O  |

## Future Considerations

- **Partial sync**: Allow syncing only specific servers from the config.
- **Validation**: Warn when a server definition is missing required fields.
- **Dry-run mode**: Preview changes without writing files.
- **Watch mode**: Automatically sync when the source file changes.

## Summary

Using JSON as the intermediary format minimizes conversion overhead for the
majority of agents. The `Formatter` interface provides a clean abstraction that
isolates agent-specific details, making it straightforward to add support for
new agents in the future.
