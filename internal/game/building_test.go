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
		if !gd.Build(Market) || before-gd.Gold != BuildingCost(lvl) {
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
	smith, train, potion := g.UpgradeCost(Weapon), g.TrainPrice(Str), g.PotionCost()
	beds, slots := gd.Beds(), gd.Slots()
	gd.Gold = 1 << 40
	for b := range NumBuildings {
		gd.Build(b)
		gd.Build(b)
	}
	g.refreshShop()
	pct := func(v int, b Building) int { return v * (100 - 2*BuildingDiscount[b]) / 100 }
	switch {
	case gd.Beds() != beds+2, gd.Slots() != slots+2:
		t.Fatalf("beds %d slots %d", gd.Beds(), gd.Slots())
	case g.UpgradeCost(Weapon) != pct(smith, Smithy), g.TrainPrice(Str) != pct(train, TrainingYard), g.PotionCost() != max(1, pct(potion, Apothecary)):
		t.Fatalf("smith %d train %d potion %d", g.UpgradeCost(Weapon), g.TrainPrice(Str), g.PotionCost())
	case len(g.Shop) != ShopSize+2*MarketItems:
		t.Fatalf("shop has %d items", len(g.Shop))
	}
	gold := gd.Gold
	g.Train(Str)
	if gold-gd.Gold != pct(train, TrainingYard) {
		t.Fatalf("training charged %d", gold-gd.Gold)
	}
}
