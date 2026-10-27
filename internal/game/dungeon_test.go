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
	g := departed(42)
	g.Auto = true
	g.Advance(10 * 60 * 1000)
	if g.MaxDepth < 3 {
		t.Fatalf("hero stuck, max depth %d", g.MaxDepth)
	}
}

func TestAdvanceChunkingIsDeterministic(t *testing.T) {
	a, b := departed(7), departed(7)
	a.Auto, b.Auto = true, true
	a.Advance(120000)
	for range 2400 {
		b.Advance(50)
	}
	if a.Hero.Pos != b.Hero.Pos || a.Hero.XP != b.Hero.XP || a.ExpNum != b.ExpNum || len(a.Log) != len(b.Log) {
		t.Fatalf("diverged: %+v vs %+v", a.Hero, b.Hero)
	}
}
