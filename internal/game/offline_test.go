package game

import (
	"testing"
	"time"
)

func TestCatchUpMatchesOnlinePlay(t *testing.T) {
	a, b := departed(5), departed(5)
	a.Auto, b.Auto = true, true
	a.CatchUp(20*time.Minute, nil)
	for range 20 * 60 * 20 {
		b.Advance(50)
	}
	if a.Gold != b.Gold || a.Hero.XP != b.Hero.XP || a.ExpNum != b.ExpNum {
		t.Fatalf("offline differs from online: gold %d/%d xp %d/%d", a.Gold, b.Gold, a.Hero.XP, b.Hero.XP)
	}
}

func TestCatchUpIsCapped(t *testing.T) {
	g := New(1, Warrior)
	s := g.CatchUp(48*time.Hour, nil)
	if !s.Capped || s.Eventful() {
		t.Fatalf("summary %+v", s)
	}
}

func TestTwelveHoursIsFastEnough(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	g := departed(8)
	g.Auto = true
	start := time.Now()
	s := g.CatchUp(OfflineCap, nil)
	took := time.Since(start)
	t.Logf("12h simulated in %v: %+v", took, s)
	if took > 15*time.Second {
		t.Fatalf("too slow: %v", took)
	}
}
