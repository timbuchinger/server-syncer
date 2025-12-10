package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"agent-align/internal/config"
)

const minFrontmatterLength = 10 // "---\nx\n---" minimum valid frontmatter

func copyExtraFileTarget(target config.ExtraFileTarget, configDir string, mcpServers map[string]interface{}) error {
	info, err := os.Stat(target.Source)
	if err != nil {
		return fmt.Errorf("failed to inspect %s: %w", target.Source, err)
	}
	if info.IsDir() {
		return fmt.Errorf("extra file target %s is a directory; use directories instead", target.Source)
	}
	for _, dest := range target.Destinations {
		if err := copyFileContentsWithSkillsAndFrontmatter(target.Source, dest, info.Mode(), configDir, mcpServers); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", target.Source, dest.Path, err)
		}
	}
	return nil
}

func copyExtraDirectoryTarget(target config.ExtraDirectoryTarget) (int, error) {
	sourceInfo, err := os.Stat(target.Source)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect %s: %w", target.Source, err)
	}
	if !sourceInfo.IsDir() {
		return 0, fmt.Errorf("extra directory target %s is not a directory", target.Source)
	}

	var total int
	for _, dest := range target.Destinations {
		count, err := copyDirectory(target.Source, dest.Path, dest.Flatten)
		if err != nil {
			return total, fmt.Errorf("failed to copy directory %s to %s: %w", target.Source, dest.Path, err)
		}
		total += count
	}
	return total, nil
}

func copyDirectory(source, destination string, flatten bool) (int, error) {
	var copied int
	walkErr := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		var destPath string
		if flatten {
			destPath = filepath.Join(destination, filepath.Base(path))
		} else {
			rel, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			destPath = filepath.Join(destination, rel)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := copyFileContents(path, destPath, info.Mode()); err != nil {
			return err
		}
		copied++
		return nil
	})
	if walkErr != nil {
		return copied, walkErr
	}
	return copied, nil
}

func copyFileContentsWithSkillsAndFrontmatter(source string, dest config.ExtraFileCopyRoute, mode os.FileMode, configDir string, mcpServers map[string]interface{}) error {
	// Read source file content
	sourceData, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dest.Path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dest.Path, err)
	}

	out, err := os.OpenFile(dest.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", dest.Path, err)
	}
	defer out.Close()

	// If FrontmatterPath is specified, use frontmatter template processing
	if dest.FrontmatterPath != "" {
		if err := processFrontmatterTemplate(out, dest.FrontmatterPath, string(sourceData), mcpServers); err != nil {
			return fmt.Errorf("failed to process frontmatter template: %w", err)
		}
	} else {
		// Otherwise, copy source content directly
		if _, err := out.Write(sourceData); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", source, dest.Path, err)
		}

		// If PathToSkills is specified (deprecated), append skills content
		if dest.PathToSkills != "" {
			if err := appendSkillsContent(out, dest.PathToSkills, configDir, nil); err != nil {
				return fmt.Errorf("failed to append skills content: %w", err)
			}
		}

		// If AppendSkills is specified (new format), append skills content with filtering
		for _, appendSkill := range dest.AppendSkills {
			if err := appendSkillsContent(out, appendSkill.Path, configDir, appendSkill.IgnoredSkills); err != nil {
				return fmt.Errorf("failed to append skills content from %s: %w", appendSkill.Path, err)
			}
		}
	}

	return nil
}

func copyFileContents(source, dest string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dest, err)
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", dest, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", source, dest, err)
	}
	return nil
}

// processFrontmatterTemplate processes a frontmatter template file, replacing [CONTENT] and [MCP] placeholders
func processFrontmatterTemplate(out *os.File, frontmatterPath, sourceContent string, mcpServers map[string]interface{}) error {
	// Read the frontmatter template
	templateData, err := os.ReadFile(frontmatterPath)
	if err != nil {
		return fmt.Errorf("failed to read frontmatter template %s: %w", frontmatterPath, err)
	}

	template := string(templateData)

	// Replace [CONTENT] with the source content
	template = strings.ReplaceAll(template, "[CONTENT]", sourceContent)

	// Build MCP server list in the format 'server_name/*'
	var mcpList []string
	for serverName := range mcpServers {
		mcpList = append(mcpList, fmt.Sprintf("'%s/*'", serverName))
	}

	// Sort for consistent output
	sort.Strings(mcpList)

	// Replace [MCP] with the comma-separated list of MCP servers
	mcpReplacement := strings.Join(mcpList, ", ")
	template = strings.ReplaceAll(template, "[MCP]", mcpReplacement)

	// Write the processed template to the output file
	if _, err := out.WriteString(template); err != nil {
		return fmt.Errorf("failed to write processed template: %w", err)
	}

	return nil
}

// appendSkillsContent reads skills.md from configDir and appends it along with discovered SKILL.md files
func appendSkillsContent(out *os.File, pathToSkills, configDir string, ignoredSkills []string) error {
	// First, try to read and append the skills.md template from configDir. If it
	// doesn't exist, fall back to the embedded default so the binary can be
	// distributed standalone.
	skillsTemplatePath := filepath.Join(configDir, "skills.md")
	templateData, err := os.ReadFile(skillsTemplatePath)
	if err != nil {
		if os.IsNotExist(err) {
			templateData = []byte(embeddedSkillsMD)
		} else {
			return fmt.Errorf("failed to read skills template %s: %w", skillsTemplatePath, err)
		}
	}

	// Write a newline before appending to ensure separation
	if _, err := out.WriteString("\n"); err != nil {
		return err
	}

	if _, err := out.Write(templateData); err != nil {
		return fmt.Errorf("failed to write skills template: %w", err)
	}

	// Discover and append SKILL.md files from pathToSkills
	skills, err := discoverSkills(pathToSkills, ignoredSkills)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	for _, skill := range skills {
		skillSection := fmt.Sprintf("\n### **Skill: %s**\n**Description / Use when:**  \n%s\n", skill.Name, skill.Description)
		if _, err := out.WriteString(skillSection); err != nil {
			return fmt.Errorf("failed to write skill %s: %w", skill.Name, err)
		}
	}

	return nil
}

// Skill represents a discovered skill with its metadata
type Skill struct {
	Name        string
	Description string
}

// discoverSkills walks the pathToSkills directory and finds all SKILL.md files
func discoverSkills(pathToSkills string, ignoredSkills []string) ([]Skill, error) {
	var skills []Skill

	// Create a map for faster lookup of ignored skills
	ignoredMap := make(map[string]bool)
	for _, ignored := range ignoredSkills {
		ignoredMap[ignored] = true
	}

	err := filepath.WalkDir(pathToSkills, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) != "SKILL.md" {
			return nil
		}

		skill, err := parseSkillFile(path)
		if err != nil {
			// Log but don't fail on individual skill parsing errors
			fmt.Fprintf(os.Stderr, "Warning: failed to parse skill file %s: %v\n", path, err)
			return nil
		}

		// Skip if skill is in the ignored list
		if ignoredMap[skill.Name] {
			return nil
		}

		skills = append(skills, skill)
		return nil
	})

	return skills, err
}

// parseSkillFile reads a SKILL.md file and extracts name and description from frontmatter
func parseSkillFile(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}

	name, description, err := parseFrontmatter(string(data))
	if err != nil {
		return Skill{}, fmt.Errorf("failed to parse frontmatter in %s: %w", path, err)
	}

	return Skill{
		Name:        name,
		Description: description,
	}, nil
}

// parseFrontmatter extracts name and description from YAML frontmatter
func parseFrontmatter(content string) (name, description string, err error) {
	// Check if content has minimum required length: "---\n" + content + "\n---"
	if len(content) < minFrontmatterLength || content[:3] != "---" {
		return "", "", fmt.Errorf("missing frontmatter delimiter")
	}

	// Find the closing delimiter - start after opening "---\n" (position 4)
	endIdx := -1
	for i := 4; i <= len(content)-4; i++ {
		if content[i] == '\n' && content[i+1:i+4] == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return "", "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	// Extract frontmatter content (skip opening "---\n", up to closing "\n---")
	frontmatter := content[4:endIdx]

	// Parse YAML frontmatter
	var fm struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
		return "", "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	if fm.Name == "" {
		return "", "", fmt.Errorf("missing 'name' field in frontmatter")
	}
	if fm.Description == "" {
		return "", "", fmt.Errorf("missing 'description' field in frontmatter")
	}

	return fm.Name, fm.Description, nil
}
