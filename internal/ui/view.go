package ui

import (
	"fmt"
	"github.com/muesli/termenv"
	"math"
	"strings"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

const (
	minW, minH       = 40, 10
	sideW            = 28
	townListW        = 70
	townSceneMinW    = 30
	campfireFrameMs  = 150
	crtFrameMs       = 100
	crtFlickerChance = 0.02
	crtFlickerSalt   = 7
	crtFlickerDim    = 0.12
	crtDimStep       = 0.05
	crtDesat         = 0.15
	crtGrayDiff      = 24
	crtGlitchChance  = 0.01
	crtGlitchSalt    = 11
)

func (m Model) View() string {
	s := m.view()
	if m.crtOff || lipgloss.ColorProfile() == termenv.Ascii {
		return s
	}
	return crt(s, m.lastTick.UnixMilli()/crtFrameMs)
}

func (m Model) view() string {
	if m.w == 0 {
		return ""
	}
	if m.w < minW || m.h < minH {
		msg := sWarn.Render(lang.T("ui.too_small")) + "\n" +
			sMuted.Render(lang.T("ui.too_small_size", minW, minH, m.w, m.h))
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, msg)
	}
	if m.choosing {
		return m.viewChoose()
	}
	bodyH := m.h - 2
	if m.catching {
		f := float64(m.progress.Load()) / 1000
		msg := sTitle.Render(lang.T("ui.simulating")) + "\n\n" +
			bar(int(f*100), 100, min(40, m.w-4), sXP) + "\n" + sMuted.Render(fmt.Sprintf("%.0f%%", f*100))
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, msg)
	}
	var body string
	switch {
	case m.confirmDo != nil:
		box := sBox.Render(sTitle.Render(m.confirmText) + "\n\n" + sMuted.Render(lang.T("ui.confirm_hint")))
		body = lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, box)
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
	case m.tab == tabSettings:
		body = m.viewSettings(bodyH)
	case m.tab == tabHero:
		lines := m.heroLines()
		top := clamp(m.heroTop, 0, max(0, len(lines)-bodyH))
		body = strings.Join(lines[top:min(len(lines), top+bodyH)], "\n")
	default:
		body = sMuted.Render(lang.T("ui.soon"))
	}
	body = lipgloss.NewStyle().Width(m.w).Height(bodyH).MaxHeight(bodyH).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, m.viewHeader(), body, m.viewFooter())
}

func (m Model) viewHeader() string {
	right := m.statusLine()
	for _, tabs := range [][]string{{"1 " + lang.T("tab.main"), "2 " + lang.T("tab.prep"), "3 " + lang.T("tab.hero"), "4 " + lang.T("tab.settings")}, {"1", "2", "3", "4"}} {
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
	where := lang.T("status.town")
	if g.Exp != nil {
		where = lang.T("status.floor", g.Exp.Level.Depth)
	}
	return hp + sMuted.Render(lang.T("status.level", h.Level, where)) + sGold.Render(lang.T("status.gold", g.Gold))
}

func (m Model) viewFooter() string {
	keys := lang.T("footer.dungeon")
	if m.g.InTown() {
		keys = lang.T("footer.town")
	}
	if m.confirmDo != nil {
		return truncate(sMuted.Render(lang.T("footer.confirm")), m.w)
	}
	if m.saveErr != nil {
		return truncate(sWarn.Render(lang.T("footer.save_error", m.saveErr.Error())), m.w)
	}
	return truncate(sMuted.Render(keys), m.w)
}

func (m Model) viewDungeon(h int) string {
	g := m.g
	if g.Exp == nil {
		return m.viewTownScreen(h)
	}
	side := m.w >= game.MapW+sideW
	mapW := min(game.MapW, m.w)
	if side {
		mapW = min(game.MapW, m.w-sideW)
	}
	mapH, logH := game.MapH, h-game.MapH
	if logH < 3 {
		mapH, logH = h-1, 1
	}
	left := m.renderMap(mapW, mapH)
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

func (m Model) viewTownScreen(h int) string {
	listW := min(townListW, m.w)
	sceneW := m.w - listW
	if sceneW < townSceneMinW {
		return m.viewTownList(m.w, h)
	}
	sceneH, logH := game.MapH, h-game.MapH
	if logH < 3 {
		sceneH, logH = h-1, 1
	}
	left := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.Place(sceneW, sceneH, lipgloss.Center, lipgloss.Center, m.viewTown()),
		m.renderLog(sceneW, logH))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sPanel.Width(listW-1).Height(h).Render(m.viewTownList(listW-2, h)))
}

func (m Model) viewSummary() string {
	s := m.summary
	title := lang.T("summary.title", fmtDuration(s.Away))
	var b strings.Builder
	b.WriteString(sTitle.Render(title) + "\n\n")
	b.WriteString(lang.T("summary.expeditions", s.Expeditions, s.Returns, s.Deaths))
	b.WriteString(lang.T("summary.kills", s.Kills))
	if s.BossKills > 0 {
		b.WriteString(logStyles[game.LogEpic].Render(lang.T("summary.bosses", s.BossKills)) + "\n")
	}
	b.WriteString(sGold.Render(lang.T("summary.gold", s.Gold)) + "\n")
	b.WriteString(lang.T("summary.level", s.LevelFrom, s.LevelTo))
	b.WriteString(lang.T("summary.depth", s.DepthFrom, s.DepthTo))
	if s.Capped {
		b.WriteString("\n" + sMuted.Render(lang.T("summary.capped", fmtDuration(game.OfflineCap))) + "\n")
	}
	b.WriteString("\n" + sMuted.Render(lang.T("summary.any_key")))
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
	art := m.campfire()
	next := sTitle.Render(lang.T("town.press_e"))
	if g.Auto {
		secs := (game.TownWaitMs - g.AccMs + 999) / 1000
		next = sMuted.Render(lang.T("town.next_in", max(0, secs)))
	}
	return lipgloss.JoinVertical(lipgloss.Center, art, "", sTitle.Render(lang.T("town.title")),
		lang.T("town.rest"), next, sMuted.Render(lang.T("town.press_prep")))
}

var (
	fireColors  = []string{"226", "220", "214", "208", "202", "196"}
	lightColors = []string{"222", "179", "137", "94", "58"}
	sDark       = lipgloss.NewStyle().Foreground(lipgloss.Color("235"))
)

// flicker is a cheap hash of the animation frame, so the fire moves without touching the game RNG.
func flicker(frame, salt int64) float64 {
	x := uint64(frame)*2654435761 + uint64(salt)*40503
	x ^= x >> 13
	x *= 0x5bd1e995
	x ^= x >> 15
	return float64(x%1000) / 1000
}

// campfire draws a small dungeon room lit by the fire the hero rests beside.
func (m Model) campfire() string {
	const w, h = 23, 9
	fx, fy := w/2, h/2
	frame := m.lastTick.UnixMilli() / campfireFrameMs
	radius := 5.5 + flicker(frame, 0)*1.5
	var b strings.Builder
	for y := range h {
		for x := range w {
			ch := '.'
			switch {
			case y == 0 || y == h-1:
				ch = '-'
			case x == 0 || x == w-1:
				ch = '|'
			}
			dx, dy := float64(x-fx), float64(y-fy)*2
			d := math.Sqrt(dx*dx + dy*dy)
			switch {
			case x == fx && y == fy:
				c := fireColors[int(flicker(frame, 1)*float64(len(fireColors)))]
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true).Render("&"))
			case x == fx-1 && y == fy:
				b.WriteString(sHero.Render("@"))
			case d > radius:
				b.WriteString(sDark.Render(string(ch)))
			default:
				c := lightColors[min(len(lightColors)-1, int(d/radius*float64(len(lightColors))))]
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(string(ch)))
			}
		}
		if y < h-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) statsLine(w int) string {
	h := m.g.Hero
	s := lang.T("stats.line",
		h.Atk(), h.Def(), h.Spd(), h.Potions, h.XP, game.XPToNext(h.Level))
	if h.MaxMP() > 0 {
		s += lang.T("stats.mp", h.MP, h.MaxMP())
	}
	if sp := h.SpellAtk(); sp > 0 {
		s += lang.T("stats.spell", sp)
	}
	if e := m.g.Exp; e != nil {
		s += lang.T("stats.bag", len(e.Bag), m.g.BagCap())
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

func (m Model) heroLines() []string {
	w := min(48, m.w-4)
	s := m.heroPanel(w, math.MaxInt/2)
	if d := m.g.LastDeath; len(d) > 0 {
		s += "\n\n" + sSection.Render(lang.T("hero.last_death")) + "\n"
		for _, e := range d {
			s += logStyles[e.Kind].Width(w).Render(e.Text) + "\n"
		}
	}
	return strings.Split(s, "\n")
}

func (m Model) heroPanel(w, maxH int) string {
	g := m.g
	h := g.Hero
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", sSection.Render(h.Class.Name()), lang.T("hero.level", h.Level))
	wrap := lipgloss.NewStyle().Width(w)
	b.WriteString(wrap.Render(lang.T("hero.attrs", h.Attr(game.Str), h.Attr(game.Dex), h.Attr(game.Int), h.Attr(game.Con))) + "\n")
	fmt.Fprintf(&b, "HP  %s\n", bar(h.HP, h.MaxHP(), w-4, hpStyle(h.HP, h.MaxHP())))
	if h.MaxMP() > 0 {
		fmt.Fprintf(&b, "MP  %s\n", bar(h.MP, h.MaxMP(), w-4, sMP))
	}
	fmt.Fprintf(&b, "XP  %s\n", bar(h.XP, game.XPToNext(h.Level), w-4, sXP))
	b.WriteString(lang.T("hero.atk_def", h.Atk(), h.Def()))
	if sp := h.SpellAtk(); sp > 0 {
		b.WriteString(lang.T("hero.spell", sp))
	}
	b.WriteString(lang.T("hero.speed", h.Spd(), game.StepsPerSec(h.Spd())))
	b.WriteString(lang.T("hero.potions_gold", h.Potions, g.Gold))
	b.WriteString("\n" + sSection.Render(lang.T("hero.equipment")) + "\n")
	for s, it := range h.Equip {
		if it == nil {
			b.WriteString(sMuted.Render(truncate(lang.T("hero.no_slot", game.Slot(s).Glyph(), lang.T(slotNames[s])), w)) + "\n")
			continue
		}
		b.WriteString(rarityStyle(it.Rarity).Width(w).Render(fmt.Sprintf("%c %s (%s)", it.Slot.Glyph(), it.Title(), it.StatLabel())) + "\n")
	}
	var abilities []string
	for a := range game.NumAbilities {
		if h.Has(a) {
			abilities = append(abilities, a.Name())
		}
	}
	if len(abilities) > 0 {
		b.WriteString("\n" + sSection.Render(lang.T("hero.abilities")) + "\n")
		b.WriteString(lipgloss.NewStyle().Width(w).Render(strings.Join(abilities, ", ")) + "\n")
	}
	if e := g.Exp; e != nil {
		fmt.Fprintf(&b, "\n%s %s", sSection.Render(lang.T("hero.expedition")), lang.T("hero.exp_floor", e.Num, e.Level.Depth))
		b.WriteString(lang.T("hero.exp_loot", e.Gold, len(e.Bag), g.BagCap()))
		if e.Retreating {
			b.WriteString(sWarn.Render(lang.T("hero.retreating")) + "\n")
		}
	}
	bottom := "\n" + sSection.Render(lang.T("hero.records")) + "\n" +
		lang.T("hero.records_line", g.MaxDepth, g.BossKills, g.Returns, g.Deaths)
	if e := g.Exp; e != nil {
		avail := maxH - strings.Count(b.String(), "\n") - strings.Count(bottom, "\n")
		shown, used := 0, 0
		for i, it := range e.Bag {
			line := rarityStyle(it.Rarity).Width(w).Render(lang.T("hero.bag_item", it.Slot.Glyph(), it.Title(), it.StatLabel(), it.Value()))
			need := lipgloss.Height(line)
			if i < len(e.Bag)-1 {
				need++
			}
			if used+need > avail {
				break
			}
			b.WriteString(line + "\n")
			used += lipgloss.Height(line)
			shown++
		}
		if rest := len(e.Bag) - shown; rest > 0 {
			b.WriteString(sMuted.Render(lang.T("hero.more_items", rest)) + "\n")
		}
	}
	return b.String() + bottom
}

func (m Model) viewHelp(h int) string {
	var b strings.Builder
	b.WriteString(sSection.Render(lang.T("help.title")) + "\n\n")
	rows := [][2]string{
		{sHero.Render("@"), lang.T("help.hero")},
		{sFloor.Render("."), lang.T("help.floor")},
		{sFloor.Render("#"), lang.T("help.corridor")},
		{sDoor.Render("+"), lang.T("help.door")},
		{sWall.Render("- |"), lang.T("help.wall")},
		{sStairs.Render(">"), lang.T("help.stairs_down")},
		{sStairs.Render("<"), lang.T("help.stairs_up")},
		{sGold.Render("$"), lang.T("help.gold")},
		{sPotion.Render("!"), lang.T("help.potion")},
		{sPlain.Render(")"), lang.T("slot.weapon")},
		{sPlain.Render("["), lang.T("slot.armor")},
		{sPlain.Render("]"), lang.T("slot.boots")},
		{sPlain.Render("^"), lang.T("slot.helmet")},
		{sPlain.Render("}"), lang.T("slot.shield")},
	}
	for _, k := range game.MonsterKinds {
		if k.MinDepth > m.g.MaxDepth {
			rows = append(rows, [2]string{sMuted.Render("?"), sMuted.Render(lang.T("help.unknown"))})
			break
		}
		rows = append(rows, [2]string{monsterStyle(k).Render(string(k.Glyph)), lang.T(k.Name)})
	}
	for i, k := range game.BossKinds {
		if (i+1)*game.BossEvery > m.g.MaxDepth {
			rows = append(rows, [2]string{sMuted.Render("?"), sMuted.Render(lang.T("help.unknown_boss"))})
			break
		}
		rows = append(rows, [2]string{monsterStyle(k).Render(string(k.Glyph)), lang.T("help.boss", lang.T(k.Name))})
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
	b.WriteString("\n" + sMuted.Render(lang.T("help.footer")))
	return b.String()
}

var slotNames = [...]string{"slot.weapon", "slot.armor", "slot.boots", "slot.helmet", "slot.shield"}

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
