package main

import (
	"bufio"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"agent-align/internal/config"
	"agent-align/internal/mcpconfig"
	"agent-align/internal/syncer"
)

// version is set at build time via -ldflags.
var version = "dev"

var (
	promptUser    = askYes
	collectConfig = promptForConfig
)

//go:embed config.example.yml
var exampleConfig string

func main() {
	// Handle -version flag before any other processing
	if len(os.Args) == 2 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Printf("agent-align version %s\n", version)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := runInitCommand(os.Args[2:]); err != nil {
			log.Fatalf("init failed: %v", err)
		}
		return
	}
	if err := validateCommand(os.Args); err != nil {
		log.Fatal(err)
	}

	defaultAgents := strings.Join(syncer.SupportedAgents(), ",")
	agents := flag.String("agents", "", fmt.Sprintf("comma-separated list of agents to keep in sync (defaults to %s)", defaultAgents))
	configPath := flag.String("config", defaultConfigPath(), "path to YAML configuration file describing target agents and overrides")
	mcpConfigPath := flag.String("mcp-config", "", "path to YAML file that defines MCP servers (defaults to agent-align-mcp.yml next to the target config)")
	dryRun := flag.Bool("dry-run", false, "only show what would be changed without applying changes")
	debug := flag.Bool("debug", false, "print shell commands to test each MCP server and exit")
	confirm := flag.Bool("confirm", false, "skip user confirmation prompt (useful for cron jobs)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "agent-align version %s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: agent-align [OPTIONS]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "  -version\n    \tprint version and exit\n")
		fmt.Fprintf(os.Stderr, "\nDefault config file location: %s\n", defaultConfigPath())
		fmt.Fprintf(os.Stderr, "Default MCP config file location: %s\n", defaultMCPConfigPath(defaultConfigPath()))
		fmt.Fprintf(os.Stderr, "\nExample config file:\n%s\n", exampleConfig)
		fmt.Fprintf(os.Stderr, "Tip: add agent-align to cron for continuous syncing, e.g.:\n")
		fmt.Fprintf(os.Stderr, "  0 * * * * agent-align -confirm >/tmp/agent-align.log 2>&1\n\n")
	}

	flag.Parse()

	resolvedConfigPath := *configPath
	resolvedMCPPath := strings.TrimSpace(*mcpConfigPath)
	agentsFlagValue := strings.TrimSpace(*agents)

	var cfg config.Config
	var haveConfig bool
	var additionalTargets []config.AdditionalJSONTarget
	var extraTargets config.ExtraTargetsConfig
	var targetAgents []syncer.AgentTarget

	if agentsFlagValue == "" {
		if err := ensureConfigFile(resolvedConfigPath); err != nil {
			log.Fatalf("configuration unavailable: %v", err)
		}
		data, err := config.Load(resolvedConfigPath)
		if err != nil {
			log.Fatalf("failed to load config %q: %v", resolvedConfigPath, err)
		}
		cfg = data
		haveConfig = true
	} else if _, err := os.Stat(resolvedConfigPath); err == nil {
		data, err := config.Load(resolvedConfigPath)
		if err != nil {
			log.Fatalf("failed to load config %q: %v", resolvedConfigPath, err)
		}
		cfg = data
		haveConfig = true
	}

	if haveConfig {
		additionalTargets = cfg.MCP.Targets.Additional.JSON
		extraTargets = cfg.ExtraTargets
		targetAgents = configTargetsToSyncer(cfg.MCP.Targets.Agents)
		if resolvedMCPPath == "" {
			resolvedMCPPath = cfg.MCP.ConfigPath
		}
	}

	if resolvedMCPPath == "" {
		resolvedMCPPath = defaultMCPConfigPath(resolvedConfigPath)
	}

	if agentsFlagValue != "" {
		names := parseAgents(agentsFlagValue)
		if len(names) == 0 {
			log.Fatal("the -agents flag must list at least one agent")
		}
		overrideLookup := make(map[string]string, len(cfg.MCP.Targets.Agents))
		for _, agent := range cfg.MCP.Targets.Agents {
			overrideLookup[agent.Name] = agent.Path
		}
		targetAgents = nil
		for _, name := range names {
			normalized := strings.ToLower(strings.TrimSpace(name))
			targetAgents = append(targetAgents, syncer.AgentTarget{
				Name:         normalized,
				PathOverride: overrideLookup[normalized],
			})
		}
	}

	if len(targetAgents) == 0 && len(additionalTargets) == 0 && extraTargets.IsZero() {
		log.Fatal("no target agents, additional destinations, or extra copy targets configured; provide agents via config/flags or add extra targets")
	}

	servers, err := mcpconfig.Load(resolvedMCPPath)
	if err != nil {
		log.Fatalf("failed to load MCP configuration %q: %v", resolvedMCPPath, err)
	}

	// If debug flag is provided, print a shell-ready command for each server and exit.
	if *debug {
		printDebugCommands(servers)
		return
	}

	s := syncer.New(targetAgents)

	syncResult, err := s.Sync(servers)
	if err != nil {
		log.Fatalf("sync failed: %v", err)
	}

	// Display the dry run results
	fmt.Println("\n=== Dry Run Results ===")
	fmt.Println("The following configuration changes will be made:")
	fmt.Println()

	var agentNames []string
	for name := range syncResult.Agents {
		agentNames = append(agentNames, name)
	}
	sort.Strings(agentNames)

	for _, agent := range agentNames {
		outputs := syncResult.Agents[agent]
		for _, output := range outputs {
			fmt.Printf("Agent: %s\n", agent)
			fmt.Printf("  File: %s\n", output.Config.FilePath)
			fmt.Printf("  Format: %s\n", output.Config.Format)
			fmt.Printf("  Content:\n")
			// Indent the content for readability
			lines := strings.Split(output.Content, "\n")
			for _, line := range lines {
				fmt.Printf("    %s\n", line)
			}
			fmt.Println()
		}
	}

	if len(additionalTargets) > 0 {
		fmt.Println("Additional destinations:")
		for _, target := range additionalTargets {
			fmt.Printf("Additional JSON: %s\n", target.FilePath)
			fmt.Printf("  JSON Path: %s\n", displayJSONPath(target.JSONPath))
			content, err := buildAdditionalJSONContent(target, syncResult.Servers)
			if err != nil {
				fmt.Printf("  (error preparing content: %v)\n\n", err)
				continue
			}
			content = strings.TrimRight(content, "\n")
			if content == "" {
				fmt.Println("  Content: <empty>")
				fmt.Println()
				continue
			}
			fmt.Println("  Content:")
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				fmt.Printf("    %s\n", line)
			}
			fmt.Println()
		}
	}

	if !extraTargets.IsZero() {
		fmt.Println("Extra copy targets:")
		for _, target := range extraTargets.Files {
			fmt.Printf("File Source: %s\n", target.Source)
			for _, dest := range target.Destinations {
				fmt.Printf("  -> %s\n", dest)
			}
			fmt.Println()
		}
		for _, target := range extraTargets.Directories {
			fmt.Printf("Directory Source: %s\n", target.Source)
			fmt.Println("  Destinations:")
			for _, dest := range target.Destinations {
				label := dest.Path
				if dest.Flatten {
					label = fmt.Sprintf("%s (flatten)", label)
				}
				fmt.Printf("    - %s\n", label)
			}
			fmt.Println()
		}
	}

	// If dry-run mode, exit without making changes
	if *dryRun {
		fmt.Println("Dry run complete. No changes were made.")
		return
	}

	// If not in confirm mode, ask for user confirmation
	if !*confirm {
		if !promptUser("Apply these changes? [y/N]: ", false) {
			fmt.Println("Changes cancelled.")
			return
		}
	}

	// Apply the changes
	fmt.Println("\nApplying changes...")
	var applyErrors []string
	for _, agent := range agentNames {
		outputs := syncResult.Agents[agent]
		for _, output := range outputs {
			if err := writeAgentConfig(output.Config.FilePath, output.Content); err != nil {
				msg := fmt.Sprintf("error writing config for %s: %v", agent, err)
				log.Print(msg)
				applyErrors = append(applyErrors, msg)
				continue
			}
			fmt.Printf("  Updated: %s\n", output.Config.FilePath)
		}
	}

	for _, target := range additionalTargets {
		content, err := buildAdditionalJSONContent(target, syncResult.Servers)
		if err != nil {
			msg := fmt.Sprintf("error preparing additional JSON %s: %v", target.FilePath, err)
			log.Print(msg)
			applyErrors = append(applyErrors, msg)
			continue
		}
		if err := writeAgentConfig(target.FilePath, content); err != nil {
			msg := fmt.Sprintf("error writing additional JSON %s: %v", target.FilePath, err)
			log.Print(msg)
			applyErrors = append(applyErrors, msg)
			continue
		}
		fmt.Printf("  Updated additional JSON: %s\n", target.FilePath)
		if target.JSONPath != "" {
			fmt.Printf("    JSON Path: %s\n", target.JSONPath)
		}
	}

	for _, target := range extraTargets.Files {
		if err := copyExtraFileTarget(target); err != nil {
			msg := fmt.Sprintf("error copying extra file %s: %v", target.Source, err)
			log.Print(msg)
			applyErrors = append(applyErrors, msg)
			continue
		}
		fmt.Printf("  Copied extra file: %s -> %d destinations\n", target.Source, len(target.Destinations))
	}
	for _, target := range extraTargets.Directories {
		count, err := copyExtraDirectoryTarget(target)
		if err != nil {
			msg := fmt.Sprintf("error copying extra directory %s: %v", target.Source, err)
			log.Print(msg)
			applyErrors = append(applyErrors, msg)
			continue
		}
		fmt.Printf("  Copied extra directory: %s -> %d destination(s) (%d files)\n", target.Source, len(target.Destinations), count)
		var flattened bool
		for _, dest := range target.Destinations {
			if dest.Flatten {
				flattened = true
				break
			}
		}
		if flattened {
			fmt.Println("    Applied flatten to some destinations")
		}
	}
	fmt.Println("\nConfiguration sync complete.")
	if len(applyErrors) > 0 {
		fmt.Println("Encountered errors while applying changes:")
		for _, msg := range applyErrors {
			fmt.Printf("  - %s\n", msg)
		}
		os.Exit(1)
	}
}

func parseAgents(agents string) []string {
	segments := strings.Split(agents, ",")
	var out []string
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func configTargetsToSyncer(targets []config.AgentTarget) []syncer.AgentTarget {
	out := make([]syncer.AgentTarget, 0, len(targets))
	for _, target := range targets {
		out = append(out, syncer.AgentTarget{
			Name:         target.Name,
			PathOverride: target.Path,
		})
	}
	return out
}

func defaultConfigPath() string {
	switch runtime.GOOS {
	case "darwin":
		return "/usr/local/etc/agent-align.yml"
	case "windows":
		if base := os.Getenv("ProgramData"); base != "" {
			return filepath.Join(base, "agent-align", "config.yml")
		}
		return `C:\ProgramData\agent-align\config.yml`
	default:
		return "/etc/agent-align.yml"
	}
}

func defaultMCPConfigPath(configPath string) string {
	dir := filepath.Dir(configPath)
	if dir == "" {
		return "agent-align-mcp.yml"
	}
	return filepath.Join(dir, "agent-align-mcp.yml")
}

func ensureConfigFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect %q: %w", path, err)
	}

	prompt := fmt.Sprintf("Configuration %s not found. %sCreate a default config? [Y/n]: ", path, configPromptSuffix(path))
	if !promptUser(prompt, true) {
		return fmt.Errorf("configuration file %s is required", path)
	}

	cfg, err := collectConfig()
	if err != nil {
		return fmt.Errorf("failed to collect configuration: %w", err)
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	fmt.Printf("Created configuration file at %s\n", path)
	return nil
}

func runInitCommand(args []string) error {
	initFlags := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := initFlags.String("config", defaultConfigPath(), "path to YAML configuration file to create")
	if err := initFlags.Parse(args); err != nil {
		return err
	}

	path := *configPath
	if _, err := os.Stat(path); err == nil {
		if !promptUser(fmt.Sprintf("Configuration already exists at %s. Overwrite? [y/N]: ", path), false) {
			fmt.Println("Init cancelled.")
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect %q: %w", path, err)
	}

	cfg, err := collectConfig()
	if err != nil {
		return fmt.Errorf("failed to collect configuration: %w", err)
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	fmt.Printf("Created configuration file at %s\n", path)
	return nil
}

func askYes(prompt string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return defaultYes
		}

		response := strings.TrimSpace(strings.ToLower(input))
		if response == "" {
			return defaultYes
		}

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please answer 'y' or 'n'.")
		}
	}
}

func promptForConfig() (config.Config, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nLet's create your agent-align configuration.")
	targets, err := promptTargetAgents(reader)
	if err != nil {
		return config.Config{}, err
	}
	additionalJSON, err := promptAdditionalJSONTargets(reader)
	if err != nil {
		return config.Config{}, err
	}
	additional := config.AdditionalTargets{}
	if len(additionalJSON) > 0 {
		additional.JSON = additionalJSON
	}
	return config.Config{
		MCP: config.MCPConfig{
			Targets: config.TargetsConfig{
				Agents:     targets,
				Additional: additional,
			},
		},
	}, nil
}

func configPromptSuffix(path string) string {
	if path == defaultConfigPath() {
		return "Use -config to choose another path. "
	}
	return ""
}

func promptTargetAgents(reader *bufio.Reader) ([]config.AgentTarget, error) {
	options := syncer.SupportedAgents()

	sort.Strings(options)
	fmt.Println("\nSelect target agents (enter comma-separated numbers, e.g. 1,3):")
	for i, agent := range options {
		fmt.Printf("  %d) %s\n", i+1, agent)
	}

	for {
		fmt.Print("Enter one or more choices: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		selections, parseErr := parseSelectionIndices(line)
		if parseErr != nil {
			fmt.Println(parseErr)
			continue
		}
		seen := make(map[int]struct{}, len(selections))
		var targets []config.AgentTarget
		valid := true
		for _, idx := range selections {
			if idx < 1 || idx > len(options) {
				fmt.Printf("Selection %d is out of range. Please use numbers from the list.\n", idx)
				valid = false
				break
			}
			if _, exists := seen[idx]; exists {
				continue
			}
			seen[idx] = struct{}{}
			targets = append(targets, config.AgentTarget{Name: options[idx-1]})
		}
		if !valid {
			continue
		}
		if len(targets) == 0 {
			fmt.Println("Please select at least one target agent.")
			continue
		}
		return targets, nil
	}
}

func parseSelectionIndices(input string) ([]int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("Please enter at least one selection.")
	}
	segments := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	if len(segments) == 0 {
		return nil, fmt.Errorf("Please enter at least one selection.")
	}
	var selections []int
	for _, segment := range segments {
		value, err := strconv.Atoi(segment)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid number", segment)
		}
		selections = append(selections, value)
	}
	return selections, nil
}

func promptAdditionalJSONTargets(reader *bufio.Reader) ([]config.AdditionalJSONTarget, error) {
	fmt.Println("\nAdd optional MCP destinations (custom files outside the built-in agents).")
	var targets []config.AdditionalJSONTarget

	for {
		addMore, err := promptYesNoInput(reader, "Add an additional JSON destination? [y/N]: ", false)
		if err != nil {
			return nil, err
		}
		if !addMore {
			return targets, nil
		}

		filePath, err := promptRequiredValue(reader, "Enter the destination file path: ", "Please enter a file path.")
		if err != nil {
			return nil, err
		}
		jsonPath, err := promptRequiredValue(reader, "Enter the JSON path within that file (e.g. .mcpServers): ", "Please enter a JSON path.")
		if err != nil {
			return nil, err
		}

		targets = append(targets, config.AdditionalJSONTarget{
			FilePath: filePath,
			JSONPath: jsonPath,
		})
	}
}

func promptYesNoInput(reader *bufio.Reader, prompt string, defaultYes bool) (bool, error) {
	for {
		fmt.Print(prompt)
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}

		response := strings.TrimSpace(strings.ToLower(input))
		if response == "" {
			return defaultYes, nil
		}

		switch response {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please answer 'y' or 'n'.")
			if err != nil && errors.Is(err, io.EOF) {
				return defaultYes, nil
			}
		}
	}
}

func promptRequiredValue(reader *bufio.Reader, prompt, emptyMsg string) (string, error) {
	for {
		fmt.Print(prompt)
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		value := strings.TrimSpace(input)
		if value == "" {
			fmt.Println(emptyMsg)
			if err != nil && errors.Is(err, io.EOF) {
				return "", errors.New(emptyMsg)
			}
			continue
		}
		return value, nil
	}
}

func writeConfigFile(path string, cfg config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate config contents: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure directory %q: %w", dir, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		printManualConfigInstructions(path, data)
		return fmt.Errorf("failed to write config %q: %w", path, err)
	}
	return nil
}

func printManualConfigInstructions(path string, contents []byte) {
	fmt.Fprintf(os.Stderr, "\nUnable to write the config file automatically. Please create %s with the following contents:\n\n%s\n", path, contents)
}

func writeAgentConfig(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure directory %q: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write config %q: %w", path, err)
	}
	return nil
}

func validateCommand(args []string) error {
	if len(args) <= 1 {
		return nil
	}
	arg := args[1]
	if arg == "" || arg == "init" || strings.HasPrefix(arg, "-") {
		return nil
	}
	return fmt.Errorf("unknown command %q. Use -h for usage or run \"init\" to create a config.", arg)
}

// printDebugCommands emits a shell-ready test command for every MCP server definition
// found in the provided map and prints them to stdout.
func printDebugCommands(servers map[string]interface{}) {
	var names []string
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		serverRaw := servers[name]
		m, ok := serverRaw.(map[string]interface{})
		if !ok {
			// skip unexpected shapes
			continue
		}
		cmd := formatServerCommand(m)
		if cmd == "" {
			fmt.Printf("%s: <cannot render command>\n", name)
			continue
		}
		fmt.Printf("%s: %s\n", name, cmd)
	}
}

// formatServerCommand builds a single-line shell command for a server mapping.
// It concatenates environment assignments (KEY=VALUE) before the command and
// properly quotes arguments.
func formatServerCommand(m map[string]interface{}) string {
	// command
	cmdVal, ok := m["command"]
	if !ok {
		return ""
	}
	cmdStr, ok := cmdVal.(string)
	if !ok || strings.TrimSpace(cmdStr) == "" {
		return ""
	}

	// args may be an array
	var args []string
	if rawArgs, ok := m["args"]; ok {
		switch v := rawArgs.(type) {
		case []interface{}:
			for _, ai := range v {
				if s, ok := ai.(string); ok {
					args = append(args, s)
				}
			}
		case []string:
			args = append(args, v...)
		case string:
			// single string argument
			args = append(args, v)
		}
	}

	// env may be a map
	var envParts []string
	if rawEnv, ok := m["env"]; ok {
		if envMap, ok := rawEnv.(map[string]interface{}); ok {
			// preserve insertion order by sorting keys
			var keys []string
			for k := range envMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := envMap[k]
				sval := fmt.Sprintf("%v", v)
				// if value looks like ${VAR} or starts with $ keep as-is
				if (strings.HasPrefix(sval, "${") && strings.HasSuffix(sval, "}")) || strings.HasPrefix(sval, "$") {
					envParts = append(envParts, fmt.Sprintf("%s=%s", k, sval))
				} else {
					envParts = append(envParts, fmt.Sprintf("%s=%s", k, shellQuote(sval)))
				}
			}
		}
	}

	// build full command
	var parts []string
	if len(envParts) > 0 {
		parts = append(parts, strings.Join(envParts, " "))
	}
	// quote the command itself if needed
	parts = append(parts, shellQuote(cmdStr))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// shellQuote applies simple single-quote quoting suitable for POSIX shells.
// If the string already looks like a shell variable reference (starts with $ or ${...})
// it is returned unchanged.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		return s
	}
	if strings.HasPrefix(s, "$") {
		return s
	}
	// safe characters: alphanum, ./@-_:+=, don't quote
	safe := true
	for _, r := range s {
		if !(r == '.' || r == '/' || r == '@' || r == '-' || r == '_' || r == ':' || r == '+' || r == '=' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	// escape single quotes by closing, inserting '\'' and reopening
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}
