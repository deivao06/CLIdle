package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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
		if g.Hero.Pos != loaded.Hero.Pos || g.Hero.XP != loaded.Hero.XP || g.Hero.MP != loaded.Hero.MP || g.Guild.Gold != loaded.Guild.Gold || len(g.Log) != len(loaded.Log) {
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
	data, _ := oldSave(1, g)
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
	data, _ := oldSave(1, g)
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
	data, _ := oldSave(2, g)
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
	g.Guild.Gold = 0
	g.Hero.Equip[Armor] = &Item{Slot: Armor, Base: 3, Upgrade: 1}
	boots := &Item{Slot: Boots, Base: 2}
	raw, _ := oldSave(3, g)
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
	if err != nil || got.Guild.Gold != want || got.Hero.Equip[Armor].Upgrade != 1 || got.BagLevel != 0 {
		t.Fatalf("gold %d want %d err %v", got.Guild.Gold, want, err)
	}
}

func TestV4ManualEquipKeepsLoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.Config.ManualEquip = true
	data, _ := oldSave(4, g)
	os.WriteFile(path, data, 0o644)
	got, _, err := Load(path)
	if err != nil || !got.Config.KeepLoot {
		t.Fatalf("keep loot %v err %v", got.Config.KeepLoot, err)
	}
}

func TestV7UpgradesKeepTheirBonus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.Hero.Equip[Weapon] = &Item{ID: 7, Slot: Weapon, Rarity: Legendary, Base: 110, Upgrade: 12}
	raw, _ := oldSave(7, g)
	os.WriteFile(path, raw, 0o644)
	got, _, err := Load(path)
	if it := got.Hero.Equip[Weapon]; err != nil || it.Bonus() != 110+12*11 {
		t.Fatalf("bonus %d err %v", it.Bonus(), err)
	}
}

func TestV8NegativeGoldBecomesMaxGold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.Guild.Gold = -9139714478572972266
	raw, _ := oldSave(8, g)
	os.WriteFile(path, raw, 0o644)
	got, _, err := Load(path)
	if err != nil || got.Guild.Gold != MaxGold {
		t.Fatalf("gold %d err %v", got.Guild.Gold, err)
	}
}

func TestV10BlessRuleBecomesHolyLight(t *testing.T) {
	for _, level := range []int{AbilityInfo[Aura].Level, AbilityInfo[HolyLight].Level} {
		path := filepath.Join(t.TempDir(), "save.json")
		g := New(1, Cleric)
		g.Hero.Level = level
		bless := Gambit{If: [2]Cond{{CondAdjacent, OpAtLeast, 1}}, Do: ActHolyLight}
		g.Config.Gambits = append([]Gambit{bless}, g.Config.Gambits...)
		raw, _ := oldSave(10, g)
		os.WriteFile(path, raw, 0o644)
		got, _, err := Load(path)
		if err != nil || slices.Contains(got.Config.Gambits, bless) || !slices.Equal(got.Config.Gambits, heroGambits(got.Hero)) {
			t.Fatalf("level %d: %+v err %v", level, got.Config.Gambits, err)
		}
	}
}

func TestGoldStopsAtMaxGold(t *testing.T) {
	if addGold(MaxGold-5, 10) != MaxGold || addGold(5, 10) != 15 {
		t.Fatal("addGold")
	}
	if pctOf(MaxGold, 200) != MaxGold || pctOf(MaxGold, 25) != MaxGold/100*25+MaxGold%100*25/100 || pctOf(1234, 150) != 1234*150/100 {
		t.Fatal("pctOf")
	}
	if goldOf(1e30) != MaxGold || goldOf(12.9) != 12 {
		t.Fatal("goldOf")
	}
}

// oldSave writes g as a save of an older version, when the guild fields still lived in the hero's game.
func oldSave(version int, g *Game) ([]byte, error) {
	raw, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	var game map[string]any
	json.Unmarshal(raw, &game)
	shared, _ := json.Marshal(g.Guild)
	json.Unmarshal(shared, &game)
	delete(game, "Heroes")
	delete(game, "Log")
	game["Log"] = g.Log
	return json.Marshal(map[string]any{"Version": version, "SavedAt": time.Now(), "Game": game})
}

func TestV11SaveBecomesAGuildWithItsHero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(31415926, Rogue)
	g.Hero.Name = "Ana"
	gd := g.Guild
	gd.Gold, gd.Stash, gd.Bell, gd.CRT, gd.Nerd, gd.Lang, gd.SeenNews = 1234, []*Item{{ID: 9, Slot: Boots}}, true, true, true, "pt", "v0.16.0"
	raw, _ := oldSave(11, g)
	os.WriteFile(path, raw, 0o644)
	got, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Guild{Gold: 1234, Bell: true, CRT: true, Nerd: true, Lang: "pt", SeenNews: "v0.16.0"}
	g2 := got.Guild
	if g2.Gold != want.Gold || g2.Bell != want.Bell || g2.CRT != want.CRT || g2.Nerd != want.Nerd || g2.Lang != want.Lang || g2.SeenNews != want.SeenNews ||
		len(g2.Stash) != 1 || g2.Stash[0].ID != 9 || len(g2.Heroes) != 1 || g2.Heroes[0] != got || got.Hero.Name != "Ana" {
		t.Fatalf("guild %+v", g2)
	}
	got.Save(path, time.Now())
	again, _, err := Load(path)
	if err != nil || again.Guild.Gold != 1234 || again.Guild.Lang != "pt" || again.Guild.Heroes[0] != again {
		t.Fatalf("after a v12 round trip: %+v err %v", again.Guild, err)
	}
}
