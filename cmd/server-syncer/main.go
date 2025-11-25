package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"server-syncer/internal/config"
	"server-syncer/internal/syncer"
)

const (
	defaultAgents         = "Copilot,Codex,ClaudeCode,Gemini"
	defaultConfigTemplate = `source: codex
targets:
  - gemini
  - copilot
  - claudecode
`
)

var promptUser = askYes

//go:embed config.example.yml
var exampleConfig string

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := runInitCommand(os.Args[2:]); err != nil {
			log.Fatalf("init failed: %v", err)
		}
		return
	}

	templatePath := flag.String("template", "", "path to the template file")
	sourceAgent := flag.String("source", "", "source-of-truth agent name")
	agents := flag.String("agents", "", "comma-separated list of agents to keep in sync (defaults to Copilot,Codex,ClaudeCode,Gemini)")
	configPath := flag.String("config", defaultConfigPath(), "path to YAML configuration file describing the source and target agents")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: server-syncer [OPTIONS]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nDefault config file location: %s\n", defaultConfigPath())
		fmt.Fprintf(os.Stderr, "\nExample config file:\n%s\n", exampleConfig)
	}

	flag.Parse()

	if err := ensureConfigFile(*configPath); err != nil {
		log.Fatalf("configuration unavailable: %v", err)
	}

	if *templatePath == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg, cfgErr := config.Load(*configPath)
	if cfgErr != nil && !errors.Is(cfgErr, os.ErrNotExist) {
		log.Fatalf("failed to load config %q: %v", *configPath, cfgErr)
	}
	useConfig := cfgErr == nil

	finalSource := strings.TrimSpace(*sourceAgent)
	if finalSource == "" && useConfig {
		finalSource = cfg.Source
	}
	if finalSource == "" {
		log.Fatalf("source agent must be provided via -source or config file at %s", *configPath)
	}

	var candidateAgents []string
	if strings.TrimSpace(*agents) != "" {
		candidateAgents = parseAgents(*agents)
	} else if useConfig {
		candidateAgents = cfg.Targets
	} else {
		candidateAgents = parseAgents(defaultAgents)
	}

	tpl, err := syncer.LoadTemplateFromFile(*templatePath)
	if err != nil {
		log.Fatalf("failed to load template: %v", err)
	}

	s := syncer.New(finalSource, candidateAgents)

	converted, err := s.Sync(tpl)
	if err != nil {
		log.Fatalf("sync failed: %v", err)
	}

	fmt.Println("Converted configurations:")
	for agent, cfg := range converted {
		fmt.Printf("  %s -> %s\n", agent, cfg)
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

func defaultConfigPath() string {
	switch runtime.GOOS {
	case "darwin":
		return "/usr/local/etc/server-syncer.yml"
	case "windows":
		if base := os.Getenv("ProgramData"); base != "" {
			return filepath.Join(base, "server-syncer", "config.yml")
		}
		return `C:\ProgramData\server-syncer\config.yml`
	default:
		return "/etc/server-syncer.yml"
	}
}

func ensureConfigFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect %q: %w", path, err)
	}

	prompt := fmt.Sprintf("Configuration %s not found. Create a default config? [Y/n]: ", path)
	if !promptUser(prompt, true) {
		return fmt.Errorf("configuration file %s is required", path)
	}

	if err := writeDefaultConfig(path); err != nil {
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

	if err := writeDefaultConfig(path); err != nil {
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

func writeDefaultConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure directory %q: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(defaultConfigTemplate), 0o644); err != nil {
		return fmt.Errorf("failed to write config %q: %w", path, err)
	}
	return nil
}
