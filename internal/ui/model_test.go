package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"clidle/internal/game"
	"clidle/internal/lang"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuitOnClassScreenDoesNotSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	m := NewWithSave(game.New(1, game.Warrior), path, 0, true)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if _, err := os.Stat(path); err == nil {
		t.Fatal("saved the placeholder hero before a class was chosen")
	}
}

func TestGameSettingsComeLastInTheirOwnSection(t *testing.T) {
	m := New(game.New(1, game.Warrior))
	rows := m.prepRows()
	for _, r := range rows[len(rows)-2:] {
		if r.section != lang.T("prep.section.game") {
			t.Fatalf("%q in section %q", r.text, r.section)
		}
	}
}

func TestHeroTabScrollsOnSmallScreens(t *testing.T) {
	g := game.New(1, game.Mage)
	g.Hero.Level = 20
	var m tea.Model = New(g)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 14})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	records := lang.T("hero.records")
	if strings.Contains(m.View(), records) {
		t.Skip("screen not small enough to need scroll")
	}
	for range 30 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if !strings.Contains(m.View(), records) {
		t.Fatal("records never scrolled into view")
	}
}
