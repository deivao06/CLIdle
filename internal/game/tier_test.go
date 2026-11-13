package game

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"clidle/internal/lang"
)

func TestGoldTurnsLinearAfterTheFirstDungeon(t *testing.T) {
	top := GoldPerPile(DungeonFloors)
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-6*b }
	if !near(top, 5*pow108(DungeonFloors-1)) || !near(GoldPerPile(DungeonFloors+1), top*1.08) || !near(GoldPerPile(DungeonFloors+100), top*9) {
		t.Fatalf("pile at 100 %.0f, 101 %.0f, 200 %.0f", top, GoldPerPile(DungeonFloors+1), GoldPerPile(DungeonFloors+100))
	}
}

func pow108(n int) float64 {
	v := 1.0
	for range n {
		v *= 1.08
	}
	return v
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
	gold, stash, dex := g.Gold, len(g.Stash), g.Hero.Attr(Dex)
	g.killBoss(boss, &round{})
	if len(g.Stash) != stash+1 || g.Stash[stash].Rarity != Legendary || g.Gold <= gold || g.Hero.Attr(Dex) != dex+DungeonAttrBonus {
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
	raw, _ := json.Marshal(saveFile{9, time.Now(), g})
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
	raw, _ := json.Marshal(saveFile{9, time.Now(), g})
	os.WriteFile(path, raw, 0o644)
	got, _, err := Load(path)
	if err != nil || got.Dungeon != 0 || got.Reached != 42 || got.Hero.Bonus != 0 || !slices.Equal(got.ShortcutList(), g.Shortcuts) {
		t.Fatalf("dungeon %d reached %d shortcuts %v err %v", got.Dungeon, got.Reached, got.ShortcutList(), err)
	}
}
