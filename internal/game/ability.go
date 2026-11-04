package game

import (
	"fmt"
	"strings"

	"clidle/internal/lang"
)

type Ability int

const (
	PowerStrike Ability = iota
	SecondWind
	Whirlwind
	SneakAttack
	Dodge
	TreasureNose
	MagicMissile
	Sleep
	Fireball
	Blink
	FrostRay
	Heal
	Bless
	TurnUndead
	NumAbilities
)

func (a Ability) Name() string { return lang.T(AbilityInfo[a].Key) }
func (a Ability) Desc() string { return lang.T(AbilityInfo[a].Key + ".desc") }

// ClassAbilities lists a class's abilities in unlock order.
func ClassAbilities(c Class) []Ability {
	var out []Ability
	for a := range NumAbilities {
		if AbilityInfo[a].Class == c {
			out = append(out, a)
		}
	}
	return out
}

func (h *Hero) Has(a Ability) bool {
	info := AbilityInfo[a]
	return h.Class == info.Class && h.Level >= info.Level
}

func (g *Game) regenMana() {
	h, e := g.Hero, g.Exp
	if h.MaxMP() == 0 {
		return
	}
	e.ManaAcc += ManaRegenPerStep + h.Attr(Int)*ManaRegenPerInt
	for e.ManaAcc >= ManaRegenDiv {
		e.ManaAcc -= ManaRegenDiv
		h.MP = min(h.MaxMP(), h.MP+1)
	}
}

func (g *Game) adjacentMonsters() []*Monster {
	l := g.Exp.Level
	var out []*Monster
	for _, m := range l.Mons {
		if m.Pos.Dist(g.Hero.Pos) == 1 && l.CanStep(g.Hero.Pos, m.Pos) {
			out = append(out, m)
		}
	}
	return out
}

func (g *Game) powerStrike(m *Monster, r *round) {
	targets := []*Monster{m}
	if adj := g.adjacentMonsters(); g.Hero.Has(Whirlwind) && len(adj) >= 2 {
		targets = adj
	}
	var hits []string
	g.strong = true
	defer func() { g.strong = false }()
	for _, t := range targets {
		d := 2 * damage(g.Exp.RNG, g.Hero.Atk(), t.Def)
		hits = append(hits, fmt.Sprintf("%s (%d)", lang.T(t.Info().Name), d))
		t.Awake = true
		g.hurt(t, d, r)
	}
	key := "ability.power_strike.log"
	if len(targets) > 1 {
		key = "ability.whirlwind.log"
	}
	r.hero = lang.T(key, strings.Join(hits, ", "))
}

func (g *Game) secondWind(r *round) bool {
	h, e := g.Hero, g.Exp
	if !h.Has(SecondWind) || e.SecondWindUsed || h.HP*100 >= h.MaxHP()*SecondWindPct {
		return false
	}
	heal := min(h.MaxHP()*SecondWindHealPct/100, h.MaxHP()-h.HP)
	h.HP += heal
	e.SecondWindUsed = true
	r.hero = lang.T("ability.second_wind.log", heal)
	return true
}
