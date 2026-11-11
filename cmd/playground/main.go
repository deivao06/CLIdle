// Command playground builds a fight on hand-made floors and plays it with the game's rules.
// It is not part of the releases: go run ./cmd/playground
package main

import (
	"fmt"
	"os"

	"clidle/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if _, err := tea.NewProgram(ui.NewLab(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
