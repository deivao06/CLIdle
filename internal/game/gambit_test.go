package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestV5RetreatOptionsBecomeGambits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	raw, _ := json.Marshal(saveFile{5, time.Now(), g})
	var f map[string]any
	json.Unmarshal(raw, &f)
	cfg := f["Game"].(map[string]any)["Config"].(map[string]any)
	delete(cfg, "Gambits")
	cfg["PotionAtPct"], cfg["RetreatHPPct"], cfg["RetreatBagFull"], cfg["LastFloor"] = 55, 20, false, 7
	data, _ := json.Marshal(f)
	os.WriteFile(path, data, 0o644)
	got, _, err := Load(path)
	want := validGambits(classGambits(gambitsV5(configV5{PotionAtPct: 55, RetreatHPPct: 20, LastFloor: 7}), Warrior))
	if err != nil || !slices.Equal(got.Config.Gambits, want) {
		t.Fatalf("gambits %+v err %v", got.Config.Gambits, err)
	}
}

func TestV6DropsRulesThatBreakTheGame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Warrior)
	stairs := [2]Cond{{CondAtStairs, OpAtLeast, 5}}
	keep := []Gambit{{If: stairs, Do: ActRetreat}, {If: [2]Cond{{CondHP, OpLess, 50}}, Do: ActPotion}}
	g.Config.Gambits = []Gambit{keep[0], {Do: ActIdle}, {If: stairs, Do: ActHunt},
		{If: [2]Cond{{CondLethal, 0, 0}}, Do: ActPotion}, {Do: ActFlee, Exit: ExitCond, Until: Cond{Kind: CondLethal}},
		keep[1], {Do: ActDescend}, {Do: ActExplore}}
	raw, _ := json.Marshal(saveFile{6, time.Now(), g})
	os.WriteFile(path, raw, 0o644)
	got, _, err := Load(path)
	if err != nil || !slices.Equal(got.Config.Gambits, keep) {
		t.Fatalf("gambits %+v err %v", got.Config.Gambits, err)
	}
}

func TestGambitsSurviveSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(1, Mage)
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondMana, OpLess, 20}}, Do: ActIdle, Exit: ExitCond, Until: Cond{CondMana, OpAtLeast, 80}}}
	g.Save(path, time.Now())
	got, _, err := Load(path)
	if err != nil || !slices.Equal(got.Config.Gambits, g.Config.Gambits) {
		t.Fatalf("gambits %+v err %v", got.Config.Gambits, err)
	}
}

func TestIdleRestsUntilManaRecovers(t *testing.T) {
	g := heroAt(Mage, 5, 3)
	g.Config.Gambits = []Gambit{
		{If: [2]Cond{{CondMana, OpLess, 20}}, Do: ActIdle, Exit: ExitCond, Until: Cond{CondMana, OpAtLeast, 80}},
		{Do: ActExplore},
	}
	h := g.Hero
	h.MP = 0
	start := h.Pos
	for h.MP*100 < h.MaxMP()*80 {
		g.step()
		if h.Pos != start && h.MP*100 < h.MaxMP()*80 {
			t.Fatalf("moved at %d/%d mana", h.MP, h.MaxMP())
		}
	}
	for range 5 {
		g.step()
	}
	if h.Pos == start {
		t.Fatal("still idle after the mana came back")
	}
}

func TestFleeGainsDistance(t *testing.T) {
	g := heroAt(Warrior, 1, 4)
	m := g.besideHero(dummy())
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondVisible, OpAtLeast, 1}}, Do: ActFlee}, {Do: ActExplore}}
	p, ok := g.fleeStep()
	if !ok || p.Dist(m.Pos) <= 1 {
		t.Fatalf("flee step %v ok %v monster %v", p, ok, m.Pos)
	}
	start := g.Hero.Pos
	g.step()
	if g.Hero.Pos == start || g.Hero.Pos.Dist(m.Pos) <= 1 {
		t.Fatalf("hero %v monster %v", g.Hero.Pos, m.Pos)
	}
}

func TestHuntWalksToTheTarget(t *testing.T) {
	g := heroAt(Warrior, 1, 5)
	l := g.Exp.Level
	r := l.Rooms[l.RoomAt(g.Hero.Pos)]
	var far Pos
	for y := r.Y + 1; y < r.Y+r.H-1; y++ {
		for x := r.X + 1; x < r.X+r.W-1; x++ {
			if p := (Pos{x, y}); p.Dist(g.Hero.Pos) > far.Dist(g.Hero.Pos) && l.At(p).Walkable() {
				far = p
			}
		}
	}
	m := dummy()
	m.Pos = far
	l.Mons = []*Monster{m}
	g.Config.Gambits = []Gambit{{If: [2]Cond{{CondVisible, OpAtLeast, 1}}, Do: ActHunt}}
	before := g.Hero.Pos.Dist(m.Pos)
	if before < 2 {
		t.Skip("room too small")
	}
	g.step()
	if g.Hero.Pos.Dist(m.Pos) >= before {
		t.Fatalf("distance %d → %d", before, g.Hero.Pos.Dist(m.Pos))
	}
}

func TestHigherRuleInterruptsTheActiveOne(t *testing.T) {
	g := heroAt(Warrior, 1, 6)
	g.Config.Gambits = []Gambit{
		{If: [2]Cond{{CondVisible, OpAtLeast, 1}}, Do: ActFlee},
		{Do: ActIdle, Exit: ExitTime, Secs: 600},
	}
	g.step()
	if g.Exp.ActiveRule != 2 {
		t.Fatalf("active %d, want idle", g.Exp.ActiveRule)
	}
	g.besideHero(dummy())
	g.step()
	if g.Exp.ActiveRule != 1 {
		t.Fatalf("active %d, want flee", g.Exp.ActiveRule)
	}
}

func TestTimedBehaviorEndsAndStaysOff(t *testing.T) {
	g := heroAt(Warrior, 1, 7)
	g.Config.Gambits = []Gambit{{Do: ActIdle, Exit: ExitTime, Secs: 1}, {Do: ActExplore}}
	start := g.Hero.Pos
	g.Advance(900)
	if g.Hero.Pos != start || g.Exp.ActiveRule != 1 {
		t.Fatalf("pos %v active %d", g.Hero.Pos, g.Exp.ActiveRule)
	}
	g.Advance(5000)
	if g.Hero.Pos == start || g.Exp.ActiveRule != 2 {
		t.Fatalf("pos %v active %d", g.Hero.Pos, g.Exp.ActiveRule)
	}
}

func TestEmptyTacticsStillExplore(t *testing.T) {
	g := heroAt(Warrior, 1, 8)
	g.Config.Gambits = nil
	start := g.Hero.Pos
	g.Advance(5000)
	if g.Exp != nil && g.Hero.Pos == start {
		t.Fatal("hero froze without tactics")
	}
}

func TestEveryConditionHasALabel(t *testing.T) {
	for k := range NumConds {
		if condKeys[k] == "" {
			t.Fatalf("condition %d has no key", k)
		}
	}
}

func TestRandomTacticsNeverBreakTheGame(t *testing.T) {
	rng := NewRNG(42)
	pick := func(n int) int { return rng.Intn(n) }
	for c := range NumClasses {
		for range 12 {
			g := New(uint64(pick(1<<30)), c)
			g.Auto = true
			var gs []Gambit
			for range 1 + pick(MaxGambits) {
				cond := func() Cond {
					c := CondKind(pick(int(NumConds))).DefaultCond()
					c.Op = Op(pick(int(NumOps)))
					return c
				}
				gs = append(gs, Gambit{If: [2]Cond{cond(), cond()}, Do: Action(pick(int(NumActions))),
					Target: Target(pick(int(NumTargets))), Exit: ExitKind(pick(int(NumExits))), Until: cond(), Secs: 1 + pick(30)})
			}
			g.Config.Gambits = gs
			g.Depart()
			for range 20 {
				g.Advance(30_000)
			}
			if g.ExpNum == 0 {
				t.Fatalf("class %d never set out with %+v", c, gs)
			}
		}
	}
}

func TestDefaultTacticsOnlyHoldWhatTheClassUses(t *testing.T) {
	for c := range NumClasses {
		g := New(1, c)
		for _, gb := range g.Config.Gambits {
			if !gb.Fits(c) {
				t.Errorf("class %d default has %v", c, gb.Do)
			}
		}
	}
	has := func(c Class, a Action) bool {
		return slices.ContainsFunc(DefaultGambits(c), func(gb Gambit) bool { return gb.Do == a })
	}
	if has(Warrior, ActSpells) || has(Mage, ActSecondWind) || has(Rogue, ActHeal) || !has(Mage, ActSpells) || !has(Cleric, ActBless) {
		t.Fatal("class filter picked the wrong rules")
	}
	g := New(1, Warrior)
	g.ChooseClass(Mage)
	for _, gb := range DefaultGambits(Mage) {
		if !gb.valid() {
			t.Fatalf("default has %+v", gb)
		}
	}
	if !slices.Equal(g.Config.Gambits, DefaultGambits(Mage)) {
		t.Fatal("choosing a class did not reset the tactics")
	}
}

func TestEveryOperatorCompares(t *testing.T) {
	g := heroAt(Warrior, 1, 9)
	g.Hero.Potions = 2
	cases := []struct {
		op   Op
		n    int
		want bool
	}{{OpLess, 2, false}, {OpLess, 3, true}, {OpGreater, 1, true}, {OpGreater, 2, false},
		{OpAtMost, 2, true}, {OpAtMost, 1, false}, {OpAtLeast, 2, true}, {OpAtLeast, 3, false}}
	for _, c := range cases {
		if got := g.holds(Cond{CondPotions, c.op, c.n}); got != c.want {
			t.Errorf("potions 2 %s %d = %v, want %v", c.op, c.n, got, c.want)
		}
	}
}

func TestUnlockingAnAbilityAddsItsRule(t *testing.T) {
	g := New(1, Cleric)
	if slices.ContainsFunc(g.Config.Gambits, func(gb Gambit) bool { return gb.Do == ActBless }) {
		t.Fatal("bless rule before bless is unlocked")
	}
	g.Hero.Level = AbilityInfo[Bless].Level
	r := &round{}
	g.unlockGambits(g.Hero.Level-1, r)
	g.unlockGambits(g.Hero.Level-1, r)
	if !slices.Equal(g.Config.Gambits, DefaultGambits(Cleric)) || len(r.after) != 1 {
		t.Fatalf("got %d log lines and %+v", len(r.after), g.Config.Gambits)
	}
}

func TestRetreatsWhenTheBagHoldsARareItem(t *testing.T) {
	g := heroAt(Warrior, 1, 10)
	g.Config.Gambits = []Gambit{{If: [2]Cond{CondLoot.DefaultCond()}, Do: ActRetreat}, {Do: ActExplore}}
	g.Exp.Bag = []*Item{{Slot: Armor, Base: 3, Rarity: Magic}}
	g.step()
	if g.Exp.Retreating {
		t.Fatal("retreated with only a magic item")
	}
	g.Exp.Bag = append(g.Exp.Bag, &Item{Slot: Armor, Base: 3, Rarity: Rare})
	g.step()
	if g.Exp != nil && !g.Exp.Retreating {
		t.Fatal("kept going with a rare item in the bag")
	}
}
