package game

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTripKeepsSimulationIdentical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(21)
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
	if g.Hero.Pos != loaded.Hero.Pos || g.Hero.XP != loaded.Hero.XP || g.Gold != loaded.Gold || len(g.Log) != len(loaded.Log) {
		t.Fatalf("diverged after load: %+v vs %+v", g.Hero, loaded.Hero)
	}
}

func TestLoadMissingSave(t *testing.T) {
	g, _, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if g != nil || err != nil {
		t.Fatalf("got %v %v", g, err)
	}
}
