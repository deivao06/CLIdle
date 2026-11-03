package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTripKeepsSimulationIdentical(t *testing.T) {
	for c := range NumClasses {
		path := filepath.Join(t.TempDir(), "save.json")
		g := heroAt(c, 15, 21)
		g.Auto = true
		g.Advance(3 * 60000)
		if err := g.Save(path, time.Now()); err != nil {
			t.Fatal(err)
		}
		loaded, _, err := Load(path)
		if err != nil || loaded == nil {
			t.Fatalf("load: %v", err)
		}
		g.Advance(5 * 60000)
		loaded.Advance(5 * 60000)
		if g.Hero.Pos != loaded.Hero.Pos || g.Hero.XP != loaded.Hero.XP || g.Hero.MP != loaded.Hero.MP || g.Gold != loaded.Gold || len(g.Log) != len(loaded.Log) {
			t.Fatalf("%s diverged after load: %+v vs %+v", c.Name(), g.Hero, loaded.Hero)
		}
	}
}

func TestLoadMissingSave(t *testing.T) {
	g, _, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if g != nil || err != nil {
		t.Fatalf("got %v %v", g, err)
	}
}

func TestV1SaveMigratesTraining(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.Hero.Training = [NumTraits]int{2, 3, 4, 5} // v1 order: Vigor, Força, Resistência, Agilidade
	data, _ := json.Marshal(saveFile{1, time.Now(), g})
	os.WriteFile(path, data, 0o644)
	got, _, err := Load(path)
	if err != nil || got.Hero.Training != [NumTraits]int{3, 5, 0, 6} || got.Hero.Class != Warrior {
		t.Fatalf("got %v %v", got.Hero.Training, err)
	}
	got.Save(path, time.Now())
	again, _, _ := Load(path)
	if again.Hero.Training != got.Hero.Training {
		t.Fatal("migration ran twice")
	}
}

func TestV1SaveMigratesItemKind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.Depart()
	g.Hero.Equip[Weapon] = &Item{Slot: Weapon, Name: "Maça", Base: 3}
	g.Exp.Bag = []*Item{{Slot: Weapon, Name: "Maça", Base: 3}, {Slot: Armor, Name: "item.plate_armor", Base: 3}}
	data, _ := json.Marshal(saveFile{1, time.Now(), g})
	os.WriteFile(path, data, 0o644)
	got, _, err := Load(path)
	if err != nil || got.Hero.Equip[Weapon].Kind != Mace || got.Exp.Bag[0].Kind != Mace || got.Exp.Bag[1].Kind != Heavy {
		t.Fatalf("kinds not migrated: %v", err)
	}
}

func TestV2SaveKeepsEpicItemsEpic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.Hero.Equip[Weapon] = &Item{Slot: Weapon, Rarity: 2, Name: "item.mace", Base: 3}
	data, _ := json.Marshal(saveFile{2, time.Now(), g})
	os.WriteFile(path, data, 0o644)
	got, _, err := Load(path)
	if err != nil || got.Hero.Equip[Weapon].Rarity != Epic {
		t.Fatalf("rarity %v err %v", got.Hero.Equip[Weapon].Rarity, err)
	}
}

func TestV3QueuedPurchasesAreRefunded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.Depart()
	g.Gold = 0
	g.Hero.Equip[Armor] = &Item{Slot: Armor, Base: 3, Upgrade: 1}
	boots := &Item{Slot: Boots, Base: 2}
	raw, _ := json.Marshal(saveFile{3, time.Now(), g})
	var f map[string]any
	json.Unmarshal(raw, &f)
	game := f["Game"].(map[string]any)
	game["Pending"] = []*Item{boots}
	game["Upgrades"] = [NumSlots]int{Armor: 2, Boots: 1}
	game["BagQueued"] = 2
	data, _ := json.Marshal(f)
	os.WriteFile(path, data, 0o644)
	got, _, err := Load(path)
	armor := g.Hero.Equip[Armor]
	want := ShopPrice(boots) + UpgradeCost(armor, 1) + UpgradeCost(armor, 2) + UpgradeCost(boots, 0) + BagCost(0) + BagCost(1)
	if err != nil || got.Gold != want || got.Hero.Equip[Armor].Upgrade != 1 || got.BagLevel != 0 {
		t.Fatalf("gold %d want %d err %v", got.Gold, want, err)
	}
}
