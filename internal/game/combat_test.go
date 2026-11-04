package game

import "testing"

func TestDamageIsAtLeastOne(t *testing.T) {
	rng := NewRNG(1)
	for range 1000 {
		if d := damage(rng, 1, 100); d < 1 {
			t.Fatalf("damage %d", d)
		}
	}
}

func TestLevelUpRaisesStats(t *testing.T) {
	h := NewHero(Warrior)
	hp, atk := h.MaxHP(), h.Atk()
	if !h.gainXP(XPToNext(1)) {
		t.Fatal("expected level up")
	}
	if h.Level != 2 || h.MaxHP() <= hp || h.Atk() < atk {
		t.Fatalf("stats did not grow: %+v", h)
	}
}

func TestDeathReturnsToTownAndRestarts(t *testing.T) {
	g := departed(3)
	g.Hero.HP = 1
	m := &Monster{Kind: 0, HP: 99, MaxHP: 99, Atk: 100, Spd: 4, Awake: true}
	g.Exp.Level.Mons = []*Monster{m}
	m.Pos = g.Hero.Pos.Add(Pos{1, 0})
	for g.Exp != nil {
		g.monsterAttack(m, &round{})
	}
	if g.Deaths != 1 || g.Hero.HP != g.Hero.MaxHP() {
		t.Fatalf("deaths=%d hp=%d", g.Deaths, g.Hero.HP)
	}
	g.Depart()
	if g.Exp == nil || g.Exp.Num != 2 {
		t.Fatal("expected a new expedition after waiting in town")
	}
}

func TestHeroKillsMonsters(t *testing.T) {
	g := departed(11)
	g.Advance(5 * 60 * 1000)
	if g.Hero.Level < 2 {
		t.Fatalf("hero never leveled: %+v", g.Hero)
	}
}

func TestRoundIsOneLogLine(t *testing.T) {
	g := departed(5)
	g.Hero.HP = 1000
	bat := &Monster{Kind: 1, HP: 99, MaxHP: 99, Atk: 2, Spd: 8, Awake: true}
	bat.Pos = g.Hero.Pos.Add(Pos{1, 0})
	if !g.Exp.Level.CanStep(g.Hero.Pos, bat.Pos) {
		bat.Pos = g.Hero.Pos.Add(Pos{-1, 0})
	}
	g.Exp.Level.Mons = []*Monster{bat}
	if g.StepMs() >= CombatRoundMs {
		t.Fatalf("first combat round should not wait a full round, got %d", g.StepMs())
	}
	before := len(g.Log)
	g.step()
	if len(g.Log) != before+1 {
		t.Fatalf("expected one line, got %v", g.Log[before:])
	}
	if g.StepMs() != CombatRoundMs {
		t.Fatalf("combat step should use fixed pace, got %d", g.StepMs())
	}
}

func TestRepeatedLinesAreCounted(t *testing.T) {
	g := New(1, Warrior)
	before := len(g.Log)
	g.logf(LogGood, "x")
	g.logf(LogGood, "x")
	if len(g.Log) != before+1 || g.Log[len(g.Log)-1].Count != 2 {
		t.Fatalf("got %+v", g.Log[before:])
	}
}

func departed(seed uint64) *Game {
	g := New(seed, Warrior)
	g.Depart()
	return g
}
