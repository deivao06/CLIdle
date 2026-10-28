package main

import (
	"fmt"
	"os"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"
	"clidle/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	path := game.SavePath()
	g, savedAt, err := game.Load(path)
	if err != nil {
		fmt.Fprint(os.Stderr, lang.T("main.load_error", path, err))
		os.Exit(1)
	}
	var away time.Duration
	if g == nil {
		g = game.New(uint64(time.Now().UnixNano()), game.Warrior)
	} else {
		away = time.Since(savedAt)
	}
	lang.Set(g.Lang)
	if _, err := tea.NewProgram(ui.NewWithSave(g, path, away), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
