package ui

import (
	"fmt"

	"clidle/internal/game"
	"clidle/internal/lang"
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
			row.text, row.style = lang.T("buildings.row", name, buildingValue(b, lvl)+" → "+buildingValue(b, lvl+1), cost), m.buyable(cost)
			row.price, row.priceSt, row.enter = cost, m.goldFor(cost), func() { gd.Build(b) }
		}
		rows = append(rows, row)
	}
	return rows
}
