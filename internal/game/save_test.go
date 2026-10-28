package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTripKeepsSimulationIdentical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := departed(21)
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
