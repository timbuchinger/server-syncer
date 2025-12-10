package mcpconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.yml")
	content := `servers:
  test:
    command: npx
    args: ["tool"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 server, got %d", len(got))
	}
	if _, ok := got["test"]; !ok {
		t.Fatalf("expected test server present")
	}
}

func TestLoadMissingServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.yml")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for missing servers")
	}
}

func TestLoadWithEnvVarExpansion(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_API_KEY", "secret-key-123")
	defer os.Unsetenv("TEST_API_KEY")
	os.Setenv("TEST_URL", "https://api.example.com")
	defer os.Unsetenv("TEST_URL")

	path := filepath.Join(t.TempDir(), "mcp.yml")
	content := `servers:
  test:
    command: npx
    args: ["@example/tool"]
    env:
      API_KEY: ${TEST_API_KEY}
      BASE_URL: $TEST_URL
      COMBINED: "prefix-${TEST_API_KEY}-suffix"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	server, ok := got["test"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test server to be a map")
	}

	env, ok := server["env"].(map[string]interface{})
	if !ok {
		t.Fatal("expected env to be a map")
	}

	if env["API_KEY"] != "secret-key-123" {
		t.Errorf("expected API_KEY to be expanded, got %v", env["API_KEY"])
	}

	if env["BASE_URL"] != "https://api.example.com" {
		t.Errorf("expected BASE_URL to be expanded, got %v", env["BASE_URL"])
	}

	if env["COMBINED"] != "prefix-secret-key-123-suffix" {
		t.Errorf("expected COMBINED to be expanded, got %v", env["COMBINED"])
	}
}

func TestLoadWithEnvVarDefault(t *testing.T) {
	// Ensure the variable is NOT set
	os.Unsetenv("TEST_MISSING_VAR")

	path := filepath.Join(t.TempDir(), "mcp.yml")
	content := `servers:
  test:
    command: npx
    env:
      DEFAULT_VAR: ${TEST_MISSING_VAR:-default-value}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	server, ok := got["test"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test server to be a map")
	}

	env, ok := server["env"].(map[string]interface{})
	if !ok {
		t.Fatal("expected env to be a map")
	}

	if env["DEFAULT_VAR"] != "default-value" {
		t.Errorf("expected DEFAULT_VAR to use default value, got %v", env["DEFAULT_VAR"])
	}
}

func TestLoadWithEmptyEnvVarUsesDefault(t *testing.T) {
	// Set the variable to empty string
	defer os.Unsetenv("TEST_EMPTY_VAR")
	os.Setenv("TEST_EMPTY_VAR", "")

	path := filepath.Join(t.TempDir(), "mcp.yml")
	content := `servers:
  test:
    command: npx
    env:
      DEFAULT_VAR: ${TEST_EMPTY_VAR:-default-value}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	server, ok := got["test"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test server to be a map")
	}

	env, ok := server["env"].(map[string]interface{})
	if !ok {
		t.Fatal("expected env to be a map")
	}

	if env["DEFAULT_VAR"] != "default-value" {
		t.Errorf("expected DEFAULT_VAR to use default value for empty string, got %v", env["DEFAULT_VAR"])
	}
}

func TestLoadWithEnvVarInNestedStructures(t *testing.T) {
	defer os.Unsetenv("TEST_TOKEN")
	os.Setenv("TEST_TOKEN", "bearer-token-xyz")

	path := filepath.Join(t.TempDir(), "mcp.yml")
	content := `servers:
  test:
    type: streamable-http
    url: https://api.example.com
    headers:
      Authorization: "Bearer ${TEST_TOKEN}"
    tools: []
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	server, ok := got["test"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test server to be a map")
	}

	headers, ok := server["headers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected headers to be a map")
	}

	if headers["Authorization"] != "Bearer bearer-token-xyz" {
		t.Errorf("expected Authorization to be expanded, got %v", headers["Authorization"])
	}
}

func TestLoadWithEnvVarInArrays(t *testing.T) {
	defer os.Unsetenv("TEST_ARG")
	os.Setenv("TEST_ARG", "custom-arg")

	path := filepath.Join(t.TempDir(), "mcp.yml")
	content := `servers:
  test:
    command: npx
    args:
      - "@example/tool"
      - "--flag=${TEST_ARG}"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	server, ok := got["test"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test server to be a map")
	}

	args, ok := server["args"].([]interface{})
	if !ok {
		t.Fatal("expected args to be an array")
	}

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	if args[1] != "--flag=custom-arg" {
		t.Errorf("expected second arg to be expanded, got %v", args[1])
	}
}
