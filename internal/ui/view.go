package ui

import (
	"fmt"
	"strings"

	"clidle/internal/game"

	"github.com/charmbracelet/lipgloss"
)

const (
	minW, minH = 40, 10
	sideW      = 28
)

func (m Model) View() string {
	if m.w == 0 {
		return ""
	}
	if m.w < minW || m.h < minH {
		msg := sWarn.Render("Janela muito pequena") + "\n" +
			sMuted.Render(fmt.Sprintf("mínimo %dx%d · atual %dx%d", minW, minH, m.w, m.h))
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, msg)
	}
	bodyH := m.h - 2
	var body string
	switch {
	case m.help:
		body = m.viewHelp(bodyH)
	case m.tab == tabDungeon:
		body = m.viewDungeon(bodyH)
	default:
		body = sMuted.Render("Em breve.")
	}
	body = lipgloss.NewStyle().Width(m.w).Height(bodyH).MaxHeight(bodyH).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, m.viewHeader(), body, m.viewFooter())
}

func (m Model) viewHeader() string {
	tabs := []string{"1 Dungeon", "2 Cidade", "3 Herói"}
	var parts []string
	for i, t := range tabs {
		if i == m.tab && !m.help {
			parts = append(parts, sTabOn.Render(" "+t+" "))
		} else {
			parts = append(parts, sTabOff.Render(" "+t+" "))
		}
	}
	left := sTitle.Render("clidle ") + strings.Join(parts, "")
	right := m.statusLine()
	gap := m.w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return truncate(left, m.w)
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) statusLine() string {
	g := m.g
	if g.Exp == nil {
		return sMuted.Render("na cidade ")
	}
	return sMuted.Render(fmt.Sprintf("andar %d ", g.Exp.Level.Depth))
}

func (m Model) viewFooter() string {
	return truncate(sMuted.Render("1-3 telas · ? legenda · q sair"), m.w)
}

func (m Model) viewDungeon(h int) string {
	g := m.g
	side := m.w >= game.MapW+sideW
	mapW := min(game.MapW, m.w)
	if side {
		mapW = min(game.MapW, m.w-sideW)
	}
	mapH, logH := game.MapH, h-game.MapH
	if logH < 3 {
		mapH, logH = h-1, 1
	}
	var left string
	if g.Exp == nil {
		left = lipgloss.Place(mapW, mapH, lipgloss.Center, lipgloss.Center, sMuted.Render("O herói descansa na cidade..."))
	} else {
		left = m.renderMap(mapW, mapH)
	}
	left = lipgloss.JoinVertical(lipgloss.Left, left, m.renderLog(mapW, logH))
	if !side {
		return left
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sPanel.Width(m.w-mapW-1).Height(h).Render(m.viewSide()))
}

func (m Model) renderMap(w, h int) string {
	g := m.g
	l := g.Exp.Level
	hp := g.Hero.Pos
	cx := clamp(hp.X-w/2, 0, game.MapW-w)
	cy := clamp(hp.Y-h/2, 0, game.MapH-h)
	var sb strings.Builder
	for y := cy; y < cy+h; y++ {
		if y > cy {
			sb.WriteByte('\n')
		}
		var run strings.Builder
		var runStyle *lipgloss.Style
		flush := func() {
			if runStyle != nil && run.Len() > 0 {
				sb.WriteString(runStyle.Render(run.String()))
			}
			run.Reset()
		}
		for x := cx; x < cx+w; x++ {
			ch, st := m.cell(l, game.Pos{X: x, Y: y})
			if runStyle != st {
				flush()
				runStyle = st
			}
			run.WriteRune(ch)
		}
		flush()
	}
	return sb.String()
}

func (m Model) cell(l *game.Level, p game.Pos) (rune, *lipgloss.Style) {
	g := m.g
	if p.Y < 0 || p.Y >= game.MapH || p.X < 0 || p.X >= game.MapW {
		return ' ', &sPlain
	}
	if p == g.Hero.Pos {
		return '@', &sHero
	}
	if !l.IsSeen(p) {
		return ' ', &sPlain
	}
	t := l.At(p)
	if !l.Visible(g.Hero.Pos, p) {
		return t.Glyph(), &sDim
	}
	return t.Glyph(), tileStyle(t)
}

func (m Model) renderLog(w, h int) string {
	entries := m.g.Log
	start := max(0, len(entries)-h)
	var lines []string
	for _, e := range entries[start:] {
		lines = append(lines, logStyles[e.Kind].Render(truncate(e.Text, w)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewSide() string {
	g := m.g
	var b strings.Builder
	b.WriteString(sSection.Render("Herói") + "\n")
	fmt.Fprintf(&b, "Velocidade %d (%.1f/s)\n", g.Hero.Spd, game.StepsPerSec(g.Hero.Spd))
	if g.Exp != nil {
		fmt.Fprintf(&b, "\n%s\nExpedição #%d\nAndar %d\n", sSection.Render("Expedição"), g.Exp.Num, g.Exp.Level.Depth)
	}
	return b.String()
}

func (m Model) viewHelp(h int) string {
	var b strings.Builder
	b.WriteString(sSection.Render("Legenda") + "\n\n")
	rows := [][2]string{
		{sHero.Render("@"), "herói"},
		{sFloor.Render("."), "chão"},
		{sFloor.Render("#"), "corredor"},
		{sDoor.Render("+"), "porta"},
		{sWall.Render("- |"), "parede"},
		{sStairs.Render(">"), "escada para baixo"},
		{sStairs.Render("<"), "escada para cima (saída)"},
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-3s %s\n", r[0], r[1])
	}
	b.WriteString("\n" + sMuted.Render("? fecha"))
	return b.String()
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return max(lo, min(v, hi))
}

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}
