package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  sourceAgent: codex
  targets:
    agents:
      - gemini
      - copilot
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := Config{
		SourceAgent: "codex",
		Targets: TargetsConfig{
			Agents: []string{"gemini", "copilot"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestLoadRejectsMissingTargets(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  sourceAgent: codex
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

func TestLoadRejectsSourceAsTarget(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  sourceAgent: copilot
  targets:
    agents:
      - copilot
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when target matches source")
	}
	if !strings.Contains(err.Error(), "both source and target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAllowsLegacyFields(t *testing.T) {
	path := writeConfigFile(t, `sourceAgent: codex
targets:
  agents:
    - copilot
`)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error for legacy format: %v", err)
	}
	if got.SourceAgent != "codex" || len(got.Targets.Agents) != 1 {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestLoadValidatesExtraTargets(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  sourceAgent: codex
  targets:
    agents:
      - gemini
extraTargets:
  files:
    - source: /tmp/one
      destinations:
        - /tmp/a
        - /tmp/b
  directories:
    - source: /srv/prompts
      destinations:
        - path: /tmp/prompts
          flatten: true
        - path: /tmp/prompts-copy
`)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.ExtraTargets.IsZero() {
		t.Fatal("expected extra targets to be populated")
	}
	if len(got.ExtraTargets.Files) != 1 || len(got.ExtraTargets.Files[0].Destinations) != 2 {
		t.Fatalf("unexpected extra file targets: %#v", got.ExtraTargets.Files)
	}
	if len(got.ExtraTargets.Directories) != 1 {
		t.Fatalf("unexpected extra directory targets: %#v", got.ExtraTargets.Directories)
	}
	destinations := got.ExtraTargets.Directories[0].Destinations
	if len(destinations) != 2 {
		t.Fatalf("unexpected extra directory destinations: %#v", destinations)
	}
	if !destinations[0].Flatten || destinations[1].Flatten {
		t.Fatalf("unexpected flatten flags: %#v", destinations)
	}
}

func TestLoadRejectsInvalidExtraTargets(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  sourceAgent: codex
  targets:
    agents:
      - gemini
extraTargets:
  files:
    - source: ""
      destinations: []
  directories:
    - source: /srv/prompts
      destinations:
        - path: ""
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "extra file target without a source") {
		t.Fatalf("expected error for invalid extra file target, got: %v", err)
	}
}

func TestLoadExpandsUserPaths(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	path := writeConfigFile(t, `mcpServers:
  sourceAgent: codex
  targets:
    agents: []
    additionalTargets:
      json:
        - filePath: ~/.agent-align/additional.json
          jsonPath: .mcpServers
extraTargets:
  files:
    - source: ~/.codex/AGENTS.md
      destinations:
        - ~/backups/AGENTS.md
  directories:
    - source: ~/prompts
      destinations:
        - path: ~/copies/prompts
          flatten: true
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	jsonTargets := got.Targets.Additional.JSON
	if len(jsonTargets) != 1 {
		t.Fatalf("expected 1 JSON target, got %d", len(jsonTargets))
	}
	expectedJSON := filepath.Join(dir, ".agent-align", "additional.json")
	if jsonTargets[0].FilePath != expectedJSON {
		t.Fatalf("expected JSON path %q, got %q", expectedJSON, jsonTargets[0].FilePath)
	}

	if len(got.ExtraTargets.Files) != 1 {
		t.Fatalf("expected 1 extra file target, got %d", len(got.ExtraTargets.Files))
	}
	fileTarget := got.ExtraTargets.Files[0]
	expectedSource := filepath.Join(dir, ".codex", "AGENTS.md")
	if fileTarget.Source != expectedSource {
		t.Fatalf("expected file source %q, got %q", expectedSource, fileTarget.Source)
	}
	if len(fileTarget.Destinations) != 1 {
		t.Fatalf("expected 1 destination, got %d", len(fileTarget.Destinations))
	}
	expectedDest := filepath.Join(dir, "backups", "AGENTS.md")
	if fileTarget.Destinations[0] != expectedDest {
		t.Fatalf("expected file destination %q, got %q", expectedDest, fileTarget.Destinations[0])
	}

	if len(got.ExtraTargets.Directories) != 1 {
		t.Fatalf("expected 1 directory target, got %d", len(got.ExtraTargets.Directories))
	}
	dirTarget := got.ExtraTargets.Directories[0]
	expectedDirSource := filepath.Join(dir, "prompts")
	if dirTarget.Source != expectedDirSource {
		t.Fatalf("expected directory source %q, got %q", expectedDirSource, dirTarget.Source)
	}
	if len(dirTarget.Destinations) != 1 {
		t.Fatalf("expected 1 directory destination, got %d", len(dirTarget.Destinations))
	}
	expectedDirDest := filepath.Join(dir, "copies", "prompts")
	if dirTarget.Destinations[0].Path != expectedDirDest {
		t.Fatalf("expected directory destination %q, got %q", expectedDirDest, dirTarget.Destinations[0].Path)
	}
	if !dirTarget.Destinations[0].Flatten {
		t.Fatal("expected directory destination to retain flatten flag")
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}
