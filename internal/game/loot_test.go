package game

import (
	"strings"
	"testing"

	"clidle/internal/lang"
)

func TestReturnDepositsGoldAndSellsBag(t *testing.T) {
	g := departed(1)
	it := &Item{Slot: Weapon, Base: 5}
	g.Exp.Gold = 10
	g.Exp.Bag = []*Item{it}
	g.returnToTown()
	if g.Exp != nil || g.Gold != 10+it.SellPrice() {
		t.Fatalf("gold %d, exp %v", g.Gold, g.Exp)
	}
}

func TestDeathLosesFoundLootButKeepsStartEquip(t *testing.T) {
	g := departed(1)
	start := &Item{Slot: Weapon, Base: 3}
	g.Hero.Equip[Weapon] = start
	g.Exp.StartEquip = g.Hero.Equip
	g.Hero.Equip[Weapon] = &Item{Slot: Weapon, Base: 9}
	g.Exp.Gold = 50
	g.Hero.HP = 1
	m := &Monster{HP: 9, Atk: 100, Spd: 4}
	for i := 0; i < 50 && g.Exp != nil; i++ {
		g.monsterAttack(m, &round{})
	}
	if g.Exp != nil || g.Gold != 50*DeathKeepGoldPct/100 || g.Hero.Equip[Weapon] != start {
		t.Fatalf("gold %d, weapon %+v", g.Gold, g.Hero.Equip[Weapon])
	}
}

func TestBagFullTriggersRetreatAndHeroLeaves(t *testing.T) {
	g := departed(2)
	for range BagSize {
		g.Exp.Bag = append(g.Exp.Bag, &Item{Base: 1})
	}
	g.Exp.Level.Mons = nil
	g.checkRetreat()
	if !g.Exp.Retreating {
		t.Fatal("expected retreat")
	}
	g.Hero.Pos = g.Exp.Level.Up
	g.step()
	if g.Exp != nil || g.Returns != 1 {
		t.Fatal("hero should be back in town")
	}
}

func TestPotionHealsWhenLow(t *testing.T) {
	g := departed(3)
	g.Exp.Level.Mons = nil
	g.Hero.Potions = 1
	g.Hero.HP = 1
	g.step()
	if g.Hero.Potions != 0 || g.Hero.HP <= 1 {
		t.Fatalf("potions %d hp %d", g.Hero.Potions, g.Hero.HP)
	}
}

func TestBetterItemIsEquipped(t *testing.T) {
	g := departed(4)
	l := g.Exp.Level
	l.Mons = nil
	it := &Item{Slot: Armor, Base: 7}
	l.Items = []*FloorItem{{Pos: g.Hero.Pos, Item: it}}
	g.pickup(&round{})
	if g.Hero.Equip[Armor] != it || g.Hero.Def() != HeroBaseDef+7 {
		t.Fatalf("armor not equipped: %+v", g.Hero.Equip)
	}
}

func TestManualRetreatTurnsOffAuto(t *testing.T) {
	g := departed(6)
	g.Auto = true
	g.Retreat()
	if !g.Exp.Retreating || g.Auto {
		t.Fatalf("retreating %v auto %v", g.Exp.Retreating, g.Auto)
	}
}

func TestAutomaticRetreatKeepsAuto(t *testing.T) {
	g := departed(6)
	g.Auto = true
	for range BagSize {
		g.Exp.Bag = append(g.Exp.Bag, &Item{Base: 1})
	}
	g.checkRetreat()
	if !g.Exp.Retreating || !g.Auto {
		t.Fatalf("retreating %v auto %v", g.Exp.Retreating, g.Auto)
	}
}

func TestLegacyItemNameIsTranslated(t *testing.T) {
	it := &Item{Name: "Adaga"}
	if got := it.Label(); got != "Dagger" {
		t.Fatalf("got %q", got)
	}
}

func TestItemsCoverEveryKind(t *testing.T) {
	rng := NewRNG(1)
	seen := map[[2]int]bool{}
	for range 5000 {
		it := newItem(rng, 30)
		seen[[2]int{int(it.Slot), it.Kind}] = true
	}
	for _, want := range [][2]int{{int(Weapon), Staff}, {int(Weapon), Dagger}, {int(Armor), Heavy}, {int(Helmet), 0}, {int(Shield), 0}} {
		if !seen[want] {
			t.Fatalf("never generated %v", want)
		}
	}
}

func TestMageStashesSword(t *testing.T) {
	g := New(1, Mage)
	g.StartExpedition()
	sword := &Item{Slot: Weapon, Kind: Blade, Base: 99}
	if g.better(sword) || g.wants(&FloorItem{Item: sword}) {
		t.Fatal("mage should ignore a sword by default")
	}
	g.Config.AnyClass = true
	if !g.wants(&FloorItem{Item: sword}) {
		t.Fatal("mage should bag a sword with other classes' items on")
	}
}

func TestEveryRarityDrops(t *testing.T) {
	rng := NewRNG(1)
	seen := map[Rarity]bool{}
	for range 5000 {
		seen[newItem(rng, 5).Rarity] = true
	}
	for r := Common; r <= Legendary; r++ {
		if !seen[r] {
			t.Fatalf("%v never drops", r)
		}
	}
}

func TestHigherRarityNeverWeakerOnSameFloor(t *testing.T) {
	for s := range NumSlots {
		for k := range ItemNames[s] {
			for d := 1; d <= 60; d++ {
				for r := Magic; r <= Legendary; r++ {
					if lo, hi := ItemRoll(s, k, d, r, 0), ItemRoll(s, k, d, r-1, 1); lo < hi {
						t.Fatalf("slot %d kind %d floor %d: %v %d < %v %d", s, k, d, r, lo, r-1, hi)
					}
				}
			}
		}
	}
}

func TestRetreatFleesInsteadOfFighting(t *testing.T) {
	g := departed(5)
	l := g.Exp.Level
	l.Mons = nil
	for _, d := range Dirs8 {
		if p := l.Up.Add(d); l.CanStep(l.Up, p) {
			g.Hero.Pos = p
			break
		}
	}
	m := dummy()
	for _, d := range Dirs8 {
		if p := g.Hero.Pos.Add(d); p != l.Up && l.CanStep(g.Hero.Pos, p) {
			m.Pos = p
			break
		}
	}
	l.Mons = []*Monster{m}
	g.startRetreat("test")
	g.step()
	if g.Exp != nil || m.HP != m.MaxHP {
		t.Fatalf("hero fought instead of fleeing: monster hp %d", m.HP)
	}
}

func TestDoingDescribesRetreatAndFight(t *testing.T) {
	g := departed(6)
	l := g.Exp.Level
	l.Mons = nil
	if g.Doing() == "" {
		t.Fatal("no status while exploring")
	}
	g.besideHero(dummy())
	if want := lang.T("doing.fight", lang.T(l.Mons[0].Info().Name)); g.Doing() != want {
		t.Fatalf("got %q, want %q", g.Doing(), want)
	}
	g.startRetreat("x")
	if g.Doing() != lang.T("doing.retreat", "x") {
		t.Fatalf("got %q", g.Doing())
	}
}

func TestDeathKeepsTheSceneAroundTheCorpse(t *testing.T) {
	g := departed(1)
	g.Hero.HP = 1
	m := &Monster{HP: 9, Atk: 100, Spd: 4, Awake: true}
	m.Pos = g.Hero.Pos.Add(Pos{1, 0})
	if !g.Exp.Level.CanStep(g.Hero.Pos, m.Pos) {
		m.Pos = g.Hero.Pos.Add(Pos{-1, 0})
	}
	g.Exp.Level.Mons = []*Monster{m}
	for i := 0; i < 50 && g.Exp != nil; i++ {
		g.monsterAttack(m, &round{})
	}
	sc := g.DeathScene
	if sc == nil || len(sc.Glyphs) != DeathSceneH {
		t.Fatal("no death scene")
	}
	all := strings.Join(sc.Glyphs, "") + strings.Join(sc.Marks, "")
	if !strings.ContainsRune(all, CorpseGlyph) || !strings.ContainsRune(strings.Join(sc.Marks, ""), MarkKiller) {
		t.Fatalf("scene misses corpse or killer:\n%s", strings.Join(sc.Glyphs, "\n"))
	}
}
