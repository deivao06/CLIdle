package ui

import (
	"fmt"

	"clidle/internal/game"
	"clidle/internal/lang"

	"github.com/charmbracelet/lipgloss"
)

// buildingValue is what a building gives at a level, in the words of its effect.
func buildingValue(b game.Building, lvl int) string {
	switch b {
	case game.Lodging:
		return lang.T("building.value.beds", game.GuildBeds+lvl)
	case game.MissionTable:
		return lang.T("building.value.slots", game.MissionSlots+lvl)
	case game.Market:
		return lang.T("building.value.items", game.ShopSize+lvl*game.MarketItems)
	}
	return lang.T("building.value.discount", lvl*game.BuildingDiscount[b])
}

func (m Model) buildingRows() []prepRow {
	gd := m.g.Guild
	sec := lang.T("buildings.section")
	var rows []prepRow
	for b := range game.NumBuildings {
		lvl := gd.Level(b)
		name := fmt.Sprintf("%-16s  %s", b.Name(), lang.T("buildings.level", lvl, game.BuildingMax[b]))
		cost, ok := gd.NextBuilding(b)
		row := prepRow{section: sec, desc: b.Desc()}
		if !ok {
			row.text, row.style = lang.T("buildings.row_max", name, buildingValue(b, lvl)), sMuted
		} else {
			row.text, row.style = lang.T("buildings.row", name, buildingValue(b, lvl)+" → "+buildingValue(b, lvl+1), game.Num(cost)), m.buyable(cost)
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
	value, right := buildingValue(b, lvl), sMuted.Render(lang.T("buildings.top"))
	if r.price > 0 {
		value += " → " + buildingValue(b, lvl+1)
		right = r.priceSt.Render(goldText(r.price))
	}
	detail := lang.T("buildings.level", lvl, game.BuildingMax[b]) + " · " + value
	return func(w int) []string {
		return []string{ends(r.style.Render(b.Name()), right, w), "  " + sMuted.Render(ellipsis(detail, w-2))}
	}
}
