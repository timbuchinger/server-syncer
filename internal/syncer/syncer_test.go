package syncer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncerSync(t *testing.T) {
	targets := []AgentTarget{
		{Name: "copilot"},
		{Name: "vscode"},
		{Name: "codex", PathOverride: "/custom/codex.toml"},
	}
	servers := map[string]interface{}{
		"command-server": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"tool"},
		},
		"http-server": map[string]interface{}{
			"type": "streamable-http",
			"url":  "https://example.test",
		},
	}

	s := New(targets)
	result, err := s.Sync(servers)
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	if len(result.Agents) != len(targets) {
		t.Fatalf("expected %d agents, got %d", len(targets), len(result.Agents))
	}

	copilot := result.Agents["copilot"]
	var copilotData map[string]interface{}
	if err := json.Unmarshal([]byte(copilot.Content), &copilotData); err != nil {
		t.Fatalf("copilot output not valid JSON: %v", err)
	}
	mcpServers, ok := copilotData["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("copilot output missing mcpServers: %v", copilotData)
	}
	for name, srv := range mcpServers {
		server := srv.(map[string]interface{})
		if _, ok := server["tools"]; !ok {
			t.Fatalf("copilot server %s missing tools array", name)
		}
	}

	vscode := result.Agents["vscode"]
	var vscodeData map[string]interface{}
	if err := json.Unmarshal([]byte(vscode.Content), &vscodeData); err != nil {
		t.Fatalf("vscode output not valid JSON: %v", err)
	}
	if _, ok := vscodeData["servers"]; !ok {
		t.Fatalf("vscode output missing servers node: %v", vscodeData)
	}
	if server := vscodeData["servers"].(map[string]interface{})["command-server"].(map[string]interface{}); server["tools"] != nil {
		t.Fatalf("vscode server should not have tools added: %v", server)
	}

	codex := result.Agents["codex"]
	if codex.Config.FilePath != "/custom/codex.toml" {
		t.Fatalf("codex override not applied, got %s", codex.Config.FilePath)
	}
	if !strings.Contains(codex.Content, "[mcp_servers.command-server]") {
		t.Fatalf("codex output missing server block: %s", codex.Content)
	}
}

func TestSupportedAgents(t *testing.T) {
	agents := SupportedAgents()
	expected := []string{"copilot", "vscode", "codex", "claudecode", "gemini", "kilocode"}
	if len(agents) != len(expected) {
		t.Fatalf("expected %d agents, got %d", len(expected), len(agents))
	}
	for i, name := range expected {
		if agents[i] != name {
			t.Fatalf("agent[%d] = %s, want %s", i, agents[i], name)
		}
	}
}

func TestFormatCodexConfigPreservesExistingSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	existing := `# Codex configuration
[general]
theme = "dark"

[mcp_servers.old]
command = "node"
args = ["--flag"]

[editor]
font_size = 12
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	servers := map[string]interface{}{
		"new": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"tool"},
		},
	}
	cfg := AgentConfig{Name: "codex", FilePath: path, Format: "toml"}
	result := formatCodexConfig(cfg, servers)

	if !strings.Contains(result, "[general]") {
		t.Fatal("general section should remain in output")
	}
	if !strings.Contains(result, "[editor]") {
		t.Fatal("editor section should remain in output")
	}
	if strings.Contains(result, "[mcp_servers.old]") {
		t.Fatal("old MCP blocks should be removed")
	}
	if !strings.Contains(result, "[mcp_servers.new]") {
		t.Fatal("new MCP block should be present")
	}
}

func TestFormatGeminiConfigPreservesExistingSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	existing := `{
  "theme": "dark",
  "nested": {
    "enabled": true
  },
  "mcpServers": {
    "old": {
      "command": "node"
    }
  }
}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	servers := map[string]interface{}{
		"new": map[string]interface{}{
			"command": "npx",
		},
	}
	cfg := AgentConfig{Name: "gemini", FilePath: path, NodeName: "mcpServers", Format: "json"}
	result := formatConfig(cfg, servers)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["theme"] != "dark" {
		t.Fatalf("theme should be preserved, got %v", parsed["theme"])
	}
	nested, ok := parsed["nested"].(map[string]interface{})
	if !ok || nested["enabled"] != true {
		t.Fatalf("nested settings should be preserved, got %v", parsed["nested"])
	}
	mcpServers, ok := parsed["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers missing in output: %v", parsed)
	}
	if _, ok := mcpServers["new"]; !ok {
		t.Fatal("new server should be present in mcpServers block")
	}
	if _, ok := mcpServers["old"]; ok {
		t.Fatal("old server should have been replaced")
	}
}

func TestFormatClaudeConfigPreservesExistingSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude.json")
	existing := `{
  "theme": "light",
  "mcpServers": {
	"old": {
	  "command": "node"
	}
  },
  "other": true
}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	servers := map[string]interface{}{
		"new": map[string]interface{}{
			"command": "npx",
		},
	}
	cfg := AgentConfig{Name: "claudecode", FilePath: path, NodeName: "mcpServers", Format: "json"}
	result := formatConfig(cfg, servers)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["theme"] != "light" {
		t.Fatalf("theme should be preserved, got %v", parsed["theme"])
	}
	if parsed["other"] != true {
		t.Fatalf("other should be preserved, got %v", parsed["other"])
	}
	mcpServers, ok := parsed["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers missing in output: %v", parsed)
	}
	if _, ok := mcpServers["new"]; !ok {
		t.Fatal("new server should be present in mcpServers block")
	}
	if _, ok := mcpServers["old"]; ok {
		t.Fatal("old server should have been replaced")
	}
}

func TestFormatGeminiConfigWithoutExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	servers := map[string]interface{}{
		"server": map[string]interface{}{
			"command": "npx",
		},
	}
	cfg := AgentConfig{Name: "gemini", FilePath: path, NodeName: "mcpServers", Format: "json"}
	result := formatConfig(cfg, servers)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected only mcpServers key, got %v", parsed)
	}
	if _, ok := parsed["mcpServers"]; !ok {
		t.Fatal("mcpServers key missing")
	}
}
