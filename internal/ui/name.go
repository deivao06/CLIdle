package ui

import (
	"strings"
	"unicode"

	"clidle/internal/lang"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const nameMax = 16

// Typing tells the page whether the phone keyboard should open; only the web build sets it.
var Typing = func(bool) {}

func (m *Model) nameKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(string(m.nameDraft))
		if name == "" {
			return
		}
		m.g.Hero.Name = name
		m.naming, m.nameDraft = false, nil
		Typing(false)
	case tea.KeyBackspace:
		if n := len(m.nameDraft); n > 0 {
			m.nameDraft = m.nameDraft[:n-1]
		}
	case tea.KeySpace:
		m.addNameRune(' ')
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.addNameRune(r)
		}
	}
}

func (m *Model) addNameRune(r rune) {
	if len(m.nameDraft) < nameMax && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(" -'.", r)) {
		m.nameDraft = append(m.nameDraft, r)
	}
}

func (m Model) viewName() string {
	field := string(m.nameDraft) + sTitle.Render("_")
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(sBar.GetForeground()).
		Padding(0, 1).Width(nameMax + 3).Render(field)
	hint := lang.T("name.keys")
	if m.narrow() {
		hint = lang.T("name.touch")
	}
	block := lipgloss.JoinVertical(lipgloss.Center, sTitle.Render(lang.T("name.title")), "", box, "", sMuted.Render(hint))
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, block)
}
