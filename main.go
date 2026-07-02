package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"kokoarch-rdp/internal/config"
	"kokoarch-rdp/internal/ui"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	exportPath := flag.String("export", "", "export session config to file")
	importPath := flag.String("import", "", "import session config from file")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if *exportPath != "" && *importPath != "" {
		fmt.Fprintln(os.Stderr, "kokoarch-rdp: use only one of --export or --import")
		os.Exit(2)
	}
	if *exportPath != "" {
		if err := exportConfig(*exportPath); err != nil {
			fmt.Fprintln(os.Stderr, "kokoarch-rdp: export failed:", err)
			os.Exit(1)
		}
		return
	}
	if *importPath != "" {
		if err := importConfig(*importPath); err != nil {
			fmt.Fprintln(os.Stderr, "kokoarch-rdp: import failed:", err)
			os.Exit(1)
		}
		return
	}

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

func exportConfig(path string) error {
	sessions, err := config.Load()
	if err != nil {
		return err
	}
	return config.SaveTo(path, sessions)
}

func importConfig(path string) error {
	sessions, err := config.LoadFrom(path)
	if err != nil {
		return err
	}
	return config.Save(sessions)
}
