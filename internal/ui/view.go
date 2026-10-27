package ui

import (
	"fmt"
	"strings"
	"time"

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
	if m.catching {
		f := float64(m.progress.Load()) / 1000
		msg := sTitle.Render("Simulando o tempo fora...") + "\n\n" +
			bar(int(f*100), 100, min(40, m.w-4), sXP) + "\n" + sMuted.Render(fmt.Sprintf("%.0f%%", f*100))
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, msg)
	}
	var body string
	switch {
	case m.summary != nil:
		block := m.viewSummary()
		block = lipgloss.NewStyle().Width(lipgloss.Width(block)).Render(block)
		body = lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, block)
	case m.help:
		body = m.viewHelp(bodyH)
	case m.tab == tabDungeon:
		body = m.viewDungeon(bodyH)
	case m.tab == tabPrep:
		body = m.viewPrep(bodyH)
	case m.tab == tabHero:
		body = m.heroPanel(min(48, m.w-4), bodyH)
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
	return hp + sMuted.Render(fmt.Sprintf(" · nv %d · %s · ", h.Level, where)) + sGold.Render(fmt.Sprintf("%d ouro ", g.Gold))
}

func (m Model) viewFooter() string {
	keys := "1-3 telas · r retirar · a auto · ? legenda · q sair"
	if m.g.InTown() {
		keys = "1-3 telas · e partir · a auto · ? legenda · q sair"
	}
	if m.confirmDo != nil {
		return truncate(sTabOn.Render(" "+m.confirmText+" ")+sTitle.Render(" (S/n)"), m.w)
	}
	if m.saveErr != nil {
		return truncate(sWarn.Render("erro ao salvar: "+m.saveErr.Error()), m.w)
	}
	return truncate(sMuted.Render(keys), m.w)
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
		left = lipgloss.Place(mapW, mapH, lipgloss.Center, lipgloss.Center, m.viewTown())
	} else {
		left = m.renderMap(mapW, mapH)
	}
	if !side && logH >= 3 {
		left = lipgloss.JoinVertical(lipgloss.Left, left, m.statsLine(mapW))
		logH--
	}
	left = lipgloss.JoinVertical(lipgloss.Left, left, m.renderLog(mapW, logH))
	if !side {
		return left
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sPanel.Width(m.w-mapW-1).Height(h).Render(m.heroPanel(min(48, m.w-mapW-3), h)))
}

func (m Model) viewSummary() string {
	s := m.summary
	title := fmt.Sprintf("Enquanto você esteve fora (%s)", fmtDuration(s.Away))
	var b strings.Builder
	b.WriteString(sTitle.Render(title) + "\n\n")
	fmt.Fprintf(&b, "Expedições   %d  (%d voltas, %d mortes)\n", s.Expeditions, s.Returns, s.Deaths)
	fmt.Fprintf(&b, "Monstros     %d mortos\n", s.Kills)
	if s.BossKills > 0 {
		b.WriteString(logStyles[game.LogEpic].Render(fmt.Sprintf("Chefes       %d derrotados", s.BossKills)) + "\n")
	}
	b.WriteString(sGold.Render(fmt.Sprintf("Ouro         %+d", s.Gold)) + "\n")
	fmt.Fprintf(&b, "Nível        %d → %d\n", s.LevelFrom, s.LevelTo)
	fmt.Fprintf(&b, "Andar máx.   %d → %d\n", s.DepthFrom, s.DepthTo)
	if s.Capped {
		b.WriteString("\n" + sMuted.Render(fmt.Sprintf("Só as primeiras %s contam.", fmtDuration(game.OfflineCap))) + "\n")
	}
	b.WriteString("\n" + sMuted.Render("aperte qualquer tecla"))
	return b.String()
}

func fmtDuration(d time.Duration) string {
	h, mins := int(d.Hours()), int(d.Minutes())%60
	if h == 0 {
		return fmt.Sprintf("%dmin", mins)
	}
	return fmt.Sprintf("%dh%02d", h, mins)
}

func (m Model) viewTown() string {
	g := m.g
	art := sWall.Render(strings.Join([]string{
		`   |>          |>   `,
		`  /_\  ____   /_\  `,
		`  |.| |.__.| |.|   `,
		`__|_|_|_||_|_|_|__ `,
	}, "\n"))
	next := sTitle.Render("Aperte e para partir")
	if g.Auto {
		secs := (game.TownWaitMs - g.AccMs + 999) / 1000
		next = sMuted.Render(fmt.Sprintf("Próxima expedição em %ds", max(0, secs)))
	}
	return lipgloss.JoinVertical(lipgloss.Center, art, "", sTitle.Render("Cidade"),
		"O herói descansa e se prepara.", next, sMuted.Render("2 para a Preparação"))
}

func (m Model) statsLine(w int) string {
	h := m.g.Hero
	s := fmt.Sprintf("Atq %d · Def %d · Vel %d · Poções %d · XP %d/%d",
		h.Atk(), h.Def(), h.Spd(), h.Potions, h.XP, game.XPToNext(h.Level))
	if e := m.g.Exp; e != nil {
		s += fmt.Sprintf(" · Mochila %d/%d", len(e.Bag), game.BagSize)
	}
	return sSection.Render(truncate(s, w))
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
	visible := l.Visible(g.Hero.Pos, p)
	if i := l.ItemAt(p); i >= 0 {
		f := l.Items[i]
		if !visible {
			return f.Glyph(), &sDim
		}
		return f.Glyph(), floorItemStyle(f)
	}
	t := l.At(p)
	if !visible {
		return t.Glyph(), &sDim
	}
	return t.Glyph(), tileStyle(t)
}

func (m Model) renderLog(w, h int) string {
	entries := m.g.Log
	start := max(0, len(entries)-h)
	var lines []string
	for _, e := range entries[start:] {
		text := e.Text
		if e.Count > 1 {
			text += fmt.Sprintf(" ×%d", e.Count)
		}
		lines = append(lines, logStyles[e.Kind].Render(truncate(text, w)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) heroPanel(w, maxH int) string {
	g := m.g
	h := g.Hero
	var b strings.Builder
	fmt.Fprintf(&b, "%s nível %d\n", sSection.Render("Herói"), h.Level)
	fmt.Fprintf(&b, "HP  %s\n", bar(h.HP, h.MaxHP(), w-4, hpStyle(h.HP, h.MaxHP())))
	fmt.Fprintf(&b, "XP  %s\n", bar(h.XP, game.XPToNext(h.Level), w-4, sXP))
	fmt.Fprintf(&b, "Ataque %d · Defesa %d\n", h.Atk(), h.Def())
	fmt.Fprintf(&b, "Velocidade %d (%.1f/s)\n", h.Spd(), game.StepsPerSec(h.Spd()))
	fmt.Fprintf(&b, "Poções %d · Ouro %d\n", h.Potions, g.Gold)
	b.WriteString("\n" + sSection.Render("Equipamento") + "\n")
	for s, it := range h.Equip {
		if it == nil {
			b.WriteString(sMuted.Render(truncate(fmt.Sprintf("%c sem %s", game.Slot(s).Glyph(), slotNames[s]), w)) + "\n")
			continue
		}
		b.WriteString(rarityStyle(it.Rarity).Render(truncate(fmt.Sprintf("%c %s (%s)", it.Slot.Glyph(), it.Title(), it.StatLabel()), w)) + "\n")
	}
	if e := g.Exp; e != nil {
		fmt.Fprintf(&b, "\n%s #%d · andar %d\n", sSection.Render("Expedição"), e.Num, e.Level.Depth)
		fmt.Fprintf(&b, "Ouro achado %d\nMochila %d/%d\n", e.Gold, len(e.Bag), game.BagSize)
		if e.Retreating {
			b.WriteString(sWarn.Render("Retirando-se") + "\n")
		}
	}
	bottom := fmt.Sprintf("\n%s\nAndar máx. %d · Chefes %d\nVoltas %d · Mortes %d\n",
		sSection.Render("Recordes"), g.MaxDepth, g.BossKills, g.Returns, g.Deaths)
	if e := g.Exp; e != nil {
		avail := maxH - strings.Count(b.String(), "\n") - strings.Count(bottom, "\n")
		bag := e.Bag
		if len(bag) > avail {
			bag = bag[:max(0, avail-1)]
		}
		for _, it := range bag {
			line := fmt.Sprintf("%c %s (%s) · %d ouro", it.Slot.Glyph(), it.Title(), it.StatLabel(), it.Value())
			b.WriteString(rarityStyle(it.Rarity).Render(truncate(line, w)) + "\n")
		}
		if rest := len(e.Bag) - len(bag); rest > 0 {
			b.WriteString(sMuted.Render(fmt.Sprintf("+%d itens", rest)) + "\n")
		}
	}
	return b.String() + bottom
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
		{sGold.Render("$"), "ouro"},
		{sPotion.Render("!"), "poção"},
		{sPlain.Render(")"), "arma"},
		{sPlain.Render("["), "armadura"},
		{sPlain.Render("]"), "botas"},
	}
	for _, k := range game.MonsterKinds {
		if k.MinDepth > m.g.MaxDepth {
			rows = append(rows, [2]string{sMuted.Render("?"), sMuted.Render("ainda não encontrado")})
			break
		}
		rows = append(rows, [2]string{monsterStyle(k).Render(string(k.Glyph)), k.Name})
	}
	for i, k := range game.BossKinds {
		if (i+1)*game.BossEvery > m.g.MaxDepth {
			rows = append(rows, [2]string{sMuted.Render("?"), sMuted.Render("chefe (a cada 10 andares)")})
			break
		}
		rows = append(rows, [2]string{monsterStyle(k).Render(string(k.Glyph)), k.Name + " (chefe)"})
	}
	perCol := max(1, h-4)
	var cols []string
	for i := 0; i < len(rows); i += perCol {
		var col strings.Builder
		for _, r := range rows[i:min(i+perCol, len(rows))] {
			col.WriteString("  " + r[0] + strings.Repeat(" ", max(1, 4-lipgloss.Width(r[0]))) + r[1] + "  \n")
		}
		cols = append(cols, col.String())
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols...))
	b.WriteString("\n" + sMuted.Render("? fecha · itens: branco comum, azul raro, roxo épico"))
	return b.String()
}

var slotNames = [...]string{"arma", "armadura", "botas"}

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
