package game

import (
	"fmt"
	"testing"

	"clidle/internal/lang"
)

const tacticsSeed = 31415926

var never = [2]Cond{{CondPotions, OpLess, 0}}

func mon(hp int) *Monster { return &Monster{HP: hp, MaxHP: hp, Spd: 1, Awake: true} }

// put places m on a visible floor tile at distance d from the hero.
func (g *Game) put(m *Monster, d int) *Monster {
	return g.putWhere(m, func(p Pos) bool { return p.Dist(g.Hero.Pos) == d })
}

// putBy places m next to other, still in spell range of the hero.
func (g *Game) putBy(m, other *Monster) *Monster {
	return g.putWhere(m, func(p Pos) bool { return p.Dist(other.Pos) == 1 && p.Dist(g.Hero.Pos) <= SpellRange })
}

func (g *Game) putWhere(m *Monster, ok func(Pos) bool) *Monster {
	l := g.Exp.Level
	h := g.Hero.Pos
	for y := range MapH {
		for x := range MapW {
			p := Pos{x, y}
			if p != h && ok(p) && l.At(p) == Floor && l.MonsterAt(p) == nil && l.Visible(h, p) {
				m.Pos = p
				l.Mons = append(l.Mons, m)
				return m
			}
		}
	}
	panic("no tile for the monster")
}

// openFloor is a tile with floor all around it.
func openFloor(l *Level) Pos {
	for y := range MapH {
		for x := range MapW {
			open := true
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					open = open && l.At(Pos{x + dx, y + dy}) == Floor
				}
			}
			if open {
				return Pos{x, y}
			}
		}
	}
	panic("no open floor")
}

func scene(c Class, level int) *Game {
	g := heroAt(c, level, tacticsSeed)
	g.Hero.MP = g.Hero.MaxMP()
	return g
}

func TestConditionsAtTheirLimits(t *testing.T) {
	cases := []struct {
		kind  CondKind
		class Class
		v     int
		set   func(g *Game)
	}{
		{CondHP, Warrior, 100, func(g *Game) {}},
		{CondMana, Mage, 100, func(g *Game) {}},
		{CondMana, Mage, 0, func(g *Game) { g.Hero.MP = 0 }},
		{CondVisible, Warrior, 3, func(g *Game) {
			g.besideHero(mon(9))
			g.besideHero(mon(9))
			g.put(mon(9), 3)
		}},
		{CondAdjacent, Warrior, 2, func(g *Game) {
			g.besideHero(mon(9))
			g.besideHero(mon(9))
			g.put(mon(9), 3)
		}},
		{CondPotions, Warrior, 3, func(g *Game) { g.Hero.Potions = 3 }},
		{CondFloor, Warrior, 7, func(g *Game) { g.Exp.Level.Depth = 7 }},
		{CondAtStairs, Warrior, 7, func(g *Game) { g.Exp.Level.Depth, g.atStairs = 7, true }},
		{CondLoot, Warrior, int(Rare), func(g *Game) { g.Exp.Bag = []*Item{{Rarity: Magic}, {Rarity: Rare}} }},
		{CondLoot, Warrior, -1, func(g *Game) { g.Exp.Bag = nil }},
	}
	for _, c := range cases {
		g := scene(c.class, 1)
		c.set(g)
		for op := range NumOps {
			for n := c.v - 1; n <= c.v+1; n++ {
				want := [NumOps]bool{c.v < n, c.v > n, c.v <= n, c.v >= n}[op]
				cond := Cond{c.kind, op, n}
				if got := g.holds(cond); got != want {
					t.Errorf("%v value %d: %s %v = %v, want %v", c.kind, c.v, op, n, got, want)
				}
			}
		}
	}
}

func TestConditionsWithoutANumber(t *testing.T) {
	cases := []struct {
		name string
		cond Cond
		set  func(g *Game)
		want bool
	}{
		{"none", Cond{Kind: CondNone}, func(g *Game) {}, true},
		{"in range", Cond{Kind: CondInRange}, func(g *Game) { g.put(mon(9), SpellRange) }, true},
		{"out of range", Cond{Kind: CondInRange}, func(g *Game) { g.put(mon(9), SpellRange+1) }, false},
		{"nobody in range", Cond{Kind: CondInRange}, func(g *Game) {}, false},
		{"boss visible", Cond{Kind: CondBossVisible}, func(g *Game) { g.put(&Monster{HP: 9, MaxHP: 9, Spd: 1, Boss: true}, 3) }, true},
		{"only a monster", Cond{Kind: CondBossVisible}, func(g *Game) { g.put(mon(9), 3) }, false},
		{"bag full", Cond{Kind: CondBagFull}, func(g *Game) { g.Exp.Bag = make([]*Item, g.BagCap()) }, true},
		{"bag one short", Cond{Kind: CondBagFull}, func(g *Game) { g.Exp.Bag = make([]*Item, g.BagCap()-1) }, false},
		{"stairs elsewhere", Cond{CondAtStairs, OpAtLeast, 1}, func(g *Game) {}, false},
		{"mana without mana", Cond{CondMana, OpAtLeast, 0}, func(g *Game) {}, false},
	}
	for _, c := range cases {
		g := scene(Warrior, 1)
		if c.cond.Kind == CondInRange {
			g = scene(Mage, 1)
		}
		c.set(g)
		if got := g.holds(c.cond); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// TestEveryActionRunsItsRule checks each action in a scene where it can act: the rule acts, and its effect shows;
// with an entry that never holds, the rule stays silent.
func TestEveryActionRunsItsRule(t *testing.T) {
	spent := func(g *Game, a Ability) func() bool {
		return func() bool { return g.Hero.MP == g.Hero.MaxMP()-AbilityInfo[a].Mana }
	}
	cases := []struct {
		do    Action
		class Class
		level int
		setup func(g *Game) func() bool
	}{
		{ActPotion, Warrior, 1, func(g *Game) func() bool {
			g.Hero.HP, g.Hero.Potions = 1, 3
			return func() bool { return g.Hero.Potions == 2 && g.Hero.HP > 1 }
		}},
		{ActSecondWind, Warrior, AbilityInfo[SecondWind].Level, func(g *Game) func() bool {
			g.Hero.HP = 1
			return func() bool { return g.Exp.SecondWindUsed && g.Hero.HP > 1 }
		}},
		{ActHeal, Cleric, AbilityInfo[Heal].Level, func(g *Game) func() bool {
			g.Hero.HP = 1
			check := spent(g, Heal)
			return func() bool { return check() && g.Hero.HP > 1 }
		}},
		{ActHolyLight, Cleric, AbilityInfo[HolyLight].Level, func(g *Game) func() bool {
			g.put(mon(999), 3)
			return spent(g, HolyLight)
		}},
		{ActAttack, Warrior, 1, func(g *Game) func() bool {
			g.besideHero(mon(999))
			return func() bool { return g.Exp.Attacks > 0 || g.Exp.HeroMisses.Run > 0 }
		}},
		{ActSpells, Mage, 1, func(g *Game) func() bool {
			g.put(mon(999), 3)
			return spent(g, MagicMissile)
		}},
		{ActMissile, Mage, AbilityInfo[MagicMissile].Level, func(g *Game) func() bool {
			g.put(mon(999), 3)
			return spent(g, MagicMissile)
		}},
		{ActSleep, Mage, AbilityInfo[Sleep].Level, func(g *Game) func() bool {
			m := g.put(mon(999), 3)
			check := spent(g, Sleep)
			return func() bool { return check() && m.SleepProof > 0 }
		}},
		{ActFireball, Mage, AbilityInfo[Fireball].Level, func(g *Game) func() bool {
			a := g.put(mon(999), 3)
			g.putBy(mon(999), a)
			return spent(g, Fireball)
		}},
		{ActBlink, Mage, AbilityInfo[Blink].Level, func(g *Game) func() bool {
			from := g.Hero.Pos
			g.besideHero(mon(999))
			check := spent(g, Blink)
			return func() bool { return check() && g.Hero.Pos != from }
		}},
		{ActFrost, Mage, AbilityInfo[FrostRay].Level, func(g *Game) func() bool {
			m := g.put(mon(999), 3)
			check := spent(g, FrostRay)
			return func() bool { return check() && m.HP < m.MaxHP }
		}},
		{ActIdle, Warrior, 1, func(g *Game) func() bool {
			from := g.Hero.Pos
			return func() bool { return g.Hero.Pos == from }
		}},
		{ActFlee, Warrior, 1, func(g *Game) func() bool {
			g.Hero.Pos = openFloor(g.Exp.Level)
			g.Exp.Level.Reveal(g.Hero.Pos)
			m := g.put(mon(999), 2)
			return func() bool { return m.Pos.Dist(g.Hero.Pos) > 2 }
		}},
		{ActHunt, Warrior, 1, func(g *Game) func() bool {
			m := g.put(mon(999), 3)
			return func() bool { return m.Pos.Dist(g.Hero.Pos) < 3 }
		}},
		{ActRetreat, Warrior, 1, func(g *Game) func() bool {
			g.Hero.Pos = farFloor(g.Exp.Level, g.Exp.Level.Up)
			return func() bool { return g.Exp.Retreating && g.Exp.RetreatRule == 1 }
		}},
	}
	seen := map[Action]bool{}
	for _, c := range cases {
		seen[c.do] = true
		for _, fires := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/fires=%v", c.do.Label(), fires), func(t *testing.T) {
				g := scene(c.class, c.level)
				effect := c.setup(g)
				rule := Gambit{Do: c.do}
				if !fires {
					rule.If = never
				}
				g.Config.Gambits = []Gambit{rule}
				g.step()
				if g.Exp == nil {
					t.Fatal("expedition ended")
				}
				if acted := g.Exp.Acted == 1; acted != fires {
					t.Fatalf("acted %d, want rule to act: %v", g.Exp.Acted, fires)
				}
				if fires && !effect() {
					t.Fatal("rule acted without its effect")
				}
				if !fires && !c.do.Behavior() && effect() {
					t.Fatal("effect without the rule")
				}
			})
		}
	}
	for a := range NumActions {
		if a.Usable() && !seen[a] {
			t.Errorf("action %s has no case", a.Label())
		}
	}
}

func TestTargetsPickTheRightMonster(t *testing.T) {
	g := scene(Mage, 1)
	weak := g.put(mon(5), 3)
	strong := g.put(mon(50), 2)
	boss := g.put(&Monster{HP: 30, MaxHP: 30, Spd: 1, Awake: true, Boss: true}, 4)
	want := [NumTargets]*Monster{TargetWeakest: weak, TargetNearest: strong, TargetStrongest: strong, TargetBoss: boss}
	for tg := range NumTargets {
		if got := g.spellTarget(tg); got != want[tg] {
			t.Errorf("spell %s: HP %d, want HP %d", tg.Label(), got.HP, want[tg].HP)
		}
		if got := g.huntTarget(tg); got != want[tg] {
			t.Errorf("hunt %s: HP %d, want HP %d", tg.Label(), got.HP, want[tg].HP)
		}
	}

	g = scene(Warrior, 1)
	a, b := g.besideHero(mon(5)), g.besideHero(mon(50))
	if g.attackTarget(TargetWeakest) != a || g.attackTarget(TargetStrongest) != b || g.attackTarget(TargetBoss) != nil {
		t.Error("attack picked the wrong neighbor")
	}

	g = scene(Mage, 1)
	weak, strong = g.put(mon(500), 3), g.put(mon(900), 2)
	g.Config.Gambits = []Gambit{{Do: ActMissile, Target: TargetStrongest}}
	g.step()
	if strong.HP == strong.MaxHP || weak.HP != weak.MaxHP {
		t.Errorf("missile at the strongest hit weak %d/%d strong %d/%d", weak.HP, weak.MaxHP, strong.HP, strong.MaxHP)
	}
}

func TestExitByEntryStopsWhenTheEntryFails(t *testing.T) {
	g := scene(Warrior, 1)
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondHP, OpLess, 100}}, Do: ActIdle}}
	g.Hero.HP = 1
	g.step()
	if g.Exp.Acted != 1 {
		t.Fatalf("hurt: acted %d", g.Exp.Acted)
	}
	g.Hero.HP = g.Hero.MaxHP()
	g.step()
	if g.Exp.Acted != 0 || g.Exp.ActiveRule != 0 {
		t.Fatalf("healed: acted %d active %d", g.Exp.Acted, g.Exp.ActiveRule)
	}
}

func TestExitByConditionWaitsForTheEntryToReset(t *testing.T) {
	g := scene(Warrior, 1)
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondPotions, OpAtLeast, 3}}, Do: ActIdle, Exit: ExitCond, Until: Cond{CondPotions, OpAtLeast, 5}}}
	steps := []struct {
		potions int
		acted   bool
	}{{3, true}, {4, true}, {5, false}, {3, false}, {1, false}, {3, true}}
	for i, s := range steps {
		g.Hero.Potions = s.potions
		g.step()
		if (g.Exp.Acted == 1) != s.acted {
			t.Fatalf("step %d with %d potions: acted %d, want %v", i, s.potions, g.Exp.Acted, s.acted)
		}
	}
}

func TestExitWhenDoneEndsWithNoMonsterInSight(t *testing.T) {
	for _, do := range []Action{ActHunt, ActFlee} {
		g := scene(Warrior, 1)
		g.Config.Gambits = []Gambit{{Do: do, Exit: ExitDone}}
		g.put(mon(999), 3)
		g.step()
		if g.Exp.Acted != 1 {
			t.Fatalf("%s with a monster: acted %d", do.Label(), g.Exp.Acted)
		}
		g.Exp.Level.Mons = nil
		g.step()
		if g.Exp.Acted != 0 || g.Exp.ActiveRule != 0 {
			t.Fatalf("%s alone: acted %d active %d", do.Label(), g.Exp.Acted, g.Exp.ActiveRule)
		}
	}
}

func TestLowerRuleWaitsForTheActiveOne(t *testing.T) {
	g := scene(Warrior, 1)
	g.Config.Gambits = []Gambit{
		{Do: ActIdle, Exit: ExitTime, Secs: 600},
		{If: [2]Cond{{CondVisible, OpAtLeast, 1}}, Do: ActHunt},
	}
	g.step()
	g.put(mon(999), 3)
	g.step()
	if g.Exp.Acted != 1 {
		t.Fatalf("acted %d, want idle to keep driving", g.Exp.Acted)
	}
}

func TestStepRulesOnlyRunAboveTheBehavior(t *testing.T) {
	potion := Gambit{If: [2]Cond{{CondHP, OpLess, 50}}, Do: ActPotion}
	idle := Gambit{Do: ActIdle}
	for _, c := range []struct {
		rules []Gambit
		acted int
		drank bool
	}{
		{[]Gambit{potion, idle}, 1, true},
		{[]Gambit{idle, potion}, 1, false},
	} {
		g := scene(Warrior, 1)
		g.Hero.HP, g.Hero.Potions = 1, 3
		g.Config.Gambits = c.rules
		g.step()
		if g.Exp.Acted != c.acted || (g.Hero.Potions == 2) != c.drank {
			t.Errorf("%s first: acted %d potions %d", c.rules[0].Do.Label(), g.Exp.Acted, g.Hero.Potions)
		}
	}
}

func TestStepRuleThatCannotActFallsThrough(t *testing.T) {
	g := scene(Warrior, 1)
	g.Hero.HP, g.Hero.Potions = 1, 0
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondHP, OpLess, 50}}, Do: ActPotion}, {Do: ActIdle}}
	g.step()
	if g.Exp.Acted != 2 {
		t.Fatalf("acted %d, want idle (2)", g.Exp.Acted)
	}
}

func TestBothConditionsMustHold(t *testing.T) {
	g := scene(Warrior, 1)
	g.Hero.HP, g.Hero.Potions = 1, 3
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondHP, OpLess, 50}, {CondVisible, OpAtLeast, 1}}, Do: ActPotion}}
	g.step()
	if g.Exp.Acted != 0 {
		t.Fatalf("one condition: acted %d", g.Exp.Acted)
	}
	g.put(mon(999), 3)
	g.step()
	if g.Exp.Acted != 1 {
		t.Fatalf("both conditions: acted %d", g.Exp.Acted)
	}
}

func TestRetreatRulesGiveTheirReason(t *testing.T) {
	cases := []struct {
		name string
		rule Gambit
		set  func(g *Game)
		why  string
	}{
		{"bag full", Gambit{If: [2]Cond{{Kind: CondBagFull}}, Do: ActRetreat},
			func(g *Game) { g.Exp.Bag = make([]*Item, g.BagCap()) }, lang.T("retreat.bag_full")},
		{"low HP", Gambit{If: [2]Cond{{CondPotions, OpLess, 1}, {CondHP, OpLess, 30}}, Do: ActRetreat},
			func(g *Game) { g.Hero.HP, g.Hero.Potions = 1, 0 }, lang.T("retreat.low_hp")},
		{"other rule", Gambit{If: [2]Cond{{CondPotions, OpAtMost, 2}}, Do: ActRetreat},
			func(g *Game) { g.Hero.Potions = 2 }, lang.T("retreat.rule", Gambit{If: [2]Cond{{CondPotions, OpAtMost, 2}}}.CondLabel())},
	}
	for _, c := range cases {
		g := scene(Warrior, 1)
		g.Hero.Pos = farFloor(g.Exp.Level, g.Exp.Level.Up)
		c.set(g)
		g.Config.Gambits = []Gambit{c.rule}
		g.step()
		if !g.Exp.Retreating || g.Exp.Acted != 1 || g.Exp.RetreatWhy != c.why {
			t.Errorf("%s: retreating %v acted %d why %q, want %q", c.name, g.Exp.Retreating, g.Exp.Acted, g.Exp.RetreatWhy, c.why)
		}
	}
}

func TestFloorDoneRetreatsAtTheStairs(t *testing.T) {
	g := scene(Warrior, 1)
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondAtStairs, OpAtLeast, 2}}, Do: ActRetreat}}
	if g.atStairsRule() {
		t.Fatal("retreated on floor 1")
	}
	g.Exp.Level.Depth = 2
	if !g.atStairsRule() || !g.Exp.Retreating || g.Exp.Acted != 1 || g.Exp.RetreatWhy != lang.T("retreat.floor_done", 2) {
		t.Fatalf("floor 2: retreating %v acted %d why %q", g.Exp.Retreating, g.Exp.Acted, g.Exp.RetreatWhy)
	}
}

func TestRetreatingSkipsTheRulesBelowIt(t *testing.T) {
	g := scene(Warrior, 1)
	g.Hero.Pos = farFloor(g.Exp.Level, g.Exp.Level.Up)
	g.Config.Gambits = []Gambit{
		{If: [2]Cond{{CondPotions, OpAtLeast, 0}}, Do: ActRetreat},
		{If: [2]Cond{{CondHP, OpLess, 100}}, Do: ActPotion},
	}
	g.Hero.HP, g.Hero.Potions = 1, 3
	g.step()
	if g.Exp.Acted != 1 || g.Hero.Potions != 3 {
		t.Fatalf("acted %d potions %d, want retreat without the potion", g.Exp.Acted, g.Hero.Potions)
	}
}
