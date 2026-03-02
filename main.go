package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denimcouch/kancli-demo/db"
	"github.com/denimcouch/kancli-demo/ui"
)

func main() {
	dbPath, err := resolveDBPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving db path: %v\n", err)
		os.Exit(1)
	}

	database, err := db.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	m, err := ui.NewModel(database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing app: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running app: %v\n", err)
		os.Exit(1)
	}
}

func resolveDBPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Fall back to home dir
		dir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	appDir := filepath.Join(dir, "kancli")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(appDir, "kancli.db"), nil
}
