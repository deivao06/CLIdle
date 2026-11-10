package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"clidle/internal/game"
	"clidle/internal/lang"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const (
	narrowW      = 60
	barBoxMinH   = 20
	maxRowLines  = 4
	maxDescLines = 4
)

func init() { zone.NewGlobal() }

// narrow is the phone layout: status on top, actions and tabs at the bottom within thumb reach.
func (m Model) narrow() bool { return m.w < narrowW }

func (m Model) barH() int {
	if m.h >= barBoxMinH {
		return 5
	}
	return 2
}

type barAction struct{ key, label string }

func (m Model) barActions() []barAction {
	g := m.g
	if m.help {
		return []barAction{{"?", lang.T("bar.close")}}
	}
	var a []barAction
	if m.tab == tabPrep && m.editing() {
		a = append(a, barAction{"esc", lang.T("bar.back")})
	}
	switch {
	case g.InTown():
		a = append(a, barAction{"e", lang.T("bar.depart")})
	case g.Exp != nil && !g.Exp.Retreating:
		a = append(a, barAction{"r", lang.T("bar.retreat")})
	}
	auto := lang.T("prep.off")
	if g.Auto {
		auto = lang.T("prep.on")
	}
	a = append(a, barAction{"a", lang.T("bar.auto", auto)})
	if m.tab == tabDungeon && g.InTown() {
		if rows := m.townRows(); m.cursor < len(rows) && rows[m.cursor].sell != nil {
			a = append(a, barAction{"x", lang.T("bar.sell")})
		}
	}
	if m.release != nil {
		a = append(a, barAction{"u", lang.T("bar.update")})
	}
	return append(a, barAction{"?", lang.T("bar.help")})
}

// spread gives each cell its text width plus an even share of the leftover space.
func spread(texts []string, w int) []int {
	widths := make([]int, len(texts))
	used := 0
	for i, t := range texts {
		widths[i] = lipgloss.Width(t)
		used += widths[i]
	}
	left := max(0, w-used)
	for i := range widths {
		widths[i] += left / len(widths)
		if i < left%len(widths) {
			widths[i]++
		}
	}
	return widths
}

func (m Model) viewActions() string {
	switch {
	case m.confirmDo != nil:
		return truncate(sMuted.Render(lang.T("footer.confirm")), m.w)
	case m.saveErr != nil:
		return truncate(sWarn.Render(lang.T("footer.save_error", m.saveErr.Error())), m.w)
	case m.updateErr != "" && time.Since(m.updateErrAt) < updateErrShow:
		return truncate(sWarn.Render(m.updateErr), m.w)
	}
	acts := m.barActions()
	keyed := func(withKeys bool) []string {
		texts := make([]string, len(acts))
		for i, a := range acts {
			texts[i] = a.label
			if withKeys {
				texts[i] = a.key + " " + a.label
			}
		}
		return texts
	}
	texts, withKeys := keyed(true), true
	if lipgloss.Width(strings.Join(texts, "")) > m.w {
		texts, withKeys = keyed(false), false
	}
	widths := spread(texts, m.w)
	var b strings.Builder
	for i, a := range acts {
		text := a.label
		if withKeys {
			text = sTitle.Render(a.key) + " " + a.label
		}
		cell := lipgloss.PlaceHorizontal(widths[i], lipgloss.Center, truncate(text, widths[i]))
		b.WriteString(zone.Mark("key:"+a.key, cell))
	}
	return b.String()
}

var tabKeys = []string{"tab.main", "tab.prep", "tab.hero", "tab.settings", "tab.nerd"}

func (m Model) tabCount() int {
	if m.g.Nerd {
		return 5
	}
	return 4
}

func (m Model) viewTabs(boxed bool) string {
	n := m.tabCount()
	widths := spread(make([]string, n), m.w-n-1)
	var cells []string
	for i := range n {
		cw := widths[i]
		label := lang.T(tabKeys[i])
		if lipgloss.Width(label) > cw-2 {
			label = lang.T(tabKeys[i] + "_short")
		}
		label = lipgloss.PlaceHorizontal(cw, lipgloss.Center, truncate(label, cw))
		if i == m.tab && !m.help {
			label = sTabOn.Render(label)
		} else {
			label = sTabOff.Render(label)
		}
		last := i == n-1
		edge := func(first, mid, end string) string {
			s := mid
			if i == 0 {
				s = first
			}
			if last {
				return s + end
			}
			return s
		}
		var cell string
		if boxed {
			cell = sBar.Render(edge("╭", "┬", "")+strings.Repeat("─", cw)+edge("", "", "╮")) + "\n" +
				sBar.Render(edge("│", "│", "")) + label + sBar.Render(edge("", "", "│")) + "\n" +
				sBar.Render(edge("╰", "┴", "")+strings.Repeat("─", cw)+edge("", "", "╯"))
		} else {
			cell = sBar.Render(edge(" ", "│", "")) + label + sBar.Render(edge("", "", " "))
		}
		cells = append(cells, zone.Mark(fmt.Sprintf("tab:%d", i), cell))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (m Model) viewBar() string {
	if m.barH() < 5 {
		return m.viewActions() + "\n" + m.viewTabs(false)
	}
	return sBar.Render(strings.Repeat("─", m.w)) + "\n" + m.viewActions() + "\n" + m.viewTabs(true)
}

var columns = regexp.MustCompile(` {2,}`)

// rowText fits a list row to w columns: cut on wide screens; on narrow ones a row with columns puts
// its name and price on the first line and the middle column below, and anything else wraps.
func (m Model) rowText(text string, w int) []string {
	if lipgloss.Width(text) <= w {
		return []string{text}
	}
	if !m.narrow() {
		return []string{truncate(text, w)}
	}
	if cols := columns.Split(strings.TrimSpace(text), -1); len(cols) >= 3 {
		head, tail := cols[0], strings.Join(cols[2:], "  ")
		if gap := w - lipgloss.Width(head) - lipgloss.Width(tail); gap >= 1 && lipgloss.Width(cols[1]) <= w-4 {
			return []string{head + strings.Repeat(" ", gap) + tail, "    " + cols[1]}
		}
	}
	lines := strings.Split(lipgloss.NewStyle().Width(w-2).Render(strings.Join(strings.Fields(text), " ")), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
		if i > 0 {
			lines[i] = "  " + lines[i]
		}
	}
	return lines[:min(len(lines), maxRowLines)]
}

func (m Model) keysHint(key string) string {
	if m.narrow() {
		return lang.T("touch.hint")
	}
	return lang.T(key)
}

func keyMsg(k string) tea.KeyMsg {
	types := map[string]tea.KeyType{"enter": tea.KeyEnter, "esc": tea.KeyEsc, "left": tea.KeyLeft,
		"right": tea.KeyRight, "up": tea.KeyUp, "down": tea.KeyDown}
	if t, ok := types[k]; ok {
		return tea.KeyMsg{Type: t}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func hit(id string, msg tea.MouseMsg) bool { return zone.Get(id).InBounds(msg) }

// mouse turns a tap or click into the key the player would press for it; the first tap on a row selects it, the second uses it.
func (m Model) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.catching {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.Update(keyMsg("up"))
	case tea.MouseButtonWheelDown:
		return m.Update(keyMsg("down"))
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if m.summary != nil {
		return m.Update(keyMsg("enter"))
	}
	if m.choosing {
		for c := range int(game.NumClasses) {
			for j := range maxDescLines + 1 {
				if hit(fmt.Sprintf("class:%d.%d", c, j), msg) {
					if c == m.cursor {
						return m.Update(keyMsg("enter"))
					}
					m.cursor = c
					return m, nil
				}
			}
		}
		return m, nil
	}
	if m.confirmDo != nil {
		if hit("btn:y", msg) {
			return m.Update(keyMsg("y"))
		}
		return m.Update(keyMsg("n"))
	}
	for i := range m.tabCount() {
		if hit(fmt.Sprintf("tab:%d", i), msg) {
			return m.Update(keyMsg(fmt.Sprint(i + 1)))
		}
	}
	for _, a := range []string{"esc", "e", "r", "a", "x", "u", "?"} {
		if hit("key:"+a, msg) {
			return m.Update(keyMsg(a))
		}
	}
	if m.help {
		for i := range helpPages {
			if hit(fmt.Sprintf("help:%d", i), msg) {
				m.helpPage, m.helpTop = i, 0
				return m, m.newsCmd()
			}
		}
		return m, nil
	}
	if z := zone.Get("val"); z.InBounds(msg) {
		if x, _ := z.Pos(msg); x < (z.EndX-z.StartX+1)/2 {
			return m.Update(keyMsg("left"))
		}
		return m.Update(keyMsg("right"))
	}
	if !m.listShown() {
		return m, nil
	}
	for i := range m.rows() {
		for j := range maxRowLines {
			if hit(fmt.Sprintf("row:%d.%d", i, j), msg) {
				if i == m.cursor {
					return m.Update(keyMsg("enter"))
				}
				m.cursor = i
				return m, nil
			}
		}
	}
	return m, nil
}

func (m Model) listShown() bool {
	return m.tab == tabPrep || m.tab == tabSettings || m.tab == tabNerd || m.tab == tabDungeon && m.g.InTown()
}
