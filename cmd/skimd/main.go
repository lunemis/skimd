package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lunemis/skimd/internal/browser"
	skimdtmux "github.com/lunemis/skimd/internal/tmux"
	"github.com/lunemis/skimd/internal/ui"
)

var version = "dev"

func main() {
	var printBinding bool
	var tmuxKey string
	var showVersion bool

	flag.BoolVar(&printBinding, "print-tmux-binding", false, "print a tmux popup binding snippet")
	flag.StringVar(&tmuxKey, "tmux-key", "v", "tmux key for the popup binding")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("skimd %s\n", version)
		return
	}

	if printBinding {
		executable, err := os.Executable()
		if err != nil {
			executable = "skimd"
		}
		fmt.Println(skimdtmux.PopupBinding(executable, tmuxKey))
		return
	}

	startArg := ""
	if flag.NArg() > 0 {
		startArg = flag.Arg(0)
	}

	start, err := browser.ResolveStartLocation(startArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve start path: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.NewModel(start), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
