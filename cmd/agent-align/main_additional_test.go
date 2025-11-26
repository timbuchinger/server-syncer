package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-align/internal/config"
)

func TestAskYes_BasicResponses(t *testing.T) {
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	// Test explicit yes
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	if _, err := w.Write([]byte("y\n")); err != nil {
		t.Fatalf("write to pipe failed: %v", err)
	}
	w.Close()
	os.Stdin = r
	if !askYes("prompt? ", false) {
		t.Fatalf("expected yes response to return true")
	}

	// Test explicit no
	r, w, err = os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	if _, err := w.Write([]byte("n\n")); err != nil {
		t.Fatalf("write to pipe failed: %v", err)
	}
	w.Close()
	os.Stdin = r
	if askYes("prompt? ", true) {
		t.Fatalf("expected no response to return false")
	}

	// Test invalid then yes
	r, w, err = os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	if _, err := w.Write([]byte("invalid\nY\n")); err != nil {
		t.Fatalf("write to pipe failed: %v", err)
	}
	w.Close()
	os.Stdin = r
	if !askYes("prompt? ", false) {
		t.Fatalf("expected eventually yes response to return true")
	}
}

func TestRunInitCommand_ExistingFileDeclineOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yml")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// override promptUser to decline
	origPrompt := promptUser
	defer func() { promptUser = origPrompt }()
	promptUser = func(string, bool) bool { return false }

	// run init with explicit config path
	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand returned error when overwrite declined: %v", err)
	}
}

func TestRunInitCommand_CreateNewAndWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yml")

	// override collectConfig to return known config
	origCollect := collectConfig
	defer func() { collectConfig = origCollect }()
	collectConfig = func() (config.Config, error) {
		return config.Config{
			SourceAgent: "copilot",
			Targets: config.TargetsConfig{
				Agents: []string{"vscode"},
			},
		}, nil
	}

	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand failed to create config: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read created config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "sourceAgent: copilot") {
		t.Fatalf("unexpected config contents: %s", data)
	}
	if !strings.Contains(content, "mcpServers:") {
		t.Fatalf("expected mcpServers block in config: %s", data)
	}
}

func TestRunInitCommand_WriteFailure(t *testing.T) {
	dir := t.TempDir()
	// create a file where a directory is expected to cause MkdirAll to fail
	fileAsDir := filepath.Join(dir, "file-as-dir")
	if err := os.WriteFile(fileAsDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write helper file: %v", err)
	}
	path := filepath.Join(fileAsDir, "agent.yml")

	// override collectConfig to return known config
	origCollect := collectConfig
	defer func() { collectConfig = origCollect }()
	collectConfig = func() (config.Config, error) {
		return config.Config{
			SourceAgent: "copilot",
			Targets: config.TargetsConfig{
				Agents: []string{"vscode"},
			},
		}, nil
	}

	if err := runInitCommand([]string{"-config", path}); err == nil {
		t.Fatalf("expected error when write cannot create directories")
	}
}
