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

func TestGameOptionsMovedToSettingsTab(t *testing.T) {
	m := New(game.New(1, game.Warrior))
	for _, r := range m.prepRows() {
		if r.section == lang.T("prep.section.game") {
			t.Fatalf("%q still in preparation", r.text)
		}
	}
	m.tab = tabSettings
	if len(m.settingsRows()) < 3 {
		t.Fatal("settings tab is missing options")
	}
}

func TestResetSaveAsksThenStartsOver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := game.New(1, game.Mage)
	g.Gold = 500
	var tm tea.Model = NewWithSave(g, path, 0, false)
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m := tm.(Model)
	m.cursor = len(m.settingsRows()) - 1
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m = tm.(Model); m.confirmDo == nil || m.g.Gold != 500 {
		t.Fatal("reset did not ask for confirmation first")
	}
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = tm.(Model)
	if m.g.Gold != 0 || !m.choosing {
		t.Fatalf("game not reset: gold %d choosing %v", m.g.Gold, m.choosing)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("old save still on disk")
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

func TestChestRowsEquipAndSell(t *testing.T) {
	g := game.New(1, game.Warrior)
	g.Hero.Equip[game.Weapon] = nil
	g.Stash = []*game.Item{{Slot: game.Weapon, Kind: game.Blade, Base: 5}, {Slot: game.Armor, Base: 3}}
	m := New(g)
	m.tab = tabDungeon
	for i, r := range m.townRows() {
		if r.sell != nil {
			m.cursor = i
			break
		}
	}
	m.prepKey("enter")
	if g.Hero.Equip[game.Weapon] == nil || len(g.Stash) != 1 {
		t.Fatalf("enter did not equip: stash %d", len(g.Stash))
	}
	for i, r := range m.townRows() {
		if r.sell != nil {
			m.cursor = i
		}
	}
	m.prepKey("x")
	if len(g.Stash) != 0 || g.Gold == 0 {
		t.Fatalf("x did not sell: stash %d gold %d", len(g.Stash), g.Gold)
	}
}

func TestHelpPagesCycleWithArrows(t *testing.T) {
	var m tea.Model = New(game.New(1, game.Warrior))
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.(Model); !got.help || got.helpPage != 1 {
		t.Fatalf("help %v page %d", got.help, got.helpPage)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if got := m.(Model); got.helpPage != len(helpPages)-1 {
		t.Fatalf("page %d did not wrap", got.helpPage)
	}
}
