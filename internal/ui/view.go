package ui

import (
	"fmt"
	"github.com/muesli/termenv"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"
	"clidle/internal/update"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const (
	minW, minH       = 40, 10
	sideW            = 28
	townListW        = 70
	gapMinH          = 20
	townSceneMinW    = 30
	townListMinH     = 10
	lampRadius       = 7.0
	lampFlicker      = 1.2
	campfireFrameMs  = 150
	crtFrameMs       = 100
	crtFlickerChance = 0.02
	crtFlickerSalt   = 7
	crtFlickerDim    = 0.12
	crtDimStep       = 0.05
	crtDesat         = 0.15
	crtTint          = 0.18
	crtGreenLift     = 22.0
	crtDefaultFg     = 204
	crtGrayDiff      = 24
	crtGlitchChance  = 0.01
	crtGlitchSalt    = 11
)

func (m Model) View() string {
	s := zone.Scan(m.view())
	if !m.crt || lipgloss.ColorProfile() == termenv.Ascii {
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
	bodyH := m.bodyH()
	if m.catching {
		f := float64(m.progress.Load()) / 1000
		msg := sTitle.Render(lang.T("ui.simulating")) + "\n\n" +
			bar(int(f*100), 100, min(40, m.w-4), sXP) + "\n" + sMuted.Render(fmt.Sprintf("%.0f%%", f*100))
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, msg)
	}
	var body string
	switch {
	case m.confirmDo != nil:
		title := sTitle.Width(min(lipgloss.Width(m.confirmText), m.w-12)).Render(m.confirmText)
		box := sBox.Render(title + "\n\n" + m.confirmButtons())
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
	case m.tab == tabNerd:
		body = m.viewNerd(bodyH)
	case m.tab == tabHero:
		lines := m.heroLines()
		top := clamp(m.heroTop, 0, max(0, len(lines)-bodyH))
		shown := lines[top:min(len(lines), top+bodyH)]
		if top+bodyH < len(lines) {
			shown[len(shown)-1] = sMuted.Render(lang.T("hero.more"))
		}
		body = strings.Join(shown, "\n")
	default:
		body = sMuted.Render(lang.T("ui.soon"))
	}
	body = lipgloss.NewStyle().Width(m.w).Height(bodyH).MaxHeight(bodyH).Render(body)
	if m.narrow() {
		status := truncate(m.updateNotice()+m.statusLine(), m.w)
		if m.h >= gapMinH {
			status += "\n"
		}
		return lipgloss.JoinVertical(lipgloss.Left, status, body, m.viewBar())
	}
	header := m.viewHeader()
	if m.h-2 > bodyH {
		header += "\n"
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, m.viewFooter())
}

// bodyH leaves a blank line under the tabs when the terminal is tall enough.
func (m Model) bodyH() int {
	if m.narrow() {
		if m.h >= gapMinH {
			return m.h - 2 - m.barH()
		}
		return m.h - 1 - m.barH()
	}
	if m.h >= gapMinH {
		return m.h - 3
	}
	return m.h - 2
}

func (m Model) confirmButtons() string {
	btn := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 3)
	yes := btn.BorderForeground(lipgloss.Color("214")).Render(sTitle.Render(lang.T("ui.yes")))
	no := btn.BorderForeground(lipgloss.Color("240")).Render(sMuted.Render(lang.T("ui.no")))
	return lipgloss.JoinHorizontal(lipgloss.Top, zone.Mark("btn:y", yes), "   ", zone.Mark("btn:n", no))
}

func (m Model) viewHeader() string {
	right := m.updateNotice() + m.statusLine()
	full := []string{"1 " + lang.T("tab.main"), "2 " + lang.T("tab.prep"), "3 " + lang.T("tab.hero"), "4 " + lang.T("tab.settings")}
	short := []string{"1", "2", "3", "4"}
	if m.g.Nerd {
		full, short = append(full, "5 "+lang.T("tab.nerd")), append(short, "5")
	}
	active := slices.Clone(short)
	if !m.help {
		active[m.tab] = full[m.tab]
	}
	for _, tabs := range [][]string{full, active, short} {
		var parts []string
		for i, t := range tabs {
			st := sTabOff
			if i == m.tab && !m.help {
				st = sTabOn
			}
			parts = append(parts, zone.Mark(fmt.Sprintf("tab:%d", i), st.Render(" "+t+" ")))
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
	auto := lang.T("prep.off")
	if m.g.Auto {
		auto = lang.T("prep.on")
	}
	tabs := 4
	if m.g.Nerd {
		tabs = 5
	}
	keys := lang.T("footer.dungeon", tabs, auto)
	if m.g.InTown() {
		keys = lang.T("footer.town", tabs, auto)
	}
	if m.confirmDo != nil {
		return truncate(sMuted.Render(lang.T("footer.confirm")), m.w)
	}
	if m.saveErr != nil {
		return truncate(sWarn.Render(lang.T("footer.save_error", m.saveErr.Error())), m.w)
	}
	if m.updateErr != "" && time.Since(m.updateErrAt) < updateErrShow {
		return truncate(sWarn.Render(m.updateErr), m.w)
	}
	for lipgloss.Width(keys) > m.w && strings.Contains(keys, " · ") {
		_, keys, _ = strings.Cut(keys, " · ")
	}
	if gap := m.w - lipgloss.Width(keys) - lipgloss.Width(update.Version); gap >= 2 {
		keys += strings.Repeat(" ", gap) + update.Version
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
	mapH, logH := game.MapH, h-game.MapH-1
	if logH < 3 {
		mapH, logH = h-2, 1
	}
	left := m.renderMap(mapW, mapH)
	if !side && logH >= 3 {
		stats := m.statsLine(mapW)
		left = lipgloss.JoinVertical(lipgloss.Left, left, stats)
		logH -= lipgloss.Height(stats)
	}
	doingStyle := sDoing
	if g.Exp.Retreating {
		doingStyle = sWarn
	}
	doing := doingStyle.Render(truncate("▸ "+g.Doing(), mapW))
	if logH >= 3 {
		doing += "\n"
		logH--
	}
	left = lipgloss.JoinVertical(lipgloss.Left, left, doing, m.renderLog(mapW, logH))
	if !side {
		return left
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sPanel.Width(m.w-mapW-1).Height(h).Render(m.heroPanel(min(48, m.w-mapW-3), h, false)))
}

func (m Model) viewTownScreen(h int) string {
	listW := min(townListW, m.w)
	sceneW := m.w - listW
	if sceneW < townSceneMinW {
		art := m.campfire()
		listH := h - lipgloss.Height(art) - 1
		if listH < townListMinH {
			return m.viewTownList(m.w, h)
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.PlaceHorizontal(m.w, lipgloss.Center, art), "", m.viewTownList(m.w, listH))
	}
	sceneH, logH := game.MapH, h-game.MapH
	if logH < 3 {
		sceneH, logH = h-1, 1
	}
	left := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.Place(sceneW, sceneH, lipgloss.Center, lipgloss.Center, m.viewTown(sceneW-2)),
		m.renderLog(sceneW, logH))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sPanel.Width(listW-1).Height(h).Render(m.viewTownList(listW-2, h)))
}

func (m Model) viewSummary() string {
	s := m.summary
	title := lang.T("summary.title", fmtDuration(s.Away))
	var b strings.Builder
	b.WriteString(sTitle.Render(title) + "\n\n")
	b.WriteString(hangLine(strings.TrimSuffix(lang.T("summary.expeditions", s.Expeditions, s.Returns, s.Deaths), "\n"), m.w) + "\n")
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

// hangLine fits a "Label  value  detail" line to w: it breaks at the double spaces first and keeps
// the following lines under the value.
func hangLine(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	i := strings.Index(s, "  ")
	if i < 0 {
		return lipgloss.NewStyle().Width(w).Render(s)
	}
	value := strings.TrimLeft(s[i:], " ")
	label := s[:len(s)-len(value)]
	pad := lipgloss.Width(label)
	if w-pad < 10 {
		return lipgloss.NewStyle().Width(w).Render(s)
	}
	var groups []string
	for _, seg := range strings.Split(value, "  ") {
		if n := len(groups); n > 0 && lipgloss.Width(groups[n-1]+"  "+seg) <= w-pad {
			groups[n-1] += "  " + seg
		} else {
			groups = append(groups, seg)
		}
	}
	var lines []string
	for _, g := range groups {
		for _, l := range strings.Split(lipgloss.NewStyle().Width(w-pad).Render(g), "\n") {
			lines = append(lines, strings.Repeat(" ", pad)+strings.TrimRight(l, " "))
		}
	}
	lines[0] = label + lines[0][pad:]
	return strings.Join(lines, "\n")
}

func fmtDuration(d time.Duration) string {
	h, mins := int(d.Hours()), int(d.Minutes())%60
	if h == 0 {
		return fmt.Sprintf("%dmin", mins)
	}
	return fmt.Sprintf("%dh%02d", h, mins)
}

func (m Model) viewTown(w int) string {
	g := m.g
	art := m.campfire()
	next := sTitle.Render(lang.T("town.press_e"))
	if g.Auto {
		secs := (game.TownWaitMs - g.AccMs + 999) / 1000
		next = sMuted.Render(lang.T("town.next_in", max(0, secs)))
	}
	text := lipgloss.NewStyle().Width(w).Align(lipgloss.Center)
	return lipgloss.JoinVertical(lipgloss.Center, art, "", sTitle.Render(lang.T("town.title")),
		text.Render(lang.T("town.rest")), text.Render(next), text.Render(sMuted.Render(lang.T("town.press_prep"))))
}

var (
	fireColors  = []string{"226", "220", "214", "208", "202", "196"}
	lightColors = []string{"222", "179", "137", "94", "58"}
	lampStyles  = func() []lipgloss.Style {
		var s []lipgloss.Style
		for _, c := range lightColors {
			s = append(s, lipgloss.NewStyle().Foreground(lipgloss.Color(c)))
		}
		return s
	}()
	sDark = lipgloss.NewStyle().Foreground(lipgloss.Color("235"))
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
	if m.narrow() {
		return sStats.Width(w).Render(s)
	}
	return sStats.Render(truncate(s, w))
}

func (m Model) renderMap(w, h int) string {
	g := m.g
	l := g.Exp.Level
	hp := g.Hero.Pos
	cx := clamp(hp.X-w/2, 0, game.MapW-w)
	cy := clamp(hp.Y-h/2, 0, game.MapH-h)
	overlay, flash, shake := m.fxLayers()
	cx -= shake
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
			p := game.Pos{X: x, Y: y}
			ch, st := m.cell(l, p)
			if o, ok := overlay[p]; ok {
				ch, st = o.r, o.st
			} else if f, ok := flash[p]; ok && ch != ' ' {
				st = f.st
			}
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
	if lit := m.lampLight(l, p, visible); lit >= 0 && t != game.StairsDown && t != game.StairsUp {
		return t.Glyph(), &lampStyles[lit]
	}
	if !visible {
		return t.Glyph(), &sDim
	}
	return t.Glyph(), tileStyle(t)
}

// lampLight returns the warmth level of a tile lit by the hero's lamp, or -1 when it is dark.
// It only lights what the hero sees or known corridors, so light never leaks into another room.
func (m Model) lampLight(l *game.Level, p game.Pos, visible bool) int {
	if m.g.Effects == game.EffectsOff || !visible && l.RoomAt(p) >= 0 {
		return -1
	}
	h := m.g.Hero.Pos
	dx, dy := float64(p.X-h.X), float64(p.Y-h.Y)*2
	radius := lampRadius
	if m.g.Effects == game.EffectsFull {
		radius += flicker(m.lastTick.UnixMilli()/campfireFrameMs, 2) * lampFlicker
	}
	d := math.Sqrt(dx*dx + dy*dy)
	if d > radius {
		return -1
	}
	return min(len(lampStyles)-1, int(d/radius*float64(len(lampStyles))))
}

func (m Model) renderLog(w, h int) string {
	entries := m.g.Log
	start := max(0, len(entries)-h)
	var lines []string
	for i := len(entries) - 1; i >= start && len(lines) < h; i-- {
		e := entries[i]
		text := e.Text
		if e.Count > 1 {
			text += fmt.Sprintf(" ×%d", e.Count)
		}
		wrapped := strings.Split(logStyles[e.Kind].Width(w).Render(text), "\n")
		if len(lines)+len(wrapped) > h {
			wrapped = []string{logStyles[e.Kind].Render(truncate(text, w))}
		}
		lines = append(wrapped, lines...)
	}
	return strings.Join(lines, "\n")
}

const (
	heroColW   = 48
	heroColGap = 4
)

// heroLines is the hero tab: one column on narrow screens, two side by side when both fit.
func (m Model) heroLines() []string {
	w := min(heroColW, m.w-4)
	left := m.heroPanel(w, math.MaxInt/2, true)
	if rw := min(56, m.w-w-heroColGap); rw >= 32 {
		right := strings.TrimPrefix(m.heroSide(rw), "\n")
		s := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(w+heroColGap).Render(left), right)
		return strings.Split(s, "\n")
	}
	return strings.Split(left+m.heroSide(w), "\n")
}

// heroSide holds what the hero tab adds beyond the panel: abilities and the last death.
func (m Model) heroSide(w int) string {
	s := m.heroAbilities(w)
	if d := m.g.LastDeath; len(d) > 0 {
		s += "\n" + sTitle.Render(lang.T("hero.last_death")) + "\n"
		if sc := m.g.DeathScene; sc != nil {
			s = strings.TrimSuffix(s, "\n") + sMuted.Render(lang.T("hero.death_floor", sc.Depth)) + "\n" + deathScene(sc, w) + "\n"
		}
		for _, e := range d {
			s += logStyles[e.Kind].Width(w).Render(e.Text) + "\n"
		}
	}
	return s
}

// abilityLabel is an ability's name with its unlock level and mana cost.
func abilityLabel(a game.Ability) string {
	info := game.AbilityInfo[a]
	if info.Mana > 0 {
		return lang.T("ability.label_mana", a.Name(), info.Level, info.Mana)
	}
	return lang.T("ability.label", a.Name(), info.Level)
}

// heroAbilities lists every ability of the hero's class, unlocked or not, with what it does.
// deathScene draws the saved map around the hero's corpse, dropping rows the hero never saw.
func deathScene(sc *game.DeathScene, w int) string {
	var rows []string
	for i, glyphs := range sc.Glyphs {
		marks := sc.Marks[i]
		if strings.TrimSpace(marks) == "" {
			continue
		}
		var b strings.Builder
		for j, r := range []rune(glyphs) {
			if j >= w || j >= len(marks) {
				break
			}
			b.WriteString(markStyle(marks[j], r).Render(string(r)))
		}
		rows = append(rows, b.String())
	}
	return strings.Join(rows, "\n")
}

func markStyle(mark byte, r rune) lipgloss.Style {
	switch mark {
	case game.MarkFloor:
		return sFloor
	case game.MarkWall:
		return sWall
	case game.MarkDoor:
		return sDoor
	case game.MarkStairs:
		return sStairs
	case game.MarkDim:
		return sDim
	case game.MarkHero:
		return sWarn
	case game.MarkMonster, game.MarkKiller:
		st := *monsterStyle(kindByGlyph(r))
		return st.Underline(mark == game.MarkKiller)
	case game.MarkGold:
		return sGold
	case game.MarkPotion:
		return sPotion
	}
	if mark >= game.MarkItem(game.Common) && mark <= game.MarkItem(game.Legendary) {
		return rarityStyles[mark-game.MarkItem(game.Common)]
	}
	return sPlain
}

func kindByGlyph(r rune) game.MonsterKind {
	for _, k := range append(slices.Clone(game.MonsterKinds), game.BossKinds...) {
		if k.Glyph == r {
			return k
		}
	}
	return game.MonsterKind{Glyph: r}
}

func (m Model) heroAbilities(w int) string {
	h := m.g.Hero
	s := "\n" + sTitle.Render(lang.T("hero.abilities")) + "\n"
	desc := lipgloss.NewStyle().Width(w - 2)
	for _, a := range game.ClassAbilities(h.Class) {
		mark, name, text := "✓ ", sValue.Bold(true), sSoft
		if !h.Has(a) {
			mark, name, text = "· ", sMuted, sMuted
		}
		s += name.Render(mark+abilityLabel(a)) + "\n"
		for _, l := range strings.Split(desc.Render(a.Desc()), "\n") {
			s += "  " + text.Render(l) + "\n"
		}
	}
	return s
}

func (m Model) heroAbilityNames(w int) string {
	var abilities []string
	for a := range game.NumAbilities {
		if m.g.Hero.Has(a) {
			abilities = append(abilities, a.Name())
		}
	}
	if len(abilities) == 0 {
		return ""
	}
	return "\n" + sSection.Render(lang.T("hero.abilities")) + "\n" + lipgloss.NewStyle().Width(w).Render(strings.Join(abilities, ", ")) + "\n"
}

func pct(f float64) int { return int(f*100 + 0.5) }

var verb = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)

// stat renders a text with its words muted and its numbers bright.
func stat(key string, args ...any) string { return statIn([]lipgloss.Style{sValue}, key, args...) }

// statIn colors the i-th number with styles[i], or with the last style when there are fewer styles.
func statIn(styles []lipgloss.Style, key string, args ...any) string {
	tpl := lang.T(key)
	var b strings.Builder
	muted := func(s string) {
		for i, line := range strings.Split(s, "\n") {
			if i > 0 {
				b.WriteString("\n")
			}
			if line != "" {
				b.WriteString(sMuted.Render(line))
			}
		}
	}
	last, n := 0, 0
	for _, loc := range verb.FindAllStringIndex(tpl, -1) {
		muted(tpl[last:loc[0]])
		st := styles[min(max(n-1, 0), len(styles)-1)]
		if v := tpl[loc[0]:loc[1]]; v == "%%" {
			b.WriteString(st.Render("%"))
		} else {
			st = styles[min(n, len(styles)-1)]
			b.WriteString(st.Render(fmt.Sprintf(v, args[n])))
			n++
		}
		last = loc[1]
	}
	muted(tpl[last:])
	return b.String()
}

// heroCombat lists the numbers that decide a fight, from the same formulas the game uses.
func (m Model) heroCombat() string {
	h := m.g.Hero
	s := "\n" + sTitle.Render(lang.T("hero.combat")) + "\n" + stat("hero.atk", h.Atk()) + stat("hero.def", h.Def())
	if sp := h.SpellAtk(); sp > 0 {
		s += statIn([]lipgloss.Style{sMP}, "hero.spell", sp)
	}
	s += stat("hero.rates", pct(h.MissChance()), pct(h.CritChance()))
	if d := h.DodgeChance(); d > 0 {
		s += stat("hero.dodge", pct(d))
	}
	s += stat("hero.speed", h.Spd(), game.StepsPerSec(h.Spd()))
	if r := h.ManaPerStep(); r > 0 {
		s += statIn([]lipgloss.Style{sMP}, "hero.mana_regen", r)
	}
	return s + statIn([]lipgloss.Style{sPotion, sHPGood}, "hero.potions_heal", h.Potions, h.PotionHeal())
}

func (m Model) heroPanel(w, maxH int, tab bool) string {
	g := m.g
	h := g.Hero
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", sSection.Render(h.Class.Name()), lang.T("hero.level", h.Level))
	wrap := lipgloss.NewStyle().Width(w)
	b.WriteString(wrap.Render(stat("hero.attrs", h.Attr(game.Str), h.Attr(game.Dex), h.Attr(game.Int), h.Attr(game.Con))) + "\n")
	fmt.Fprintf(&b, "HP  %s\n", numBar(h.HP, h.MaxHP(), w-4, hpStyle(h.HP, h.MaxHP())))
	if h.MaxMP() > 0 {
		fmt.Fprintf(&b, "MP  %s\n", numBar(h.MP, h.MaxMP(), w-4, sMP))
	}
	fmt.Fprintf(&b, "XP  %s\n", numBar(h.XP, game.XPToNext(h.Level), w-4, sXP))
	if tab {
		b.WriteString(m.heroCombat())
	} else {
		b.WriteString(stat("hero.atk_def", h.Atk(), h.Def()))
		if sp := h.SpellAtk(); sp > 0 {
			b.WriteString(statIn([]lipgloss.Style{sMP}, "hero.spell", sp))
		}
		b.WriteString(stat("hero.speed", h.Spd(), game.StepsPerSec(h.Spd())))
		b.WriteString(statIn([]lipgloss.Style{sPotion, sGold}, "hero.potions_gold", h.Potions, g.Gold))
	}
	b.WriteString("\n" + sTitle.Render(lang.T("hero.equipment")) + "\n")
	for s, it := range h.Equip {
		if it == nil {
			b.WriteString(sMuted.Render(truncate(lang.T("hero.no_slot", game.Slot(s).Glyph(), lang.T(slotNames[s])), w)) + "\n")
			continue
		}
		b.WriteString(rarityStyle(it.Rarity).Width(w).Render(fmt.Sprintf("%c %s (%s)", it.Slot.Glyph(), it.Title(), it.StatLabel())) + "\n")
	}
	if !tab {
		b.WriteString(m.heroAbilityNames(w))
	}
	if e := g.Exp; e != nil {
		fmt.Fprintf(&b, "\n%s %s", sTitle.Render(lang.T("hero.expedition")), stat("hero.exp_floor", e.Num, e.Level.Depth))
		b.WriteString(statIn([]lipgloss.Style{sGold, sValue}, "hero.exp_loot", e.Gold, len(e.Bag), g.BagCap()))
		if e.Retreating {
			b.WriteString(sWarn.Render(lang.T("hero.retreating")) + "\n")
		}
	}
	records := "hero.records_line"
	if tab {
		records = "hero.records_tab"
	}
	bottom := "\n" + sTitle.Render(lang.T("hero.records")) + "\n" +
		stat(records, g.MaxDepth, g.BossKills, g.Returns, g.Deaths)
	if e := g.Exp; e != nil {
		avail := maxH - strings.Count(b.String(), "\n") - strings.Count(bottom, "\n")
		shown, used := 0, 0
		for i, it := range e.Bag {
			line := rarityStyle(it.Rarity).Width(w).Render(lang.T("hero.bag_item", it.Slot.Glyph(), it.Title(), it.StatLabel(), it.SellPrice()))
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

var helpPages = []string{"help.page.symbols", "help.page.how", "help.page.tactics", "help.page.attrs", "help.page.classes"}

func (m Model) viewHelp(h int) string {
	var tabs []string
	for i, p := range helpPages {
		st := sTabOff
		if i == m.helpPage {
			st = sTabOn
		}
		name := strings.ReplaceAll(lang.T(p), " ", "\u00a0")
		tabs = append(tabs, zone.Mark(fmt.Sprintf("help:%d", i), st.Render(" "+name+" ")))
	}
	head := lipgloss.NewStyle().Width(m.w).Render(strings.Join(tabs, " ")) + "\n\n"
	foot := "\n" + sMuted.Render(truncate(lang.T("help.footer"), m.w))
	bodyH := max(1, h-lipgloss.Height(head))
	if m.helpPage == 0 {
		return head + m.viewSymbols(bodyH) + "\n" + sMuted.Render(truncate(lang.T("help.footer_symbols"), m.w))
	}
	w := min(m.w, 80)
	var lines []string
	for _, p := range strings.Split(helpText(helpPages[m.helpPage]), "\n") {
		if title, ok := strings.CutPrefix(p, "# "); ok {
			lines = append(lines, sSection.Render(truncate(title, w)))
			continue
		}
		lines = append(lines, hangWrap(p, w)...)
	}
	top := clamp(m.helpTop, 0, max(0, len(lines)-bodyH))
	lines = lines[top:min(len(lines), top+bodyH)]
	for len(lines) < bodyH {
		lines = append(lines, "")
	}
	return head + strings.Join(lines, "\n") + foot
}

// helpText is a help page; the classes page is built from the ability table so it never goes stale.
func helpText(page string) string {
	if page != "help.page.classes" {
		return lang.T(page + ".text")
	}
	s := lang.T("help.classes.intro")
	for c := range game.NumClasses {
		main := lang.T([...]string{"stat.str", "stat.dex", "stat.int", "stat.con"}[c.MainAttr()])
		desc := []rune(c.Desc() + lang.T("class.main", main) + ".")
		s += "\n\n# " + c.Name() + "\n" + strings.ToUpper(string(desc[:1])) + string(desc[1:])
		for _, a := range game.ClassAbilities(c) {
			s += "\n  " + abilityLabel(a) + "\n    " + a.Desc()
		}
	}
	return s
}

// hangWrap wraps an indented line so continuation lines align with its text column ("  FOR  text...").
func hangWrap(p string, w int) []string {
	body := strings.TrimLeft(p, " ")
	cut := len(p) - len(body)
	if cut > 0 {
		if i := strings.Index(body, "  "); i > 0 {
			rest := strings.TrimLeft(body[i:], " ")
			cut = len(p) - len(rest)
		}
	}
	pad := lipgloss.Width(p[:cut])
	wrapped := strings.Split(lipgloss.NewStyle().Width(max(10, w-pad)).Render(p[cut:]), "\n")
	for i := range wrapped {
		if i == 0 {
			wrapped[i] = p[:cut] + wrapped[i]
		} else {
			wrapped[i] = strings.Repeat(" ", pad) + wrapped[i]
		}
	}
	return wrapped
}

func (m Model) viewSymbols(h int) string {
	var b strings.Builder
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
	perCol := max(1, h-2)
	var cols []string
	for i := 0; i < len(rows); i += perCol {
		var col strings.Builder
		for _, r := range rows[i:min(i+perCol, len(rows))] {
			col.WriteString("  " + r[0] + strings.Repeat(" ", max(1, 4-lipgloss.Width(r[0]))) + r[1] + "  \n")
		}
		cols = append(cols, col.String())
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols...))
	rarity := truncate(lang.T("help.rarity"), m.w)
	if m.narrow() {
		rarity = lipgloss.NewStyle().Width(m.w).Render(lang.T("help.rarity"))
	}
	b.WriteString("\n" + sMuted.Render(rarity))
	return b.String()
}

var slotNames = [...]string{"slot.weapon", "slot.armor", "slot.boots", "slot.helmet", "slot.shield"}

func numBar(v, maxV, w int, st lipgloss.Style) string {
	num := fmt.Sprintf(" %9s", fmt.Sprintf("%d/%d", v, maxV))
	return bar(v, maxV, w-len(num), st) + sMuted.Render(num)
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
