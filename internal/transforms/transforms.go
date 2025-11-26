package transforms

import (
	"fmt"
	"strings"
)

// Transformer defines the interface for destination-specific transformations.
// Each target agent can have its own transformer that manipulates server
// configurations before they are written.
type Transformer interface {
	// Transform modifies servers in place and returns an error if validation fails.
	Transform(servers map[string]interface{}) error
}

// GetTransformer returns the appropriate transformer for a given agent.
// If no specific transformer exists, it returns a no-op transformer.
func GetTransformer(agent string) Transformer {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "copilot":
		return &CopilotTransformer{}
	default:
		return &NoOpTransformer{}
	}
}

// NoOpTransformer performs no transformations.
type NoOpTransformer struct{}

// Transform returns nil without modifying servers.
func (t *NoOpTransformer) Transform(servers map[string]interface{}) error {
	return nil
}

// CopilotTransformer handles Copilot-specific transformations and validations.
type CopilotTransformer struct{}

// Transform applies Copilot-specific modifications:
// - Adds an empty "tools" array to command-type servers if not present
// - Validates that network-based servers have both "type" and "url" fields
func (t *CopilotTransformer) Transform(servers map[string]interface{}) error {
	for name, serverRaw := range servers {
		server, ok := serverRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if err := t.transformServer(name, server); err != nil {
			return err
		}
	}
	return nil
}

// transformServer applies transformations to a single server configuration.
func (t *CopilotTransformer) transformServer(name string, server map[string]interface{}) error {
	if isCommandServer(server) {
		addToolsArrayIfMissing(server)
		return nil
	}

	if isNetworkServer(server) {
		if err := validateNetworkServer(name, server); err != nil {
			return err
		}
	}

	return nil
}

// isCommandServer returns true if the server is a command-type server.
// A command-type server has a "command" field.
func isCommandServer(server map[string]interface{}) bool {
	_, hasCommand := server["command"]
	return hasCommand
}

// isNetworkServer returns true if the server appears to be a network-based server.
// A network-based server has either "type" or "url" field (or both).
func isNetworkServer(server map[string]interface{}) bool {
	_, hasType := server["type"]
	_, hasURL := server["url"]
	return hasType || hasURL
}

// addToolsArrayIfMissing adds an empty "tools" array to the server if not present.
func addToolsArrayIfMissing(server map[string]interface{}) {
	if _, hasTools := server["tools"]; !hasTools {
		server["tools"] = []interface{}{}
	}
}

// validateNetworkServer ensures that network-based servers have both "type" and "url" fields.
func validateNetworkServer(name string, server map[string]interface{}) error {
	_, hasType := server["type"]
	_, hasURL := server["url"]

	if !hasType && !hasURL {
		// Not a network server, nothing to validate
		return nil
	}

	var missing []string
	if !hasType {
		missing = append(missing, "type")
	}
	if !hasURL {
		missing = append(missing, "url")
	}

	if len(missing) > 0 {
		return fmt.Errorf("copilot validation error: network-based server %q is missing required field(s): %s. Network servers must have both 'type' and 'url' fields",
			name, strings.Join(missing, ", "))
	}

	return nil
}
