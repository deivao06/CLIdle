package game

import (
	"testing"

	"clidle/internal/lang"
)

func TestStreakOddsAndRecord(t *testing.T) {
	var s, best Streak
	for i := range LuckStreakMin {
		if shown := s.add(0.25, &best); shown != (i == LuckStreakMin-1) {
			t.Fatalf("miss %d shown %v", i+1, shown)
		}
	}
	if got := s.Odds(); got != "(1 in 64 · 1.6%)" {
		t.Fatalf("odds %q", got)
	}
	lang.Set("pt")
	defer lang.Set("en")
	s.add(0.01, &best)
	if got := s.Odds(); got != "(1 em 6.400 · 0,016%)" || best != s {
		t.Fatalf("odds %q best %+v", got, best)
	}
	s = Streak{}
	for range LuckStreakMin {
		s.add(0.5, &best)
	}
	if best.Run != LuckStreakMin+1 {
		t.Fatalf("a likelier streak replaced the record: %+v", best)
	}
}

func TestHitEndsTheMissStreak(t *testing.T) {
	g := heroAt(Warrior, 1, 1)
	m := g.besideHero(dummy())
	g.Exp.HeroMisses = Streak{Run: 5, Chance: 0.001}
	for g.Exp.HeroMisses.Run >= 5 {
		g.heroAttack(m, &round{})
	}
	if g.Exp.HeroMisses.Run != 0 {
		t.Fatalf("streak %+v after a hit", g.Exp.HeroMisses)
	}
}
