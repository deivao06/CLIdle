package game

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"clidle/internal/lang"
)

func TestGoldPerPileIsQuadratic(t *testing.T) {
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-6*b }
	if !near(GoldPerPile(1), 5) || !near(GoldPerPile(11), 20) || !near(GoldPerPile(101), 605) {
		t.Fatalf("pile at 1 %.0f, 11 %.0f, 101 %.0f", GoldPerPile(1), GoldPerPile(11), GoldPerPile(101))
	}
}

func TestBossesFollowTheDungeonFloors(t *testing.T) {
	g := heroAt(Warrior, 1, 1)
	g.Dungeon = 1
	g.enterLevel(g.Offset() + 10)
	if g.Exp.Level.boss() == nil || g.Floor(g.Exp.Level.Depth) != 10 {
		t.Fatal("no boss on floor 10 of the second dungeon")
	}
	g.enterLevel(g.Offset() + 5)
	if g.Exp.Level.boss() != nil {
		t.Fatal("boss on floor 5")
	}
}

func TestClearingADungeonRewardsAndOpensTheNext(t *testing.T) {
	g := heroAt(Rogue, 1, 1)
	g.enterLevel(DungeonFloors)
	boss := g.Exp.Level.boss()
	gold, stash, dex := g.Guild.Gold, len(g.Guild.Stash), g.Hero.Attr(Dex)
	g.killBoss(boss, &round{})
	if len(g.Guild.Stash) != stash+1 || g.Guild.Stash[stash].Rarity != Legendary || g.Guild.Gold <= gold || g.Hero.Attr(Dex) != dex+DungeonAttrBonus {
		t.Fatal("no reward for clearing the dungeon")
	}
	if !g.Exp.Retreating || !g.Exp.DungeonDone || g.Dungeon != 0 {
		t.Fatal("the hero should head home before the next dungeon opens")
	}
	name := g.DungeonName()
	g.returnToTown()
	if g.Dungeon != 1 || g.Reached != 0 || !slices.Equal(g.ShortcutList(), []int{DungeonStep + 1}) || g.startFloor() != DungeonStep+1 {
		t.Fatalf("dungeon %d reached %d shortcuts %v", g.Dungeon, g.Reached, g.ShortcutList())
	}
	if g.DungeonName() == name && g.Noun == 0 && g.Adj == 0 {
		t.Fatal("next dungeon has no name")
	}
	g.StartExpedition()
	if g.Exp.Level.Depth != DungeonStep+1 || g.Floor(g.Exp.Level.Depth) != 1 {
		t.Fatalf("next expedition starts at depth %d", g.Exp.Level.Depth)
	}
}

func TestDyingAfterTheLastBossStillOpensTheNextDungeon(t *testing.T) {
	g := heroAt(Warrior, 1, 1)
	g.enterLevel(DungeonFloors)
	g.killBoss(g.Exp.Level.boss(), &round{})
	g.die(dummy())
	if g.Dungeon != 1 || g.Hero.Bonus != DungeonAttrBonus {
		t.Fatalf("dungeon %d bonus %d", g.Dungeon, g.Hero.Bonus)
	}
}

func TestDungeonNameShowsTier(t *testing.T) {
	g := New(1, Warrior)
	g.Noun, g.Adj, g.Dungeon = 0, 0, 1
	if got := g.DungeonName(); got != lang.T("dungeon.adj.infernal")+" "+lang.T("dungeon.noun.cave")+" (Tier 2)" {
		t.Fatalf("name %q", got)
	}
}

func TestV9DeepSaveMovesToItsDungeon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.MaxDepth = 330
	g.Shortcuts = []int{1, 11, 101, 311, 321, 331}
	raw, _ := oldSave(9, g)
	os.WriteFile(path, raw, 0o644)
	got, _, err := Load(path)
	if err != nil || got.Dungeon != 4 || got.Reached != 30 || got.Hero.Bonus != 4*DungeonAttrBonus {
		t.Fatalf("dungeon %d reached %d bonus %d err %v", got.Dungeon, got.Reached, got.Hero.Bonus, err)
	}
	if !slices.Equal(got.ShortcutList(), []int{301, 311, 321, 331}) {
		t.Fatalf("shortcuts %v", got.ShortcutList())
	}
}

func TestV9ShallowSaveStaysInTheFirstDungeon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	g.MaxDepth = 42
	g.Shortcuts = []int{1, 11, 21, 31, 41}
	raw, _ := oldSave(9, g)
	os.WriteFile(path, raw, 0o644)
	got, _, err := Load(path)
	if err != nil || got.Dungeon != 0 || got.Reached != 42 || got.Hero.Bonus != 0 || !slices.Equal(got.ShortcutList(), g.Shortcuts) {
		t.Fatalf("dungeon %d reached %d shortcuts %v err %v", got.Dungeon, got.Reached, got.ShortcutList(), err)
	}
}

func TestPrestigeStartsOverAndKeepsTheLegacy(t *testing.T) {
	g := New(1, Warrior)
	g.Hero.Name, g.Hero.Level, g.Hero.Training[Str], g.Hero.Bonus = "Ana", 40, 3, 2*DungeonAttrBonus
	g.Guild.Gold, g.Guild.BagLevel, g.Dungeon, g.Reached = 5000, 3, 2, 40
	g.Guild.Stash = []*Item{{ID: 99, Slot: Weapon, Rarity: Legendary, Base: 50}}
	if g.PrestigeGain() != int(NumTraits)*PrestigePerTier*(1+2) {
		t.Fatalf("gain %d", g.PrestigeGain())
	}
	g.Prestige(Mage)
	h := g.Hero
	if h.Class != Mage || h.Level != 1 || h.Name != "Ana" || h.LegacyPoints != 24 || h.FreePoints != CreationPoints+24 || h.Bonus != 0 || h.Training[Str] != 0 {
		t.Fatalf("hero %+v", h)
	}
	if g.Guild.Gold != 5000 || g.Guild.BagLevel != 3 || g.Dungeon != 0 || g.Reached != 0 || len(g.Guild.Stash) != 1 || g.Prestiges != 1 || g.startFloor() != 1 {
		t.Fatal("prestige did not start over")
	}
	if h.Spend([NumTraits]int{Con: CreationPoints + 24}) != nil || h.Attr(Con) != CreationPoints+24 || g.CanPrestige() {
		t.Fatal("legacy missing, or prestige available with no dungeon cleared")
	}
	g.Dungeon = 1
	g.Prestige(Mage)
	if g.Prestiges != 1 {
		t.Fatal("prestige below the level")
	}
	g.Hero.Level = PrestigeLevel
	g.Prestige(Mage)
	if g.Hero.LegacyPoints != 24+int(NumTraits)*PrestigePerTier || g.Hero.FreePoints != CreationPoints+g.Hero.LegacyPoints || g.Prestiges != 2 {
		t.Fatalf("legacy %d", g.Hero.LegacyPoints)
	}
}
