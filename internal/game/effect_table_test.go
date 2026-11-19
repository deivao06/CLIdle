package game

import (
	"fmt"
	"math"
	"testing"
)

// effectProbe measures one effect where the game uses it: the gain the hero gets with that effect on.
type effectProbe struct {
	kind  Effect
	class Class
	gain  func(g *Game) int
	want  func(g *Game, pct int) int
}

var effectProbes = []effectProbe{
	{LifeSteal, Warrior, func(g *Game) int {
		g.Hero.HP = 1
		g.hurt(g.besideHero(dummy()), 100, &round{})
		return g.Hero.HP - 1
	}, func(g *Game, pct int) int { return min(100*pct/100, g.Hero.MaxHP()-1) }},
	{Crit, Warrior, func(g *Game) int {
		return int(math.Round(g.Hero.CritChance(0) * 100))
	}, func(g *Game, pct int) int { return int(math.Round(CritChance(g.Hero.Attr(Dex), 0)*100)) + pct }},
	{Vigor, Warrior, func(g *Game) int { return g.Hero.MaxHP() }, nil},
	{GoldFind, Warrior, func(g *Game) int {
		l := g.Exp.Level
		l.Items = append(l.Items, &FloorItem{Pos: g.Hero.Pos, Gold: 100})
		g.pickup(&round{})
		m := g.besideHero(dummy())
		m.HP, m.XP = 1, 100*GoldKillDiv
		g.hurt(m, 1, &round{})
		return g.Exp.Gold
	}, func(g *Game, pct int) int { return 2 * boosted(100, pct) }},
	{PotionBoost, Warrior, func(g *Game) int { return g.Hero.PotionHeal() },
		func(g *Game, pct int) int { return g.Hero.MaxHP() * (PotionHealPct + pct) / 100 }},
	{XPBoost, Warrior, func(g *Game) int {
		before := g.Hero.XP
		m := g.besideHero(dummy())
		m.HP, m.XP = 1, 100
		g.hurt(m, 1, &round{})
		return g.Hero.XP - before
	}, func(g *Game, pct int) int { return boosted(100, pct) }},
}

func effectItem(slot Slot, k Effect, v int) *Item {
	return &Item{Slot: slot, Rarity: Epic, Effects: []ItemEffect{{k, v}}}
}

// TestEveryEffectWorksOnlyWhileEquipped checks each effect where the game uses it: nothing without it,
// nothing from the bag, the exact gain from one item, two items adding up, and the cap.
func TestEveryEffectWorksOnlyWhileEquipped(t *testing.T) {
	seen := map[Effect]bool{}
	for _, p := range effectProbes {
		seen[p.kind] = true
		v := EffectRange[p.kind][1]
		cases := []struct {
			name  string
			equip []*Item
			bag   []*Item
			pct   int
		}{
			{"none", nil, nil, 0},
			{"in the bag", nil, []*Item{effectItem(Helmet, p.kind, v)}, 0},
			{"one item", []*Item{effectItem(Helmet, p.kind, v)}, nil, v},
			{"two items", []*Item{effectItem(Helmet, p.kind, v), effectItem(Boots, p.kind, v)}, nil, min(2*v, EffectCap[p.kind])},
			{"over the cap", []*Item{effectItem(Helmet, p.kind, EffectCap[p.kind]), effectItem(Boots, p.kind, v)}, nil, EffectCap[p.kind]},
		}
		base := 0
		for _, c := range cases {
			t.Run(fmt.Sprintf("%v/%s", p.kind, c.name), func(t *testing.T) {
				g := heroAt(p.class, 20, 27182818)
				for _, it := range c.equip {
					g.Hero.Equip[it.Slot] = it
				}
				g.Exp.Bag = c.bag
				if got := g.Hero.Effect(p.kind); got != c.pct {
					t.Fatalf("effect %d%%, want %d%%", got, c.pct)
				}
				want := 0
				if p.want != nil {
					want = p.want(g, c.pct)
				}
				got := p.gain(g)
				if c.name == "none" {
					base = got
				}
				if p.want == nil {
					want = base * (100 + c.pct) / 100
				}
				if got != want {
					t.Fatalf("gain %d, want %d", got, want)
				}
			})
		}
	}
	for k := range NumEffects {
		if !seen[k] {
			t.Errorf("effect %v has no probe", k)
		}
	}
}

func TestTreasureNoseFindsBiggerGoldPiles(t *testing.T) {
	g := heroAt(Rogue, AbilityInfo[TreasureNose].Level, 16180339)
	l := g.Exp.Level
	piles := func(nose bool) []int {
		l.Items = nil
		l.placeLoot(NewRNG(16180339), 0, nose)
		var gold []int
		for _, f := range l.Items {
			if f.Gold > 0 {
				gold = append(gold, f.Gold)
			}
		}
		return gold
	}
	plain, nose := piles(false), piles(true)
	if len(plain) == 0 || len(plain) != len(nose) {
		t.Fatalf("piles %v vs %v", plain, nose)
	}
	for i := range plain {
		if want := float64(plain[i]) * NoseGoldMult; math.Abs(float64(nose[i])-want) > NoseGoldMult {
			t.Errorf("pile %d: %d with the nose, want about %.0f", i, nose[i], want)
		}
	}
}
