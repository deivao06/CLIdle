package game

import (
	"testing"
	"time"
)

func TestLevelFullyConnected(t *testing.T) {
	for seed := range uint64(200) {
		l := GenerateLevel(NewRNG(seed), 1)
		dist := l.distMap(l.Up)
		for y := range MapH {
			for x := range MapW {
				p := Pos{x, y}
				if l.At(p).Walkable() && dist[y*MapW+x] < 0 {
					t.Fatalf("seed %d: tile %v unreachable", seed, p)
				}
			}
		}
		if l.Up == l.Down {
			t.Fatalf("seed %d: stairs overlap", seed)
		}
	}
}

func TestHeroExploresAndDescends(t *testing.T) {
	g := departed(42)
	g.Auto = true
	g.Advance(10 * 60 * 1000)
	if g.MaxDepth < 3 {
		t.Fatalf("hero stuck, max depth %d", g.MaxDepth)
	}
}

func TestAdvanceChunkingIsDeterministic(t *testing.T) {
	for c := range NumClasses {
		a, b := New(7, c), New(7, c)
		a.Depart()
		b.Depart()
		a.Auto, b.Auto = true, true
		a.Advance(120000)
		for range 2400 {
			b.Advance(50)
		}
		if a.Hero.Pos != b.Hero.Pos || a.Hero.XP != b.Hero.XP || a.ExpNum != b.ExpNum || len(a.Log) != len(b.Log) {
			t.Fatalf("class %d diverged: %+v vs %+v", c, a.Hero, b.Hero)
		}
	}
}

func TestRandomFloorFullRoomDoesNotHang(t *testing.T) {
	l := GenerateLevel(NewRNG(1), 1)
	r := l.Rooms[0]
	for y := r.Y + 1; y <= r.Y+r.H-2; y++ {
		for x := r.X + 1; x <= r.X+r.W-2; x++ {
			if p := (Pos{x, y}); l.At(p) == Floor && l.MonsterAt(p) == nil {
				l.Mons = append(l.Mons, &Monster{Pos: p, HP: 1})
			}
		}
	}
	done := make(chan Pos)
	go func() { done <- l.RandomFloor(NewRNG(2), 0) }()
	select {
	case p := <-done:
		if l.At(p) != Floor || l.MonsterAt(p) != nil || l.ItemAt(p) >= 0 {
			t.Fatalf("got occupied tile %v", p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RandomFloor hangs on a full room")
	}
}
