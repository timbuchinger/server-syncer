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

func TestResolveExecutionMode(t *testing.T) {
	cases := []struct {
		name          string
		source        string
		agents        string
		configFlag    bool
		wantUseConfig bool
		wantErr       bool
	}{
		{"defaults", "", "", false, true, false},
		{"explicit", "copilot", "codex", false, false, false},
		{"missingAgents", "copilot", "", false, true, true},
		{"configOnly", "", "", true, true, false},
		{"conflicting", "copilot", "codex", true, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveExecutionMode(tc.source, tc.agents, tc.configFlag)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if got != tc.wantUseConfig {
				t.Fatalf("got %v, want %v", got, tc.wantUseConfig)
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	if err := validateCommand([]string{"agent-align"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateCommand([]string{"agent-align", "init"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateCommand([]string{"agent-align", "-source"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateCommand([]string{"agent-align", "run"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestPromptSourceAgent(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("invalid\n1\n"))
	got, err := promptSourceAgent(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Choice 1 should return the first agent from the sorted list, which is "claudecode"
	if got != "claudecode" {
		t.Fatalf("expected %q, got %q", "claudecode", got)
	}
}

func TestPromptTargetAgents(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("0\n1,3\n"))
	targets, err := promptTargetAgents(reader, "copilot")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(targets) != 2 || targets[0] == "" {
		t.Fatalf("unexpected targets: %v", targets)
	}
}

func TestWriteConfigFileCreatesPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "agent.yml")

	cfg := config.Config{
		SourceAgent: "copilot",
		Targets: config.TargetsConfig{
			Agents: []string{"vscode"},
		},
	}
	if err := writeConfigFile(path, cfg); err != nil {
		t.Fatalf("writeConfigFile failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "sourceAgent: copilot") {
		t.Fatalf("unexpected config contents: %s", data)
	}
	if !strings.Contains(content, "agents:") {
		t.Fatalf("expected agents block in config: %s", data)
	}
	if !strings.Contains(content, "mcpServers:") {
		t.Fatalf("expected mcpServers block in config: %s", data)
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
			SourceAgent: "copilot",
			Targets: config.TargetsConfig{
				Agents: []string{"vscode"},
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
	if !strings.Contains(content, "sourceAgent: copilot") {
		t.Fatalf("unexpected config contents: %s", data)
	}
	if !strings.Contains(content, "agents:") {
		t.Fatalf("expected agents block in config: %s", data)
	}
	if !strings.Contains(content, "mcpServers:") {
		t.Fatalf("expected mcpServers block in config: %s", data)
	}
}

func TestPromptAdditionalJSONTargetsNone(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("n\n"))
	targets, err := promptAdditionalJSONTargets(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("expected no targets, got %v", targets)
	}
}

func TestPromptAdditionalJSONTargetsMultiple(t *testing.T) {
	input := strings.NewReader("y\n\n/path/one.json\n.mcpServers\nY\n/tmp/two.json\n\n.payload\nn\n")
	reader := bufio.NewReader(input)
	targets, err := promptAdditionalJSONTargets(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %v", targets)
	}
	if targets[0].FilePath != "/path/one.json" || targets[0].JSONPath != ".mcpServers" {
		t.Fatalf("unexpected first target: %#v", targets[0])
	}
	if targets[1].FilePath != "/tmp/two.json" || targets[1].JSONPath != ".payload" {
		t.Fatalf("unexpected second target: %#v", targets[1])
	}
}
