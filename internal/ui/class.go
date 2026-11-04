package ui

import (
	"strings"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) chooseKey(key string) {
	switch key {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(int(game.NumClasses)-1, m.cursor+1)
	case "enter", " ":
		m.g.ChooseClass(game.Class(m.cursor))
		m.choosing, m.cursor = false, 0
	}
}

func (m Model) viewChoose() string {
	var b strings.Builder
	b.WriteString(sTitle.Render(lang.T("choose.title")) + "\n\n")
	for c := range game.NumClasses {
		name := sPlain.Render(c.Name())
		prefix := "  "
		if int(c) == m.cursor {
			name, prefix = sTitle.Render(c.Name()), sTabOn.Render("›")+" "
		}
		b.WriteString(prefix + name + "\n    " + sMuted.Render(c.Desc()+lang.T("class.main", lang.T([...]string{"stat.str", "stat.dex", "stat.int", "stat.con"}[c.MainAttr()]))) + "\n")
	}
	b.WriteString("\n" + sMuted.Render(lang.T("choose.keys")))
	block := b.String()
	block = lipgloss.NewStyle().Width(lipgloss.Width(block)).Render(block)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, block)
}
