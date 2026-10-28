package game

import "testing"

// spotAt finds a free walkable tile visible from the hero at exactly dist, next to near when given.
func (g *Game) spotAt(dist int, near *Pos) Pos {
	l := g.Exp.Level
	h := g.Hero.Pos
	for y := h.Y - dist; y <= h.Y+dist; y++ {
		for x := h.X - dist; x <= h.X+dist; x++ {
			p := Pos{x, y}
			if p.Dist(h) == dist && l.At(p).Walkable() && l.Visible(h, p) && l.MonsterAt(p) == nil && (near == nil || p.Dist(*near) == 1) {
				return p
			}
		}
	}
	panic("no spot")
}

func TestMageCastsFromRange(t *testing.T) {
	g := heroAt(Mage, 1, 1)
	m := dummy()
	m.Pos = g.spotAt(4, nil)
	g.Exp.Level.Mons = []*Monster{m}
	start := g.Hero.Pos
	g.step()
	if g.Hero.Pos != start || m.HP == m.MaxHP || g.Hero.MP >= g.Hero.MaxMP() {
		t.Fatalf("pos %v→%v hp %d mp %d", start, g.Hero.Pos, m.HP, g.Hero.MP)
	}
	if g.StepMs() != CombatRoundMs {
		t.Fatal("ranged combat should use combat pace")
	}
}

func TestMageWithoutManaMelees(t *testing.T) {
	g := heroAt(Mage, 1, 1)
	g.Hero.MP = 0
	m := g.besideHero(dummy())
	for i := 0; i < 3 && m.HP == m.MaxHP; i++ {
		g.step()
	}
	if m.HP == m.MaxHP || g.Hero.MP != 0 {
		t.Fatalf("hp %d mp %d", m.HP, g.Hero.MP)
	}
}

func TestSleepSkipsMonsterTurns(t *testing.T) {
	g := heroAt(Mage, 4, 1)
	m := g.besideHero(&Monster{HP: 9, MaxHP: 9, Atk: 100, Spd: 1, Awake: true, Asleep: 3})
	r := &round{}
	g.monstersAct(r, false)
	if len(r.mons) != 0 || m.Asleep != 2 {
		t.Fatalf("sleeping monster acted, asleep %d", m.Asleep)
	}
}

func TestBlinkNeedsDestination(t *testing.T) {
	g := heroAt(Mage, 10, 1)
	l := g.Exp.Level
	g.besideHero(dummy())
	g.besideHero(dummy())
	for i := range l.Tiles {
		p := Pos{i % MapW, i / MapW}
		if p != g.Hero.Pos && l.MonsterAt(p) == nil {
			l.Tiles[i] = byte(Rock)
		}
	}
	g.Hero.HP = 1
	mp := g.Hero.MP
	if g.blink(&round{}) || g.Hero.MP != mp {
		t.Fatal("blinked with nowhere to go")
	}
}

func TestFireballHitsNeighbors(t *testing.T) {
	g := heroAt(Mage, 7, 1)
	a := &Monster{HP: 999, MaxHP: 4000, Spd: 1, Awake: true}
	a.Pos = g.spotAt(3, nil)
	g.Exp.Level.Mons = []*Monster{a}
	b := &Monster{HP: 999, MaxHP: 4000, Spd: 1, Awake: true}
	b.Pos = g.spotAt(3, &a.Pos)
	g.Exp.Level.Mons = append(g.Exp.Level.Mons, b)
	if !g.cast(&round{}) || a.HP == 999 || b.HP == 999 {
		t.Fatalf("fireball hp %d %d", a.HP, b.HP)
	}
}
