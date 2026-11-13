package ui

import (
	"fmt"
	"slices"
	"strings"

	"clidle/internal/game"
	"clidle/internal/lang"
	"clidle/internal/update"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type prepRow struct {
	section string
	text    string
	label   string
	value   string
	style   lipgloss.Style
	adjust  func(delta int)
	enter   func()
	act     func(*Model)
	do      func(*Model)
	sell    func()
	sellAsk string
	confirm string
	desc    string
}

func onOff(b bool) string {
	if b {
		return lang.T("prep.yes")
	}
	return lang.T("prep.no")
}

func crtLabel(on bool) string {
	if on {
		return lang.T("prep.crt_on")
	}
	return lang.T("prep.crt_off")
}

func step(v *int, delta, by, lo, hi int) { *v = clamp(*v+delta*by, lo, hi) }

func cfgRow(section, label, value string, adjust func(int)) prepRow {
	return prepRow{
		section: section,
		text:    fmt.Sprintf("%-24s %s", lang.T(label), value),
		label:   lang.T(label),
		value:   value,
		style:   sPlain,
		adjust:  adjust,
		enter:   func() { adjust(1) },
		desc:    lang.T(label + ".desc"),
	}
}

func (m Model) rows() []prepRow {
	switch m.tab {
	case tabSettings:
		return m.settingsRows()
	case tabDungeon:
		return m.townRows()
	case tabNerd:
		return m.nerdRows()
	}
	if m.editing() {
		return m.ruleRows(m.edit - 1)
	}
	return m.prepRows()
}

func (m Model) settingsRows() []prepRow {
	g := m.g
	sec := lang.T("prep.section.game")
	return []prepRow{
		cfgRow(sec, "prep.bell", onOff(g.Bell), func(int) { g.Bell = !g.Bell }),
		cfgRow(sec, "prep.crt", crtLabel(g.CRT), func(int) { g.CRT = !g.CRT }),
		cfgRow(sec, "prep.nerd", crtLabel(g.Nerd), func(int) { g.Nerd = !g.Nerd }),
		cfgRow(sec, "prep.effects", lang.T([...]string{"prep.effects_full", "prep.effects_subtle", "prep.effects_off"}[g.Effects]), func(d int) {
			g.Effects = (g.Effects + game.Effects(d) + game.NumEffectLevels) % game.NumEffectLevels
		}),
		cfgRow(sec, "prep.language", lang.T("lang.name"), func(int) {
			g.Lang = lang.Next(lang.Current())
			lang.Set(g.Lang)
		}),
		{section: lang.T("settings.section.save"), text: lang.T("settings.reset"), style: sWarn,
			confirm: lang.T("confirm.reset"), do: (*Model).resetSave, desc: lang.T("settings.reset.desc")},
	}
}

func (m Model) prepRows() []prepRow {
	g := m.g
	c := &g.Config
	var rows []prepRow
	section := lang.T("prep.section.next")
	cfg := func(label, value string, adjust func(int)) {
		rows = append(rows, cfgRow(section, label, value, adjust))
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
	floors := g.ShortcutList()
	at := max(0, slices.Index(floors, c.StartFloor))
	cfg("prep.start_floor", lang.T("prep.start_floor_value", g.Floor(floors[at]), len(floors)-1), func(d int) {
		c.StartFloor = floors[(at+d+len(floors))%len(floors)]
	})
	section = lang.T("prep.section.potions")
	cfg("prep.potions", lang.T("prep.potions_value", c.PotionsToBring),
		func(d int) { step(&c.PotionsToBring, d, 1, 0, game.MaxPotions) })
	section = lang.T("prep.section.loot")
	cfg("prep.auto_equip", onOff(!c.ManualEquip), func(int) { c.ManualEquip = !c.ManualEquip })
	cfg("prep.auto_sell", onOff(!c.KeepLoot), func(int) { c.KeepLoot = !c.KeepLoot })
	cfg("prep.min_rarity", lang.T("prep.min_rarity_value", c.MinRarity.Name()), func(d int) {
		c.MinRarity = (c.MinRarity + game.Rarity(d) + game.NumRarities) % game.NumRarities
	})
	cfg("prep.any_class", onOff(c.AnyClass), func(int) { c.AnyClass = !c.AnyClass })
	return append(rows, m.tacticsRows(len(rows))...)
}

func (m Model) townRows() []prepRow {
	g := m.g
	rows := []prepRow{{text: lang.T("prep.depart_now"), style: sTitle, enter: g.Depart, confirm: lang.T("confirm.depart"), desc: lang.T("town.depart.desc")}}

	for s := range game.NumSlots {
		it := g.Hero.Equip[s]
		if it == nil {
			rows = append(rows, prepRow{section: lang.T("prep.section.smith"), text: lang.T("hero.no_slot", s.Glyph(), lang.T(slotNames[s])), style: sMuted, desc: lang.T("town.no_slot.desc")})
			continue
		}
		if it.Upgrade >= game.UpgradeMax {
			text := lang.T("prep.smith_max", s.Glyph(), it.Label()+fmt.Sprintf(" +%d", it.Upgrade)+it.ILvl())
			rows = append(rows, prepRow{section: lang.T("prep.section.smith"), text: text, style: rarityStyle(it.Rarity), desc: lang.T("town.smith_max.desc")})
			continue
		}
		next := *it
		next.Upgrade++
		next.Forged += it.UpgradeStep()
		cost := g.UpgradeCost(s)
		text := lang.T("prep.smith_row", s.Glyph(), it.Label()+fmt.Sprintf(" +%d", it.Upgrade)+it.ILvl(), next.StatLabel(), cost)
		rows = append(rows, prepRow{section: lang.T("prep.section.smith"), text: text, style: m.priced(rarityStyle(it.Rarity), cost),
			enter: func() { g.Upgrade(s) }, desc: lang.T("town.smith.desc")})
	}

	for t := range game.NumTraits {
		lvl := g.Hero.Training[t]
		cost := game.TrainCost(lvl)
		text := lang.T("prep.train_row", t.Name(), lvl, t.Effect(), cost)
		rows = append(rows, prepRow{section: lang.T("prep.section.training"), text: text, style: m.buyable(cost),
			enter: func() { g.Train(t) }, desc: m.trainDesc(t)})
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
			enter: func() { g.BuyShop(i) }, desc: m.itemDesc(it)})
	}
	if len(g.Shop) == 0 {
		rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: lang.T("prep.sold_out"), style: sMuted})
	}
	potion := m.buyable(game.PotionPrice)
	if g.Hero.Potions >= game.MaxPotions {
		potion = sMuted
	}
	rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: lang.T("prep.potion_row", g.Hero.Potions, game.MaxPotions, game.PotionPrice),
		style: potion, enter: g.BuyPotion, desc: lang.T("town.potion.desc", game.MaxPotions)})
	if size, cost, ok := g.NextBag(); ok {
		text := lang.T("prep.bag_row", size-game.BagSlotsPerLevel, size, cost)
		rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: text, style: m.buyable(cost),
			enter: g.UpgradeBag, desc: lang.T("town.bag.desc")})
	} else {
		rows = append(rows, prepRow{section: lang.T("prep.section.shop"), text: lang.T("prep.bag_max", size-game.BagSlotsPerLevel), style: sMuted})
	}

	if g.Dungeon > 0 {
		row := prepRow{section: lang.T("prep.section.prestige"), text: lang.T("prep.prestige_row", g.PrestigeGain()), style: logStyles[game.LogEpic],
			confirm: lang.T("confirm.prestige", g.PrestigeGain()), do: func(m *Model) { m.choosing, m.prestige, m.cursor = true, true, int(m.g.Hero.Class) },
			desc: lang.T("town.prestige.desc", g.Dungeon, g.PrestigeGain())}
		if !g.CanPrestige() {
			row.text += lang.T("prep.prestige_level", game.PrestigeLevel)
			row.style, row.confirm, row.do = sMuted, "", nil
		}
		rows = append(rows, row)
	}

	chest := lang.T("prep.section.chest", len(g.Stash))
	for i, it := range g.Stash {
		text := lang.T("prep.chest_row", it.Slot.Glyph(), it.Title(), it.StatLabel(), it.SellPrice(), m.compare(it))
		st := rarityStyle(it.Rarity)
		if !g.Hero.Class.CanUse(it) {
			text += lang.T("prep.other_class")
			st = sMuted
		}
		row := prepRow{section: chest, text: text, style: st,
			enter: func() { g.EquipStash(i) }, sell: func() { g.SellStash(i) }, desc: m.itemDesc(it) + " " + lang.T("chest.desc", it.SellPrice())}
		if it.Rarity == game.Legendary {
			row.sellAsk = lang.T("confirm.sell_legendary", it.Title(), it.SellPrice())
		}
		rows = append(rows, row)
	}
	return rows
}

// trainDesc explains a trait and previews what one more training level changes.
func (m Model) trainDesc(t game.Trait) string {
	before := *m.g.Hero
	after := before
	after.Training[t]++
	stats := []struct {
		key string
		f   func(*game.Hero) int
	}{
		{"stat.atk", (*game.Hero).Atk}, {"stat.spell", (*game.Hero).SpellAtk}, {"stat.spd", (*game.Hero).Spd},
		{"stat.hp", (*game.Hero).MaxHP}, {"stat.mp", (*game.Hero).MaxMP},
	}
	var diffs []string
	for _, s := range stats {
		if a, b := s.f(&before), s.f(&after); a != b {
			diffs = append(diffs, fmt.Sprintf("%s %d → %d", lang.T(s.key), a, b))
		}
	}
	effect := lang.T("train.no_effect")
	if len(diffs) > 0 {
		effect = strings.Join(diffs, " · ")
	}
	return lang.T([...]string{"train.str.desc", "train.dex.desc", "train.int.desc", "train.con.desc"}[t]) + " " + lang.T("train.effect", effect)
}

// itemDesc says how an item compares with what the hero has equipped in that slot.
func (m Model) itemDesc(it *game.Item) string {
	rarity := lang.T("item.rarity.desc", it.Rarity.Name()) + " "
	if !m.g.Hero.Class.CanUse(it) {
		return rarity + lang.T("item.other_class.desc")
	}
	if cur := m.g.Hero.Equip[it.Slot]; cur != nil && cur.ID != it.ID {
		return rarity + lang.T("item.vs.desc", cur.Title(), cur.StatLabel())
	}
	return rarity + lang.T("item.vs_none.desc")
}

func (m Model) buyable(cost int) lipgloss.Style {
	if cost > m.g.Gold {
		return sMuted
	}
	return sBuy
}

func (m Model) priced(st lipgloss.Style, cost int) lipgloss.Style {
	if cost > m.g.Gold {
		return st.Faint(true)
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
	if m.editing() {
		return m.viewRule(h)
	}
	g := m.g
	rows := m.prepRows()
	head := []string{sGold.Render(lang.T("prep.gold", g.Gold))}
	if !g.InTown() {
		head = append(head, sMuted.Render(lang.T("prep.away_config")))
	}
	head = append(head, sMuted.Render(m.keysHint("prep.keys")))
	return m.viewRows(head, rows, m.w, h)
}

func (m Model) viewTownList(w, h int) string {
	head := []string{sGold.Render(lang.T("prep.gold", m.g.Gold)), sMuted.Render(m.keysHint("town.keys"))}
	return m.viewRows(head, m.townRows(), w, h)
}

func (m Model) viewSettings(h int) string {
	head := []string{sMuted.Render(m.keysHint("settings.keys"))}
	if m.narrow() {
		head = append(head, sMuted.Render(lang.T("settings.version", update.Version)))
	}
	return m.viewRows(head, m.settingsRows(), m.w, h)
}

// rowLines styles a row; the selected adjustable row shows ‹ value › so a tap on either side changes it.
func (m Model) rowLines(r prepRow, selected bool, w int) []string {
	if !selected || r.adjust == nil || r.value == "" {
		lines := m.rowText(r.text, w)
		for i := range lines {
			lines[i] = r.style.Render(lines[i])
		}
		return lines
	}
	val := zone.Mark("val", sTitle.Render("‹ ")+r.style.Render(r.value)+sTitle.Render(" ›"))
	label := fmt.Sprintf("%-24s ", r.label)
	if lipgloss.Width(label)+lipgloss.Width(val) <= w {
		return []string{r.style.Render(label) + val}
	}
	return []string{r.style.Render(truncate(r.label, w)), "  " + val}
}

func (m Model) viewRows(head []string, rows []prepRow, w, h int) string {
	var fit []string
	for _, l := range head {
		if m.narrow() {
			fit = append(fit, strings.Split(lipgloss.NewStyle().Width(w).Render(l), "\n")...)
		} else {
			fit = append(fit, truncate(l, w))
		}
	}
	head = fit
	var foot []string
	if m.cursor < len(rows) && rows[m.cursor].desc != "" {
		foot = append([]string{""}, strings.Split(sMuted.Width(w).Render(rows[m.cursor].desc), "\n")...)
		if len(head)+len(foot) >= h {
			foot = nil
		}
	}
	var lines []string
	cursorLine := 0
	section := ""
	for i, r := range rows {
		if r.section != section {
			section = r.section
			lines = append(lines, "", sSection.Render(section))
		}
		prefix := "  "
		if i == m.cursor {
			cursorLine = len(lines)
			prefix = sTabOn.Render("›") + " "
		}
		for j, text := range m.rowLines(r, i == m.cursor, w-4) {
			line := "  " + text
			if j == 0 {
				line = prefix + text
			}
			line += strings.Repeat(" ", max(0, w-lipgloss.Width(line)))
			lines = append(lines, zone.Mark(fmt.Sprintf("row:%d.%d", i, j), line))
		}
	}
	avail := max(1, h-len(head)-len(foot))
	top := cursorLine - avail/2
	if m.listFree {
		top = m.listTop
	}
	top = clamp(top, 0, max(0, len(lines)-avail))
	*m.shown = scrolled{top, max(0, len(lines)-avail)}
	lines = lines[top:min(len(lines), top+avail)]
	for len(foot) > 0 && len(lines) < avail {
		lines = append(lines, "")
	}
	return strings.Join(append(append(head, lines...), foot...), "\n")
}
