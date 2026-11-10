package game

import (
	"strings"
	"testing"

	"clidle/internal/lang"
)

func TestEffectCountGrowsWithRarity(t *testing.T) {
	rng := NewRNG(1)
	for range 50 {
		for r, want := range []int{0, 0, 1, 2, 3} {
			fx := newItemOf(rng, 5, Rarity(r)).Effects
			if len(fx) != want {
				t.Fatalf("rarity %d with %d effects", r, len(fx))
			}
			for i := range fx {
				if rg := EffectRange[fx[i].Kind]; fx[i].Value < rg[0] || fx[i].Value > rg[1] {
					t.Fatalf("effect %v out of range", fx[i])
				}
				for j := range i {
					if fx[i].Kind == fx[j].Kind {
						t.Fatalf("repeated effect %v", fx)
					}
				}
			}
		}
	}
}

func TestSameEffectStopsAtItsCap(t *testing.T) {
	g := New(1, Warrior)
	withEffect(g, LifeSteal, 20)
	g.Hero.Equip[Boots] = &Item{Slot: Boots, Rarity: Rare, Base: 1, Effects: []ItemEffect{{LifeSteal, 20}}}
	if got := g.Hero.Effect(LifeSteal); got != EffectCap[LifeSteal] {
		t.Fatalf("life steal %d, cap %d", got, EffectCap[LifeSteal])
	}
}

func withEffect(g *Game, k Effect, v int) {
	g.Hero.Equip[Helmet] = &Item{Slot: Helmet, Rarity: Epic, Base: 1, Effects: []ItemEffect{{k, v}}}
}

func TestVigorRaisesMaxHP(t *testing.T) {
	g := New(1, Warrior)
	base := g.Hero.MaxHP()
	withEffect(g, Vigor, 10)
	if want := (base + g.Hero.Equip[Helmet].Bonus()*ArmorHPPct/100) * 110 / 100; g.Hero.MaxHP() != want {
		t.Fatalf("max hp %d, base %d", g.Hero.MaxHP(), base)
	}
}

func TestLifeStealHealsOnHit(t *testing.T) {
	g := heroAt(Warrior, 5, 1)
	withEffect(g, LifeSteal, 30)
	g.Hero.HP = 1
	m := g.besideHero(dummy())
	g.hurt(m, 10, &round{})
	if g.Hero.HP != 4 {
		t.Fatalf("hp %d", g.Hero.HP)
	}
}

func TestGoldAndXPFindOnKill(t *testing.T) {
	g := heroAt(Warrior, 1, 1)
	withEffect(g, GoldFind, 100)
	g.Hero.Equip[Boots] = &Item{Slot: Boots, Rarity: Epic, Base: 1, Effects: []ItemEffect{{XPBoost, 50}}}
	m := g.besideHero(dummy())
	m.HP, m.XP = 1, 5
	g.hurt(m, 5, &round{})
	if g.Exp.Gold != 2*GoldPerKill(5) || g.Hero.XP != boosted(5, 50) {
		t.Fatalf("gold %d xp %d", g.Exp.Gold, g.Hero.XP)
	}
}

func TestCritDoublesAndLogs(t *testing.T) {
	g := heroAt(Warrior, 5, 1)
	withEffect(g, Crit, 25)
	m := g.besideHero(dummy())
	m.HP = 1 << 30
	name := lang.T(m.Info().Name)
	want := lang.T("combat.hero_crit", name, 0)
	want = want[:strings.Index(want, "(")]
	for range 200 {
		r := &round{}
		g.heroAttack(m, r)
		if strings.HasPrefix(r.hero, want) {
			return
		}
	}
	t.Fatal("no critical hit in 200 attacks")
}

func TestPotionBoostHealsMore(t *testing.T) {
	g := heroAt(Rogue, 5, 1)
	withEffect(g, PotionBoost, 20)
	g.Hero.HP, g.Hero.Potions = 1, 1
	g.step()
	if want := 1 + g.Hero.MaxHP()*(PotionHealPct+20)/100; g.Hero.HP != want {
		t.Fatalf("hp %d want %d", g.Hero.HP, want)
	}
}

func TestRemovingVigorCapsHP(t *testing.T) {
	g := New(1, Warrior)
	withEffect(g, Vigor, 50)
	g.Hero.HP = g.Hero.MaxHP()
	g.Unequip(Helmet)
	if g.Hero.HP > g.Hero.MaxHP() {
		t.Fatalf("hp %d > max %d", g.Hero.HP, g.Hero.MaxHP())
	}
}
