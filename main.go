package main

import (
	"fmt"
	"os"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"
	"clidle/internal/ui"
	"clidle/internal/update"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if exe, err := update.Executable(); err == nil {
		update.Cleanup(exe)
	}
	path := game.SavePath()
	g, savedAt, err := game.Load(path)
	if err != nil {
		fmt.Fprint(os.Stderr, lang.T("main.load_error", path, err))
		os.Exit(1)
	}
	var away time.Duration
	fresh := g == nil
	if fresh {
		g = game.New(uint64(time.Now().UnixNano()), game.Warrior)
	} else {
		away = time.Since(savedAt)
	}
	lang.Set(g.Lang)
	final, err := tea.NewProgram(ui.NewWithSave(g, path, away, fresh), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if m, ok := final.(ui.Model); ok && m.Restarting() {
		exe, err := update.Executable()
		if err == nil {
			err = update.Restart(exe)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
