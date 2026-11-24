package ui

import (
	"errors"
	"fmt"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

// allocDraft is a hero getting its attribute points: a recruit (index), one made by hand (-1 with a name and
// class), or a hero of the guild with free points (hero).
type allocDraft struct {
	recruit int
	name    string
	class   game.Class
	hero    *game.Game
	pts     [game.NumTraits]int
}

func (a *allocDraft) left(free int) int {
	for _, p := range a.pts {
		free -= p
	}
	return free
}

func (a *allocDraft) free() int {
	if a.hero != nil {
		return a.hero.Hero.FreePoints
	}
	return game.CreationPoints
}

var statKeys = [game.NumTraits]string{"stat.str", "stat.dex", "stat.int", "stat.con"}

func (m Model) hireBlock() string {
	switch err := m.g.Guild.CanHire(); {
	case errors.Is(err, game.ErrNoBed):
		return lang.T("recruit.no_bed", m.g.Guild.Beds())
	case errors.Is(err, game.ErrNoGold):
		return lang.T("recruit.no_gold", m.g.Guild.HirePrice())
	}
	return ""
}

func (m Model) recruitRows() []prepRow {
	gd := m.g.Guild
	price, block := gd.HirePrice(), m.hireBlock()
	sec := lang.T("recruit.section", len(gd.Heroes), gd.Beds())
	next := lang.T("recruit.next", (game.RecruitRefreshMs-gd.RecruitMs+59999)/60000)
	style := m.buyable(price)
	if block != "" {
		style = sMuted
	}
	var rows []prepRow
	for i, r := range gd.Recruits {
		row := prepRow{section: sec, text: lang.T("recruit.row", r.Name, r.Class.Name(), price), style: style,
			price: price, priceSt: m.goldFor(price), desc: r.Class.Desc() + " " + next}
		name := sTitle
		if block != "" {
			name = sMuted
		}
		row.card = func(w int) []string {
			who := name.Render(fmt.Sprintf("%-12s", r.Name)) + " " + sMuted.Render(r.Class.Name())
			return []string{ends(who, row.priceSt.Render(goldText(price)), min(w, lipgloss.Width(lang.T("recruit.make", price))))}
		}
		if block != "" {
			row.desc = block
		} else {
			row.act = func(m *Model) { m.alloc, m.cursor = &allocDraft{recruit: i, name: r.Name, class: r.Class}, 0 }
		}
		rows = append(rows, row)
	}
	own := prepRow{section: sec, text: lang.T("recruit.make", price), style: style, price: price, priceSt: m.goldFor(price),
		desc: lang.T("recruit.make.desc")}
	if block != "" {
		own.desc = block
	} else {
		own.act = func(m *Model) {
			m.alloc, m.naming, m.nameDraft = &allocDraft{recruit: -1}, true, nil
			Typing(true)
		}
	}
	reroll := prepRow{section: sec, text: lang.T("recruit.reroll", game.RecruitReroll), style: m.buyable(game.RecruitReroll),
		price: game.RecruitReroll, priceSt: m.goldFor(game.RecruitReroll), desc: lang.T("recruit.reroll.desc") + " " + next,
		enter: func() { gd.RerollRecruits() }}
	return append(rows, own, reroll)
}

// allocRows spends the points one attribute at a time, showing what each attribute becomes.
func (m Model) allocRows() []prepRow {
	a := m.alloc
	left := a.left(a.free())
	sec := lang.T("alloc.section", a.name, a.class.Name(), left)
	preview := game.NewHero(a.class)
	if a.hero != nil {
		cp := *a.hero.Hero
		preview = &cp
	}
	for t, p := range a.pts {
		preview.Points[t] += p
	}
	var rows []prepRow
	for t := range game.NumTraits {
		value := fmt.Sprintf("%d  → %s %d", a.pts[t], lang.T(statKeys[t]), preview.Attr(t))
		if t == game.Con {
			value = fmt.Sprintf("%d  → HP %d", a.pts[t], preview.MaxHP())
		}
		adjust := func(d int) {
			if n := a.pts[t] + d; n >= 0 && a.left(a.free())-d >= 0 {
				a.pts[t] = n
			}
		}
		rows = append(rows, prepRow{section: sec, label: t.Name(), value: value, text: fmt.Sprintf("%-24s %s", t.Name(), value),
			style: sPlain, adjust: adjust, enter: func() { adjust(1) }, desc: lang.T("alloc.desc", t.Effect())})
	}
	done := prepRow{section: sec, style: sMuted, desc: lang.T("alloc.left", left)}
	switch {
	case a.hero != nil:
		done.text = lang.T("alloc.confirm")
	default:
		price := m.g.Guild.HirePrice()
		done.text, done.price, done.priceSt = lang.T("alloc.hire", price), price, m.goldFor(price)
	}
	if left == 0 {
		done.style, done.desc, done.act = sTitle, "", (*Model).finishAlloc
		if b := m.hireBlock(); a.hero == nil && b != "" {
			done.style, done.desc, done.act = sMuted, b, nil
		}
	}
	return append(rows, done)
}

func (m *Model) finishAlloc() {
	a := m.alloc
	if a.hero != nil {
		if a.hero.Hero.Spend(a.pts) == nil {
			m.alloc, m.cursor = nil, 0
			m.save(m.lastTick)
		}
		return
	}
	if _, err := m.g.Guild.Hire(a.recruit, a.name, a.class, a.pts); err != nil {
		return
	}
	m.alloc = nil
	m.guild, m.tab = true, tabGuild
	m.cursor = len(m.g.Guild.Heroes) - 1
	m.save(m.lastTick)
}
