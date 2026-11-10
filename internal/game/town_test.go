package game

import "testing"

func townGame() *Game {
	g := New(9, Warrior)
	g.Gold = 100000
	return g
}

func TestBuyInTownEquipsAndSellsOld(t *testing.T) {
	g := townGame()
	old := &Item{Slot: Weapon, Base: 1}
	g.Hero.Equip[Weapon] = old
	g.Shop = []*Item{{Slot: Weapon, Base: 10}}
	price := ShopPrice(g.Shop[0])
	g.BuyShop(0)
	if g.Hero.Equip[Weapon].Base != 10 || g.Gold != 100000-price+old.SellPrice() {
		t.Fatalf("equip %+v gold %d", g.Hero.Equip[Weapon], g.Gold)
	}
}

func TestTownActionsDoNothingAway(t *testing.T) {
	g := departed(9)
	g.Gold = 100000
	g.Hero.Equip[Armor] = &Item{Slot: Armor, Base: 3}
	g.Shop = []*Item{{Slot: Boots, Base: 3}}
	g.Stash = []*Item{{Slot: Weapon, Base: 3}}
	g.BuyShop(0)
	g.Upgrade(Armor)
	g.Train(Con)
	g.UpgradeBag()
	g.SellStash(0)
	if g.Gold != 100000 || len(g.Shop) != 1 || len(g.Stash) != 1 || g.Hero.Equip[Armor].Upgrade != 0 || g.Hero.Training[Con] != 0 || g.BagLevel != 0 {
		t.Fatal("a town action worked in the dungeon")
	}
}

func TestNotEnoughGold(t *testing.T) {
	g := townGame()
	g.Gold = 0
	g.Train(Con)
	if g.Hero.Training[Con] != 0 {
		t.Fatal("trained without gold")
	}
}

func TestTrainingRaisesStat(t *testing.T) {
	g := townGame()
	hp := g.Hero.MaxHP()
	g.Train(Con)
	if g.Hero.MaxHP() != hp+TrainHP {
		t.Fatalf("max hp %d", g.Hero.MaxHP())
	}
}

func TestAutoOffKeepsHeroInTown(t *testing.T) {
	g := townGame()
	g.Auto = false
	g.Advance(60000)
	if !g.InTown() {
		t.Fatal("hero left with auto off")
	}
	g.Depart()
	if g.InTown() {
		t.Fatal("depart did not start expedition")
	}
}

func TestPotionsBoughtBeforeExpedition(t *testing.T) {
	g := townGame()
	g.Hero.Potions = 0
	g.Config.PotionsToBring = 4
	g.StartExpedition()
	if g.Hero.Potions != 4 || g.Gold != 100000-4*PotionPrice {
		t.Fatalf("potions %d gold %d", g.Hero.Potions, g.Gold)
	}
}

func TestNewGameStartsInTown(t *testing.T) {
	g := New(1, Warrior)
	if !g.InTown() {
		t.Fatal("new game should start in town")
	}
	g.Advance(TownWaitMs)
	if !g.InTown() {
		t.Fatal("new game should wait for the player to depart")
	}
	g.Auto = true
	g.Advance(TownWaitMs)
	if g.InTown() {
		t.Fatal("auto-expedition should depart after the town wait")
	}
}

func TestShopRefusesOtherClassItem(t *testing.T) {
	g := New(1, Mage)
	g.Gold = 100000
	g.Shop = []*Item{{Slot: Shield, Base: 3}}
	g.BuyShop(0)
	if g.Gold != 100000 || g.Hero.Equip[Shield] != nil {
		t.Fatal("bought a shield as mage")
	}
}

func TestExtraPotionsSoldInTown(t *testing.T) {
	g := townGame()
	g.Hero.Potions = MaxPotions + 3
	g.arriveInTown()
	if g.Hero.Potions != MaxPotions || g.Gold != 100000+3*PotionSellPrice {
		t.Fatalf("potions %d gold %d", g.Hero.Potions, g.Gold)
	}
}

func TestAutoStopsAfterDeathLimit(t *testing.T) {
	g := townGame()
	g.Config.DeathLimit = 2
	g.ToggleAuto()
	for range 2 {
		g.Depart()
		g.die(nil)
	}
	if g.Auto {
		t.Fatal("auto still on after the death limit")
	}
	g.ToggleAuto()
	g.Depart()
	g.die(nil)
	if !g.Auto {
		t.Fatal("turning auto back on did not reset the count")
	}
}

func TestShopNeverSellsLegendaries(t *testing.T) {
	g := New(1, Warrior)
	for range 2000 {
		g.refreshShop()
		for _, it := range g.Shop {
			if it.Rarity == Legendary {
				t.Fatal("shop rolled a legendary")
			}
		}
	}
}

func TestSmithIgnoresRarityAndStopsAtTheMax(t *testing.T) {
	g := New(1, Warrior)
	common := &Item{Slot: Weapon, Base: 38}
	legend := &Item{Slot: Weapon, Rarity: Legendary, Base: 110}
	if common.UpgradeStep() != legend.UpgradeStep() || UpgradeCost(legend, 3) <= UpgradeCost(common, 3) {
		t.Fatalf("steps %d/%d", common.UpgradeStep(), legend.UpgradeStep())
	}
	g.Hero.Equip[Weapon] = legend
	g.Gold = 1 << 40
	for range UpgradeMax + 3 {
		g.Upgrade(Weapon)
	}
	if legend.Upgrade != UpgradeMax || legend.Bonus() != 110+UpgradeMax*legend.UpgradeStep() {
		t.Fatalf("upgrade +%d bonus %d", legend.Upgrade, legend.Bonus())
	}
}
