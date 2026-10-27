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
	right := m.statusLine()
	for _, tabs := range [][]string{{"1 Principal", "2 Preparação", "3 Herói"}, {"1", "2", "3"}} {
		var parts []string
		for i, t := range tabs {
			if i == m.tab && !m.help {
				parts = append(parts, sTabOn.Render(" "+t+" "))
			} else {
				parts = append(parts, sTabOff.Render(" "+t+" "))
			}
		}
		left := strings.Join(parts, "")
		if gap := m.w - lipgloss.Width(left) - lipgloss.Width(right); gap >= 1 {
			return left + strings.Repeat(" ", gap) + right
		}
	}
	return truncate(right, m.w)
}

func (m Model) statusLine() string {
	g := m.g
	h := g.Hero
	hp := hpStyle(h.HP, h.MaxHP()).Render(fmt.Sprintf("HP %d/%d", h.HP, h.MaxHP()))
	where := "cidade"
	if g.Exp != nil {
		where = fmt.Sprintf("andar %d", g.Exp.Level.Depth)
	}
	return hp + sMuted.Render(fmt.Sprintf(" · nv %d · %s ", h.Level, where))
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
	if mon := l.MonsterAt(p); mon != nil && l.Visible(g.Hero.Pos, p) {
		k := mon.Info()
		return k.Glyph, monsterStyle(k)
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
	h := g.Hero
	w := sideW - 3
	var b strings.Builder
	fmt.Fprintf(&b, "%s nível %d\n", sSection.Render("Herói"), h.Level)
	fmt.Fprintf(&b, "HP  %s\n", bar(h.HP, h.MaxHP(), w-4, hpStyle(h.HP, h.MaxHP())))
	fmt.Fprintf(&b, "XP  %s\n", bar(h.XP, game.XPToNext(h.Level), w-4, sXP))
	fmt.Fprintf(&b, "Ataque %d · Defesa %d\n", h.Atk(), h.Def())
	fmt.Fprintf(&b, "Velocidade %d (%.1f/s)\n", h.Spd(), game.StepsPerSec(h.Spd()))
	if g.Exp != nil {
		fmt.Fprintf(&b, "\n%s #%d\nAndar %d\n", sSection.Render("Expedição"), g.Exp.Num, g.Exp.Level.Depth)
	}
	fmt.Fprintf(&b, "\n%s\nAndar máx. %d\nMortes %d\n", sSection.Render("Recordes"), g.MaxDepth, g.Deaths)
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
	for _, k := range game.MonsterKinds {
		if k.MinDepth > m.g.MaxDepth {
			rows = append(rows, [2]string{sMuted.Render("?"), sMuted.Render("ainda não encontrado")})
			break
		}
		rows = append(rows, [2]string{monsterStyle(k).Render(string(k.Glyph)), k.Name})
	}
	for _, r := range rows {
		b.WriteString("  " + r[0] + strings.Repeat(" ", max(1, 4-lipgloss.Width(r[0]))) + r[1] + "\n")
	}
	b.WriteString("\n" + sMuted.Render("? fecha"))
	return b.String()
}

func bar(v, maxV, w int, st lipgloss.Style) string {
	w = max(w, 4)
	filled := 0
	if maxV > 0 {
		filled = clamp(v*w/maxV, 0, w)
	}
	return st.Render(strings.Repeat("█", filled)) + sDim.Render(strings.Repeat("░", w-filled))
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
