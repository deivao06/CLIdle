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
	path := game.SavePath()
	g, savedAt, err := game.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "não consegui ler o save %s: %v\n", path, err)
		os.Exit(1)
	}
	var away time.Duration
	if g == nil {
		g = game.New(uint64(time.Now().UnixNano()))
	} else {
		away = time.Since(savedAt)
	}
	if _, err := tea.NewProgram(ui.NewWithSave(g, path, away), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
