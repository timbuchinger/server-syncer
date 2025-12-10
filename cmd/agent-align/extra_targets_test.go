package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-align/internal/config"
)

func TestCopyExtraFileTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	dest1 := filepath.Join(dir, "dest", "one.md")
	dest2 := filepath.Join(dir, "dest-two.md")
	target := config.ExtraFileTarget{
		Source: source,
		Destinations: []config.ExtraFileCopyRoute{
			{Path: dest1},
			{Path: dest2},
		},
	}
	mcpServers := map[string]interface{}{}
	if err := copyExtraFileTarget(target, dir, mcpServers); err != nil {
		t.Fatalf("copyExtraFileTarget returned error: %v", err)
	}

	for _, dest := range []string{dest1, dest2} {
		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("failed to read %s: %v", dest, err)
		}
		if string(data) != "hello" {
			t.Fatalf("unexpected file contents for %s: %q", dest, data)
		}
	}
}

func TestCopyExtraDirectoryTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("failed to create source tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "root.txt"), []byte("root"), 0o644); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "child.txt"), []byte("child"), 0o644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	dest := filepath.Join(dir, "dest")
	target := config.ExtraDirectoryTarget{
		Source: source,
		Destinations: []config.ExtraDirectoryCopyRoute{
			{Path: dest},
		},
	}
	count, err := copyExtraDirectoryTarget(target)
	if err != nil {
		t.Fatalf("copyExtraDirectoryTarget returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 files copied, got %d", count)
	}

	wantFiles := []string{
		filepath.Join(dest, "root.txt"),
		filepath.Join(dest, "nested", "child.txt"),
	}
	for _, path := range wantFiles {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestCopyExtraDirectoryTargetMultipleDestinations(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "one.txt"), []byte("one"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	dest1 := filepath.Join(dir, "dest1")
	dest2 := filepath.Join(dir, "dest2")
	target := config.ExtraDirectoryTarget{
		Source: source,
		Destinations: []config.ExtraDirectoryCopyRoute{
			{Path: dest1},
			{Path: dest2},
		},
	}
	count, err := copyExtraDirectoryTarget(target)
	if err != nil {
		t.Fatalf("copyExtraDirectoryTarget returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected total copies to be 2, got %d", count)
	}

	for _, dest := range []string{dest1, dest2} {
		if _, err := os.Stat(filepath.Join(dest, "one.txt")); err != nil {
			t.Fatalf("expected %s to contain copied file: %v", dest, err)
		}
	}
}

func TestCopyExtraDirectoryTargetFlatten(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("failed to create source tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "child.txt"), []byte("child"), 0o644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	dest := filepath.Join(dir, "dest")
	target := config.ExtraDirectoryTarget{
		Source: source,
		Destinations: []config.ExtraDirectoryCopyRoute{
			{Path: dest, Flatten: true},
		},
	}
	count, err := copyExtraDirectoryTarget(target)
	if err != nil {
		t.Fatalf("copyExtraDirectoryTarget returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 file copied, got %d", count)
	}

	if _, err := os.Stat(filepath.Join(dest, "child.txt")); err != nil {
		t.Fatalf("expected flattened file to exist: %v", err)
	}
}

func TestCopyExtraFileTargetWithSkills(t *testing.T) {
	dir := t.TempDir()
	
	// Create source file
	source := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(source, []byte("# Original Content\n"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	// Create skills.md template
	skillsTemplate := filepath.Join(dir, "skills.md")
	templateContent := `# Skills Section

## Available Skills
`
	if err := os.WriteFile(skillsTemplate, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("failed to write skills template: %v", err)
	}

	// Create skills directory with SKILL.md files
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(filepath.Join(skillsDir, "bash-patterns"), 0o755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	skill1 := filepath.Join(skillsDir, "bash-patterns", "SKILL.md")
	skill1Content := `---
name: bash-defensive-patterns
description: Use when writing or reviewing Bash scripts to apply defensive programming patterns
---

# Bash Defensive Patterns
`
	if err := os.WriteFile(skill1, []byte(skill1Content), 0o644); err != nil {
		t.Fatalf("failed to write skill1: %v", err)
	}

	skill2 := filepath.Join(skillsDir, "SKILL.md")
	skill2Content := `---
name: code-review
description: Use when reviewing code for best practices and common issues
---

# Code Review Skill
`
	if err := os.WriteFile(skill2, []byte(skill2Content), 0o644); err != nil {
		t.Fatalf("failed to write skill2: %v", err)
	}

	// Create destination
	dest := filepath.Join(dir, "output", "AGENTS.md")
	target := config.ExtraFileTarget{
		Source: source,
		Destinations: []config.ExtraFileCopyRoute{
			{
				Path:         dest,
				PathToSkills: skillsDir,
			},
		},
	}

	mcpServers := map[string]interface{}{}
	if err := copyExtraFileTarget(target, dir, mcpServers); err != nil {
		t.Fatalf("copyExtraFileTarget returned error: %v", err)
	}

	// Verify the output contains original content and skills
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Original Content") {
		t.Errorf("destination missing original content")
	}
	if !strings.Contains(content, "# Skills Section") {
		t.Errorf("destination missing skills template")
	}
	if !strings.Contains(content, "bash-defensive-patterns") {
		t.Errorf("destination missing skill1 name")
	}
	if !strings.Contains(content, "code-review") {
		t.Errorf("destination missing skill2 name")
	}
	if !strings.Contains(content, "defensive programming patterns") {
		t.Errorf("destination missing skill1 description")
	}
}

func TestCopyExtraFileTargetMixedDestinations(t *testing.T) {
	dir := t.TempDir()
	
	// Create source file
	source := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(source, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	// Create skills.md template
	if err := os.WriteFile(filepath.Join(dir, "skills.md"), []byte("# Skills\n"), 0o644); err != nil {
		t.Fatalf("failed to write skills template: %v", err)
	}

	// Create a minimal skills directory
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	dest1 := filepath.Join(dir, "dest1.md")
	dest2 := filepath.Join(dir, "dest2.md")
	
	target := config.ExtraFileTarget{
		Source: source,
		Destinations: []config.ExtraFileCopyRoute{
			{Path: dest1, PathToSkills: skillsDir}, // With skills
			{Path: dest2},                          // Without skills
		},
	}

	mcpServers := map[string]interface{}{}
	if err := copyExtraFileTarget(target, dir, mcpServers); err != nil {
		t.Fatalf("copyExtraFileTarget returned error: %v", err)
	}

	// dest1 should have skills appended
	data1, err := os.ReadFile(dest1)
	if err != nil {
		t.Fatalf("failed to read dest1: %v", err)
	}
	if !strings.Contains(string(data1), "# Skills") {
		t.Errorf("dest1 should contain skills content")
	}

	// dest2 should not have skills appended
	data2, err := os.ReadFile(dest2)
	if err != nil {
		t.Fatalf("failed to read dest2: %v", err)
	}
	if string(data2) != "content" {
		t.Errorf("dest2 should only contain original content, got: %q", string(data2))
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantDesc    string
		wantErr     bool
	}{
		{
			name: "valid frontmatter",
			content: `---
name: test-skill
description: A test skill description
---

# Content`,
			wantName: "test-skill",
			wantDesc: "A test skill description",
			wantErr:  false,
		},
		{
			name:     "missing frontmatter",
			content:  "# No frontmatter",
			wantErr:  true,
		},
		{
			name: "missing closing delimiter",
			content: `---
name: test-skill
description: A test skill

# Content`,
			wantErr: true,
		},
		{
			name: "missing name field",
			content: `---
description: A test skill description
---`,
			wantErr: true,
		},
		{
			name: "missing description field",
			content: `---
name: test-skill
---`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, desc, err := parseFrontmatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if name != tt.wantName {
					t.Errorf("parseFrontmatter() name = %v, want %v", name, tt.wantName)
				}
				if desc != tt.wantDesc {
					t.Errorf("parseFrontmatter() description = %v, want %v", desc, tt.wantDesc)
				}
			}
		})
	}
}

func TestCopyExtraFileTargetWithFrontmatter(t *testing.T) {
	dir := t.TempDir()

	// Create source file
	source := filepath.Join(dir, "AGENTS.md")
	sourceContent := "# My Agent Instructions\n\nFollow these rules."
	if err := os.WriteFile(source, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	// Create frontmatter template
	frontmatterPath := filepath.Join(dir, "frontmatter.md")
	frontmatterContent := `---
description: 'Agent instructions'
tools: ['edit', 'view', [MCP]]
---

[CONTENT]`
	if err := os.WriteFile(frontmatterPath, []byte(frontmatterContent), 0o644); err != nil {
		t.Fatalf("failed to write frontmatter template: %v", err)
	}

	// Create destination
	dest := filepath.Join(dir, "output.md")
	target := config.ExtraFileTarget{
		Source: source,
		Destinations: []config.ExtraFileCopyRoute{
			{
				Path:            dest,
				FrontmatterPath: frontmatterPath,
			},
		},
	}

	// Create MCP servers
	mcpServers := map[string]interface{}{
		"github":  map[string]interface{}{"command": "npx"},
		"azure":   map[string]interface{}{"command": "docker"},
		"qdrant":  map[string]interface{}{"command": "uvx"},
	}

	if err := copyExtraFileTarget(target, dir, mcpServers); err != nil {
		t.Fatalf("copyExtraFileTarget returned error: %v", err)
	}

	// Verify the output
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}

	content := string(data)

	// Check that [CONTENT] was replaced
	if !strings.Contains(content, "# My Agent Instructions") {
		t.Errorf("destination missing source content")
	}
	if strings.Contains(content, "[CONTENT]") {
		t.Errorf("destination still contains [CONTENT] placeholder")
	}

	// Check that [MCP] was replaced with MCP server list
	if strings.Contains(content, "[MCP]") {
		t.Errorf("destination still contains [MCP] placeholder")
	}
	if !strings.Contains(content, "'azure/*'") {
		t.Errorf("destination missing azure MCP server")
	}
	if !strings.Contains(content, "'github/*'") {
		t.Errorf("destination missing github MCP server")
	}
	if !strings.Contains(content, "'qdrant/*'") {
		t.Errorf("destination missing qdrant MCP server")
	}

	// Check that frontmatter structure is preserved
	if !strings.Contains(content, "description: 'Agent instructions'") {
		t.Errorf("destination missing frontmatter description")
	}
	if !strings.Contains(content, "tools:") {
		t.Errorf("destination missing tools array")
	}
}

func TestDiscoverSkills(t *testing.T) {
	dir := t.TempDir()
	
	// Create nested directory structure with SKILL.md files
	if err := os.MkdirAll(filepath.Join(dir, "skill1"), 0o755); err != nil {
		t.Fatalf("failed to create skill1 dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "nested", "skill2"), 0o755); err != nil {
		t.Fatalf("failed to create nested skill2 dir: %v", err)
	}

	skill1Content := `---
name: skill-one
description: First skill
---`
	if err := os.WriteFile(filepath.Join(dir, "skill1", "SKILL.md"), []byte(skill1Content), 0o644); err != nil {
		t.Fatalf("failed to write skill1: %v", err)
	}

	skill2Content := `---
name: skill-two
description: Second skill
---`
	if err := os.WriteFile(filepath.Join(dir, "nested", "skill2", "SKILL.md"), []byte(skill2Content), 0o644); err != nil {
		t.Fatalf("failed to write skill2: %v", err)
	}

	// Create a non-SKILL.md file that should be ignored
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Readme"), 0o644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	skills, err := discoverSkills(dir)
	if err != nil {
		t.Fatalf("discoverSkills returned error: %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	// Verify skills are discovered (order may vary)
	foundSkills := make(map[string]string)
	for _, skill := range skills {
		foundSkills[skill.Name] = skill.Description
	}

	if desc, ok := foundSkills["skill-one"]; !ok || desc != "First skill" {
		t.Errorf("skill-one not found or has wrong description")
	}
	if desc, ok := foundSkills["skill-two"]; !ok || desc != "Second skill" {
		t.Errorf("skill-two not found or has wrong description")
	}
}
