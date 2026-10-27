package game

import "testing"

func TestReturnDepositsGoldAndSellsBag(t *testing.T) {
	g := New(1)
	it := &Item{Slot: Weapon, Base: 5}
	g.Exp.Gold = 10
	g.Exp.Bag = []*Item{it}
	g.returnToTown()
	if g.Exp != nil || g.Gold != 10+it.Value() {
		t.Fatalf("gold %d, exp %v", g.Gold, g.Exp)
	}
}

func TestDeathLosesFoundLootButKeepsStartEquip(t *testing.T) {
	g := New(1)
	start := &Item{Slot: Weapon, Base: 3}
	g.Hero.Equip[Weapon] = start
	g.Exp.StartEquip = g.Hero.Equip
	g.Hero.Equip[Weapon] = &Item{Slot: Weapon, Base: 9}
	g.Exp.Gold = 50
	g.Hero.HP = 1
	m := &Monster{HP: 9, Atk: 100, Spd: 4}
	g.monsterAttack(m, &round{})
	if g.Exp != nil || g.Gold != 0 || g.Hero.Equip[Weapon] != start {
		t.Fatalf("gold %d, weapon %+v", g.Gold, g.Hero.Equip[Weapon])
	}
}

func TestBagFullTriggersRetreatAndHeroLeaves(t *testing.T) {
	g := New(2)
	for range BagSize {
		g.Exp.Bag = append(g.Exp.Bag, &Item{Base: 1})
	}
	g.Exp.Level.Mons = nil
	g.checkRetreat()
	if !g.Exp.Retreating {
		t.Fatal("expected retreat")
	}
	g.Hero.Pos = g.Exp.Level.Up
	g.step()
	if g.Exp != nil || g.Returns != 1 {
		t.Fatal("hero should be back in town")
	}
}

func TestPotionHealsWhenLow(t *testing.T) {
	g := New(3)
	g.Exp.Level.Mons = nil
	g.Hero.Potions = 1
	g.Hero.HP = 1
	g.step()
	if g.Hero.Potions != 0 || g.Hero.HP <= 1 {
		t.Fatalf("potions %d hp %d", g.Hero.Potions, g.Hero.HP)
	}
}

func TestBetterItemIsEquipped(t *testing.T) {
	g := New(4)
	l := g.Exp.Level
	l.Mons = nil
	it := &Item{Slot: Armor, Base: 7}
	l.Items = []*FloorItem{{Pos: g.Hero.Pos, Item: it}}
	g.pickup(&round{})
	if g.Hero.Equip[Armor] != it || g.Hero.Def() != HeroBaseDef+7 {
		t.Fatalf("armor not equipped: %+v", g.Hero.Equip)
	}
}
