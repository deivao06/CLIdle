package game

import (
	"fmt"
	"slices"

	"clidle/internal/lang"
)

// LabRooms is the number of rooms of a playground floor, the same 3x3 grid as the dungeon.
const LabRooms = gridRooms

// LabMon is a group of identical monsters placed in one room of the playground.
type LabMon struct {
	Kind  int
	Boss  bool
	Depth int
	Count int
}

// LabGear is one equipped item of the playground hero; Kind indexes ItemNames.
type LabGear struct {
	On      bool
	Kind    int
	Rarity  Rarity
	Depth   int
	Upgrade int
	Effects []ItemEffect
}

// LabFloor is one hand-made floor; a floor without rooms does not exist.
type LabFloor struct {
	Depth int
	Rooms [LabRooms]bool
	Up    int
	Down  int
	Mons  [LabRooms][]LabMon
}

// Scenario is everything the playground builds a fight from. Floors are sorted by depth.
type Scenario struct {
	Seed    uint64
	Class   Class
	Level   int
	Train   [NumTraits]int
	Potions int
	Gear    [NumSlots]LabGear
	Gambits []Gambit
	Mode    ExploreMode
	Floors  []LabFloor
}

// DefaultScenario is a level 1 warrior on floor 1, which has a single room with both stairs.
func DefaultScenario() Scenario {
	s := Scenario{Seed: 1, Level: 1, Potions: StartPotions}
	s.EditFloor(1).Toggle(LabRooms / 2)
	s.SetClass(Warrior)
	return s
}

// Floor is the floor at depth, or nil when it has no rooms.
func (s *Scenario) Floor(depth int) *LabFloor {
	for i := range s.Floors {
		if f := &s.Floors[i]; f.Depth == depth && f.Exists() {
			return f
		}
	}
	return nil
}

// EditFloor returns the floor at depth, adding an empty one when there is none.
func (s *Scenario) EditFloor(depth int) *LabFloor {
	i, found := slices.BinarySearchFunc(s.Floors, depth, func(f LabFloor, d int) int { return f.Depth - d })
	if !found {
		s.Floors = slices.Insert(s.Floors, i, LabFloor{Depth: depth})
	}
	return &s.Floors[i]
}

// Prune drops the floors that have no rooms.
func (s *Scenario) Prune() {
	s.Floors = slices.DeleteFunc(s.Floors, func(f LabFloor) bool { return !f.Exists() })
}

func (f *LabFloor) Exists() bool { return slices.Contains(f.Rooms[:], true) }

// Toggle turns a room on or off. The first room gets both stairs; stairs of a room turned off move to another
// room. It refuses, returning false, when the rooms left on would not touch each other.
func (f *LabFloor) Toggle(room int) bool {
	f.Rooms[room] = !f.Rooms[room]
	if f.Exists() && !f.Linked() {
		f.Rooms[room] = !f.Rooms[room]
		return false
	}
	first := slices.Index(f.Rooms[:], true)
	switch {
	case first < 0:
		f.Up, f.Down = 0, 0
	case f.Rooms[room] && first == room && !slices.Contains(f.Rooms[room+1:], true):
		f.Up, f.Down = room, room
	}
	if first >= 0 && !f.Rooms[f.Up] {
		f.Up = first
	}
	if first >= 0 && !f.Rooms[f.Down] {
		f.Down = first
	}
	return true
}

// SetClass switches the class, keeping only gear the class can use, and resets the rules to its defaults.
func (s *Scenario) SetClass(c Class) {
	s.Class = c
	gear := ClassGear[c]
	w := &s.Gear[Weapon]
	if !slices.Contains(gear.Weapon, w.Kind) || !w.On {
		*w = LabGear{On: true, Kind: gear.Weapon[0], Depth: max(1, w.Depth)}
	}
	if a := &s.Gear[Armor]; a.On && !slices.Contains(gear.Armor, a.Kind) {
		a.Kind = gear.Armor[0]
	}
	if !gear.Shield {
		s.Gear[Shield].On = false
	}
	s.Gambits = heroGambits(s.hero())
}

// SetLevel changes the hero level; abilities unlocked by going up get their default rule, like a level up.
func (s *Scenario) SetLevel(n int) {
	old := s.Level
	s.Level = n
	g := &Game{Hero: s.hero(), Config: Config{Gambits: slices.Clone(s.Gambits)}}
	g.unlockGambits(old, &round{})
	s.Gambits = g.Config.Gambits
}

// Item builds the item of a gear slot as the dungeon would roll it on its floor.
func (lg LabGear) Item(slot Slot, id uint64) *Item {
	if !lg.On {
		return nil
	}
	names := ItemNames[slot][lg.Kind]
	name := names[0].name
	for _, n := range names {
		if n.depth <= lg.Depth {
			name = n.name
		}
	}
	it := &Item{ID: id, Slot: slot, Kind: lg.Kind, Rarity: lg.Rarity, Name: name,
		Base: ItemRoll(slot, lg.Kind, lg.Depth, lg.Rarity, 0.5), Upgrade: lg.Upgrade, Effects: slices.Clone(lg.Effects)}
	it.Forged = lg.Upgrade * it.UpgradeStep()
	return it
}

func (s *Scenario) hero() *Hero {
	h := NewHero(s.Class)
	h.Level = s.Level
	h.Training = s.Train
	h.Potions = s.Potions
	for slot, lg := range s.Gear {
		h.Equip[slot] = lg.Item(Slot(slot), uint64(slot+1))
	}
	h.HP, h.MP = h.MaxHP(), h.MaxMP()
	return h
}

// Linked reports whether the rooms that are on form one block, so every room can be reached.
func (f *LabFloor) Linked() bool {
	start := slices.Index(f.Rooms[:], true)
	if start < 0 {
		return false
	}
	var seen [LabRooms]bool
	seen[start] = true
	queue := []int{start}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		for _, j := range []int{i - gridCols, i + gridCols, i - 1, i + 1} {
			sameRow := j/gridCols == i/gridCols
			if j < 0 || j >= LabRooms || (j == i-1 || j == i+1) && !sameRow || !f.Rooms[j] || seen[j] {
				continue
			}
			seen[j] = true
			queue = append(queue, j)
		}
	}
	return seen == f.Rooms
}

// MinDepth is the weakest floor a monster group can be built for.
func (m LabMon) MinDepth() int {
	if m.Boss {
		return BossEvery
	}
	return MonsterKinds[m.Kind].MinDepth
}

// Monster is the group's monster as it spawns, or nil for a boss, whose stats depend on the seed.
func (m LabMon) Monster() *Monster {
	if m.Boss {
		return nil
	}
	return newMonster(m.Kind, max(m.Depth, m.MinDepth()))
}

type LabEnd int

const (
	LabRunning LabEnd = iota
	LabDead
	LabOut
	LabDown
)

// LabRun is what the playground counts during a run.
type LabRun struct {
	End     LabEnd
	Steps   int
	Ms      int64
	Dealt   int
	Taken   int
	Kills   int
	Potions int
	down    bool
	sc      Scenario
}

// NewLab builds a game whose only expedition starts on the scenario's first floor and goes down only
// to floors that exist. Without floors there is no expedition.
func NewLab(s Scenario) *Game {
	h := s.hero()
	seed := s.Seed
	g := &Game{RNG: NewRNG(seed), Hero: h, Config: DefaultConfig(), lab: &LabRun{sc: s}, ExpNum: 1}
	g.Config.Gambits = slices.Clone(s.Gambits)
	g.Config.Mode = s.Mode
	s.Prune()
	if len(s.Floors) > 0 {
		g.Exp = &Expedition{Num: 1, Seed: seed, RNG: NewRNG(seed), StartEquip: h.Equip}
		g.enterLab(&s.Floors[0])
	}
	return g
}

// LabPreview is the scenario's game standing at the start of floor depth, to draw it; nil when it does not exist.
func LabPreview(s Scenario, depth int) *Game {
	f := s.Floor(depth)
	if f == nil {
		return nil
	}
	g := NewLab(s)
	g.enterLab(f)
	return g
}

// enterLab builds a playground floor. Its layout and monsters come from their own RNG, so every floor looks
// the same whatever happened above it; the fight uses the expedition's RNG.
func (g *Game) enterLab(f *LabFloor) {
	rng := NewRNG(g.lab.sc.Seed ^ uint64(f.Depth)*0x9e3779b97f4a7c15)
	l := carveLevel(rng, f.Depth, f.Rooms)
	l.Up = l.RandomFloor(rng, f.Up)
	l.Set(l.Up, StairsUp)
	l.Down = l.RandomFloor(rng, f.Down)
	l.Set(l.Down, StairsDown)
	for room, groups := range f.Mons {
		if !f.Rooms[room] {
			continue
		}
		for _, mg := range groups {
			depth := max(mg.Depth, mg.MinDepth())
			for range mg.Count {
				var m *Monster
				if mg.Boss {
					m = spawnBoss(rng, depth/BossEvery*BossEvery)
					m.Kind = mg.Kind
				} else {
					m = newMonster(mg.Kind, depth)
				}
				m.Pos = l.RandomFloor(rng, room)
				l.Mons = append(l.Mons, m)
			}
		}
	}
	var rooms []Room
	for i, r := range l.Rooms {
		if f.Rooms[i] {
			rooms = append(rooms, r)
		}
	}
	l.Rooms = rooms
	g.Exp.Level = l
	g.Exp.SecondWindUsed = false
	g.MaxDepth = max(g.MaxDepth, f.Depth)
	g.Hero.Pos = l.Up
	l.Reveal(g.Hero.Pos)
	g.logf(LogInfo, "%s", lang.T("log.arrive_floor", f.Depth))
}

// labDescend goes down to the next floor when it exists, or ends the run on the stairs.
func (g *Game) labDescend() {
	if f := g.lab.sc.Floor(g.Exp.Level.Depth + 1); f != nil {
		g.enterLab(f)
		return
	}
	g.lab.down = true
}

// Lab is the playground run, or nil in the real game.
func (g *Game) Lab() *LabRun { return g.lab }

// LabStep plays one step of the playground and stops it when the hero dies, leaves by < or reaches >.
func (g *Game) LabStep() {
	r := g.lab
	if r == nil || r.End != LabRunning || g.Exp == nil {
		return
	}
	e, h := g.Exp, g.Hero
	deaths, returns, kills, potions, hp := g.Deaths, g.Returns, g.Kills, h.Potions, h.HP
	prev := g.Events
	g.Events = nil
	ms := g.StepMs()
	g.Advance(ms)
	taken := 0
	for _, ev := range g.Events {
		switch ev.Kind {
		case EventHit:
			r.Dealt += ev.Value
		case EventHurt:
			taken += ev.Value
		}
	}
	r.Taken += taken
	g.Events = append(prev, g.Events...)
	g.Events = g.Events[max(0, len(g.Events)-maxEvents):]
	r.Steps++
	r.Ms += ms
	r.Kills += g.Kills - kills
	r.Potions += max(0, potions-h.Potions)
	switch {
	case g.Deaths > deaths:
		g.Exp, h.HP = e, 0
		r.End = LabDead
	case g.Returns > returns:
		g.Exp, h.HP = e, max(1, hp-taken)
		r.End = LabOut
	case r.down:
		r.End = LabDown
	}
	if r.End != LabRunning {
		g.logf(LogEpic, "%s", lang.T([...]string{"", "lab.end.dead", "lab.end.out", "lab.end.down"}[r.End])+
			lang.T("lab.end.stats", fmt.Sprintf("%d:%02d", r.Ms/60000, r.Ms/1000%60), r.Steps, r.Kills, r.Dealt, r.Taken, r.Potions))
	}
}
