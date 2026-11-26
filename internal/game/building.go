package game

import "clidle/internal/lang"

// Building is a part of the guild that gets better with gold, for good.
type Building int

const (
	Lodging Building = iota
	MissionTable
	Smithy
	TrainingYard
	Market
	Apothecary
	NumBuildings
)

var buildingKeys = [NumBuildings]string{"lodging", "mission_table", "smithy", "training_yard", "market", "apothecary"}

func (b Building) Key() string         { return buildingKeys[b] }
func (b Building) Name() string        { return lang.T("building." + buildingKeys[b]) }
func (b Building) Desc() string        { return lang.T("building." + buildingKeys[b] + ".desc") }
func (gd *Guild) Level(b Building) int { return gd.Buildings[b] }

// NextBuilding is the price of the building's next level; ok is false at the top.
func (gd *Guild) NextBuilding(b Building) (cost int, ok bool) {
	lvl := gd.Buildings[b]
	return BuildingCost(b, lvl), lvl < BuildingMax[b] && !(b == Market && gd.MarketLocked())
}

// Deepest is the deepest floor any hero of the guild has reached.
func (gd *Guild) Deepest() int {
	d := 1
	for _, h := range gd.Heroes {
		d = max(d, h.MaxDepth)
	}
	return d
}

// ShopLevel is the floor the shop's items come from: the Market's, never past the guild's deepest floor.
func (gd *Guild) ShopLevel() int { return min(MarketILvl(gd.Buildings[Market]), gd.Deepest()) }

// MarketLocked says the Market's items already come from the guild's deepest floor.
func (gd *Guild) MarketLocked() bool { return MarketILvl(gd.Buildings[Market]) >= gd.Deepest() }

// Build pays for the building's next level.
func (gd *Guild) Build(b Building) bool {
	cost, ok := gd.NextBuilding(b)
	if !ok || gd.Gold < cost {
		return false
	}
	gd.Gold -= cost
	gd.Buildings[b]++
	return true
}

// discounted takes the building's discount off a price, never below 1.
func (gd *Guild) discounted(price int, b Building) int {
	if price <= 0 {
		return price
	}
	return max(1, price*(100-BuildingDiscountAt(b, gd.Buildings[b]))/100)
}

func (g *Game) TrainPrice(t Trait) int {
	return g.Guild.discounted(TrainCost(g.Hero.Training[t]), TrainingYard)
}

func (g *Game) PotionCost() int { return g.Guild.discounted(PotionPrice, Apothecary) }
