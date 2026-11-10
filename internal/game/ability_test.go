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
	g := heroAt(Warrior, AbilityInfo[SecondWind].Level, 1)
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
	g := heroAt(Warrior, AbilityInfo[Whirlwind].Level, 1)
	g.Exp.Attacks = PowerStrikeEvery - 1
	m1, m2 := g.besideHero(dummy()), g.besideHero(dummy())
	for g.Exp.Attacks < PowerStrikeEvery {
		g.heroAttack(m1, &round{})
	}
	if m1.HP == m1.MaxHP || m2.HP == m2.MaxHP {
		t.Fatalf("whirlwind hp %d %d", m1.HP, m2.HP)
	}
}

func TestSneakAttackOnSleeper(t *testing.T) {
	rogue, warrior := heroAt(Rogue, 1, 1), heroAt(Warrior, 1, 1)
	warrior.Hero.Training[Str] = rogue.Hero.Atk() - warrior.Hero.Atk()
	mr, mw := rogue.besideHero(dummy()), warrior.besideHero(dummy())
	mr.Awake, mw.Awake = false, false
	for mr.HP == mr.MaxHP {
		rogue.heroAttack(mr, &round{})
		warrior.heroAttack(mw, &round{})
	}
	if mr.MaxHP-mr.HP <= mw.MaxHP-mw.HP {
		t.Fatalf("sneak %d, plain %d", mr.MaxHP-mr.HP, mw.MaxHP-mw.HP)
	}
}

func TestDodgeAvoidsSomeHits(t *testing.T) {
	g := heroAt(Rogue, AbilityInfo[Dodge].Level, 1)
	g.Hero.HP = 1 << 30
	m := g.besideHero(&Monster{HP: 1, MaxHP: 1, Atk: 1, Spd: 1, Awake: true})
	r := &round{}
	for range 1000 {
		g.monsterAttack(m, r)
	}
	if n := r.of(m).misses; n < 170 || n > 400 {
		t.Fatalf("misses %d", n)
	}
}

func TestNoseFindsMoreGold(t *testing.T) {
	gold := func(nose bool) (sum int) {
		for seed := range uint64(200) {
			rng := NewRNG(seed)
			l := GenerateLevel(rng, 5)
			l.placeLoot(rng, l.RoomAt(l.Up), nose)
			for _, f := range l.Items {
				sum += f.Gold
			}
		}
		return sum
	}
	if plain, nose := gold(false), gold(true); nose*10 < plain*12 {
		t.Fatalf("nose %d, plain %d", nose, plain)
	}
}

func TestManaRegenIsChunkIndependent(t *testing.T) {
	a, b := heroAt(Mage, 1, 3), heroAt(Mage, 1, 3)
	a.Hero.MP, b.Hero.MP = 0, 0
	a.Advance(120000)
	for range 2400 {
		b.Advance(50)
	}
	if a.Hero.MP != b.Hero.MP || a.Exp.ManaAcc != b.Exp.ManaAcc || a.Hero.MP == 0 {
		t.Fatalf("mp %d/%d acc %d/%d", a.Hero.MP, b.Hero.MP, a.Exp.ManaAcc, b.Exp.ManaAcc)
	}
}
