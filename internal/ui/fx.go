package ui

import (
	"fmt"
	"math"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

const (
	fxFlash     = 150 * time.Millisecond
	fxFloat     = 700 * time.Millisecond
	fxShake     = 120 * time.Millisecond
	fxShakeStep = 40 * time.Millisecond
	fxBurst     = 300 * time.Millisecond
	fxSparkle   = 600 * time.Millisecond
	fxTwinkle   = 100 * time.Millisecond
	fxRing      = 700 * time.Millisecond
	fxRingMax   = 7.0
	fxLife      = 800 * time.Millisecond
)

var (
	sFlashHit  = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("88")).Bold(true)
	sFlashHurt = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	sDmgHit    = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
	sDmgBig    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	sDmgHurt   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	sFxMiss    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	sFxDodge   = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	sFxSkill   = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	sFxHeal    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
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
		if f.lvl == lvl && now.Sub(f.at) < fxLife {
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
	full := m.g.Effects == game.EffectsFull
	if m.g.Effects == game.EffectsOff {
		return overlay, flash, 0
	}
	for _, f := range m.fx {
		if age := m.lastTick.Sub(f.at); age >= 0 && full {
			m.moment(f, age, overlay)
		}
	}
	for _, f := range m.fx {
		age := m.lastTick.Sub(f.at)
		text, st, fl := floatText(f.Event)
		if age < 0 || age >= fxFloat || text == "" {
			continue
		}
		if full && f.Big && age < fxShake {
			shake = 1 - 2*int(age/fxShakeStep%2)
		}
		if fl != nil && age < fxFlash {
			flash[f.Pos] = fxCell{st: fl}
		}
		y := f.Pos.Y - 1
		if age >= fxFloat/2 {
			y--
		}
		runes := []rune(text)
		x := f.Pos.X - len(runes)/2
		for taken(overlay, x, y, len(runes)) {
			y--
		}
		for i, r := range runes {
			overlay[game.Pos{X: x + i, Y: y}] = fxCell{r, st}
		}
	}
	return overlay, flash, shake
}

// taken tells whether any of the n tiles from x on row y already holds floating text.
func taken(overlay map[game.Pos]fxCell, x, y, n int) bool {
	for i := range n + 2 {
		if _, ok := overlay[game.Pos{X: x - 1 + i, Y: y}]; ok {
			return true
		}
	}
	return false
}

// floatText is the text that rises from an event, its style and the flash on the tile (nil for none).
func floatText(e game.Event) (string, *lipgloss.Style, *lipgloss.Style) {
	switch e.Kind {
	case game.EventHit:
		if e.Big {
			return fmt.Sprint(e.Value) + "!", &sDmgBig, &sFlashHit
		}
		return fmt.Sprint(e.Value), &sDmgHit, &sFlashHit
	case game.EventHurt:
		return "-" + fmt.Sprint(e.Value), &sDmgHurt, &sFlashHurt
	case game.EventMiss:
		return lang.T("fx.miss"), &sFxMiss, nil
	case game.EventDodge:
		return lang.T("fx.dodge"), &sFxDodge, nil
	case game.EventSkill:
		return game.Ability(e.Value).Name(), &sFxSkill, nil
	case game.EventHeal:
		return "+" + fmt.Sprint(e.Value), &sFxHeal, nil
	}
	return "", nil, nil
}

var (
	sBurst = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	sRing  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
)

// moment draws kill bursts, loot sparkles and the level-up ring.
func (m Model) moment(f fx, age time.Duration, overlay map[game.Pos]fxCell) {
	l := m.g.Exp.Level
	put := func(p game.Pos, r rune, st *lipgloss.Style) {
		if p != m.g.Hero.Pos && l.IsSeen(p) {
			overlay[p] = fxCell{r, st}
		}
	}
	switch f.Kind {
	case game.EventKill:
		switch {
		case age < fxBurst/3:
			put(f.Pos, '*', &sBurst)
			if f.Big {
				for _, d := range game.Dirs8 {
					put(f.Pos.Add(d), '*', &sBurst)
				}
			}
		case age < fxBurst:
			put(f.Pos, '·', &sMuted)
		}
	case game.EventLoot:
		if f.Value < int(game.Rare) || age >= fxSparkle {
			return
		}
		st := rarityStyles[f.Value]
		glyph := [...]rune{'*', '+', '·'}[age/fxTwinkle%3]
		ring := []game.Pos{{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1}}
		if f.Value == int(game.Legendary) {
			ring = append(ring, game.Pos{X: -2}, game.Pos{X: 2}, game.Pos{Y: -2}, game.Pos{Y: 2})
		}
		for _, d := range ring {
			put(f.Pos.Add(d), glyph, &st)
		}
	case game.EventLevelUp:
		if age >= fxRing {
			return
		}
		r := float64(age) / float64(fxRing) * fxRingMax
		for y := f.Pos.Y - int(r/2) - 1; y <= f.Pos.Y+int(r/2)+1; y++ {
			for x := f.Pos.X - int(r) - 1; x <= f.Pos.X+int(r)+1; x++ {
				dx, dy := float64(x-f.Pos.X), float64(y-f.Pos.Y)*2
				if d := math.Sqrt(dx*dx + dy*dy); math.Abs(d-r) < 0.7 {
					put(game.Pos{X: x, Y: y}, '*', &sRing)
				}
			}
		}
	}
}
