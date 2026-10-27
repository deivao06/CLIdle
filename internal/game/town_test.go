package game

import "testing"

func townGame() *Game {
	g := New(9)
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
	if g.Hero.Equip[Weapon].Base != 10 || g.Gold != 100000-price+old.Value() {
		t.Fatalf("equip %+v gold %d", g.Hero.Equip[Weapon], g.Gold)
	}
}

func TestBuyAwayIsAppliedOnReturn(t *testing.T) {
	g := departed(9)
	g.Gold = 100000
	g.Shop = []*Item{{Slot: Armor, Base: 10}}
	g.BuyShop(0)
	if g.Hero.Equip[Armor] != nil && g.Hero.Equip[Armor].Base == 10 {
		t.Fatal("should not equip while away")
	}
	g.Upgrade(Armor)
	g.returnToTown()
	if a := g.Hero.Equip[Armor]; a == nil || a.Base != 10 || a.Upgrade != 1 {
		t.Fatalf("armor %+v", a)
	}
}

func TestPurchaseSurvivesDeath(t *testing.T) {
	g := departed(9)
	g.Gold = 100000
	g.Shop = []*Item{{Slot: Boots, Base: 3}}
	g.BuyShop(0)
	g.Hero.HP = 1
	g.monsterAttack(&Monster{Atk: 100}, &round{})
	if b := g.Hero.Equip[Boots]; b == nil || b.Base != 3 {
		t.Fatalf("boots %+v", b)
	}
}

func TestNotEnoughGold(t *testing.T) {
	g := townGame()
	g.Gold = 0
	g.Train(Vigor)
	if g.Hero.Training[Vigor] != 0 {
		t.Fatal("trained without gold")
	}
}

func TestTrainingRaisesStat(t *testing.T) {
	g := townGame()
	hp := g.Hero.MaxHP()
	g.Train(Vigor)
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
	g := New(1)
	if !g.InTown() {
		t.Fatal("new game should start in town")
	}
	g.Advance(TownWaitMs)
	if g.InTown() {
		t.Fatal("auto-expedition should depart after the town wait")
	}
}
