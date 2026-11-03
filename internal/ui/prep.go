package ui

import (
	"fmt"
	"slices"
	"strings"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

type prepRow struct {
	section string
	text    string
	style   lipgloss.Style
	adjust  func(delta int)
	enter   func()
	do      func(*Model)
	sell    func()
	confirm string
}

func onOff(b bool) string {
	if b {
		return lang.T("prep.yes")
	}
	return lang.T("prep.no")
}

func step(v *int, delta, by, lo, hi int) { *v = clamp(*v+delta*by, lo, hi) }

func cfgRow(section, label, value string, adjust func(int)) prepRow {
	return prepRow{
		section: section,
		text:    fmt.Sprintf("%-22s %s", lang.T(label), value),
		style:   sPlain,
		adjust:  adjust,
		enter:   func() { adjust(1) },
	}
}

func (m Model) rows() []prepRow {
	switch m.tab {
	case tabSettings:
		return m.settingsRows()
	case tabDungeon:
		return m.townRows()
	}
	return m.prepRows()
}

func (m Model) settingsRows() []prepRow {
	g := m.g
	sec := lang.T("prep.section.game")
	return []prepRow{
		cfgRow(sec, "prep.bell", onOff(g.Bell), func(int) { g.Bell = !g.Bell }),
		cfgRow(sec, "prep.language", lang.T("lang.name"), func(int) {
			g.Lang = lang.Next(lang.Current())
			lang.Set(g.Lang)
		}),
		{section: lang.T("settings.section.save"), text: lang.T("settings.reset"), style: sWarn,
			confirm: lang.T("confirm.reset"), do: (*Model).resetSave},
	}
}

func (m Model) prepRows() []prepRow {
	g := m.g
	c := &g.Config
	var rows []prepRow
	nextSection := lang.T("prep.section.next")
	cfg := func(label, value string, adjust func(int)) {
		rows = append(rows, cfgRow(nextSection, label, value, adjust))
	}
	auto := lang.T("prep.off")
	if g.Auto {
		auto = lang.T("prep.on")
	}
	cfg("prep.auto", auto, func(int) { g.ToggleAuto() })
	limit := lang.T("prep.no_limit")
	if c.DeathLimit > 0 {
		limit = lang.T("prep.deaths_value", g.AutoDeaths, c.DeathLimit)
	}
	cfg("prep.death_limit", limit, func(d int) { step(&c.DeathLimit, d, 1, 0, 99) })
	mode := lang.T("prep.mode.explore")
	if c.Mode == game.DescendFast {
		mode = lang.T("prep.mode.descend")
	}
	cfg("prep.mode", mode, func(int) { c.Mode = 1 - c.Mode })
	cfg("prep.potions", lang.T("prep.potions_value", c.PotionsToBring, game.PotionPrice),
		func(d int) { step(&c.PotionsToBring, d, 1, 0, game.MaxPotions) })
	cfg("prep.potion_at", fmt.Sprintf("%d%%", c.PotionAtPct), func(d int) { step(&c.PotionAtPct, d, 5, 5, 90) })
	retreat := lang.T("prep.never")
	if c.RetreatHPPct > 0 {
		retreat = fmt.Sprintf("%d%%", c.RetreatHPPct)
	}
	cfg("prep.retreat_hp", retreat, func(d int) { step(&c.RetreatHPPct, d, 5, 0, 90) })
	cfg("prep.auto_equip", onOff(!c.ManualEquip), func(int) { c.ManualEquip = !c.ManualEquip })
	cfg("prep.min_rarity", lang.T("prep.min_rarity_value", c.MinRarity.Name()), func(d int) {
		c.MinRarity = (c.MinRarity + game.Rarity(d) + game.NumRarities) % game.NumRarities
	})
	cfg("prep.any_class", onOff(c.AnyClass), func(int) { c.AnyClass = !c.AnyClass })
	cfg("prep.retreat_bag", onOff(c.RetreatBagFull), func(int) { c.RetreatBagFull = !c.RetreatBagFull })
	last := lang.T("prep.no_limit")
	if c.LastFloor > 0 {
		last = lang.T("status.floor", c.LastFloor)
	}
	cfg("prep.last_floor", last, func(d int) { step(&c.LastFloor, d, 1, 0, 999) })
	floors := g.ShortcutList()
	at := max(0, slices.Index(floors, c.StartFloor))
	cfg("prep.start_floor", lang.T("prep.start_floor_value", floors[at], len(floors)-1), func(d int) {
		c.StartFloor = floors[(at+d+len(floors))%len(floors)]
	})
	return rows
}

func (m Model) townRows() []prepRow {
	g := m.g
	rows := []prepRow{{text: lang.T("prep.depart_now"), style: sTitle, enter: g.Depart, confirm: lang.T("confirm.depart")}}

	for s := range game.NumSlots {
		it := g.Hero.Equip[s]
		if it == nil {
			rows = append(rows, prepRow{section: lang.T("prep.section.smith"), text: lang.T("hero.no_slot", s.Glyph(), lang.T(slotNames[s])), style: sMuted})
			continue
		}
		next := *it
		next.Upgrade++
		cost := g.UpgradeCost(s)
		text := lang.T("prep.smith_row", s.Glyph(), it.Label()+fmt.Sprintf(" +%d", it.Upgrade), next.StatLabel(), cost)
		rows = append(rows, prepRow{section: lang.T("prep.section.smith"), text: text, style: m.priced(sPlain, cost),
			enter: func() { g.Upgrade(s) }})
	}

	for t := range game.NumTraits {
		lvl := g.Hero.Training[t]
		cost := game.TrainCost(t, lvl)
		text := lang.T("prep.train_row", t.Name(), lvl, t.Effect(), cost)
		rows = append(rows, prepRow{section: lang.T("prep.section.training"), text: text, style: m.priced(sPlain, cost),
			enter: func() { g.Train(t) }})
	}

	for i, it := range g.Shop {
		price := game.ShopPrice(it)
		text := lang.T("prep.shop_row", it.Slot.Glyph(), it.Title(), it.StatLabel(), price, m.compare(it))
		st := m.priced(rarityStyle(it.Rarity), price)
		if !g.Hero.Class.CanUse(it) {
			text += lang.T("prep.other_class")
			st = sMuted
		}
		rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: text, style: st,
			enter: func() { g.BuyShop(i) }})
	}
	if len(g.Shop) == 0 {
		rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: lang.T("prep.sold_out"), style: sMuted})
	}
	if size, cost, ok := g.NextBag(); ok {
		text := lang.T("prep.bag_row", size-game.BagSlotsPerLevel, size, cost)
		rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: text, style: m.priced(sPlain, cost),
			enter: g.UpgradeBag})
	} else {
		rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: lang.T("prep.bag_max", size-game.BagSlotsPerLevel), style: sMuted})
	}

	if g.Config.ManualEquip || len(g.Stash) > 0 {
		chest := lang.T("prep.section.chest", len(g.Stash), game.StashSize)
		for s, it := range g.Hero.Equip {
			if it == nil {
				continue
			}
			rows = append(rows, prepRow{section: chest, text: lang.T("prep.equipped_row", it.Slot.Glyph(), it.Title(), it.StatLabel()),
				style: rarityStyle(it.Rarity), enter: func() { g.Unequip(game.Slot(s)) }})
		}
		for i, it := range g.Stash {
			text := lang.T("prep.chest_row", it.Slot.Glyph(), it.Title(), it.StatLabel(), it.Value(), m.compare(it))
			st := rarityStyle(it.Rarity)
			if !g.Hero.Class.CanUse(it) {
				text += lang.T("prep.other_class")
				st = sMuted
			}
			rows = append(rows, prepRow{section: chest, text: text, style: st,
				enter: func() { g.EquipStash(i) }, sell: func() { g.SellStash(i) }})
		}
	}
	return rows
}

func (m Model) priced(st lipgloss.Style, cost int) lipgloss.Style {
	if cost > m.g.Gold {
		return sMuted
	}
	return st
}

// compare shows how a shop item changes the stat of the currently equipped one.
func (m Model) compare(it *game.Item) string {
	cur := m.g.Hero.Equip[it.Slot]
	diff := it.Bonus()
	if cur != nil {
		diff -= cur.Bonus()
	}
	if diff > 0 {
		return fmt.Sprintf("  ▲%d", diff)
	}
	return fmt.Sprintf("  ▼%d", -diff)
}

func (m Model) viewPrep(h int) string {
	g := m.g
	rows := m.prepRows()
	head := []string{sGold.Render(lang.T("prep.gold", g.Gold))}
	if !g.InTown() {
		head = append(head, sMuted.Render(lang.T("prep.away_config")))
	}
	head = append(head, sMuted.Render(lang.T("prep.keys")))
	return m.viewRows(head, rows, m.w, h)
}

func (m Model) viewTownList(w, h int) string {
	head := []string{sGold.Render(lang.T("prep.gold", m.g.Gold)), sMuted.Render(lang.T("town.keys"))}
	return m.viewRows(head, m.townRows(), w, h)
}

func (m Model) viewSettings(h int) string {
	return m.viewRows([]string{sMuted.Render(lang.T("settings.keys"))}, m.settingsRows(), m.w, h)
}

func (m Model) viewRows(head []string, rows []prepRow, w, h int) string {
	for i := range head {
		head[i] = truncate(head[i], w)
	}
	var lines []string
	cursorLine := 0
	section := ""
	for i, r := range rows {
		if r.section != section {
			section = r.section
			lines = append(lines, "", sSection.Render(section))
		}
		text := truncate(r.text, w-4)
		if i == m.cursor {
			cursorLine = len(lines)
			lines = append(lines, sTabOn.Render("›")+" "+r.style.Render(text))
		} else {
			lines = append(lines, "  "+r.style.Render(text))
		}
	}
	avail := max(1, h-len(head))
	top := clamp(cursorLine-avail/2, 0, max(0, len(lines)-avail))
	lines = lines[top:min(len(lines), top+avail)]
	return strings.Join(append(head, lines...), "\n")
}
