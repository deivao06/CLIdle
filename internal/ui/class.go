package ui

import (
	"fmt"
	"strings"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
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
	descW := min(m.w-6, 72)
	var b strings.Builder
	b.WriteString(sTitle.Render(lang.T("choose.title")) + "\n\n")
	for c := range game.NumClasses {
		name := sPlain.Render(c.Name())
		prefix := "  "
		if int(c) == m.cursor {
			name, prefix = sTitle.Render(c.Name()), sTabOn.Render("›")+" "
		}
		desc := c.Desc() + lang.T("class.main", lang.T([...]string{"stat.str", "stat.dex", "stat.int", "stat.con"}[c.MainAttr()]))
		lines := []string{prefix + name}
		wrapped := strings.Split(lipgloss.NewStyle().Width(descW).Render(desc), "\n")
		for _, l := range wrapped[:min(len(wrapped), maxDescLines)] {
			lines = append(lines, "    "+sMuted.Render(strings.TrimRight(l, " ")))
		}
		for j, l := range lines {
			b.WriteString(zone.Mark(fmt.Sprintf("class:%d.%d", c, j), l+strings.Repeat(" ", max(0, descW+4-lipgloss.Width(l)))) + "\n")
		}
	}
	b.WriteString("\n" + sMuted.Render(m.keysHint("choose.keys")))
	block := b.String()
	block = lipgloss.NewStyle().Width(lipgloss.Width(block)).Render(block)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, block)
}
