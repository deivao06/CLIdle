package game

import (
	"strings"
	"testing"

	"clidle/internal/lang"
)

func TestOnlyEpicAndAboveRollEffects(t *testing.T) {
	rng := NewRNG(1)
	for range 50 {
		if n := len(newItemOf(rng, 5, Rare).Effects); n != 0 {
			t.Fatalf("magic item with %d effects", n)
		}
		if n := len(newItemOf(rng, 5, Epic).Effects); n != 1 {
			t.Fatalf("epic item with %d effects", n)
		}
		fx := newItemOf(rng, 5, Legendary).Effects
		if len(fx) != 2 || fx[0].Kind == fx[1].Kind {
			t.Fatalf("legendary effects %v", fx)
		}
	}
}

func withEffect(g *Game, k Effect, v int) {
	g.Hero.Equip[Helmet] = &Item{Slot: Helmet, Rarity: Epic, Base: 1, Effects: []ItemEffect{{k, v}}}
}

func TestVigorRaisesMaxHP(t *testing.T) {
	g := New(1, Warrior)
	base := g.Hero.MaxHP()
	withEffect(g, Vigor, 10)
	if g.Hero.MaxHP() != base*110/100 {
		t.Fatalf("max hp %d, base %d", g.Hero.MaxHP(), base)
	}
}

func TestLifeStealHealsOnHit(t *testing.T) {
	g := heroAt(Warrior, 5, 1)
	withEffect(g, LifeSteal, 50)
	g.Hero.HP = 1
	m := g.besideHero(dummy())
	g.hurt(m, 10, &round{})
	if g.Hero.HP != 6 {
		t.Fatalf("hp %d", g.Hero.HP)
	}
}

func TestGoldAndXPFindOnKill(t *testing.T) {
	g := heroAt(Warrior, 1, 1)
	withEffect(g, GoldFind, 100)
	g.Hero.Equip[Boots] = &Item{Slot: Boots, Rarity: Epic, Base: 1, Effects: []ItemEffect{{XPBoost, 100}}}
	m := g.besideHero(dummy())
	m.HP, m.XP = 1, 5
	g.hurt(m, 5, &round{})
	if g.Exp.Gold != 2*GoldPerKill(5) || g.Hero.XP != 10 {
		t.Fatalf("gold %d xp %d", g.Exp.Gold, g.Hero.XP)
	}
}

func TestCritDoublesAndLogs(t *testing.T) {
	g := heroAt(Warrior, 5, 1)
	withEffect(g, Crit, 100)
	m := g.besideHero(dummy())
	name := lang.T(m.Info().Name)
	r := &round{}
	for r.hero == "" || r.hero == lang.T("combat.hero_miss", name) {
		r = &round{}
		g.heroAttack(m, r)
	}
	want := lang.T("combat.hero_crit", name, 0)
	if !strings.HasPrefix(r.hero, want[:strings.Index(want, "(")]) {
		t.Fatalf("log %q", r.hero)
	}
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
