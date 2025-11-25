package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"server-syncer/internal/config"
)

type promptStub struct {
	responses []bool
	idx       int
	prompts   []string
}

func (s *promptStub) answer(prompt string, defaultYes bool) bool {
	s.prompts = append(s.prompts, prompt)
	if s.idx >= len(s.responses) {
		return defaultYes
	}
	resp := s.responses[s.idx]
	s.idx++
	return resp
}

func TestEnsureConfigFileCreatesDefault(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "conf", "server-syncer.yml")

	stub := &promptStub{responses: []bool{true}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })
	origCollect := collectConfig
	collectConfig = func() (config.Config, error) {
		return config.Config{Source: "codex", Targets: []string{"gemini", "copilot"}}, nil
	}
	t.Cleanup(func() { collectConfig = origCollect })

	if err := ensureConfigFile(path); err != nil {
		t.Fatalf("ensureConfigFile failed: %v", err)
	}

	createdCfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("failed to load created config: %v", err)
	}
	expected := config.Config{Source: "codex", Targets: []string{"gemini", "copilot"}}
	if !reflect.DeepEqual(createdCfg, expected) {
		t.Fatalf("config mismatch. got %#v", createdCfg)
	}

	if len(stub.prompts) != 1 {
		t.Fatalf("expected one prompt, got %d", len(stub.prompts))
	}
}

func TestEnsureConfigFileDeclined(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")

	stub := &promptStub{responses: []bool{false}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })

	if err := ensureConfigFile(path); err == nil {
		t.Fatal("expected error when user declines to create config")
	}
}

func TestRunInitCommandOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "server-syncer.yml")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	stub := &promptStub{responses: []bool{true}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })
	origCollect := collectConfig
	collectConfig = func() (config.Config, error) {
		return config.Config{Source: "gemini", Targets: []string{"codex"}}, nil
	}
	t.Cleanup(func() { collectConfig = origCollect })

	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	createdCfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	expected := config.Config{Source: "gemini", Targets: []string{"codex"}}
	if !reflect.DeepEqual(createdCfg, expected) {
		t.Fatalf("config not overwritten. got %#v", createdCfg)
	}
}

func TestRunInitCommandCancelOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "server-syncer.yml")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	stub := &promptStub{responses: []bool{false}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })

	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("config should not have changed, got %q", string(data))
	}
}

func TestRunInitCommandCreatesMissing(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "server-syncer.yml")

	stub := &promptStub{responses: []bool{true}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })
	origCollect := collectConfig
	collectConfig = func() (config.Config, error) {
		return config.Config{Source: "copilot", Targets: []string{"claudecode"}}, nil
	}
	t.Cleanup(func() { collectConfig = origCollect })

	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	createdCfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	expected := config.Config{Source: "copilot", Targets: []string{"claudecode"}}
	if !reflect.DeepEqual(createdCfg, expected) {
		t.Fatalf("config mismatch. got %#v", createdCfg)
	}
}

func TestWriteAgentConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("creates directories and file", func(t *testing.T) {
		path := filepath.Join(tempDir, "nested", "dir", "config.json")
		content := `{"test": "value"}`

		if err := writeAgentConfig(path, content); err != nil {
			t.Fatalf("writeAgentConfig failed: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}
		if string(data) != content {
			t.Fatalf("content mismatch. got %q, want %q", string(data), content)
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		path := filepath.Join(tempDir, "existing.json")
		oldContent := `{"old": "data"}`
		newContent := `{"new": "data"}`

		if err := os.WriteFile(path, []byte(oldContent), 0o644); err != nil {
			t.Fatalf("failed to write existing file: %v", err)
		}

		if err := writeAgentConfig(path, newContent); err != nil {
			t.Fatalf("writeAgentConfig failed: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}
		if string(data) != newContent {
			t.Fatalf("content mismatch. got %q, want %q", string(data), newContent)
		}
	})
}

func TestParseAgents(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Copilot,Codex", []string{"Copilot", "Codex"}},
		{"Copilot, Codex, ClaudeCode", []string{"Copilot", "Codex", "ClaudeCode"}},
		{"  Copilot  ,  Codex  ", []string{"Copilot", "Codex"}},
		{"", []string{}},
		{",,,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAgents(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d agents, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("agent[%d] = %q, want %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestDefaultAgentsAreLowercase(t *testing.T) {
	if defaultAgents != strings.ToLower(defaultAgents) {
		t.Fatalf("defaultAgents must be lowercase, got %q", defaultAgents)
	}
	if !strings.Contains(defaultAgents, "vscode") {
		t.Error("defaultAgents should include vscode")
	}
}
