package syncer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"agent-align/internal/transforms"
)

// AgentTarget allows overrides for an agent destination.
type AgentTarget struct {
	Name         string
	PathOverride string
}

// AgentConfig holds information about an agent's configuration file.
type AgentConfig struct {
	Name     string // Normalized agent name
	FilePath string // Path to the config file
	NodeName string // Name of the node where servers are stored
	Format   string // "json" or "toml"
}

// AgentResult is the rendered output for a single agent.
type AgentResult struct {
	Config  AgentConfig
	Content string
}

var supportedAgentList = []string{"copilot", "vscode", "codex", "claudecode", "gemini", "kilocode"}

// SupportedAgents returns a list of supported agent names.
func SupportedAgents() []string {
	return append([]string(nil), supportedAgentList...)
}

// GetAgentConfig returns the configuration information for a given agent.
func GetAgentConfig(agent, overridePath string) (AgentConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return AgentConfig{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	name := normalizeAgent(agent)
	switch name {
	case "copilot":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".copilot", "mcp-config.json")),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "vscode":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".config", "Code", "User", "mcp.json")),
			NodeName: "servers",
			Format:   "json",
		}, nil
	case "codex":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".codex", "config.toml")),
			NodeName: "",
			Format:   "toml",
		}, nil
	case "claudecode":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".claude.json")),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "gemini":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".gemini", "settings.json")),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "kilocode":
		var defaultPath string
		if runtime.GOOS == "windows" {
			defaultPath = filepath.Join(homeDir, "AppData", "Roaming", "Code", "user", "mcp.json")
		} else {
			defaultPath = filepath.Join(homeDir, ".config", "Code", "User", "globalStorage", "kilocode.kilo-code", "settings", "mcp_settings.json")
		}
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, defaultPath),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	default:
		return AgentConfig{}, fmt.Errorf("unsupported agent: %s", agent)
	}
}

// Syncer renders MCP server definitions into the supported agent formats.
type Syncer struct {
	Agents []AgentTarget
}

func New(agents []AgentTarget) *Syncer {
	return &Syncer{Agents: dedupeTargets(agents)}
}

// SyncResult contains the output per agent plus the parsed server data.
type SyncResult struct {
	Agents  map[string]AgentResult
	Servers map[string]interface{}
}

func (s *Syncer) Sync(servers map[string]interface{}) (SyncResult, error) {
	if len(servers) == 0 {
		return SyncResult{}, fmt.Errorf("server list cannot be empty")
	}

	outputs := make(map[string]AgentResult, len(s.Agents))
	for _, agent := range s.Agents {
		cfg, err := GetAgentConfig(agent.Name, agent.PathOverride)
		if err != nil {
			return SyncResult{}, fmt.Errorf("target agent %q not supported: %w", agent.Name, err)
		}

		agentServers, err := deepCopyServers(servers)
		if err != nil {
			return SyncResult{}, err
		}

		transformer := transforms.GetTransformer(cfg.Name)
		if err := transformer.Transform(agentServers); err != nil {
			return SyncResult{}, err
		}

		outputs[cfg.Name] = AgentResult{
			Config:  cfg,
			Content: formatConfig(cfg, agentServers),
		}
	}

	return SyncResult{Agents: outputs, Servers: servers}, nil
}

// deepCopyServers creates a deep copy of the servers map to avoid
// transformations from one agent affecting another.
func deepCopyServers(servers map[string]interface{}) (map[string]interface{}, error) {
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(servers)
	if err != nil {
		return nil, fmt.Errorf("failed to copy server configuration: %w", err)
	}
	var copy map[string]interface{}
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to copy server configuration: %w", err)
	}
	return copy, nil
}

func formatConfig(config AgentConfig, servers map[string]interface{}) string {
	if config.Format == "toml" {
		return formatCodexConfig(config, servers)
	}

	switch config.Name {
	case "gemini":
		return formatGeminiConfig(config, servers)
	default:
		return formatToJSON(config.NodeName, servers)
	}
}

// formatToJSON converts servers to JSON format with the specified node name
func formatGeminiConfig(cfg AgentConfig, servers map[string]interface{}) string {
	var existing map[string]interface{}
	if data, err := os.ReadFile(cfg.FilePath); err == nil {
		if err := json.Unmarshal(data, &existing); err != nil {
			existing = make(map[string]interface{})
		}
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}

	existing[cfg.NodeName] = servers
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func formatToJSON(nodeName string, servers map[string]interface{}) string {
	var output map[string]interface{}
	if nodeName != "" {
		output = map[string]interface{}{
			nodeName: servers,
		}
	} else {
		output = servers
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

// formatToTOML converts servers to Codex TOML format
func formatToTOML(servers map[string]interface{}) string {
	var sb strings.Builder

	// Sort server names for consistent output
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		serverData, ok := servers[name].(map[string]interface{})
		if !ok {
			continue
		}

		formatServerToTOML(&sb, "mcp_servers."+name, serverData)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// formatServerToTOML recursively formats a server and its nested sections to TOML
func formatServerToTOML(sb *strings.Builder, sectionPath string, data map[string]interface{}) {
	// Separate nested maps from simple values
	simpleValues := make(map[string]interface{})
	nestedMaps := make(map[string]map[string]interface{})

	for k, v := range data {
		if nested, ok := v.(map[string]interface{}); ok {
			nestedMaps[k] = nested
		} else {
			simpleValues[k] = v
		}
	}

	// Write the section header and simple values
	sb.WriteString(fmt.Sprintf("[%s]\n", sectionPath))

	// Sort keys for consistent output
	keys := make([]string, 0, len(simpleValues))
	for k := range simpleValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := simpleValues[k]
		switch val := v.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("%s = \"%s\"\n", k, val))
		case []interface{}:
			arr := make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					arr = append(arr, fmt.Sprintf("\"%s\"", s))
				}
			}
			sb.WriteString(fmt.Sprintf("%s = [%s]\n", k, strings.Join(arr, ", ")))
		case []string:
			arr := make([]string, 0, len(val))
			for _, s := range val {
				arr = append(arr, fmt.Sprintf("\"%s\"", s))
			}
			sb.WriteString(fmt.Sprintf("%s = [%s]\n", k, strings.Join(arr, ", ")))
		default:
			sb.WriteString(fmt.Sprintf("%s = %v\n", k, val))
		}
	}
	sb.WriteString("\n")

	// Sort nested map keys for consistent output
	nestedKeys := make([]string, 0, len(nestedMaps))
	for k := range nestedMaps {
		nestedKeys = append(nestedKeys, k)
	}
	sort.Strings(nestedKeys)

	// Recursively format nested maps as separate sections
	for _, k := range nestedKeys {
		formatServerToTOML(sb, sectionPath+"."+k, nestedMaps[k])
	}
}

func formatCodexConfig(cfg AgentConfig, servers map[string]interface{}) string {
	var existing string
	if data, err := os.ReadFile(cfg.FilePath); err == nil {
		existing = string(data)
	}

	preserved := strings.TrimRight(stripMCPServersSections(existing), "\r\n")
	newSections := strings.TrimRight(formatToTOML(servers), "\r\n")

	var parts []string
	if preserved != "" {
		parts = append(parts, preserved)
	}
	if newSections != "" {
		parts = append(parts, newSections)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n") + "\n"
}

func stripMCPServersSections(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var sb strings.Builder
	insideMCP := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if strings.HasPrefix(trimmed, "[mcp_servers.") {
				insideMCP = true
				continue
			}
			insideMCP = false
		}
		if insideMCP {
			continue
		}
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func normalizeAgent(agent string) string {
	return strings.ToLower(strings.TrimSpace(agent))
}

func dedupeTargets(targets []AgentTarget) []AgentTarget {
	seen := make(map[string]struct{}, len(targets))
	var out []AgentTarget
	for _, target := range targets {
		name := normalizeAgent(target.Name)
		if name == "" {
			continue
		}
		key := name + "|" + strings.TrimSpace(target.PathOverride)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, AgentTarget{
			Name:         name,
			PathOverride: strings.TrimSpace(target.PathOverride),
		})
	}
	return out
}

func applyOverride(overridePath, defaultPath string) string {
	if trimmed := strings.TrimSpace(overridePath); trimmed != "" {
		return trimmed
	}
	return defaultPath
}
