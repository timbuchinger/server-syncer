package syncer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agent-align/internal/transforms"
)

type Template struct {
	Name    string
	Payload string
}

// AgentConfig holds information about an agent's configuration file.
type AgentConfig struct {
	FilePath string // Path to the config file
	NodeName string // Name of the node where servers are stored
	Format   string // "json" or "toml"
}

// SupportedAgents returns a list of supported agent names.
func SupportedAgents() []string {
	return []string{"copilot", "vscode", "codex", "claudecode", "gemini"}
}

// GetAgentConfig returns the configuration information for a given agent.
func GetAgentConfig(agent string) (AgentConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return AgentConfig{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	switch normalizeAgent(agent) {
	case "copilot":
		return AgentConfig{
			FilePath: filepath.Join(homeDir, ".copilot", "mcp-config.json"),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "vscode":
		return AgentConfig{
			FilePath: filepath.Join(homeDir, ".config", "Code", "User", "mcp.json"),
			NodeName: "servers",
			Format:   "json",
		}, nil
	case "codex":
		return AgentConfig{
			FilePath: filepath.Join(homeDir, ".codex", "config.toml"),
			NodeName: "",
			Format:   "toml",
		}, nil
	case "claudecode":
		return AgentConfig{
			FilePath: filepath.Join(homeDir, ".claude.json"),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "gemini":
		return AgentConfig{
			FilePath: filepath.Join(homeDir, ".gemini", "settings.json"),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	default:
		return AgentConfig{}, fmt.Errorf("unsupported agent: %s", agent)
	}
}

func LoadTemplateFromFile(path string) (Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Template{}, err
	}

	payload := strings.TrimSpace(string(data))
	return Template{
		Name:    filepath.Base(path),
		Payload: payload,
	}, nil
}

type Syncer struct {
	SourceAgent string
	Agents      []string
}

func New(sourceAgent string, agents []string) *Syncer {
	normalized := normalizeAgent(sourceAgent)
	cleanAgents := uniqueAgents(agents)
	return &Syncer{
		SourceAgent: normalized,
		Agents:      cleanAgents,
	}
}

// SyncResult contains the output per agent plus the parsed server data.
type SyncResult struct {
	Agents  map[string]string
	Servers map[string]interface{}
}

func (s *Syncer) Sync(template Template) (SyncResult, error) {
	if strings.TrimSpace(template.Name) == "" {
		return SyncResult{}, fmt.Errorf("template requires a name")
	}
	if strings.TrimSpace(template.Payload) == "" {
		return SyncResult{}, fmt.Errorf("template payload cannot be empty")
	}
	if _, err := GetAgentConfig(s.SourceAgent); err != nil {
		return SyncResult{}, fmt.Errorf("source agent %q not supported: %w", s.SourceAgent, err)
	}

	servers, err := parseServersFromSource(s.SourceAgent, template.Payload)
	if err != nil {
		return SyncResult{}, err
	}

	result := make(map[string]string, len(s.Agents))
	for _, agent := range s.Agents {
		// Deep copy servers for each agent to avoid cross-agent transformation interference
		agentServers, err := deepCopyServers(servers)
		if err != nil {
			return SyncResult{}, err
		}

		// Apply agent-specific transformations
		transformer := transforms.GetTransformer(agent)
		if err := transformer.Transform(agentServers); err != nil {
			return SyncResult{}, err
		}

		result[agent] = formatConfig(agent, agentServers)
	}

	return SyncResult{Agents: result, Servers: servers}, nil
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

func formatConfig(agent string, servers map[string]interface{}) string {
	config, err := GetAgentConfig(agent)
	if err != nil {
		return ""
	}

	if config.Format == "toml" {
		return formatCodexConfig(config, servers)
	}
	return formatToJSON(config.NodeName, servers)
}

// parseServersFromSource extracts MCP server definitions from the source template
func parseServersFromSource(source, payload string) (map[string]interface{}, error) {
	sourceConfig, err := GetAgentConfig(source)
	if err != nil {
		return nil, err
	}

	if sourceConfig.Format == "toml" {
		return parseServersFromTOML(payload)
	}
	return parseServersFromJSON(sourceConfig.NodeName, payload)
}

// parseServersFromJSON extracts servers from a JSON config
func parseServersFromJSON(nodeName, payload string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// If nodeName is empty, assume the entire payload is the servers map
	if nodeName == "" {
		return data, nil
	}

	servers, ok := data[nodeName].(map[string]interface{})
	if !ok {
		// Return empty map if node doesn't exist
		return make(map[string]interface{}), nil
	}
	return servers, nil
}

// parseServersFromTOML extracts servers from a TOML config (Codex format)
func parseServersFromTOML(payload string) (map[string]interface{}, error) {
	servers := make(map[string]interface{})
	lines := strings.Split(payload, "\n")

	var currentServer string
	var serverData map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for server section header [mcp_servers.servername]
		if strings.HasPrefix(line, "[mcp_servers.") && strings.HasSuffix(line, "]") {
			if currentServer != "" && serverData != nil {
				servers[currentServer] = serverData
			}
			currentServer = strings.TrimPrefix(line, "[mcp_servers.")
			currentServer = strings.TrimSuffix(currentServer, "]")
			serverData = make(map[string]interface{})
			continue
		}

		// Parse key-value pairs within a server section
		if currentServer != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Handle string values
				if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
					value = strings.Trim(value, "\"")
					serverData[key] = value
				} else if strings.HasPrefix(value, "[") {
					// Handle array values
					arr := parseTOMLArray(value)
					serverData[key] = arr
				} else {
					serverData[key] = value
				}
			}
		}
	}

	// Don't forget the last server
	if currentServer != "" && serverData != nil {
		servers[currentServer] = serverData
	}

	return servers, nil
}

// parseTOMLArray parses a simple TOML array like ["a", "b", "c"]
// This handles basic quoted strings but does not support escaped quotes within values
func parseTOMLArray(value string) []string {
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	var result []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case ch == '"' && !inQuotes:
			inQuotes = true
		case ch == '"' && inQuotes:
			inQuotes = false
			result = append(result, current.String())
			current.Reset()
		case ch == ',' && !inQuotes:
			// Skip commas outside quotes (between elements)
			continue
		case inQuotes:
			current.WriteByte(ch)
		}
	}

	return result
}

// formatToJSON converts servers to JSON format with the specified node name
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

		sb.WriteString(fmt.Sprintf("[mcp_servers.%s]\n", name))

		// Sort keys for consistent output
		keys := make([]string, 0, len(serverData))
		for k := range serverData {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := serverData[k]
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
	}

	return strings.TrimRight(sb.String(), "\n")
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

func uniqueAgents(agents []string) []string {
	seen := make(map[string]struct{}, len(agents))
	var out []string
	for _, agent := range agents {
		normalized := normalizeAgent(agent)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
