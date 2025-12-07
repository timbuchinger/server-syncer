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
	MCP          MCPConfig          `yaml:"mcpServers"`
	ExtraTargets ExtraTargetsConfig `yaml:"extraTargets"`
}

// MCPConfig groups the MCP definition source and the target agents.
type MCPConfig struct {
	ConfigPath string        `yaml:"configPath"`
	Targets    TargetsConfig `yaml:"targets"`
}

// TargetsConfig groups agent targets and additional destinations.
type TargetsConfig struct {
	Agents     []AgentTarget     `yaml:"agents"`
	Additional AdditionalTargets `yaml:"additionalTargets"`
}

// AgentTarget allows overriding the destination path for an agent.
type AgentTarget struct {
	Name string `yaml:"name"`
	Path string `yaml:"path,omitempty"`
	// DisabledMcpServers lists MCP IDs that should be omitted for this agent.
	DisabledMcpServers []string `yaml:"disabledMcpServers,omitempty"`
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
	Source       string                  `yaml:"source"`
	Destinations []ExtraFileCopyRoute    `yaml:"destinations"`
}

// ExtraFileCopyRoute describes how a single file destination should be written.
type ExtraFileCopyRoute struct {
	Path         string `yaml:"path"`
	PathToSkills string `yaml:"pathToSkills,omitempty"`
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

// UnmarshalYAML lets file destinations be provided as either strings or mappings.
func (e *ExtraFileCopyRoute) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.ScalarNode:
		var path string
		if err := node.Decode(&path); err != nil {
			return err
		}
		e.Path = path
		return nil
	case yaml.MappingNode:
		type raw ExtraFileCopyRoute
		var r raw
		if err := node.Decode(&r); err != nil {
			return err
		}
		e.Path = r.Path
		e.PathToSkills = r.PathToSkills
		return nil
	default:
		return fmt.Errorf("file destination entry must be a string or mapping")
	}
}

// UnmarshalYAML lets agent targets be provided as either strings or mappings.
func (a *AgentTarget) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.ScalarNode:
		var name string
		if err := node.Decode(&name); err != nil {
			return err
		}
		a.Name = name
		return nil
	case yaml.MappingNode:
		type raw AgentTarget
		var r raw
		if err := node.Decode(&r); err != nil {
			return err
		}
		a.Name = r.Name
		a.Path = r.Path
		a.DisabledMcpServers = r.DisabledMcpServers
		return nil
	default:
		return fmt.Errorf("agent entry must be a string or mapping")
	}
}

// UnmarshalYAML accepts both a sequence of agents and a mapping with additional targets.
func (t *TargetsConfig) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.SequenceNode:
		var agents []AgentTarget
		if err := node.Decode(&agents); err != nil {
			return err
		}
		t.Agents = agents
		return nil
	case yaml.MappingNode:
		type raw struct {
			Agents            []AgentTarget     `yaml:"agents"`
			Additional        AdditionalTargets `yaml:"additional"`
			AdditionalTargets AdditionalTargets `yaml:"additionalTargets"`
		}
		var r raw
		if err := node.Decode(&r); err != nil {
			return err
		}
		t.Agents = r.Agents
		if len(r.AdditionalTargets.JSON) > 0 {
			t.Additional = r.AdditionalTargets
		} else {
			t.Additional = r.Additional
		}
		return nil
	default:
		return fmt.Errorf("unexpected targets format, expected sequence or mapping")
	}
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

	cfg.MCP.ConfigPath = strings.TrimSpace(cfg.MCP.ConfigPath)
	if cfg.MCP.ConfigPath != "" {
		expanded, err := expandUserPath(cfg.MCP.ConfigPath)
		if err != nil {
			return Config{}, fmt.Errorf("config at %q has an invalid MCP configPath %q: %w", path, cfg.MCP.ConfigPath, err)
		}
		cfg.MCP.ConfigPath = expanded
	}

	cfg.MCP.Targets = normalizeTargets(cfg.MCP.Targets)

	for i := range cfg.MCP.Targets.Additional.JSON {
		cfg.MCP.Targets.Additional.JSON[i].FilePath = strings.TrimSpace(cfg.MCP.Targets.Additional.JSON[i].FilePath)
		cfg.MCP.Targets.Additional.JSON[i].JSONPath = strings.TrimSpace(cfg.MCP.Targets.Additional.JSON[i].JSONPath)
		if cfg.MCP.Targets.Additional.JSON[i].FilePath == "" {
			return Config{}, fmt.Errorf("config at %q has an additional JSON target without a filePath", path)
		}
		expanded, err := expandUserPath(cfg.MCP.Targets.Additional.JSON[i].FilePath)
		if err != nil {
			return Config{}, fmt.Errorf("config at %q has an additional JSON target with invalid filePath %q: %w", path, cfg.MCP.Targets.Additional.JSON[i].FilePath, err)
		}
		cfg.MCP.Targets.Additional.JSON[i].FilePath = expanded
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
		var routes []ExtraFileCopyRoute
		for _, dest := range cfg.ExtraTargets.Files[i].Destinations {
			trimmedPath := strings.TrimSpace(dest.Path)
			if trimmedPath == "" {
				continue
			}
			expandedPath, err := expandUserPath(trimmedPath)
			if err != nil {
				return Config{}, fmt.Errorf("config at %q has an extra file target destination %q: %w", path, trimmedPath, err)
			}
			trimmedSkills := strings.TrimSpace(dest.PathToSkills)
			var expandedSkills string
			if trimmedSkills != "" {
				expandedSkills, err = expandUserPath(trimmedSkills)
				if err != nil {
					return Config{}, fmt.Errorf("config at %q has an extra file target pathToSkills %q: %w", path, trimmedSkills, err)
				}
			}
			routes = append(routes, ExtraFileCopyRoute{
				Path:         expandedPath,
				PathToSkills: expandedSkills,
			})
		}
		if len(routes) == 0 {
			return Config{}, fmt.Errorf("config at %q has an extra file target for %q without destinations", path, source)
		}
		cfg.ExtraTargets.Files[i].Destinations = routes
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

	if len(cfg.MCP.Targets.Agents) == 0 &&
		len(cfg.MCP.Targets.Additional.JSON) == 0 &&
		cfg.ExtraTargets.IsZero() {
		return Config{}, fmt.Errorf("config at %q must define at least one target", path)
	}

	return cfg, nil
}

func normalizeAgent(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeTargets(targets TargetsConfig) TargetsConfig {
	seen := make(map[string]struct{}, len(targets.Agents))
	var agents []AgentTarget
	for _, target := range targets.Agents {
		name := normalizeAgent(target.Name)
		if name == "" {
			continue
		}
		path := strings.TrimSpace(target.Path)
		if path != "" {
			expanded, err := expandUserPath(path)
			if err == nil {
				path = expanded
			}
		}
		// Normalize disabled MCP list: trim entries and skip empty
		var disabled []string
		for _, d := range target.DisabledMcpServers {
			t := strings.TrimSpace(d)
			if t == "" {
				continue
			}
			disabled = append(disabled, t)
		}
		key := name + "|" + path + "|" + strings.Join(disabled, ",")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		agents = append(agents, AgentTarget{
			Name:               name,
			Path:               path,
			DisabledMcpServers: disabled,
		})
	}
	targets.Agents = agents
	return targets
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

// IsZero reports whether any extra targets are configured.
func (e ExtraTargetsConfig) IsZero() bool {
	return len(e.Files) == 0 && len(e.Directories) == 0
}

// IsZero reports whether additional JSON targets are configured.
func (a AdditionalTargets) IsZero() bool {
	return len(a.JSON) == 0
}
