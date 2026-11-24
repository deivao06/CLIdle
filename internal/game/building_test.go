package game

import "testing"

func TestBuildingsCostGoldAndStopAtTheTop(t *testing.T) {
	gd := New(14142135, Warrior).Guild
	if gd.Build(Lodging) {
		t.Fatal("built without gold")
	}
	gd.Gold = 1 << 40
	for lvl := range BuildingMax[Market] {
		before := gd.Gold
		if !gd.Build(Market) || before-gd.Gold != BuildingCost(Market, lvl) {
			t.Fatalf("level %d cost %d", lvl+1, before-gd.Gold)
		}
	}
	if gd.Build(Market) || gd.Level(Market) != BuildingMax[Market] {
		t.Fatal("built past the top")
	}
}

func TestBuildingsChangeTheGuild(t *testing.T) {
	g := New(17320508, Warrior)
	gd := g.Guild
	smith, train, potion, heal := g.UpgradeCost(Weapon), g.TrainPrice(Str), g.PotionCost(), g.PotionHeal()
	beds, slots := gd.Beds(), gd.Slots()
	gd.Gold = 1 << 40
	for b := range NumBuildings {
		gd.Build(b)
		gd.Build(b)
	}
	g.refreshShop()
	pct := func(v int, b Building) int { return v * (100 - BuildingDiscountAt(b, 2)) / 100 }
	switch {
	case gd.Beds() != beds+2, gd.Slots() != slots+2:
		t.Fatalf("beds %d slots %d", gd.Beds(), gd.Slots())
	case g.UpgradeCost(Weapon) != pct(smith, Smithy), g.TrainPrice(Str) != pct(train, TrainingYard), g.PotionCost() != max(1, pct(potion, Apothecary)):
		t.Fatalf("smith %d train %d potion %d", g.UpgradeCost(Weapon), g.TrainPrice(Str), g.PotionCost())
	case g.PotionHeal() != heal+g.Hero.MaxHP()*2*ApothecaryHealPct/100:
		t.Fatalf("potion heals %d", g.PotionHeal())
	case len(g.Shop) != ShopSize+2*MarketItems:
		t.Fatalf("shop has %d items", len(g.Shop))
	}
	gold := gd.Gold
	g.Train(Str)
	if gold-gd.Gold != pct(train, TrainingYard) {
		t.Fatalf("training charged %d", gold-gd.Gold)
	}
}

func TestApothecaryHealsFirstThenDiscounts(t *testing.T) {
	h, d := ApothecaryHealPct, BuildingDiscount[Apothecary]
	for lvl, want := range [][2]int{{0, 0}, {h, 0}, {2 * h, 0}, {3 * h, 0}, {3 * h, d}, {3 * h, 2 * d}} {
		if got := [2]int{ApothecaryHeal(lvl), BuildingDiscountAt(Apothecary, lvl)}; got != want {
			t.Fatalf("level %d: heal and discount %v, want %v", lvl, got, want)
		}
	}
}

func TestMarketAddsItemsFirstThenRarity(t *testing.T) {
	r := MarketRarityPct
	for lvl, want := range [][2]int{{0, 0}, {MarketItems, 0}, {2 * MarketItems, 0}, {2 * MarketItems, r}, {2 * MarketItems, 2 * r}, {2 * MarketItems, 3 * r}} {
		if got := [2]int{MarketItemsAt(lvl), MarketRarity(lvl)}; got != want {
			t.Fatalf("level %d: items and rarity %v, want %v", lvl, got, want)
		}
	}
	rare := func(boost int) int {
		rng, n := NewRNG(27182818), 0
		for range 20000 {
			if rollRarity(rng, Epic, boost) > Common {
				n++
			}
		}
		return n
	}
	if base, top := rare(0), rare(MarketRarity(BuildingMax[Market])); top < base*2 {
		t.Fatalf("the Market's top level gives %d non-common items against %d", top, base)
	}
}
