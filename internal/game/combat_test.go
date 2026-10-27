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
	h := NewHero()
	hp, atk := h.MaxHP(), h.Atk()
	if !h.gainXP(XPToNext(1)) {
		t.Fatal("expected level up")
	}
	if h.Level != 2 || h.MaxHP() <= hp || h.Atk() < atk {
		t.Fatalf("stats did not grow: %+v", h)
	}
}

func TestDeathReturnsToTownAndRestarts(t *testing.T) {
	g := New(3)
	g.Hero.HP = 1
	m := &Monster{Kind: 0, HP: 99, MaxHP: 99, Atk: 100, Spd: 4, Awake: true}
	g.Exp.Level.Mons = []*Monster{m}
	m.Pos = g.Hero.Pos.Add(Pos{1, 0})
	for g.Exp != nil {
		g.monsterAttack(m)
	}
	if g.Deaths != 1 || g.Hero.HP != g.Hero.MaxHP() {
		t.Fatalf("deaths=%d hp=%d", g.Deaths, g.Hero.HP)
	}
	g.Advance(TownWaitMs)
	if g.Exp == nil || g.Exp.Num != 2 {
		t.Fatal("expected a new expedition after waiting in town")
	}
}

func TestHeroKillsMonsters(t *testing.T) {
	g := New(11)
	g.Advance(5 * 60 * 1000)
	if g.Hero.Level < 2 {
		t.Fatalf("hero never leveled: %+v", g.Hero)
	}
}
