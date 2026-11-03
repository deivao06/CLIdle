package game

import (
	"fmt"
	"strings"

	"clidle/internal/lang"
)

func (g *Game) ready(a Ability) bool {
	return g.Hero.Has(a) && g.Hero.MP >= AbilityInfo[a].Mana
}

func (g *Game) useMana(a Ability) { g.Hero.MP -= AbilityInfo[a].Mana }

func (g *Game) healSpell(r *round) bool {
	h := g.Hero
	if !g.ready(Heal) || h.HP >= h.MaxHP() {
		return false
	}
	g.useMana(Heal)
	heal := min(h.MaxHP()*HealPct/100+h.Attr(Int), h.MaxHP()-h.HP)
	h.HP += heal
	r.hero = lang.T("spell.heal.log", heal)
	return true
}

func (g *Game) bless(r *round) bool {
	if g.Exp.Blessed > 0 || !g.ready(Bless) {
		return false
	}
	g.useMana(Bless)
	g.Exp.Blessed = BlessRounds
	r.hero = lang.T("spell.bless.log")
	return true
}

// castTarget prefers the nearest monster in range that is not asleep.
func (g *Game) castTarget() *Monster {
	l := g.Exp.Level
	h := g.Hero.Pos
	var best *Monster
	for _, m := range l.Mons {
		d := m.Pos.Dist(h)
		if d > SpellRange || !l.Visible(h, m.Pos) || m.Boss && !m.Awake && !g.canBeat(m) {
			continue
		}
		if best == nil || (m.Asleep == 0) != (best.Asleep == 0) && m.Asleep == 0 ||
			(m.Asleep == 0) == (best.Asleep == 0) && d < best.Pos.Dist(h) {
			best = m
		}
	}
	return best
}

func (g *Game) cast(r *round) bool {
	if !g.ready(MagicMissile) {
		return false
	}
	t := g.castTarget()
	if t == nil {
		return false
	}
	h := g.Hero
	name := lang.T(t.Info().Name)
	switch {
	case g.blink(r):
	case g.ready(Sleep) && t.Awake && t.Asleep == 0 && t.HP*100 > t.MaxHP*SleepMinHPPct && g.awakeInRange() >= 2:
		g.useMana(Sleep)
		t.Asleep = SleepRounds
		if t.Boss {
			t.Asleep = SleepBossRounds
		}
		r.hero = lang.T("spell.sleep.log", name)
	case g.ready(Fireball) && len(g.around(t)) >= 2:
		g.useMana(Fireball)
		var hits []string
		for _, m := range g.around(t) {
			d := g.spellDamage(Fireball, m)
			hits = append(hits, fmt.Sprintf("%s (%d)", lang.T(m.Info().Name), d))
			g.hurt(m, d, r)
		}
		r.hero = lang.T("spell.fireball.log", strings.Join(hits, ", "))
	case g.ready(FrostRay) && t.Spd > h.Spd():
		g.useMana(FrostRay)
		d := g.spellDamage(FrostRay, t)
		t.Slowed = FrostSlowRounds
		r.hero = lang.T("spell.frost_ray.log", name, d)
		g.hurt(t, d, r)
	default:
		g.useMana(MagicMissile)
		d := g.spellDamage(MagicMissile, t)
		r.hero = lang.T("spell.magic_missile.log", name, d)
		g.hurt(t, d, r)
	}
	return true
}

func (g *Game) spellDamage(a Ability, m *Monster) int {
	m.Awake = true
	return damage(g.Exp.RNG, g.Hero.spellAtk(a), m.Def)
}

func (h *Hero) spellAtk(a Ability) int { return SpellPower[a] + h.Attr(Int) + h.bonus(Weapon) }

// SpellAtk is the attack of the strongest damage spell the hero knows, or 0.
func (h *Hero) SpellAtk() int {
	best := 0
	for a := range NumAbilities {
		if SpellPower[a] > 0 && h.Has(a) {
			best = max(best, h.spellAtk(a))
		}
	}
	return best
}

func (g *Game) awakeInRange() int {
	l := g.Exp.Level
	h := g.Hero.Pos
	n := 0
	for _, m := range l.Mons {
		if m.Asleep == 0 && m.Pos.Dist(h) <= SpellRange && l.Visible(h, m.Pos) {
			n++
		}
	}
	return n
}

// around lists the monsters within one tile of t, t included.
func (g *Game) around(t *Monster) []*Monster {
	var out []*Monster
	for _, m := range g.Exp.Level.Mons {
		if m.Pos.Dist(t.Pos) <= 1 {
			out = append(out, m)
		}
	}
	return out
}

// blink jumps to the visible tile in range farthest from the nearest monster.
func (g *Game) blink(r *round) bool {
	h := g.Hero
	l := g.Exp.Level
	if !g.ready(Blink) || len(g.adjacentMonsters()) < 2 || h.HP*100 >= h.MaxHP()*BlinkHPPct {
		return false
	}
	best, bestD := h.Pos, 1
	for y := h.Pos.Y - SpellRange; y <= h.Pos.Y+SpellRange; y++ {
		for x := h.Pos.X - SpellRange; x <= h.Pos.X+SpellRange; x++ {
			p := Pos{x, y}
			if !l.At(p).Walkable() || !l.Visible(h.Pos, p) || l.MonsterAt(p) != nil {
				continue
			}
			near := MapW
			for _, m := range l.Mons {
				near = min(near, m.Pos.Dist(p))
			}
			if near > bestD {
				best, bestD = p, near
			}
		}
	}
	if best == h.Pos {
		return false
	}
	g.useMana(Blink)
	h.Pos = best
	l.Reveal(h.Pos)
	r.hero = lang.T("spell.blink.log")
	return true
}
