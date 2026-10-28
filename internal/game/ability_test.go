package game

import "testing"

func heroAt(class Class, level int, seed uint64) *Game {
	g := New(seed, class)
	g.Hero.Level = level
	g.StartExpedition()
	g.Exp.Level.Mons = nil
	return g
}

func (g *Game) besideHero(m *Monster) *Monster {
	l := g.Exp.Level
	for _, d := range Dirs8 {
		p := g.Hero.Pos.Add(d)
		if l.CanStep(g.Hero.Pos, p) && l.MonsterAt(p) == nil {
			m.Pos = p
			l.Mons = append(l.Mons, m)
			return m
		}
	}
	panic("no room beside hero")
}

func dummy() *Monster { return &Monster{HP: 999, MaxHP: 999, Spd: 1, Awake: true} }

func TestPowerStrikeDoublesEveryN(t *testing.T) {
	a, b := heroAt(Warrior, 1, 1), heroAt(Warrior, 1, 1)
	a.Exp.Attacks = PowerStrikeEvery - 1
	ma, mb := a.besideHero(dummy()), b.besideHero(dummy())
	for ma.HP == ma.MaxHP {
		a.heroAttack(ma, &round{})
		b.heroAttack(mb, &round{})
	}
	if lost, normal := ma.MaxHP-ma.HP, mb.MaxHP-mb.HP; lost != 2*normal {
		t.Fatalf("power strike %d, normal %d", lost, normal)
	}
}

func TestSecondWindOncePerFloor(t *testing.T) {
	g := heroAt(Warrior, 5, 1)
	g.Hero.HP = 1
	if !g.secondWind(&round{}) || g.Hero.HP <= 1 {
		t.Fatal("second wind did not heal")
	}
	g.Hero.HP = 1
	if g.secondWind(&round{}) {
		t.Fatal("second wind twice on one floor")
	}
	g.enterLevel(2)
	g.Hero.HP = 1
	if !g.secondWind(&round{}) {
		t.Fatal("second wind did not reset on a new floor")
	}
}

func TestWhirlwindHitsAllAdjacent(t *testing.T) {
	g := heroAt(Warrior, 10, 1)
	g.Exp.Attacks = PowerStrikeEvery - 1
	m1, m2 := g.besideHero(dummy()), g.besideHero(dummy())
	for g.Exp.Attacks < PowerStrikeEvery {
		g.heroAttack(m1, &round{})
	}
	if m1.HP == m1.MaxHP || m2.HP == m2.MaxHP {
		t.Fatalf("whirlwind hp %d %d", m1.HP, m2.HP)
	}
}
