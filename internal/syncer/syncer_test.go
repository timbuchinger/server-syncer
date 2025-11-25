package syncer

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestSyncer_Sync(t *testing.T) {
    t.Run("success with JSON source", func(t *testing.T) {
        // Create a valid JSON payload for copilot format
        payload := `{
            "mcpServers": {
                "test-server": {
                    "command": "npx",
                    "args": ["test-mcp"]
                }
            }
        }`
        s := New("Copilot", []string{"Copilot", "Codex", "VSCode", "ClaudeCode", "Gemini"})
        template := Template{Name: "test-config", Payload: payload}

        got, err := s.Sync(template)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(got) != 5 {
            t.Fatalf("expected 5 agents, got %d", len(got))
        }

        // Verify copilot output (JSON with mcpServers)
        var copilotData map[string]interface{}
        if err := json.Unmarshal([]byte(got["copilot"]), &copilotData); err != nil {
            t.Fatalf("copilot output is not valid JSON: %v", err)
        }
        if _, ok := copilotData["mcpServers"]; !ok {
            t.Error("copilot output should have mcpServers node")
        }

        // Verify vscode output (JSON with servers)
        var vscodeData map[string]interface{}
        if err := json.Unmarshal([]byte(got["vscode"]), &vscodeData); err != nil {
            t.Fatalf("vscode output is not valid JSON: %v", err)
        }
        if _, ok := vscodeData["servers"]; !ok {
            t.Error("vscode output should have servers node")
        }

        // Verify codex output (TOML format)
        if !strings.Contains(got["codex"], "[mcp_servers.test-server]") {
            t.Errorf("codex output should be in TOML format, got: %s", got["codex"])
        }

        // Verify claudecode output (JSON with mcpServers)
        var claudeData map[string]interface{}
        if err := json.Unmarshal([]byte(got["claudecode"]), &claudeData); err != nil {
            t.Fatalf("claudecode output is not valid JSON: %v", err)
        }
        if _, ok := claudeData["mcpServers"]; !ok {
            t.Error("claudecode output should have mcpServers node")
        }

        // Verify gemini output (JSON with mcpServers)
        var geminiData map[string]interface{}
        if err := json.Unmarshal([]byte(got["gemini"]), &geminiData); err != nil {
            t.Fatalf("gemini output is not valid JSON: %v", err)
        }
        if _, ok := geminiData["mcpServers"]; !ok {
            t.Error("gemini output should have mcpServers node")
        }
    })

    t.Run("success with TOML source", func(t *testing.T) {
        // Create a valid TOML payload for codex format
        payload := `[mcp_servers.test-server]
command = "npx"
args = ["test-mcp"]`
        s := New("Codex", []string{"Copilot", "Codex"})
        template := Template{Name: "test-config", Payload: payload}

        got, err := s.Sync(template)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(got) != 2 {
            t.Fatalf("expected 2 agents, got %d", len(got))
        }

        // Verify copilot output (JSON with mcpServers)
        var copilotData map[string]interface{}
        if err := json.Unmarshal([]byte(got["copilot"]), &copilotData); err != nil {
            t.Fatalf("copilot output is not valid JSON: %v", err)
        }
        if _, ok := copilotData["mcpServers"]; !ok {
            t.Error("copilot output should have mcpServers node")
        }

        // Verify codex output (TOML format)
        if !strings.Contains(got["codex"], "[mcp_servers.test-server]") {
            t.Errorf("codex output should be in TOML format, got: %s", got["codex"])
        }
    })

    t.Run("missing template payload", func(t *testing.T) {
        s := New("copilot", []string{"copilot"})
        _, err := s.Sync(Template{Name: "empty", Payload: ""})
        if err == nil {
            t.Fatal("expected error for empty payload")
        }
    })

    t.Run("source agent not registered", func(t *testing.T) {
        s := New("unknown", []string{"copilot"})
        _, err := s.Sync(Template{Name: "payload", Payload: `{"mcpServers": {}}`})
        if err == nil || !strings.Contains(err.Error(), "source agent") {
            t.Fatalf("unexpected error state: %v", err)
        }
    })
}

func TestLoadTemplateFromFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "template.txt")
    if err := os.WriteFile(path, []byte("  contents with whitespace\n"), 0o644); err != nil {
        t.Fatalf("failed to write template: %v", err)
    }

    tpl, err := LoadTemplateFromFile(path)
    if err != nil {
        t.Fatalf("failed to load template: %v", err)
    }

    if tpl.Name != "template.txt" {
        t.Fatalf("unexpected template name %q", tpl.Name)
    }
    if tpl.Payload != "contents with whitespace" {
        t.Fatalf("unexpected payload %q", tpl.Payload)
    }
}

func TestGetAgentConfig(t *testing.T) {
    tests := []struct {
        agent    string
        nodeName string
        format   string
    }{
        {"copilot", "mcpServers", "json"},
        {"vscode", "servers", "json"},
        {"codex", "", "toml"},
        {"claudecode", "mcpServers", "json"},
        {"gemini", "mcpServers", "json"},
    }

    for _, tt := range tests {
        t.Run(tt.agent, func(t *testing.T) {
            config, err := GetAgentConfig(tt.agent)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if config.NodeName != tt.nodeName {
                t.Errorf("expected node name %q, got %q", tt.nodeName, config.NodeName)
            }
            if config.Format != tt.format {
                t.Errorf("expected format %q, got %q", tt.format, config.Format)
            }
            if config.FilePath == "" {
                t.Error("expected non-empty file path")
            }
        })
    }

    t.Run("unsupported agent", func(t *testing.T) {
        _, err := GetAgentConfig("unknown")
        if err == nil {
            t.Error("expected error for unsupported agent")
        }
    })
}

func TestSupportedAgents(t *testing.T) {
    agents := SupportedAgents()
    expected := []string{"copilot", "vscode", "codex", "claudecode", "gemini"}
    if len(agents) != len(expected) {
        t.Fatalf("expected %d agents, got %d", len(expected), len(agents))
    }
    for i, agent := range expected {
        if agents[i] != agent {
            t.Errorf("expected agent %q at index %d, got %q", agent, i, agents[i])
        }
    }
}

func TestFormatConversion(t *testing.T) {
    t.Run("JSON to TOML conversion", func(t *testing.T) {
        payload := `{
            "mcpServers": {
                "server1": {
                    "command": "npx",
                    "args": ["arg1", "arg2"]
                }
            }
        }`
        servers, err := parseServersFromJSON("mcpServers", payload)
        if err != nil {
            t.Fatalf("failed to parse JSON: %v", err)
        }

        toml := formatToTOML(servers)
        if !strings.Contains(toml, "[mcp_servers.server1]") {
            t.Errorf("TOML should contain server section, got: %s", toml)
        }
        if !strings.Contains(toml, "command = \"npx\"") {
            t.Errorf("TOML should contain command, got: %s", toml)
        }
    })

    t.Run("TOML to JSON conversion", func(t *testing.T) {
        payload := `[mcp_servers.server1]
command = "npx"
args = ["arg1", "arg2"]`
        servers, err := parseServersFromTOML(payload)
        if err != nil {
            t.Fatalf("failed to parse TOML: %v", err)
        }

        jsonOutput := formatToJSON("mcpServers", servers)
        var data map[string]interface{}
        if err := json.Unmarshal([]byte(jsonOutput), &data); err != nil {
            t.Fatalf("output is not valid JSON: %v", err)
        }
        if _, ok := data["mcpServers"]; !ok {
            t.Error("JSON should have mcpServers node")
        }
    })
}

func TestParseTOMLArray(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []string
    }{
        {"simple array", `["a", "b", "c"]`, []string{"a", "b", "c"}},
        {"array with spaces", `[ "a" , "b" , "c" ]`, []string{"a", "b", "c"}},
        {"array with comma in value", `["a,b", "c"]`, []string{"a,b", "c"}},
        {"empty array", `[]`, nil},
        {"single element", `["only"]`, []string{"only"}},
        {"path arguments", `["-y", "@modelcontextprotocol/server-github"]`, []string{"-y", "@modelcontextprotocol/server-github"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parseTOMLArray(tt.input)
            if len(result) != len(tt.expected) {
                t.Fatalf("expected %d elements, got %d: %v", len(tt.expected), len(result), result)
            }
            for i, expected := range tt.expected {
                if result[i] != expected {
                    t.Errorf("element[%d] = %q, want %q", i, result[i], expected)
                }
            }
        })
    }
}
