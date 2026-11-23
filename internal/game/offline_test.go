package game

import (
	"slices"
	"testing"
	"time"
)

func TestCatchUpMatchesOnlinePlay(t *testing.T) {
	for c := range NumClasses {
		a, b := heroAt(c, 15, 5), heroAt(c, 15, 5)
		a.Auto, b.Auto = true, true
		a.Guild.CatchUp(20*time.Minute, nil)
		for range 20 * 60 * 20 {
			b.Guild.Advance(50)
		}
		if a.Guild.Gold != b.Guild.Gold || a.Hero.XP != b.Hero.XP || a.Hero.MP != b.Hero.MP || a.ExpNum != b.ExpNum {
			t.Fatalf("%s offline differs from online: gold %d/%d xp %d/%d", c.Name(), a.Guild.Gold, b.Guild.Gold, a.Hero.XP, b.Hero.XP)
		}
	}
}

func TestCatchUpIsCapped(t *testing.T) {
	g := New(1, Warrior)
	s := g.Guild.CatchUp(48*time.Hour, nil)
	if !s.Capped || s.Eventful() {
		t.Fatalf("summary %+v", s)
	}
}

func TestTwelveHoursIsFastEnough(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	g := New(8, Rogue)
	g.StartExpedition()
	g.Auto = true
	start := time.Now()
	s := g.Guild.CatchUp(OfflineCap, nil)
	took := time.Since(start)
	t.Logf("12h simulated in %v: %+v", took, s)
	if took > 15*time.Second {
		t.Fatalf("too slow: %v", took)
	}
}

// guildOf builds a guild of heroes on auto, one of each class, all at the given level.
func guildOf(level int, seed uint64) *Guild {
	gd := New(seed, Warrior).Guild
	for c := Mage; c < NumClasses; c++ {
		h := New(seed+uint64(c), c)
		h.Guild = gd
		gd.Heroes = append(gd.Heroes, h)
	}
	for _, h := range gd.Heroes {
		h.Auto, h.Hero.Level = true, level
	}
	return gd
}

func TestGuildChunkingIsDeterministic(t *testing.T) {
	a, b := guildOf(12, 31415926), guildOf(12, 31415926)
	a.Advance(10 * 60 * 1000)
	for range 10 * 60 * 20 {
		b.Advance(50)
	}
	if a.Gold != b.Gold {
		t.Fatalf("gold %d/%d", a.Gold, b.Gold)
	}
	for i := range a.Heroes {
		x, y := a.Heroes[i], b.Heroes[i]
		if x.ExpNum != y.ExpNum || x.Hero.XP != y.Hero.XP || x.Hero.HP != y.Hero.HP || x.Queue != y.Queue {
			t.Fatalf("hero %d differs: exp %d/%d xp %d/%d", i, x.ExpNum, y.ExpNum, x.Hero.XP, y.Hero.XP)
		}
	}
}

func TestMissionTableTakesTurns(t *testing.T) {
	gd := guildOf(12, 27182818)
	waited, most := false, 0
	for range 30 * 60 * 10 {
		gd.Advance(100)
		most = max(most, gd.Running())
		for _, h := range gd.Heroes {
			waited = waited || h.Queue != 0
		}
	}
	if most != MissionSlots || !waited {
		t.Fatalf("at most %d out at once, someone waited %v", most, waited)
	}
	for _, h := range gd.Heroes {
		if h.ExpNum == 0 {
			t.Fatalf("%s never left", h.Hero.Class.Name())
		}
	}
	idle := gd.Heroes[0]
	for idle.Exp != nil || gd.Running() < MissionSlots {
		gd.Advance(100)
		idle = gd.Heroes[slices.IndexFunc(gd.Heroes, func(h *Game) bool { return h.Exp == nil })]
	}
	if idle.Depart() {
		t.Fatal("departed with every mission taken")
	}
}

func TestTwelveHoursForAGuild(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	gd := guildOf(1, 8)
	start := time.Now()
	s := gd.CatchUp(OfflineCap, nil)
	took := time.Since(start)
	t.Logf("12h for %d heroes in %v: %+v", len(gd.Heroes), took, s)
}
