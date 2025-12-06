package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rgrouls/terminal-wrapped/internal/analyzer"
	"github.com/rgrouls/terminal-wrapped/internal/parser"
	"github.com/rgrouls/terminal-wrapped/internal/ui"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// 1. Parse History
    // Check for ZSH history file
	historyFile := filepath.Join(homeDir, ".zsh_history")
    if _, err := os.Stat(historyFile); os.IsNotExist(err) {
        // Fallback or error?
        // Let's try bash history if zsh not found
        bashHistory := filepath.Join(homeDir, ".bash_history")
        if _, err := os.Stat(bashHistory); err == nil {
            historyFile = bashHistory
        } else {
             fmt.Fprintf(os.Stderr, "Could not find .zsh_history or .bash_history in %s\n", homeDir)
             os.Exit(1)
        }
    }

	fmt.Println("Parsing history file...")
	commands, err := parser.ParseZshHistory(historyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing history: %v\n", err)
		os.Exit(1)
	}

	// 2. Parse Aliases
	aliases, err := analyzer.ParseAliases(homeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse aliases: %v\n", err)
	}

	// 3. Analyze Data
	fmt.Println("Analyzing data...")
	stats := analyzer.Analyze(commands, aliases)

	// 4. Run UI
	p := tea.NewProgram(ui.NewModel(stats), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running UI: %v\n", err)
		os.Exit(1)
	}
}
