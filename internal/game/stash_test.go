package game

import "testing"

func TestManualEquipDoesNotEquip(t *testing.T) {
	g := New(1, Warrior)
	g.Config.ManualEquip = true
	g.Depart()
	if g.better(&Item{Slot: Weapon, Kind: Blade, Base: 99}) {
		t.Fatal("hero equips by himself in manual mode")
	}
}

func TestKeepLootSendsBagToStash(t *testing.T) {
	g := New(1, Warrior)
	g.Config.KeepLoot = true
	g.Depart()
	it := &Item{Slot: Weapon, Kind: Blade, Base: 99}
	if !g.better(it) {
		t.Fatal("keeping loot turned auto-equip off")
	}
	g.Exp.Bag = []*Item{it}
	g.returnToTown()
	if len(g.Guild.Stash) != 1 || g.Guild.Stash[0] != it || g.Guild.Gold != 0 {
		t.Fatalf("stash %v gold %d", g.Guild.Stash, g.Guild.Gold)
	}
}

func TestManualEquipStillSellsByDefault(t *testing.T) {
	g := New(1, Warrior)
	g.Config.ManualEquip = true
	g.Depart()
	it := &Item{Slot: Weapon, Base: 5}
	g.Exp.Bag = []*Item{it}
	g.returnToTown()
	if len(g.Guild.Stash) != 0 || g.Guild.Gold != it.SellPrice() {
		t.Fatalf("stash %v gold %d", g.Guild.Stash, g.Guild.Gold)
	}
}

func TestStashHasNoLimit(t *testing.T) {
	g := New(1, Warrior)
	g.Config.KeepLoot = true
	g.Guild.Stash = make([]*Item, 50)
	g.Depart()
	it := &Item{Slot: Weapon, Base: 5}
	g.Exp.Bag = []*Item{it}
	g.returnToTown()
	if len(g.Guild.Stash) != 51 || g.Guild.Stash[50] != it {
		t.Fatalf("stash %d gold %d", len(g.Guild.Stash), g.Guild.Gold)
	}
}

func TestEquipFromStashSwapsWithEquipped(t *testing.T) {
	g := New(1, Warrior)
	old := &Item{Slot: Weapon, Kind: Blade, Base: 2}
	it := &Item{Slot: Weapon, Kind: Blade, Base: 7}
	g.Hero.Equip[Weapon] = old
	g.Guild.Stash = []*Item{it}
	g.EquipStash(0)
	if g.Hero.Equip[Weapon] != it || len(g.Guild.Stash) != 1 || g.Guild.Stash[0] != old {
		t.Fatalf("equip %v stash %v", g.Hero.Equip[Weapon], g.Guild.Stash)
	}
}

func TestEquipFromStashRefusesOtherClassAndWhileAway(t *testing.T) {
	g := New(1, Warrior)
	g.Hero.Equip[Weapon] = nil
	g.Guild.Stash = []*Item{{Slot: Weapon, Kind: Staff, Base: 7}}
	g.EquipStash(0)
	if g.Hero.Equip[Weapon] != nil {
		t.Fatal("warrior equipped a staff")
	}
	g.Guild.Stash = []*Item{{Slot: Weapon, Kind: Blade, Base: 7}}
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
	if g.Hero.Equip[Armor] != nil || len(g.Guild.Stash) != 1 {
		t.Fatalf("equip %v stash %v", g.Hero.Equip[Armor], g.Guild.Stash)
	}
	g.SellStash(0)
	if len(g.Guild.Stash) != 0 || g.Guild.Gold != it.SellPrice() {
		t.Fatalf("stash %v gold %d", g.Guild.Stash, g.Guild.Gold)
	}
}

func TestLegendaryIsNeverSoldOnItsOwn(t *testing.T) {
	g := New(1, Warrior)
	g.Depart()
	leg := &Item{Slot: Weapon, Base: 5, Rarity: Legendary}
	g.Exp.Bag = []*Item{leg, {Slot: Weapon, Base: 5}}
	g.returnToTown()
	if len(g.Guild.Stash) != 1 || g.Guild.Stash[0] != leg {
		t.Fatalf("legendary sold on return, stash %d", len(g.Guild.Stash))
	}
	old := &Item{Slot: Armor, Base: 2, Rarity: Legendary}
	g.Hero.Equip[Armor] = old
	g.Guild.Gold = 0
	g.equipBought(&Item{Slot: Armor, Base: 9})
	if g.Guild.Gold != 0 || g.Guild.Stash[len(g.Guild.Stash)-1] != old {
		t.Fatalf("legendary sold when buying, gold %d", g.Guild.Gold)
	}
}

func TestUpgradesRaiseSellPriceFairly(t *testing.T) {
	plain := &Item{Slot: Weapon, Base: 20}
	up := &Item{Slot: Weapon, Base: 20, Upgrade: 2}
	spent := UpgradeCost(&Item{Slot: Weapon, Base: 20}, 0) + UpgradeCost(&Item{Slot: Weapon, Base: 20, Upgrade: 1}, 1)
	if got, want := up.SellPrice(), up.Value()+spent*UpgradeRefundPct/100; got != want || got <= plain.SellPrice() || got >= spent {
		t.Fatalf("sell %d want %d plain %d spent %d", got, want, plain.SellPrice(), spent)
	}
}
