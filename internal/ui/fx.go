package ui

import (
	"fmt"
	"time"

	"clidle/internal/game"

	"github.com/charmbracelet/lipgloss"
)

const (
	fxFlash     = 150 * time.Millisecond
	fxFloat     = 700 * time.Millisecond
	fxShake     = 120 * time.Millisecond
	fxShakeStep = 40 * time.Millisecond
)

var (
	sFlashHit  = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("88")).Bold(true)
	sFlashHurt = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	sDmgHit    = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
	sDmgBig    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	sDmgHurt   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
)

// fx is a short-lived visual effect born from a game event, timed by the UI clock.
type fx struct {
	game.Event
	at  time.Time
	lvl *game.Level
}

type fxCell struct {
	r  rune
	st *lipgloss.Style
}

func (m *Model) collectFx(now time.Time) {
	var lvl *game.Level
	if m.g.Exp != nil {
		lvl = m.g.Exp.Level
	}
	keep := m.fx[:0]
	for _, f := range m.fx {
		if f.lvl == lvl && now.Sub(f.at) < fxFloat {
			keep = append(keep, f)
		}
	}
	m.fx = keep
	if lvl != nil {
		for _, e := range m.g.Events {
			m.fx = append(m.fx, fx{Event: e, at: now, lvl: lvl})
		}
	}
	m.g.Events = m.g.Events[:0]
}

// fxLayers returns the floating numbers and flashes to draw over the map, and the camera shake offset.
func (m Model) fxLayers() (overlay, flash map[game.Pos]fxCell, shake int) {
	overlay, flash = map[game.Pos]fxCell{}, map[game.Pos]fxCell{}
	for _, f := range m.fx {
		age := m.lastTick.Sub(f.at)
		if age < 0 || age >= fxFloat {
			continue
		}
		if f.Big && age < fxShake {
			shake = 1 - 2*int(age/fxShakeStep%2)
		}
		text, st, fl := fmt.Sprint(f.Value), &sDmgHit, &sFlashHit
		switch {
		case f.Kind == game.EventHurt:
			text, st, fl = "-"+text, &sDmgHurt, &sFlashHurt
		case f.Big:
			text, st = text+"!", &sDmgBig
		}
		if age < fxFlash {
			flash[f.Pos] = fxCell{st: fl}
		}
		y := f.Pos.Y - 1
		if age >= fxFloat/2 {
			y--
		}
		x := f.Pos.X - len(text)/2
		for i, r := range text {
			overlay[game.Pos{X: x + i, Y: y}] = fxCell{r, st}
		}
	}
	return overlay, flash, shake
}
