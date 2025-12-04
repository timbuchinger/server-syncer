package main

import (
	"bufio"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"agent-align/internal/config"
)

func TestParseAgents(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "copilot,vscode", []string{"copilot", "vscode"}},
		{"spaces", " copilot , , gemini ", []string{"copilot", "gemini"}},
		{"empty", "", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAgents(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseAgents(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseSelectionIndices(t *testing.T) {
	t.Run("valid commas", func(t *testing.T) {
		got, err := parseSelectionIndices("1, 2,3")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !reflect.DeepEqual(got, []int{1, 2, 3}) {
			t.Fatalf("unexpected result: %v", got)
		}
	})

	t.Run("space separated", func(t *testing.T) {
		got, err := parseSelectionIndices("1  4")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !reflect.DeepEqual(got, []int{1, 4}) {
			t.Fatalf("unexpected result: %v", got)
		}
	})

	if _, err := parseSelectionIndices(""); err == nil {
		t.Fatal("expected error for empty input")
	}

	if _, err := parseSelectionIndices("abc"); err == nil {
		t.Fatal("expected error for invalid number")
	}
}

func TestValidateCommand(t *testing.T) {
	if err := validateCommand([]string{"agent-align"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateCommand([]string{"agent-align", "init"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateCommand([]string{"agent-align", "-config"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateCommand([]string{"agent-align", "run"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestDefaultMCPConfigPath(t *testing.T) {
	path := "/etc/agent-align.yml"
	got := defaultMCPConfigPath(path)
	if got != "/etc/agent-align-mcp.yml" {
		t.Fatalf("unexpected default MCP path: %s", got)
	}
}

func TestConfigTargetsToSyncer(t *testing.T) {
	targets := []config.AgentTarget{
		{Name: "Copilot", Path: "/tmp/custom"},
	}
	got := configTargetsToSyncer(targets)
	if len(got) != 1 {
		t.Fatalf("expected 1 target, got %d", len(got))
	}
	if got[0].Name != "Copilot" || got[0].PathOverride != "/tmp/custom" {
		t.Fatalf("unexpected conversion: %#v", got[0])
	}
}

func TestEnsureConfigFileCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yml")

	origPrompt := promptUser
	origCollect := collectConfig
	defer func() {
		promptUser = origPrompt
		collectConfig = origCollect
	}()

	promptUser = func(string, bool) bool { return true }
	collectConfig = func() (config.Config, error) {
		return config.Config{
			MCP: config.MCPConfig{
				Targets: config.TargetsConfig{
					Agents: []config.AgentTarget{{Name: "vscode"}},
				},
			},
		}, nil
	}

	if err := ensureConfigFile(path); err != nil {
		t.Fatalf("ensureConfigFile failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "mcpServers:") {
		t.Fatalf("expected mcpServers block in config: %s", data)
	}
	if !strings.Contains(content, "agents:") {
		t.Fatalf("expected agents block in config: %s", data)
	}
}

func TestPromptTargetAgents(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("0\n1,3\n"))
	targets, err := promptTargetAgents(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(targets) != 2 || targets[0].Name == "" {
		t.Fatalf("unexpected targets: %v", targets)
	}
}

func TestVersionVariableDefault(t *testing.T) {
	// The version variable should have a default value of "dev"
	if version != "dev" {
		t.Fatalf("expected default version to be 'dev', got %q", version)
	}
}
