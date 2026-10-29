package game

import "testing"

func TestManualEquipSendsFoundItemsToStash(t *testing.T) {
	g := New(1, Warrior)
	g.Config.ManualEquip = true
	g.Depart()
	it := &Item{Slot: Weapon, Kind: Blade, Base: 99}
	if g.better(it) {
		t.Fatal("hero equips by himself in manual mode")
	}
	g.Exp.Bag = []*Item{it}
	g.returnToTown()
	if len(g.Stash) != 1 || g.Stash[0] != it || g.Gold != 0 {
		t.Fatalf("stash %v gold %d", g.Stash, g.Gold)
	}
}

func TestStashOverflowIsSold(t *testing.T) {
	g := New(1, Warrior)
	g.Config.ManualEquip = true
	g.Stash = make([]*Item, StashSize)
	g.Depart()
	it := &Item{Slot: Weapon, Base: 5}
	g.Exp.Bag = []*Item{it}
	g.returnToTown()
	if len(g.Stash) != StashSize || g.Gold != it.Value() {
		t.Fatalf("stash %d gold %d", len(g.Stash), g.Gold)
	}
}

func TestEquipFromStashSwapsWithEquipped(t *testing.T) {
	g := New(1, Warrior)
	old := &Item{Slot: Weapon, Kind: Blade, Base: 2}
	it := &Item{Slot: Weapon, Kind: Blade, Base: 7}
	g.Hero.Equip[Weapon] = old
	g.Stash = []*Item{it}
	g.EquipStash(0)
	if g.Hero.Equip[Weapon] != it || len(g.Stash) != 1 || g.Stash[0] != old {
		t.Fatalf("equip %v stash %v", g.Hero.Equip[Weapon], g.Stash)
	}
}

func TestEquipFromStashRefusesOtherClassAndWhileAway(t *testing.T) {
	g := New(1, Warrior)
	g.Stash = []*Item{{Slot: Weapon, Kind: Staff, Base: 7}}
	g.EquipStash(0)
	if g.Hero.Equip[Weapon] != nil {
		t.Fatal("warrior equipped a staff")
	}
	g.Stash = []*Item{{Slot: Weapon, Kind: Blade, Base: 7}}
	g.Depart()
	g.EquipStash(0)
	if g.Hero.Equip[Weapon] != nil {
		t.Fatal("equipped while the hero is in the dungeon")
	}
}

func TestUnequipAndSellStash(t *testing.T) {
	g := New(1, Warrior)
	it := &Item{Slot: Armor, Base: 4}
	g.Hero.Equip[Armor] = it
	g.Unequip(Armor)
	if g.Hero.Equip[Armor] != nil || len(g.Stash) != 1 {
		t.Fatalf("equip %v stash %v", g.Hero.Equip[Armor], g.Stash)
	}
	g.SellStash(0)
	if len(g.Stash) != 0 || g.Gold != it.Value() {
		t.Fatalf("stash %v gold %d", g.Stash, g.Gold)
	}
}
