package main

import (
	"bufio"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"agent-align/internal/config"
	"agent-align/internal/syncer"
)

const (
	defaultAgents = "copilot,vscode,codex,claudecode,gemini"
)

var (
	promptUser      = askYes
	collectConfig   = promptForConfig
	supportedAgents = []string{"codex", "vscode", "gemini", "copilot", "claudecode"}
)

//go:embed config.example.yml
var exampleConfig string

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := runInitCommand(os.Args[2:]); err != nil {
			log.Fatalf("init failed: %v", err)
		}
		return
	}
	if err := validateCommand(os.Args); err != nil {
		log.Fatal(err)
	}

	sourceAgent := flag.String("source", "", "source-of-truth agent name")
	agents := flag.String("agents", "", fmt.Sprintf("comma-separated list of agents to keep in sync (defaults to %s)", defaultAgents))
	configPath := flag.String("config", defaultConfigPath(), "path to YAML configuration file describing the source and target agents")
	dryRun := flag.Bool("dry-run", false, "only show what would be changed without applying changes")
	confirm := flag.Bool("confirm", false, "skip user confirmation prompt (useful for cron jobs)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: agent-align [OPTIONS]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nDefault config file location: %s\n", defaultConfigPath())
		fmt.Fprintf(os.Stderr, "\nExample config file:\n%s\n", exampleConfig)
		fmt.Fprintf(os.Stderr, "Tip: add agent-align to cron for continuous syncing, e.g.:\n")
		fmt.Fprintf(os.Stderr, "  0 * * * * /usr/local/bin/agent-align -config /etc/agent-align.yml -confirm >> /var/log/agent-align.log 2>&1\n\n")
	}

	flag.Parse()

	var configFlagUsed bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			configFlagUsed = true
		}
	})

	useConfig, modeErr := resolveExecutionMode(*sourceAgent, *agents, configFlagUsed)
	if modeErr != nil {
		log.Fatal(modeErr)
	}

	var finalSource string
	var candidateAgents []string

	if useConfig {
		if err := ensureConfigFile(*configPath); err != nil {
			log.Fatalf("configuration unavailable: %v", err)
		}

		cfg, cfgErr := config.Load(*configPath)
		if cfgErr != nil {
			log.Fatalf("failed to load config %q: %v", *configPath, cfgErr)
		}
		finalSource = cfg.Source
		candidateAgents = cfg.Targets
	} else {
		finalSource = strings.TrimSpace(*sourceAgent)
		candidateAgents = parseAgents(*agents)
		if len(candidateAgents) == 0 {
			log.Fatal("the -agents flag must list at least one agent")
		}
	}

	if strings.TrimSpace(finalSource) == "" {
		log.Fatal("source agent must be provided via a config file or -source/-agents")
	}
	if len(candidateAgents) == 0 {
		log.Fatal("no target agents configured; provide them via config or -agents")
	}

	sourceCfg, err := syncer.GetAgentConfig(finalSource)
	if err != nil {
		log.Fatalf("failed to locate source agent config for %s: %v", finalSource, err)
	}
	tpl, err := syncer.LoadTemplateFromFile(sourceCfg.FilePath)
	if err != nil {
		log.Fatalf("failed to load template: %v", err)
	}

	s := syncer.New(finalSource, candidateAgents)

	converted, err := s.Sync(tpl)
	if err != nil {
		log.Fatalf("sync failed: %v", err)
	}

	// Display the dry run results
	fmt.Println("\n=== Dry Run Results ===")
	fmt.Println("The following configuration changes will be made:")
	fmt.Println()

	for agent, cfgContent := range converted {
		agentCfg, err := syncer.GetAgentConfig(agent)
		if err != nil {
			log.Printf("Warning: could not get config for agent %s: %v", agent, err)
			continue
		}
		fmt.Printf("Agent: %s\n", agent)
		fmt.Printf("  File: %s\n", agentCfg.FilePath)
		fmt.Printf("  Format: %s\n", agentCfg.Format)
		fmt.Printf("  Content:\n")
		// Indent the content for readability
		lines := strings.Split(cfgContent, "\n")
		for _, line := range lines {
			fmt.Printf("    %s\n", line)
		}
		fmt.Println()
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
	for agent, cfgContent := range converted {
		agentCfg, err := syncer.GetAgentConfig(agent)
		if err != nil {
			log.Printf("Warning: could not get config for agent %s: %v", agent, err)
			continue
		}

		if err := writeAgentConfig(agentCfg.FilePath, cfgContent); err != nil {
			log.Printf("Error writing config for %s: %v", agent, err)
			continue
		}
		fmt.Printf("  Updated: %s\n", agentCfg.FilePath)
	}
	fmt.Println("\nConfiguration sync complete.")
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
	source, err := promptSourceAgent(reader)
	if err != nil {
		return config.Config{}, err
	}
	targets, err := promptTargetAgents(reader, source)
	if err != nil {
		return config.Config{}, err
	}
	return config.Config{Source: source, Targets: targets}, nil
}

func configPromptSuffix(path string) string {
	if path == defaultConfigPath() {
		return "Use -config to choose another path. "
	}
	return ""
}

func promptSourceAgent(reader *bufio.Reader) (string, error) {
	fmt.Println("\nSelect the source agent:")
	for i, agent := range supportedAgents {
		fmt.Printf("  %d) %s\n", i+1, agent)
	}
	for {
		fmt.Printf("Enter choice [1-%d]: ", len(supportedAgents))
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		choice, convErr := strconv.Atoi(strings.TrimSpace(input))
		if convErr != nil || choice < 1 || choice > len(supportedAgents) {
			fmt.Println("Please enter a number from the list above.")
			continue
		}
		return supportedAgents[choice-1], nil
	}
}

func promptTargetAgents(reader *bufio.Reader, source string) ([]string, error) {
	options := make([]string, 0, len(supportedAgents)-1)
	for _, agent := range supportedAgents {
		if agent == source {
			continue
		}
		options = append(options, agent)
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("no target agents available for source %q", source)
	}

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
		var targets []string
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
			targets = append(targets, options[idx-1])
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

func resolveExecutionMode(sourceFlag, agentsFlag string, configFlagUsed bool) (bool, error) {
	sourceProvided := strings.TrimSpace(sourceFlag) != ""
	agentsProvided := strings.TrimSpace(agentsFlag) != ""

	if configFlagUsed && (sourceProvided || agentsProvided) {
		return true, fmt.Errorf("-config cannot be combined with -source or -agents")
	}
	if sourceProvided != agentsProvided {
		return true, fmt.Errorf("use -source and -agents together or rely entirely on the config file")
	}
	return !sourceProvided, nil
}
