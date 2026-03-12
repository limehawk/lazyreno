package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/limehawk/lazyreno/internal/app"
	"github.com/limehawk/lazyreno/internal/config"
)

var version = "dev"

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\nlazyreno panicked: %v\n", r)
			os.Exit(1)
		}
	}()

	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("lazyreno " + version)
		return
	}

	// Find config file
	configPath := ""
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".config", "lazyreno", "config.toml")
		if _, err := os.Stat(candidate); err == nil {
			configPath = candidate
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	m := app.NewModel(cfg)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
