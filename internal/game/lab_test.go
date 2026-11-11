package game

import (
	"encoding/json"
	"slices"
	"testing"
)

// fourRooms is the top-left 2x2 block with rats next to the entrance and a goblin at the exit.
func fourRooms() Scenario {
	s := DefaultScenario()
	f := &s.Floors[0]
	for _, i := range []int{0, 1, 3} {
		f.Rooms[i] = true
	}
	f.Up, f.Down = 0, 4
	f.Mons[1] = []LabMon{{Kind: 0, Depth: 1, Count: 2}}
	f.Mons[4] = []LabMon{{Kind: 3, Depth: 4, Count: 1}}
	return s
}

func runLab(t *testing.T, s Scenario) *Game {
	t.Helper()
	g := NewLab(s)
	for range 20000 {
		if g.Lab().End != LabRunning {
			return g
		}
		g.LabStep()
	}
	t.Fatal("the run never ended")
	return nil
}

func TestLabBuildsOnlyTheChosenRooms(t *testing.T) {
	s := fourRooms()
	s.Floors[0].Mons[1] = []LabMon{{Kind: 2, Depth: 3, Count: 3}}
	g := NewLab(s)
	l := g.Exp.Level
	if len(l.Rooms) != 4 {
		t.Fatalf("rooms = %d, want 4", len(l.Rooms))
	}
	if len(l.Mons) != 4 {
		t.Fatalf("monsters = %d, want 4", len(l.Mons))
	}
	want := newMonster(2, 3)
	for _, m := range l.Mons[:3] {
		if m.Kind != 2 || m.MaxHP != want.MaxHP || m.Atk != want.Atk {
			t.Fatalf("monster %+v, want kobold of floor 3 %+v", m, want)
		}
	}
	dist := l.distMap(l.Up)
	for _, p := range append([]Pos{l.Down}, l.Mons[0].Pos, l.Mons[3].Pos) {
		if dist[p.Y*MapW+p.X] < 0 {
			t.Fatalf("%v cannot be reached from the entrance", p)
		}
	}
	if g.Hero.Pos != l.Up || l.At(l.Up) != StairsUp || l.At(l.Down) != StairsDown {
		t.Fatal("stairs or hero misplaced")
	}
}

func TestLabGearIsBuiltAsSpecified(t *testing.T) {
	s := DefaultScenario()
	s.Gear[Weapon] = LabGear{On: true, Kind: Blade, Rarity: Epic, Depth: 20, Upgrade: 3,
		Effects: []ItemEffect{{LifeSteal, 10}, {Crit, 7}}}
	h := NewLab(s).Hero
	it := h.Equip[Weapon]
	if it.Rarity != Epic || it.Upgrade != 3 || it.Base != ItemRoll(Weapon, Blade, 20, Epic, 0.5) || it.Forged != 3*it.UpgradeStep() {
		t.Fatalf("weapon %+v", it)
	}
	if h.Effect(LifeSteal) != 10 || h.Effect(Crit) != 7 {
		t.Fatal("effects missing")
	}
	if h.Equip[Armor] != nil {
		t.Fatal("armor should be empty")
	}
}

func TestLabLinked(t *testing.T) {
	s := LabFloor{}
	s.Rooms[0], s.Rooms[2] = true, true
	if s.Linked() {
		t.Fatal("rooms 0 and 2 are not neighbours")
	}
	s.Rooms[1] = true
	if !s.Linked() {
		t.Fatal("0-1-2 is a row")
	}
	s = LabFloor{}
	s.Rooms[2], s.Rooms[3] = true, true
	if s.Linked() {
		t.Fatal("2 and 3 are on different rows")
	}
}

func TestLabStartsWithOneEmptyRoom(t *testing.T) {
	g := NewLab(DefaultScenario())
	if l := g.Exp.Level; len(l.Rooms) != 1 || len(l.Mons) != 0 || len(l.Items) != 0 {
		t.Fatalf("rooms %d, monsters %d, items %d", len(l.Rooms), len(l.Mons), len(l.Items))
	}
}

func TestLabEndsAtTheStairsWithoutMonsters(t *testing.T) {
	s := fourRooms()
	s.Floors[0].Mons = [LabRooms][]LabMon{}
	g := runLab(t, s)
	if g.Lab().End != LabDown || g.Exp == nil || g.Exp.Level.Depth != 1 || g.Hero.Pos != g.Exp.Level.Down {
		t.Fatalf("end %v", g.Lab().End)
	}
}

func TestLabEndsWhenTheHeroDies(t *testing.T) {
	s := fourRooms()
	s.Gambits = slices.DeleteFunc(s.Gambits, func(gb Gambit) bool { return gb.Do == ActRetreat })
	s.Floors[0].Mons[1] = []LabMon{{Kind: 9, Depth: 40, Count: 3}}
	g := runLab(t, s)
	r := g.Lab()
	if r.End != LabDead || g.Exp == nil || g.Hero.HP != 0 || r.Taken == 0 {
		t.Fatalf("run %+v", r)
	}
}

func TestLabIsReproducible(t *testing.T) {
	s := fourRooms()
	s.Floors[0].Mons[4] = []LabMon{{Kind: 0, Boss: true, Depth: 10, Count: 1}}
	s.Level = 15
	a, _ := json.Marshal(runLab(t, s))
	b, _ := json.Marshal(runLab(t, s))
	if string(a) != string(b) {
		t.Fatal("same scenario, different result")
	}
}

func TestLabLevelUnlocksRules(t *testing.T) {
	s := DefaultScenario()
	s.SetClass(Cleric)
	n := len(s.Gambits)
	s.SetLevel(15)
	if len(s.Gambits) != n+1 || !slices.ContainsFunc(s.Gambits, func(gb Gambit) bool { return gb.Do == ActBless }) {
		t.Fatal("the bless rule should be added")
	}
	s.SetClass(Rogue)
	if s.Gear[Weapon].Kind != Dagger || s.Gear[Shield].On {
		t.Fatal("rogue gear")
	}
}

func TestLabFloorToggle(t *testing.T) {
	var f LabFloor
	if !f.Toggle(4) || f.Up != 4 || f.Down != 4 {
		t.Fatalf("first room gets both stairs: %+v", f)
	}
	if f.Toggle(0) {
		t.Fatal("room 0 does not touch room 4")
	}
	f.Toggle(1)
	f.Toggle(4)
	if f.Rooms[4] || f.Up != 1 || f.Down != 1 {
		t.Fatalf("stairs move with the room: %+v", f)
	}
	f.Toggle(1)
	if f.Exists() {
		t.Fatal("no rooms left")
	}
}

func TestLabGoesDownOnlyToFloorsThatExist(t *testing.T) {
	s := DefaultScenario()
	s.EditFloor(3).Toggle(0)
	s.EditFloor(2).Toggle(8)
	if g := runLab(t, s); g.Lab().End != LabDown || g.Exp.Level.Depth != 3 {
		t.Fatalf("end %v on floor %d", g.Lab().End, g.Exp.Level.Depth)
	}
	s.EditFloor(2).Toggle(8)
	if g := runLab(t, s); g.Lab().End != LabDown || g.Exp.Level.Depth != 1 {
		t.Fatalf("floor 2 has no rooms: end on floor %d", g.Exp.Level.Depth)
	}
	if NewLab(Scenario{Seed: 7, Level: 1}).Exp != nil {
		t.Fatal("no floors, no expedition")
	}
}

func TestLabFloorLooksTheSameWhateverHappensAbove(t *testing.T) {
	s := fourRooms()
	s.EditFloor(2).Toggle(4)
	a := LabPreview(s, 2).Exp.Level
	s.Floors[0].Mons[1] = nil
	b := LabPreview(s, 2).Exp.Level
	if string(a.Tiles) != string(b.Tiles) || a.Up != b.Up {
		t.Fatal("floor 2 changed")
	}
}
