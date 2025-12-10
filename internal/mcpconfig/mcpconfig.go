package mcpconfig

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads the MCP server definitions from a YAML file.
// It accepts either a top-level "servers" or "mcpServers" mapping.
func Load(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Servers    map[string]interface{} `yaml:"servers"`
		MCPServers map[string]interface{} `yaml:"mcpServers"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config at %q: %w", path, err)
	}

	servers := raw.Servers
	if len(servers) == 0 {
		servers = raw.MCPServers
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no MCP servers found in %s", path)
	}

	for name, server := range servers {
		if _, ok := server.(map[string]interface{}); !ok {
			return nil, fmt.Errorf("server %q must be a mapping", name)
		}
	}

	// Expand environment variables in all string values
	expandEnvInMap(servers)

	return servers, nil
}

// expandEnvInMap recursively expands environment variables in all string
// values within a map[string]interface{}. It supports ${VAR} and $VAR syntax.
func expandEnvInMap(m map[string]interface{}) {
	for key, value := range m {
		m[key] = expandEnvInValue(value)
	}
}

// expandEnvInValue recursively expands environment variables in a value.
// It handles strings, maps, slices, and nested structures.
func expandEnvInValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return expandEnv(v)
	case map[string]interface{}:
		expandEnvInMap(v)
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = expandEnvInValue(item)
		}
		return v
	default:
		return value
	}
}

// expandEnv expands environment variables in a string.
// It supports both ${VAR} and $VAR syntax.
func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		// Support ${VAR:-default} syntax
		if strings.Contains(key, ":-") {
			parts := strings.SplitN(key, ":-", 2)
			val := os.Getenv(parts[0])
			if val == "" {
				return parts[1]
			}
			return val
		}
		return os.Getenv(key)
	})
}
