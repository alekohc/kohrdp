package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"kohrdp/internal/config"
	"kohrdp/internal/ui"
)

var version = "dev"

func main() {
	if name, multimon, ok, err := parseNamedArgs(os.Args[1:]); ok {
		if err != nil {
			fmt.Fprintln(os.Stderr, "kohrdp:", err)
			os.Exit(2)
		}
		if err := runNamedConnect(name, multimon); err != nil {
			fmt.Fprintln(os.Stderr, "kohrdp:", err)
			os.Exit(1)
		}
		return
	}

	if direct, ok, err := parseDirectArgs(os.Args[1:]); ok {
		if err != nil {
			fmt.Fprintln(os.Stderr, "kohrdp:", err)
			os.Exit(2)
		}
		if err := runDirectConnect(direct); err != nil {
			fmt.Fprintln(os.Stderr, "kohrdp:", err)
			os.Exit(1)
		}
		return
	}

	showVersion := flag.Bool("version", false, "print version")
	exportPath := flag.String("export", "", "export session config to file")
	importPath := flag.String("import", "", "import session config from file")
	importMode := flag.String("import-mode", "replace", "import mode: replace or merge")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if *exportPath != "" && *importPath != "" {
		fmt.Fprintln(os.Stderr, "kohrdp: use only one of --export or --import")
		os.Exit(2)
	}
	if *exportPath != "" {
		if err := exportConfig(*exportPath); err != nil {
			fmt.Fprintln(os.Stderr, "kohrdp: export failed:", err)
			os.Exit(1)
		}
		return
	}
	if *importPath != "" {
		if err := importSessions(*importPath, *importMode); err != nil {
			fmt.Fprintln(os.Stderr, "kohrdp: import failed:", err)
			os.Exit(1)
		}
		return
	}

	sessions, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kohrdp: failed to load config:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(sessions), tea.WithAltScreen())
	model, err := p.Run()
	if m, ok := model.(ui.Model); ok {
		m.Cleanup()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "kohrdp:", err)
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

func importSessions(path, mode string) error {
	imported, err := config.LoadFrom(path)
	if err != nil {
		return err
	}
	switch mode {
	case "replace":
		return config.Save(imported)
	case "merge":
		current, err := config.Load()
		if err != nil {
			return err
		}
		for name, session := range imported {
			current[name] = session
		}
		return config.Save(current)
	default:
		return fmt.Errorf("invalid import mode %q", mode)
	}
}
