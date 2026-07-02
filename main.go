package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"kokoarch-rdp/internal/config"
	"kokoarch-rdp/internal/ui"
)

func main() {
	sessions, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kokoarch-rdp: failed to load config:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(sessions), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "kokoarch-rdp:", err)
		os.Exit(1)
	}
}
