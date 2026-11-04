package ui

import (
	"strings"
	"testing"
	"time"

	"clidle/internal/game"
	"clidle/internal/update"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsShowVersion(t *testing.T) {
	m := New(game.New(1, game.Warrior))
	m.w, m.h, m.tab = 100, 30, tabSettings
	lines := strings.Split(m.View(), "\n")
	if !strings.Contains(lines[len(lines)-2], update.Version) {
		t.Fatalf("version not at the bottom of the settings tab: %q", lines[len(lines)-2])
	}
}

func TestUpdateNoticeAndKey(t *testing.T) {
	var tm tea.Model = New(game.New(1, game.Warrior))
	m := tm.(Model)
	m.w, m.h = 110, 30
	m.release = &update.Release{Version: "v9.9.9"}
	if !strings.Contains(m.View(), "v9.9.9") {
		t.Fatal("no update notice in the header")
	}
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = tm.(Model)
	if m.confirmDo == nil || !strings.Contains(m.confirmText, "v9.9.9") {
		t.Fatal("u did not ask to update")
	}
	m.g.Depart()
	tm, _ = m.Update(tickMsg(time.Now()))
	if tm.(Model).confirmDo == nil {
		t.Fatal("update question closed when the hero left town")
	}
}

func TestTownArrivalChecksAtMostHourly(t *testing.T) {
	old := update.Version
	t.Cleanup(func() { update.Version = old })
	m := New(game.New(1, game.Warrior))
	update.Version = "dev"
	if m.townArrival(time.Now()) != nil {
		t.Fatal("dev build looked for updates")
	}
	update.Version = "v0.1.0"
	now := time.Now()
	m.wasInTown = false
	if m.townArrival(now) == nil {
		t.Fatal("arriving in town did not check")
	}
	m.checking, m.wasInTown = false, false
	if m.townArrival(now.Add(10*time.Minute)) != nil {
		t.Fatal("checked again within the hour")
	}
	m.wasInTown = false
	if m.townArrival(now.Add(61*time.Minute)) == nil {
		t.Fatal("did not check after an hour")
	}
}
