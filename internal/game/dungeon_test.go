package game

import "testing"

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
	g := New(42)
	g.Advance(10 * 60 * 1000)
	if g.Exp.Level.Depth < 3 {
		t.Fatalf("hero stuck on depth %d", g.Exp.Level.Depth)
	}
}

func TestAdvanceChunkingIsDeterministic(t *testing.T) {
	a, b := New(7), New(7)
	a.Advance(120000)
	for range 2400 {
		b.Advance(50)
	}
	if a.Hero.Pos != b.Hero.Pos || a.Exp.Level.Depth != b.Exp.Level.Depth {
		t.Fatalf("diverged: %v/%d vs %v/%d", a.Hero.Pos, a.Exp.Level.Depth, b.Hero.Pos, b.Exp.Level.Depth)
	}
}
