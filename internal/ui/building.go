package ui

import (
	"fmt"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

// buildingValue is what a building gives at a level, in the words of its effect.
func buildingValue(b game.Building, lvl int) string { return capValue(b, lvl, 1<<30) }

// capValue is what the building gives at a level, with the shop's items never past the floor deep.
func capValue(b game.Building, lvl, deep int) string {
	switch b {
	case game.Lodging:
		return lang.T("building.value.beds", game.GuildBeds+lvl)
	case game.MissionTable:
		return lang.T("building.value.slots", game.MissionSlots+lvl)
	case game.Market:
		v := lang.T("building.value.ilvl", min(game.MarketILvl(lvl), deep)) + " · " + lang.T("building.value.items", game.ShopSize+game.MarketItemsAt(lvl))
		if r := game.MarketRarity(lvl); r > 0 {
			v += " · " + lang.T("building.value.rarity", r)
		}
		return v
	case game.Apothecary:
		v := lang.T("building.value.heal", game.PotionHealPct+game.ApothecaryHeal(lvl))
		if d := game.BuildingDiscountAt(b, lvl); d > 0 {
			v += " · " + lang.T("building.value.discount", d)
		}
		return v
	}
	return lang.T("building.value.discount", game.BuildingDiscountAt(b, lvl))
}

func (m Model) buildingRows() []prepRow {
	gd := m.g.Guild
	sec := lang.T("buildings.section")
	var rows []prepRow
	for b := range game.NumBuildings {
		lvl := gd.Level(b)
		name := fmt.Sprintf("%-16s  %-8s", b.Name(), levelLabel(b, lvl))
		cost, ok := gd.NextBuilding(b)
		row := prepRow{section: sec, desc: b.Desc()}
		switch {
		case b == game.Market && gd.MarketLocked():
			row.text, row.style = lang.T("buildings.row_locked", name, m.valueAt(b, lvl)), sMuted
		case !ok:
			row.text, row.style = lang.T("buildings.row_max", name, m.valueAt(b, lvl)), sMuted
		default:
			row.text, row.style = lang.T("buildings.row", name, m.step(b, lvl), game.Num(cost)), m.buyable(cost)
			row.price, row.priceSt, row.enter = cost, m.goldFor(cost), func() { gd.Build(b) }
		}
		if m.narrow() || lipgloss.Width(row.text) > m.w-4 {
			row.card = m.buildingCard(b, row)
		}
		rows = append(rows, row)
	}
	return rows
}

// buildingCard is a building on a phone: its name and price, then its level and what the next level gives.
func (m Model) buildingCard(b game.Building, r prepRow) func(w int) []string {
	lvl := m.g.Guild.Level(b)
	value, right := m.valueAt(b, lvl), sMuted.Render(lang.T("buildings.top"))
	switch {
	case r.price > 0:
		value = m.step(b, lvl)
		right = r.priceSt.Render(goldText(r.price))
	case b == game.Market:
		right = sMuted.Render(lang.T("buildings.locked"))
	}
	detail := levelLabel(b, lvl) + " · " + value
	return func(w int) []string {
		return []string{ends(r.style.Render(b.Name()), right, w), "  " + sMuted.Render(ellipsis(detail, w-2))}
	}
}

func (m Model) valueAt(b game.Building, lvl int) string { return capValue(b, lvl, m.g.Guild.Deepest()) }

func levelLabel(b game.Building, lvl int) string {
	if b == game.Market {
		return lang.T("buildings.level_open", lvl)
	}
	return lang.T("buildings.level", lvl, game.BuildingMax[b])
}

// step is what the next level changes; the Market names only the parts that change.
func (m Model) step(b game.Building, lvl int) string {
	if b != game.Market {
		return m.valueAt(b, lvl) + " → " + m.valueAt(b, lvl+1)
	}
	deep := m.g.Guild.Deepest()
	v := lang.T("building.value.ilvl", min(game.MarketILvl(lvl), deep)) + " → " + fmt.Sprint(min(game.MarketILvl(lvl+1), deep))
	if a, z := game.ShopSize+game.MarketItemsAt(lvl), game.ShopSize+game.MarketItemsAt(lvl+1); a != z {
		v += " · " + lang.T("building.value.items", z)
	}
	if a, z := game.MarketRarity(lvl), game.MarketRarity(lvl+1); a != z {
		v += " · " + lang.T("building.value.rarity", z)
	}
	return v
}
