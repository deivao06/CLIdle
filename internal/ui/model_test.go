package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"clidle/internal/game"
	"clidle/internal/lang"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	seen := false
	for range 60 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		seen = seen || strings.Contains(m.View(), records)
	}
	if !seen {
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

func TestEffectsOffDrawsNoLampOrFx(t *testing.T) {
	g := game.New(1, game.Warrior)
	g.Depart()
	m := New(g)
	m.fx = []fx{{Event: game.Event{Kind: game.EventHit, Pos: g.Hero.Pos, Value: 9, Big: true}, at: m.lastTick, lvl: g.Exp.Level}}
	if o, _, shake := m.fxLayers(); len(o) == 0 || shake == 0 {
		t.Fatal("full effects should draw the hit and shake")
	}
	g.Effects = game.EffectsOff
	if o, f, shake := m.fxLayers(); len(o)+len(f) != 0 || shake != 0 {
		t.Fatal("effects off still draws")
	}
	if m.lampLight(g.Exp.Level, g.Hero.Pos, true) >= 0 {
		t.Fatal("effects off still lights the lamp")
	}
}

func TestSellingLegendaryAsksFirst(t *testing.T) {
	g := game.New(1, game.Warrior)
	g.Stash = []*game.Item{{Slot: game.Armor, Base: 3, Rarity: game.Legendary}}
	m := New(g)
	m.tab = tabDungeon
	for i, r := range m.townRows() {
		if r.sell != nil {
			m.cursor = i
		}
	}
	m.prepKey("x")
	if len(g.Stash) != 1 || m.confirmDo == nil {
		t.Fatalf("legendary sold without asking: stash %d", len(g.Stash))
	}
	m.confirmDo(&m)
	if len(g.Stash) != 0 || g.Gold == 0 {
		t.Fatalf("confirming did not sell: stash %d gold %d", len(g.Stash), g.Gold)
	}
}

func TestClassesHelpListsEveryAbility(t *testing.T) {
	text := helpText("help.page.classes")
	for a := range game.NumAbilities {
		if !strings.Contains(text, a.Name()) || !strings.Contains(text, a.Desc()) || strings.HasSuffix(a.Desc(), ".desc") {
			t.Fatalf("ability %s missing or without description", a.Name())
		}
	}
}

func TestHangLineKeepsDetailUnderTheValue(t *testing.T) {
	s := "Expeditions  102  (101 returns, 0 deaths)"
	if got := hangLine(s, 60); got != s {
		t.Errorf("fitting line changed: %q", got)
	}
	want := "Expeditions  102\n             (101 returns, 0 deaths)"
	if got := hangLine(s, 36); got != want {
		t.Errorf("hangLine(36) =\n%s\nwant\n%s", got, want)
	}
	for _, l := range strings.Split(hangLine(s, 20), "\n") {
		if lipgloss.Width(l) > 20 {
			t.Errorf("line %q wider than 20", l)
		}
	}
}

func press(m Model, keys ...string) Model {
	for _, k := range keys {
		next, _ := m.Update(keyMsg(k))
		m = next.(Model)
	}
	return m
}

func TestRuleEditorChangesAndClosesARule(t *testing.T) {
	g := game.New(1, game.Warrior)
	m := New(g)
	m.w, m.h = 100, 30
	m = press(m, "2")
	rows := m.rows()
	first := len(rows) - len(g.Config.Gambits) - 2
	m.cursor = first
	m = press(m, "enter")
	if m.edit != 1 {
		t.Fatalf("editor not open: edit %d", m.edit)
	}
	for i, r := range m.rows() {
		if r.label == lang.T("tactics.field.do") {
			m.cursor = i
		}
	}
	if g.Config.Gambits[0].Do != game.ActSecondWind {
		t.Fatalf("rule 1 is %v", g.Config.Gambits[0].Do)
	}
	m = press(m, "right")
	if g.Config.Gambits[0].Do != game.ActAttack {
		t.Fatalf("action %v, want attack (heal and bless are cleric-only)", g.Config.Gambits[0].Do)
	}
	m = press(m, "esc")
	if m.edit != 0 || m.cursor != first {
		t.Fatalf("edit %d cursor %d, want list at %d", m.edit, m.cursor, first)
	}
}

func TestAddAndDeleteRule(t *testing.T) {
	g := game.New(1, game.Warrior)
	m := New(g)
	m.w, m.h = 100, 30
	m = press(m, "2")
	n := len(g.Config.Gambits)
	m.cursor = len(m.rows()) - 2
	m = press(m, "enter")
	if len(g.Config.Gambits) != n+1 || m.edit != n+1 {
		t.Fatalf("rules %d edit %d", len(g.Config.Gambits), m.edit)
	}
	for i, r := range m.rows() {
		if r.text == lang.T("tactics.delete") {
			m.cursor = i
		}
	}
	m = press(m, "enter", "y")
	if len(g.Config.Gambits) != n || m.edit != 0 {
		t.Fatalf("rules %d edit %d after delete", len(g.Config.Gambits), m.edit)
	}
}
