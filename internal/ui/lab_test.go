package ui

import (
	"strings"
	"testing"

	"clidle/internal/game"

	tea "github.com/charmbracelet/bubbletea"
)

func labPress(l Lab, keys ...string) Lab {
	for _, k := range keys {
		m, _ := l.Update(keyMsg(k))
		l = m.(Lab)
	}
	return l
}

func TestLabPagesRender(t *testing.T) {
	m, _ := NewLab().Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	l := m.(Lab)
	for _, page := range []string{"1", "2", "3", "4", "5", "6"} {
		l = labPress(l, page)
		if v := l.View(); strings.Count(v, "\n") != 39 {
			t.Fatalf("page %s has %d lines", page, strings.Count(v, "\n")+1)
		}
	}
}

func TestLabChangeRebuildsTheRun(t *testing.T) {
	l := labPress(NewLab(), "p")
	if l.g.Lab().Steps != 1 {
		t.Fatalf("steps = %d", l.g.Lab().Steps)
	}
	l = labPress(l, "2", "right")
	if l.sc.Class != game.Mage || l.g.Hero.Class != game.Mage || l.g.Lab().Steps != 0 {
		t.Fatal("changing the class should rebuild the run as a mage")
	}
}

func TestLabSavedRuleGoesToTheScenario(t *testing.T) {
	l := NewLab()
	l = labPress(l, "5", "enter", "down", "down", "right")
	for l.editing() && l.cursor < len(l.rows())-2 {
		l = labPress(l, "down")
	}
	l = labPress(l, "enter", "r")
	if l.sc.Gambits[0] != l.g.Config.Gambits[0] || l.sc.Gambits[0] == game.DefaultScenario().Gambits[0] {
		t.Fatalf("rule %v not kept", l.sc.Gambits[0])
	}
}

func TestLabSeedIsTyped(t *testing.T) {
	l := labPress(NewLab(), "4", "down", "down", "enter", "backspace", "4", "x", "2", "q", "enter")
	if l.sc.Seed != 42 || l.st.typing {
		t.Fatalf("seed %d", l.sc.Seed)
	}
	l = labPress(l, "enter", "9", "esc")
	if l.sc.Seed != 42 {
		t.Fatal("esc keeps the old seed")
	}
}

func TestLabNewFloorStartsEmpty(t *testing.T) {
	l := labPress(NewLab(), "4", "right")
	if l.st.floor != 2 || l.sc.Floor(2) != nil {
		t.Fatal("floor 2 should not exist yet")
	}
	l = labPress(l, "down", "down", "down", "right", "enter", "enter")
	if f := l.sc.Floor(2); f == nil || !f.Rooms[2] || f.Up != 2 || !l.st.inRoom {
		t.Fatalf("enter opens room 3 and its first row turns it on: %+v", l.sc.Floors)
	}
	l = labPress(l, "enter", "esc")
	if l.sc.Floor(2) != nil || len(l.sc.Floors) != 1 {
		t.Fatal("a floor without rooms is dropped")
	}
	if l.st.inRoom || l.cursor != labGridRow {
		t.Fatal("esc goes back to the grid, on the room's row")
	}
}
