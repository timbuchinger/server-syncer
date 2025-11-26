package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config describes the MCP sync behavior and extra file/directory copies.
type Config struct {
	SourceAgent  string
	Targets      TargetsConfig
	ExtraTargets ExtraTargetsConfig
}

// TargetsConfig groups agent targets and additional destinations.
type TargetsConfig struct {
	Agents     []string          `yaml:"agents"`
	Additional AdditionalTargets `yaml:"additionalTargets"`
}

// AdditionalTargets lists paths for JSON-style destinations.
type AdditionalTargets struct {
	JSON []AdditionalJSONTarget `yaml:"json"`
}

// ExtraTargetsConfig describes file/directory copy operations outside the MCP sync.
type ExtraTargetsConfig struct {
	Files       []ExtraFileTarget      `yaml:"files"`
	Directories []ExtraDirectoryTarget `yaml:"directories"`
}

// ExtraFileTarget copies a single source file into multiple destinations.
type ExtraFileTarget struct {
	Source       string   `yaml:"source"`
	Destinations []string `yaml:"destinations"`
}

// ExtraDirectoryTarget copies an entire directory, optionally flattening the files.
type ExtraDirectoryTarget struct {
	Source       string                    `yaml:"source"`
	Destinations []ExtraDirectoryCopyRoute `yaml:"destinations"`
}

// ExtraDirectoryCopyRoute describes how a single destination should be written.
type ExtraDirectoryCopyRoute struct {
	Path    string `yaml:"path"`
	Flatten bool   `yaml:"flatten"`
}

// AdditionalJSONTarget describes a JSON file that should receive the MCP payload.
type AdditionalJSONTarget struct {
	FilePath string `yaml:"filePath"`
	JSONPath string `yaml:"jsonPath"`
}

// Load reads the YAML configuration from the given path and validates it.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config at %q: %w", path, err)
	}

	cfg.SourceAgent = normalizeAgent(cfg.SourceAgent)
	if cfg.SourceAgent == "" {
		return Config{}, fmt.Errorf("config at %q must define a source agent", path)
	}

	cleanAgents := make([]string, 0, len(cfg.Targets.Agents))
	for _, target := range cfg.Targets.Agents {
		trimmed := normalizeAgent(target)
		if trimmed == "" {
			continue
		}
		cleanAgents = append(cleanAgents, trimmed)
	}

	for _, target := range cleanAgents {
		if target == cfg.SourceAgent {
			return Config{}, fmt.Errorf("config at %q lists %q as both source and target; remove it from targets", path, target)
		}
	}

	cfg.Targets.Agents = cleanAgents
	for i := range cfg.Targets.Additional.JSON {
		cfg.Targets.Additional.JSON[i].FilePath = strings.TrimSpace(cfg.Targets.Additional.JSON[i].FilePath)
		cfg.Targets.Additional.JSON[i].JSONPath = strings.TrimSpace(cfg.Targets.Additional.JSON[i].JSONPath)
		if cfg.Targets.Additional.JSON[i].FilePath == "" {
			return Config{}, fmt.Errorf("config at %q has an additional JSON target without a filePath", path)
		}
		expanded, err := expandUserPath(cfg.Targets.Additional.JSON[i].FilePath)
		if err != nil {
			return Config{}, fmt.Errorf("config at %q has an additional JSON target with invalid filePath %q: %w", path, cfg.Targets.Additional.JSON[i].FilePath, err)
		}
		cfg.Targets.Additional.JSON[i].FilePath = expanded
	}

	for i := range cfg.ExtraTargets.Files {
		source := strings.TrimSpace(cfg.ExtraTargets.Files[i].Source)
		if source == "" {
			return Config{}, fmt.Errorf("config at %q has an extra file target without a source", path)
		}
		expandedSource, err := expandUserPath(source)
		if err != nil {
			return Config{}, fmt.Errorf("config at %q has an extra file target with invalid source %q: %w", path, source, err)
		}
		cfg.ExtraTargets.Files[i].Source = expandedSource
		var dests []string
		for _, dest := range cfg.ExtraTargets.Files[i].Destinations {
			trimmed := strings.TrimSpace(dest)
			if trimmed == "" {
				continue
			}
			expandedDest, err := expandUserPath(trimmed)
			if err != nil {
				return Config{}, fmt.Errorf("config at %q has an extra file target destination %q: %w", path, trimmed, err)
			}
			dests = append(dests, expandedDest)
		}
		if len(dests) == 0 {
			return Config{}, fmt.Errorf("config at %q has an extra file target for %q without destinations", path, source)
		}
		cfg.ExtraTargets.Files[i].Destinations = dests
	}

	for i := range cfg.ExtraTargets.Directories {
		source := strings.TrimSpace(cfg.ExtraTargets.Directories[i].Source)
		if source == "" {
			return Config{}, fmt.Errorf("config at %q has an extra directory target without a source", path)
		}
		expandedSource, err := expandUserPath(source)
		if err != nil {
			return Config{}, fmt.Errorf("config at %q has an extra directory target with invalid source %q: %w", path, source, err)
		}
		cfg.ExtraTargets.Directories[i].Source = expandedSource
		var routes []ExtraDirectoryCopyRoute
		for _, dest := range cfg.ExtraTargets.Directories[i].Destinations {
			trimmed := strings.TrimSpace(dest.Path)
			if trimmed == "" {
				continue
			}
			expandedPath, err := expandUserPath(trimmed)
			if err != nil {
				return Config{}, fmt.Errorf("config at %q has an extra directory destination %q: %w", path, trimmed, err)
			}
			routes = append(routes, ExtraDirectoryCopyRoute{
				Path:    expandedPath,
				Flatten: dest.Flatten,
			})
		}
		if len(routes) == 0 {
			return Config{}, fmt.Errorf("config at %q has an extra directory target for %q without destinations", path, source)
		}
		cfg.ExtraTargets.Directories[i].Destinations = routes
	}

	if len(cfg.Targets.Agents) == 0 &&
		len(cfg.Targets.Additional.JSON) == 0 &&
		cfg.ExtraTargets.IsZero() {
		return Config{}, fmt.Errorf("config at %q must define at least one target", path)
	}

	return cfg, nil
}

func normalizeAgent(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func expandUserPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value[0] != '~' {
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if len(value) == 1 {
		return home, nil
	}
	switch value[1] {
	case '/', '\\':
		remainder := strings.TrimLeft(value[1:], "/\\")
		if remainder == "" {
			return home, nil
		}
		return filepath.Clean(filepath.Join(home, remainder)), nil
	default:
		return value, nil
	}
}

// MarshalYAML outputs the nested structure with mcpServers/extraTargets.
func (c Config) MarshalYAML() (interface{}, error) {
	type mcpServers struct {
		SourceAgent string        `yaml:"sourceAgent"`
		Targets     TargetsConfig `yaml:"targets"`
	}
	type output struct {
		MCPServers   mcpServers         `yaml:"mcpServers"`
		ExtraTargets ExtraTargetsConfig `yaml:"extraTargets,omitempty"`
	}
	return output{
		MCPServers: mcpServers{
			SourceAgent: c.SourceAgent,
			Targets:     c.Targets,
		},
		ExtraTargets: c.ExtraTargets,
	}, nil
}

// UnmarshalYAML supports legacy top-level fields in addition to mcpServers.
func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	type mcpBlock struct {
		Source      string        `yaml:"source"`
		SourceAgent string        `yaml:"sourceAgent"`
		Targets     TargetsConfig `yaml:"targets"`
	}
	type rawConfig struct {
		MCPServers   *mcpBlock          `yaml:"mcpServers"`
		Source       string             `yaml:"source"`
		SourceAgent  string             `yaml:"sourceAgent"`
		Targets      TargetsConfig      `yaml:"targets"`
		ExtraTargets ExtraTargetsConfig `yaml:"extraTargets"`
	}

	var raw rawConfig
	if err := node.Decode(&raw); err != nil {
		return err
	}

	c.ExtraTargets = raw.ExtraTargets
	switch {
	case raw.MCPServers != nil:
		c.Targets = raw.MCPServers.Targets
		if raw.MCPServers.SourceAgent != "" {
			c.SourceAgent = raw.MCPServers.SourceAgent
		} else {
			c.SourceAgent = raw.MCPServers.Source
		}
	default:
		c.Targets = raw.Targets
		if raw.SourceAgent != "" {
			c.SourceAgent = raw.SourceAgent
		} else {
			c.SourceAgent = raw.Source
		}
	}
	return nil
}

// UnmarshalYAML accepts both the legacy target list and the new mapping.
func (t *TargetsConfig) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.SequenceNode:
		var agents []string
		if err := node.Decode(&agents); err != nil {
			return err
		}
		t.Agents = agents
		return nil
	case yaml.MappingNode:
		type rawTargets struct {
			Agents            []string          `yaml:"agents"`
			Additional        AdditionalTargets `yaml:"additional"`
			AdditionalTargets AdditionalTargets `yaml:"additionalTargets"`
		}
		var raw rawTargets
		if err := node.Decode(&raw); err != nil {
			return err
		}
		t.Agents = raw.Agents
		if len(raw.AdditionalTargets.JSON) > 0 {
			t.Additional = raw.AdditionalTargets
		} else {
			t.Additional = raw.Additional
		}
		return nil
	default:
		return fmt.Errorf("unexpected targets format, expected sequence or mapping")
	}
}

// IsZero reports whether any extra targets are configured.
func (e ExtraTargetsConfig) IsZero() bool {
	return len(e.Files) == 0 && len(e.Directories) == 0
}

// IsZero reports whether additional JSON targets are configured.
func (a AdditionalTargets) IsZero() bool {
	return len(a.JSON) == 0
}
