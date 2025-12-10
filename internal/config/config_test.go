package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	content := `mcpServers:
  configPath: ~/agent-align-mcp.yml
  targets:
    agents:
      - copilot
      - name: codex
        path: ~/custom.toml
    additionalTargets:
      json:
        - filePath: ~/extra.json
          jsonPath: .mcpServers
extraTargets:
  files:
    - source: ~/AGENTS.md
      destinations:
        - ~/dest/AGENTS.md
`

	path := writeConfigFile(t, content)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.MCP.ConfigPath != filepath.Join(dir, "agent-align-mcp.yml") {
		t.Fatalf("configPath not expanded, got %s", got.MCP.ConfigPath)
	}

	if len(got.MCP.Targets.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(got.MCP.Targets.Agents))
	}
	if got.MCP.Targets.Agents[0].Name != "copilot" {
		t.Fatalf("unexpected agent name: %s", got.MCP.Targets.Agents[0].Name)
	}
	if got.MCP.Targets.Agents[1].Path != filepath.Join(dir, "custom.toml") {
		t.Fatalf("agent override path not expanded, got %s", got.MCP.Targets.Agents[1].Path)
	}

	if got.ExtraTargets.IsZero() {
		t.Fatalf("expected extraTargets to be populated")
	}
	if got.MCP.Targets.Additional.IsZero() {
		t.Fatalf("expected additional targets to be populated")
	}
}

func TestLoadRejectsMissingTargets(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  targets:
    agents: []
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing targets")
	}
	if !strings.Contains(err.Error(), "at least one target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidAdditionalTarget(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  targets:
    agents: [copilot]
    additionalTargets:
      json:
        - filePath: ""
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing filePath")
	}
}

func TestLoadExtraFileTargetsBackwardCompatibility(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Test that both old string format and new object format work
	content := `mcpServers:
  targets:
    agents:
      - copilot
extraTargets:
  files:
    - source: ~/source.md
      destinations:
        - ~/dest1.md
        - path: ~/dest2.md
        - path: ~/dest3.md
          pathToSkills: ~/skills
`

	path := writeConfigFile(t, content)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(got.ExtraTargets.Files) != 1 {
		t.Fatalf("expected 1 file target, got %d", len(got.ExtraTargets.Files))
	}

	fileTarget := got.ExtraTargets.Files[0]
	if len(fileTarget.Destinations) != 3 {
		t.Fatalf("expected 3 destinations, got %d", len(fileTarget.Destinations))
	}

	// Check first destination (string format, no skills)
	if fileTarget.Destinations[0].Path != filepath.Join(dir, "dest1.md") {
		t.Errorf("dest1 path not correct: %s", fileTarget.Destinations[0].Path)
	}
	if fileTarget.Destinations[0].PathToSkills != "" {
		t.Errorf("dest1 should not have pathToSkills: %s", fileTarget.Destinations[0].PathToSkills)
	}

	// Check second destination (object format, no skills)
	if fileTarget.Destinations[1].Path != filepath.Join(dir, "dest2.md") {
		t.Errorf("dest2 path not correct: %s", fileTarget.Destinations[1].Path)
	}
	if fileTarget.Destinations[1].PathToSkills != "" {
		t.Errorf("dest2 should not have pathToSkills: %s", fileTarget.Destinations[1].PathToSkills)
	}

	// Check third destination (object format, with skills)
	if fileTarget.Destinations[2].Path != filepath.Join(dir, "dest3.md") {
		t.Errorf("dest3 path not correct: %s", fileTarget.Destinations[2].Path)
	}
	if fileTarget.Destinations[2].PathToSkills != filepath.Join(dir, "skills") {
		t.Errorf("dest3 pathToSkills not correct: %s", fileTarget.Destinations[2].PathToSkills)
	}
}

func TestLoadExtraFileTargetsWithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	content := `mcpServers:
  targets:
    agents:
      - copilot
extraTargets:
  files:
    - source: ~/source.md
      destinations:
        - path: ~/dest1.md
          frontmatterPath: ~/frontmatter.md
        - path: ~/dest2.md
          pathToSkills: ~/skills
        - path: ~/dest3.md
`

	path := writeConfigFile(t, content)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(got.ExtraTargets.Files) != 1 {
		t.Fatalf("expected 1 file target, got %d", len(got.ExtraTargets.Files))
	}

	fileTarget := got.ExtraTargets.Files[0]
	if len(fileTarget.Destinations) != 3 {
		t.Fatalf("expected 3 destinations, got %d", len(fileTarget.Destinations))
	}

	// Check first destination with frontmatter
	if fileTarget.Destinations[0].Path != filepath.Join(dir, "dest1.md") {
		t.Errorf("dest1 path not correct: %s", fileTarget.Destinations[0].Path)
	}
	if fileTarget.Destinations[0].FrontmatterPath != filepath.Join(dir, "frontmatter.md") {
		t.Errorf("dest1 frontmatterPath not correct: %s", fileTarget.Destinations[0].FrontmatterPath)
	}

	// Check second destination with skills
	if fileTarget.Destinations[1].Path != filepath.Join(dir, "dest2.md") {
		t.Errorf("dest2 path not correct: %s", fileTarget.Destinations[1].Path)
	}
	if fileTarget.Destinations[1].PathToSkills != filepath.Join(dir, "skills") {
		t.Errorf("dest2 pathToSkills not correct: %s", fileTarget.Destinations[1].PathToSkills)
	}

	// Check third destination without extras
	if fileTarget.Destinations[2].Path != filepath.Join(dir, "dest3.md") {
		t.Errorf("dest3 path not correct: %s", fileTarget.Destinations[2].Path)
	}
	if fileTarget.Destinations[2].FrontmatterPath != "" {
		t.Errorf("dest3 should not have frontmatterPath: %s", fileTarget.Destinations[2].FrontmatterPath)
	}
	if fileTarget.Destinations[2].PathToSkills != "" {
		t.Errorf("dest3 should not have pathToSkills: %s", fileTarget.Destinations[2].PathToSkills)
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return path
}
