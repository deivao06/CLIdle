package ui

import (
	"fmt"

	"clidle/internal/game"
	"clidle/internal/lang"
)

func cycle[T ~int](v *T, d int, n T) { *v = (*v + T(d) + n) % n }

func (m *Model) openRule(i, back int) {
	m.edit, m.editBack, m.cursor = i+1, back, 0
}

func (m Model) editing() bool { return m.edit > 0 && m.edit <= len(m.g.Config.Gambits) }

func (m *Model) closeRule() {
	m.edit, m.cursor = 0, m.editBack
}

// tacticsRows lists the rules in the preparation screen; first is the index of the first row.
func (m Model) tacticsRows(first int) []prepRow {
	g := m.g
	sec := lang.T("tactics.section")
	var rows []prepRow
	for i, gb := range g.Config.Gambits {
		back := first + i
		rows = append(rows, prepRow{section: sec, text: fmt.Sprintf("%2d. %s", i+1, gb.Label()), style: sPlain,
			act: func(m *Model) { m.openRule(i, back) }, desc: gb.Do.Desc() + " " + lang.T("tactics.rule.desc")})
	}
	if len(g.Config.Gambits) < game.MaxGambits {
		back := first + len(rows)
		rows = append(rows, prepRow{section: sec, text: lang.T("tactics.add"), style: sTitle, desc: lang.T("tactics.add.desc"),
			act: func(m *Model) {
				if i := m.g.AddGambit(); i >= 0 {
					m.openRule(i, back)
				}
			}})
	}
	return append(rows, prepRow{section: sec, text: lang.T("tactics.reset"), style: sWarn, desc: lang.T("tactics.reset.desc"),
		confirm: lang.T("confirm.tactics_reset"), do: func(m *Model) { m.g.ResetGambits() }})
}

func nLabel(c game.Cond) string {
	switch c.Kind {
	case game.CondHP, game.CondMana:
		return fmt.Sprintf("%d%%", c.N)
	}
	return fmt.Sprint(c.N)
}

func condValue(c game.Cond) string {
	if c.Kind == game.CondNone {
		return lang.T("gambit.none")
	}
	return c.Label()
}

// ruleRows is the editor of rule i: each field cycles with ‹ ›, and only the fields the action uses show up.
func (m Model) ruleRows(i int) []prepRow {
	g := m.g
	gb := &g.Config.Gambits[i]
	class := g.Hero.Class
	sec := lang.T("tactics.edit", i+1)
	var rows []prepRow
	field := func(key, value string, adjust func(int)) {
		rows = append(rows, cfgRow(sec, key, value, adjust))
	}
	cond := func(key string, c *game.Cond, first game.CondKind) {
		field(key, condValue(*c), func(d int) {
			n := game.NumConds - first
			k := c.Kind - first
			for {
				cycle(&k, d, n)
				if kk := k + first; kk.Fits(class) && kk.Usable() && (kk != game.CondAtStairs || gb.Do == game.ActRetreat) {
					break
				}
			}
			*c = (k + first).DefaultCond()
		})
		if c.Kind.HasN() {
			field("tactics.field.op", c.Op.String(), func(d int) { cycle(&c.Op, d, game.NumOps) })
			field("tactics.field.n", nLabel(*c), c.StepN)
		}
	}
	cond("tactics.field.if", &gb.If[0], game.CondNone)
	cond("tactics.field.and", &gb.If[1], game.CondNone)
	field("tactics.field.do", gb.Do.Label(), func(d int) {
		for {
			cycle(&gb.Do, d, game.NumActions)
			if gb.Do.Fits(class) && gb.Do.Usable() {
				break
			}
		}
		for j, c := range gb.If {
			if c.Kind == game.CondAtStairs && gb.Do != game.ActRetreat {
				gb.If[j] = game.Cond{}
			}
		}
	})
	rows[len(rows)-1].desc = gb.Do.Desc()
	if gb.Do.HasTarget() {
		field("tactics.field.target", gb.Target.Label(), func(d int) { cycle(&gb.Target, d, game.NumTargets) })
	}
	if gb.Do.HasExit() {
		field("tactics.field.exit", gb.Exit.Label(), func(d int) {
			cycle(&gb.Exit, d, game.NumExits)
			if gb.Exit == game.ExitCond && gb.Until.Kind == game.CondNone {
				gb.Until = game.Cond{Kind: game.CondHP, Op: game.OpAtLeast, N: 80}
			}
			if gb.Exit == game.ExitTime && gb.Secs == 0 {
				gb.Secs = 10
			}
		})
		switch gb.Exit {
		case game.ExitCond:
			cond("tactics.field.until", &gb.Until, game.CondNone+1)
		case game.ExitTime:
			field("tactics.field.secs", lang.T("tactics.secs", gb.Secs), func(d int) { step(&gb.Secs, d, 5, 5, 600) })
		}
	}
	act := func(key string, f func(*Model)) {
		rows = append(rows, prepRow{section: sec, text: lang.T(key), style: sTitle, act: f, desc: lang.T(key + ".desc")})
	}
	act("tactics.up", func(m *Model) { m.edit = m.g.MoveGambit(i, -1) + 1 })
	act("tactics.down", func(m *Model) { m.edit = m.g.MoveGambit(i, 1) + 1 })
	rows = append(rows, prepRow{section: sec, text: lang.T("tactics.delete"), style: sWarn, desc: lang.T("tactics.delete.desc"),
		confirm: lang.T("confirm.tactics_delete", i+1), do: func(m *Model) { m.g.DeleteGambit(i); m.closeRule() }})
	act("tactics.done", (*Model).closeRule)
	return rows
}

func (m Model) viewRule(h int) string {
	i := m.edit - 1
	head := []string{sTitle.Render(m.g.Config.Gambits[i].Label()), sMuted.Render(m.keysHint("tactics.keys"))}
	return m.viewRows(head, m.ruleRows(i), m.w, h)
}
