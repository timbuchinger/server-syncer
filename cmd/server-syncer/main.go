package main

import (
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

const defaultAgents = "Copilot,Codex,ClaudeCode,Gemini"

func main() {
	templatePath := flag.String("template", "", "path to the template file")
	sourceAgent := flag.String("source", "", "source-of-truth agent name")
	agents := flag.String("agents", "", "comma-separated list of agents to keep in sync (defaults to Copilot,Codex,ClaudeCode,Gemini)")
	configPath := flag.String("config", defaultConfigPath(), "path to YAML configuration file describing the source and target agents")
	flag.Parse()

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
