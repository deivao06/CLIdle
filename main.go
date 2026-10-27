package main

import (
	"fmt"
	"os"
	"time"

	"clidle/internal/game"
	"clidle/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	g := game.New(uint64(time.Now().UnixNano()))
	if _, err := tea.NewProgram(ui.New(g), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
