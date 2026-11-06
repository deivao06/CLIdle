package game

import (
	"slices"

	"clidle/internal/lang"
)

type CondKind int

const (
	CondNone CondKind = iota
	CondHPBelow
	CondHPAtLeast
	CondManaBelow
	CondManaAtLeast
	CondVisibleAtLeast
	CondAdjacentAtLeast
	CondInRange
	CondLethal
	CondBossVisible
	CondBagFull
	CondPotionsAtMost
	CondFloorAtLeast
	CondAtStairs
	NumConds
)

// Cond is one test on the hero's situation; N is a percent, a count or a floor, depending on Kind.
type Cond struct {
	Kind CondKind
	N    int
}

type Action int

const (
	ActPotion Action = iota
	ActSecondWind
	ActHeal
	ActBless
	ActAttack
	ActSpells
	ActMissile
	ActFireball
	ActFrost
	ActSleep
	ActBlink
	ActIdle
	ActFlee
	ActHunt
	ActExplore
	ActDescend
	ActRetreat
	NumActions
)

// Behavior actions last until their exit; the others take a single step.
func (a Action) Behavior() bool { return a >= ActIdle }

type Target int

const (
	TargetWeakest Target = iota
	TargetNearest
	TargetStrongest
	TargetBoss
	NumTargets
)

type ExitKind int

const (
	ExitEntry ExitKind = iota
	ExitCond
	ExitTime
	ExitDone
	NumExits
)

// Gambit is one rule of the hero's tactics: when both conditions hold, do the action on the target;
// a behavior keeps going until its exit.
type Gambit struct {
	If     [2]Cond
	Do     Action
	Target Target
	Exit   ExitKind
	Until  Cond
	Secs   int
}

// configV5 holds the retreat and potion options that v6 turned into gambits.
type configV5 struct {
	PotionAtPct    int
	RetreatHPPct   int
	RetreatBagFull bool
	LastFloor      int
}

// gambitsV5 builds the default tactics from the v5 options: the fixed hero AI of v5, without its potion before a lethal hit.
func gambitsV5(c configV5) []Gambit {
	when := func(k CondKind, n int) [2]Cond { return [2]Cond{{k, n}} }
	gs := []Gambit{
		{If: when(CondHPBelow, SecondWindPct), Do: ActSecondWind},
		{If: when(CondHPBelow, c.PotionAtPct), Do: ActHeal},
		{If: when(CondHPBelow, c.PotionAtPct), Do: ActPotion},
	}
	if c.RetreatBagFull {
		gs = append(gs, Gambit{If: when(CondBagFull, 0), Do: ActRetreat})
	}
	if c.RetreatHPPct > 0 {
		gs = append(gs, Gambit{If: [2]Cond{{CondPotionsAtMost, 0}, {CondHPBelow, c.RetreatHPPct}}, Do: ActRetreat})
	}
	if c.LastFloor > 0 {
		gs = append(gs, Gambit{If: when(CondAtStairs, c.LastFloor), Do: ActRetreat})
	}
	return append(gs,
		Gambit{If: when(CondInRange, 0), Do: ActSpells},
		Gambit{If: when(CondAdjacentAtLeast, 1), Do: ActBless},
		Gambit{If: when(CondAdjacentAtLeast, 1), Do: ActAttack},
		Gambit{Do: ActExplore},
	)
}

func DefaultGambits() []Gambit {
	return gambitsV5(configV5{PotionAtPct: 40, RetreatHPPct: 30, RetreatBagFull: true})
}

func (g *Game) visibleMonsters() int {
	l := g.Exp.Level
	n := 0
	for _, m := range l.Mons {
		if l.Visible(g.Hero.Pos, m.Pos) {
			n++
		}
	}
	return n
}

func (g *Game) holds(c Cond) bool {
	h, e := g.Hero, g.Exp
	switch c.Kind {
	case CondNone:
		return true
	case CondHPBelow:
		return h.HP*100 < h.MaxHP()*c.N
	case CondHPAtLeast:
		return h.HP*100 >= h.MaxHP()*c.N
	case CondManaBelow:
		return h.MaxMP() > 0 && h.MP*100 < h.MaxMP()*c.N
	case CondManaAtLeast:
		return h.MaxMP() > 0 && h.MP*100 >= h.MaxMP()*c.N
	case CondVisibleAtLeast:
		return g.visibleMonsters() >= c.N
	case CondAdjacentAtLeast:
		return len(g.adjacentMonsters()) >= max(1, c.N)
	case CondInRange:
		return g.castTarget() != nil
	case CondLethal:
		worst, avg := g.incoming()
		heal := g.potionHeal()
		return (h.MaxHP()-h.HP)*2 >= heal && worst >= h.HP && avg < heal
	case CondBossVisible:
		b := e.Level.boss()
		return b != nil && e.Level.Visible(h.Pos, b.Pos)
	case CondBagFull:
		return len(e.Bag) >= g.BagCap()
	case CondPotionsAtMost:
		return h.Potions <= c.N
	case CondFloorAtLeast:
		return e.Level.Depth >= c.N
	case CondAtStairs:
		return g.atStairs && e.Level.Depth >= c.N
	}
	return false
}

func (g *Game) entry(gb Gambit) bool { return g.holds(gb.If[0]) && g.holds(gb.If[1]) }

func (gb Gambit) waitsForStairs() bool {
	return gb.If[0].Kind == CondAtStairs || gb.If[1].Kind == CondAtStairs
}

// exited reports whether the active behavior gb should stop.
func (g *Game) exited(gb Gambit) bool {
	switch gb.Exit {
	case ExitCond:
		return g.holds(gb.Until)
	case ExitTime:
		return g.Exp.ActiveMs >= int64(gb.Secs)*1000
	case ExitDone:
		switch gb.Do {
		case ActHunt, ActFlee:
			return g.visibleMonsters() == 0
		case ActExplore:
			return g.explored()
		}
		return false
	}
	return !g.entry(gb)
}

// possible reports whether a behavior has anything to do right now.
func (g *Game) possible(gb Gambit) bool {
	switch gb.Do {
	case ActHunt:
		return g.huntTarget(gb.Target) != nil
	case ActFlee:
		return g.nearestThreat() != nil
	}
	return true
}

// retreatIndex is where the current retreat sits in the list: the rule that started it, or the first retreat rule.
func (g *Game) retreatIndex() int {
	gs := g.Config.Gambits
	if i := g.Exp.RetreatRule - 1; i >= 0 && i < len(gs) && gs[i].Do == ActRetreat {
		return i
	}
	for i, gb := range gs {
		if gb.Do == ActRetreat {
			return i
		}
	}
	return len(gs)
}

// behavior picks the rule that drives the hero this step: a higher rule whose entry holds takes over
// from the active one, a lower one waits for it to exit. It returns len(Gambits) when none applies.
func (g *Game) behavior() int {
	e := g.Exp
	gs := g.Config.Gambits
	retreatAt := -1
	if e.Retreating {
		retreatAt = g.retreatIndex()
	}
	for i, gb := range gs {
		if i == retreatAt {
			return i
		}
		if !gb.Do.Behavior() || gb.waitsForStairs() {
			continue
		}
		if i == e.Disarmed-1 {
			if !g.entry(gb) {
				e.Disarmed = 0
			}
			continue
		}
		if i == e.ActiveRule-1 && gb.Do != ActRetreat {
			if !g.exited(gb) && g.possible(gb) {
				return i
			}
			e.ActiveRule = 0
			if g.entry(gb) {
				e.Disarmed = i + 1
			}
			continue
		}
		if !g.entry(gb) || !g.possible(gb) {
			continue
		}
		if gb.Do == ActRetreat {
			if !e.Retreating {
				g.startRetreat(g.retreatWhy(gb))
				e.RetreatRule = i + 1
			}
			return i
		}
		e.ActiveRule, e.ActiveMs = i+1, 0
		return i
	}
	return len(gs)
}

// atStairsRule checks the rules that wait for the stairs, right before the hero would descend.
func (g *Game) atStairsRule() bool {
	e := g.Exp
	g.atStairs = true
	defer func() { g.atStairs = false }()
	for i, gb := range g.Config.Gambits {
		if !gb.Do.Behavior() || !gb.waitsForStairs() || !g.entry(gb) {
			continue
		}
		if gb.Do == ActRetreat {
			g.startRetreat(g.retreatWhy(gb))
			e.RetreatRule = i + 1
		} else {
			e.ActiveRule, e.ActiveMs = i+1, 0
		}
		return true
	}
	return false
}

func (g *Game) retreatWhy(gb Gambit) string {
	for _, c := range gb.If {
		switch c.Kind {
		case CondBagFull:
			return lang.T("retreat.bag_full")
		case CondAtStairs:
			return lang.T("retreat.floor_done", g.Exp.Level.Depth)
		}
	}
	if gb.If[0].Kind == CondPotionsAtMost && gb.If[1].Kind == CondHPBelow {
		return lang.T("retreat.low_hp")
	}
	return lang.T("retreat.rule", gb.CondLabel())
}

// offensive actions never run while the hero retreats without a retreat rule to rank them.
func (a Action) offensive() bool {
	return a == ActBless || a == ActAttack || a >= ActSpells && a <= ActSleep
}

// instant runs the first single-step rule above the driving behavior that can act now.
func (g *Game) instant(limit int, r *round) bool {
	gs := g.Config.Gambits
	manual := g.Exp.Retreating && limit == len(gs)
	for _, gb := range gs[:limit] {
		if gb.Do.Behavior() || manual && gb.Do.offensive() || !g.entry(gb) {
			continue
		}
		if g.act(gb, r) {
			return true
		}
	}
	return false
}

func (g *Game) act(gb Gambit, r *round) bool {
	h := g.Hero
	switch gb.Do {
	case ActPotion:
		if h.Potions == 0 || h.HP >= h.MaxHP() {
			return false
		}
		heal := min(g.potionHeal(), h.MaxHP()-h.HP)
		h.HP += heal
		h.Potions--
		r.hero = lang.T("log.drink", heal)
		return true
	case ActSecondWind:
		return g.secondWind(r)
	case ActHeal:
		return g.healSpell(r)
	case ActBless:
		return g.bless(r)
	case ActAttack:
		if m := g.attackTarget(gb.Target); m != nil {
			g.heroAttack(m, r)
			return true
		}
		return false
	case ActSpells:
		return g.cast(r)
	case ActBlink:
		return g.ready(Blink) && len(g.adjacentMonsters()) > 0 && g.blinkAway(r)
	}
	spell := map[Action]Ability{ActMissile: MagicMissile, ActFireball: Fireball, ActFrost: FrostRay, ActSleep: Sleep}[gb.Do]
	t := g.spellTarget(gb.Target)
	if !g.ready(spell) || t == nil || spell == Sleep && t.Asleep > 0 {
		return false
	}
	g.castSpell(spell, t, r)
	return true
}

// pick returns the monster among ms that the target asks for.
func (g *Game) pick(ms []*Monster, t Target) *Monster {
	var best *Monster
	h := g.Hero.Pos
	for _, m := range ms {
		switch {
		case t == TargetBoss && !m.Boss:
		case best == nil,
			t == TargetWeakest && m.HP < best.HP,
			t == TargetStrongest && m.HP > best.HP,
			(t == TargetNearest || t == TargetBoss) && m.Pos.Dist(h) < best.Pos.Dist(h):
			best = m
		}
	}
	return best
}

func (g *Game) attackTarget(t Target) *Monster {
	if t == TargetWeakest {
		return g.adjacentMonster()
	}
	return g.pick(g.adjacentMonsters(), t)
}

func (g *Game) spellTarget(t Target) *Monster {
	if t == TargetNearest {
		return g.castTarget()
	}
	l := g.Exp.Level
	h := g.Hero.Pos
	var ms []*Monster
	for _, m := range l.Mons {
		if m.Pos.Dist(h) <= SpellRange && l.Visible(h, m.Pos) {
			ms = append(ms, m)
		}
	}
	return g.pick(ms, t)
}

func (g *Game) huntTarget(t Target) *Monster {
	l := g.Exp.Level
	var ms []*Monster
	for _, m := range l.Mons {
		if l.Visible(g.Hero.Pos, m.Pos) {
			ms = append(ms, m)
		}
	}
	return g.pick(ms, t)
}

// nearestThreat is the closest visible monster that is not asleep, the one fleeing runs from.
func (g *Game) nearestThreat() *Monster {
	l := g.Exp.Level
	h := g.Hero.Pos
	var best *Monster
	for _, m := range l.Mons {
		if m.Asleep == 0 && l.Visible(h, m.Pos) && (best == nil || m.Pos.Dist(h) < best.Pos.Dist(h)) {
			best = m
		}
	}
	return best
}

// perform runs the driving behavior; i == len(Gambits) means no rule applies and the hero explores.
func (g *Game) perform(i int, r *round) {
	h := g.Hero
	l := g.Exp.Level
	gb := Gambit{Do: ActExplore}
	if i < len(g.Config.Gambits) {
		gb = g.Config.Gambits[i]
	}
	move := func(p Pos) {
		h.Pos = p
		l.Reveal(h.Pos)
		g.pickup(r)
	}
	switch gb.Do {
	case ActIdle:
	case ActFlee:
		if p, ok := g.fleeStep(); ok {
			move(p)
		}
	case ActHunt:
		t := g.huntTarget(gb.Target)
		if t == nil {
			return
		}
		if t.Pos.Dist(h.Pos) == 1 && l.CanStep(h.Pos, t.Pos) {
			g.heroAttack(t, r)
		} else if p, ok := l.pathStep(h.Pos, g.passable, func(p Pos) bool { return p == t.Pos }); ok {
			move(p)
		}
	default:
		if next, ok := g.chooseStep(g.modeFor(gb.Do)); ok {
			move(next)
		} else if m := g.adjacentMonster(); m != nil {
			g.heroAttack(m, r)
		}
	}
}

func (g *Game) modeFor(a Action) ExploreMode {
	if a == ActDescend {
		return DescendFast
	}
	return g.Config.Mode
}

// fleeStep moves to the free neighbor farthest from the nearest awake visible monster, if it gains distance.
func (g *Game) fleeStep() (Pos, bool) {
	h := g.Hero.Pos
	l := g.Exp.Level
	away := func(p Pos) int {
		d := MapW
		for _, m := range l.Mons {
			if m.Asleep == 0 && l.Visible(h, m.Pos) {
				d = min(d, m.Pos.Dist(p))
			}
		}
		return d
	}
	best, bestD := h, away(h)
	for _, d := range Dirs8 {
		n := h.Add(d)
		if g.passable(n) && l.CanStep(h, n) && away(n) > bestD {
			best, bestD = n, away(n)
		}
	}
	return best, best != h
}

func (g *Game) driving(i int) Action {
	if i < len(g.Config.Gambits) {
		return g.Config.Gambits[i].Do
	}
	if g.Exp.Retreating {
		return ActRetreat
	}
	return ActExplore
}

var condKeys = [NumConds]string{"none", "hp_below", "hp_at_least", "mana_below", "mana_at_least", "visible", "adjacent",
	"in_range", "lethal", "boss_visible", "bag_full", "potions_at_most", "floor", "at_stairs"}

// HasN reports whether the condition takes a number.
func (k CondKind) HasN() bool {
	switch k {
	case CondNone, CondInRange, CondLethal, CondBossVisible, CondBagFull:
		return false
	}
	return true
}

func (c Cond) Label() string {
	key := "gambit.cond." + condKeys[c.Kind]
	if c.Kind.HasN() {
		return lang.T(key, c.N)
	}
	return lang.T(key)
}

func (gb Gambit) condLabel(c Cond) string { return c.Label() }

// CondLabel describes the entry conditions, for the tactics list and the retreat reason.
func (gb Gambit) CondLabel() string {
	switch {
	case gb.If[0].Kind == CondNone && gb.If[1].Kind == CondNone:
		return lang.T("gambit.always")
	case gb.If[1].Kind == CondNone:
		return gb.condLabel(gb.If[0])
	case gb.If[0].Kind == CondNone:
		return gb.condLabel(gb.If[1])
	}
	return lang.T("gambit.and", gb.condLabel(gb.If[0]), gb.condLabel(gb.If[1]))
}

const MaxGambits = 16

// TacticsChanged forgets which rule was active, since the indices may point elsewhere now.
func (g *Game) TacticsChanged() {
	if e := g.Exp; e != nil {
		e.ActiveRule, e.Disarmed, e.RetreatRule = 0, 0, 0
	}
}

func (g *Game) AddGambit() int {
	gs := &g.Config.Gambits
	if len(*gs) >= MaxGambits {
		return -1
	}
	*gs = append(*gs, Gambit{If: [2]Cond{{CondHPBelow, 50}}, Do: ActPotion})
	g.TacticsChanged()
	return len(*gs) - 1
}

func (g *Game) DeleteGambit(i int) {
	g.Config.Gambits = slices.Delete(g.Config.Gambits, i, i+1)
	g.TacticsChanged()
}

// MoveGambit swaps rule i with its neighbor in direction d and returns where it went.
func (g *Game) MoveGambit(i, d int) int {
	gs := g.Config.Gambits
	j := i + d
	if j < 0 || j >= len(gs) {
		return i
	}
	gs[i], gs[j] = gs[j], gs[i]
	g.TacticsChanged()
	return j
}

func (g *Game) ResetGambits() {
	g.Config.Gambits = DefaultGambits()
	g.TacticsChanged()
}

// HasTarget and HasExit say which fields a rule's action uses, so the editor shows only those.
func (a Action) HasTarget() bool {
	return a == ActAttack || a == ActHunt || a >= ActMissile && a <= ActSleep
}

func (a Action) HasExit() bool { return a.Behavior() && a != ActRetreat }

var (
	actionKeys = [NumActions]string{"potion", "second_wind", "heal", "bless", "attack", "spells", "missile", "fireball",
		"frost", "sleep", "blink", "idle", "flee", "hunt", "explore", "descend", "retreat"}
	targetKeys = [NumTargets]string{"weakest", "nearest", "strongest", "boss"}
	exitKeys   = [NumExits]string{"entry", "cond", "time", "done"}
)

func (a Action) Label() string   { return lang.T("gambit.do." + actionKeys[a]) }
func (t Target) Label() string   { return lang.T("gambit.target." + targetKeys[t]) }
func (x ExitKind) Label() string { return lang.T("gambit.exit." + exitKeys[x]) }
func (a Action) Desc() string    { return lang.T("gambit.do." + actionKeys[a] + ".desc") }

// Label is the one-line summary of a rule: conditions → action (target), until exit.
func (gb Gambit) Label() string {
	s := gb.CondLabel() + " → " + gb.Do.Label()
	if gb.Do.HasTarget() {
		s += " (" + gb.Target.Label() + ")"
	}
	if gb.Do.HasExit() {
		switch gb.Exit {
		case ExitCond:
			s += " " + lang.T("gambit.until", gb.Until.Label())
		case ExitTime:
			s += " " + lang.T("gambit.for", gb.Secs)
		case ExitDone:
			s += " " + lang.T("gambit.until_done")
		}
	}
	return s
}

// DefaultN is the starting number when a condition kind is picked in the editor.
func (k CondKind) DefaultN() int {
	switch k {
	case CondVisibleAtLeast:
		return 2
	case CondAdjacentAtLeast:
		return 1
	case CondPotionsAtMost:
		return 0
	case CondFloorAtLeast, CondAtStairs:
		return 10
	}
	return 50
}

// StepN changes a condition number by d steps within its range.
func (c *Cond) StepN(d int) {
	lo, hi, by := 0, 100, 5
	switch c.Kind {
	case CondVisibleAtLeast, CondAdjacentAtLeast:
		lo, hi, by = 1, 20, 1
	case CondPotionsAtMost:
		hi, by = MaxPotions, 1
	case CondFloorAtLeast, CondAtStairs:
		lo, hi, by = 1, 999, 1
	}
	c.N = max(lo, min(hi, c.N+d*by))
}
