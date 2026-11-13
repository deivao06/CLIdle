package game

import (
	"slices"

	"clidle/internal/lang"
)

type CondKind int

const (
	CondNone CondKind = iota
	CondHP
	CondMana
	CondVisible
	CondAdjacent
	CondInRange
	CondLethal
	CondBossVisible
	CondBagFull
	CondPotions
	CondFloor
	CondAtStairs
	CondLoot
	NumConds
)

type Op int

const (
	OpLess Op = iota
	OpGreater
	OpAtMost
	OpAtLeast
	NumOps
)

var opSymbols = [NumOps]string{"<", ">", "≤", "≥"}

func (o Op) String() string { return opSymbols[o] }

func (o Op) cmp(a, b int) bool {
	switch o {
	case OpGreater:
		return a > b
	case OpAtMost:
		return a <= b
	case OpAtLeast:
		return a >= b
	}
	return a < b
}

// Cond is one test on the hero's situation: what Kind measures compared with N by Op.
// N is a percent for HP and mana, a count or a floor otherwise.
type Cond struct {
	Kind CondKind
	Op   Op
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
	when := func(k CondKind, op Op, n int) [2]Cond { return [2]Cond{{k, op, n}} }
	gs := []Gambit{
		{If: when(CondHP, OpLess, SecondWindPct), Do: ActSecondWind},
		{If: when(CondHP, OpLess, c.PotionAtPct), Do: ActHeal},
		{If: when(CondHP, OpLess, c.PotionAtPct), Do: ActPotion},
	}
	if c.RetreatBagFull {
		gs = append(gs, Gambit{If: when(CondBagFull, 0, 0), Do: ActRetreat})
	}
	if c.RetreatHPPct > 0 {
		gs = append(gs, Gambit{If: [2]Cond{{CondPotions, OpAtMost, 0}, {CondHP, OpLess, c.RetreatHPPct}}, Do: ActRetreat})
	}
	if c.LastFloor > 0 {
		gs = append(gs, Gambit{If: when(CondAtStairs, OpAtLeast, c.LastFloor), Do: ActRetreat})
	}
	return append(gs,
		Gambit{If: when(CondInRange, 0, 0), Do: ActSpells},
		Gambit{If: when(CondAdjacent, OpAtLeast, 1), Do: ActBless},
		Gambit{If: when(CondAdjacent, OpAtLeast, 1), Do: ActAttack},
		Gambit{Do: ActExplore},
	)
}

func defaultV5() []Gambit {
	return validGambits(gambitsV5(configV5{PotionAtPct: 40, RetreatHPPct: 30, RetreatBagFull: true}))
}

// Usable reports whether the tactics may hold the action: walking the dungeon follows the expedition mode.
func (a Action) Usable() bool { return a != ActIdle && a != ActExplore && a != ActDescend }

// Usable reports whether the tactics may test the condition: the hero never knows a hit would kill him.
func (k CondKind) Usable() bool { return k != CondLethal }

// valid reports whether the rule can stay in the tactics; only a retreat may wait for the stairs.
func (gb Gambit) valid() bool {
	conds := gb.If[0].Kind.Usable() && gb.If[1].Kind.Usable() && (gb.Exit != ExitCond || gb.Until.Kind.Usable())
	return conds && gb.Do.Usable() && (gb.Do == ActRetreat || !gb.waitsForStairs())
}

func validGambits(gs []Gambit) []Gambit {
	return slices.DeleteFunc(gs, func(gb Gambit) bool { return !gb.valid() })
}

// DefaultGambits is the default list without the rules class c can never use.
func DefaultGambits(c Class) []Gambit { return classGambits(defaultV5(), c) }

// heroGambits is the default list with only the abilities the hero has already unlocked.
func heroGambits(h *Hero) []Gambit {
	return slices.DeleteFunc(DefaultGambits(h.Class), func(gb Gambit) bool { return !gb.unlocked(h) })
}

func (gb Gambit) unlocked(h *Hero) bool {
	ab, ok := actionAbility[gb.Do]
	return !ok || h.Has(ab)
}

// unlockGambits adds the default rule of each ability unlocked since level old, next to the rules
// that come before it in the default list; a rule the player already has for that action is kept.
func (g *Game) unlockGambits(old int, r *round) {
	h := g.Hero
	defs := DefaultGambits(h.Class)
	for i, gb := range defs {
		ab, ok := actionAbility[gb.Do]
		gs := g.Config.Gambits
		if !ok || AbilityInfo[ab].Level <= old || !h.Has(ab) || len(gs) >= MaxGambits ||
			slices.ContainsFunc(gs, func(o Gambit) bool { return o.Do == gb.Do }) {
			continue
		}
		at := 0
		for j, o := range gs {
			if slices.ContainsFunc(defs[:i], func(d Gambit) bool { return d.Do == o.Do }) {
				at = j + 1
			}
		}
		g.Config.Gambits = slices.Insert(gs, at, gb)
		if e := g.Exp; e != nil {
			for _, idx := range []*int{&e.ActiveRule, &e.RetreatRule, &e.Disarmed} {
				if *idx > at {
					*idx++
				}
			}
		}
		r.after = append(r.after, LogEntry{Text: lang.T("log.tactic_added", gb.Label()), Kind: LogGood})
	}
}

func classGambits(gs []Gambit, c Class) []Gambit {
	return slices.DeleteFunc(slices.Clone(gs), func(gb Gambit) bool { return !gb.Fits(c) })
}

var actionAbility = map[Action]Ability{ActSecondWind: SecondWind, ActHeal: Heal, ActBless: Bless, ActSpells: MagicMissile,
	ActMissile: MagicMissile, ActFireball: Fireball, ActFrost: FrostRay, ActSleep: Sleep, ActBlink: Blink}

// Fits reports whether class c can ever use the action.
func (a Action) Fits(c Class) bool {
	ab, ok := actionAbility[a]
	return !ok || AbilityInfo[ab].Class == c
}

func usesMana(c Class) bool {
	for _, a := range ClassAbilities(c) {
		if AbilityInfo[a].Mana > 0 {
			return true
		}
	}
	return false
}

// Fits reports whether the condition means anything for class c.
func (k CondKind) Fits(c Class) bool {
	switch k {
	case CondMana:
		return usesMana(c)
	case CondInRange:
		return AbilityInfo[MagicMissile].Class == c
	}
	return true
}

func (gb Gambit) Fits(c Class) bool {
	return gb.Do.Fits(c) && gb.If[0].Kind.Fits(c) && gb.If[1].Kind.Fits(c)
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
	case CondHP:
		return c.Op.cmp(h.HP*100, h.MaxHP()*c.N)
	case CondMana:
		return h.MaxMP() > 0 && c.Op.cmp(h.MP*100, h.MaxMP()*c.N)
	case CondVisible:
		return c.Op.cmp(g.visibleMonsters(), c.N)
	case CondAdjacent:
		return c.Op.cmp(len(g.adjacentMonsters()), c.N)
	case CondInRange:
		return g.castTarget() != nil
	case CondBossVisible:
		b := e.Level.boss()
		return b != nil && e.Level.Visible(h.Pos, b.Pos)
	case CondBagFull:
		return len(e.Bag) >= g.BagCap()
	case CondPotions:
		return c.Op.cmp(h.Potions, c.N)
	case CondFloor:
		return c.Op.cmp(g.Floor(e.Level.Depth), c.N)
	case CondAtStairs:
		return g.atStairs && c.Op.cmp(g.Floor(e.Level.Depth), c.N)
	case CondLoot:
		best := -1
		for _, it := range e.Bag {
			best = max(best, int(it.Rarity))
		}
		return c.Op.cmp(best, c.N)
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
			return lang.T("retreat.floor_done", g.Floor(g.Exp.Level.Depth))
		}
	}
	if gb.If[0].Kind == CondPotions && gb.If[1].Kind == CondHP {
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

var fleeScratch = struct {
	safe         []int
	taken        []bool
	queue, seeds []Pos
	relaxed      []Pos
}{safe: make([]int, MapW*MapH), taken: make([]bool, MapW*MapH)}

// fleeStep walks downhill on a safety map (the classic roguelike flee map): the walking distance from the
// awake visible monsters, negated and scaled past the step cost, then relaxed, so the hero heads for doors,
// corridors and other rooms, even around a monster, instead of the farthest corner.
func (g *Game) fleeStep() (Pos, bool) {
	const unset, stepCost, distWeight, reach = 1 << 30, 10, 12, 25
	h := g.Hero.Pos
	l := g.Exp.Level
	at := func(p Pos) int { return p.Y*MapW + p.X }
	safe, taken := fleeScratch.safe, fleeScratch.taken
	for i := range safe {
		safe[i], taken[i] = unset, false
	}
	queue := fleeScratch.queue[:0]
	for _, m := range l.Mons {
		taken[at(m.Pos)] = true
		if m.Asleep == 0 && l.Visible(h, m.Pos) {
			safe[at(m.Pos)] = 0
			queue = append(queue, m.Pos)
		}
	}
	if len(queue) == 0 {
		return h, false
	}
	for head := 0; head < len(queue); head++ {
		p := queue[head]
		if safe[at(p)] >= reach {
			continue
		}
		for _, d := range Dirs8 {
			if n := p.Add(d); l.In(n) && g.known(n) && safe[at(n)] == unset && l.CanStep(p, n) {
				safe[at(n)] = safe[at(p)] + 1
				queue = append(queue, n)
			}
		}
	}
	free := func(p Pos) bool { return !taken[at(p)] }
	seeds := fleeScratch.seeds[:0]
	for _, p := range queue {
		safe[at(p)] *= -distWeight
		if free(p) {
			seeds = append(seeds, p)
		}
	}
	slices.SortStableFunc(seeds, func(a, b Pos) int { return safe[at(a)] - safe[at(b)] })
	relaxed := fleeScratch.relaxed[:0]
	for si, ri := 0, 0; si < len(seeds) || ri < len(relaxed); {
		var p Pos
		if ri >= len(relaxed) || si < len(seeds) && safe[at(seeds[si])] <= safe[at(relaxed[ri])] {
			p, si = seeds[si], si+1
		} else {
			p, ri = relaxed[ri], ri+1
		}
		for _, d := range Dirs8 {
			if n := p.Add(d); l.In(n) && safe[at(n)] != unset && free(n) && safe[at(p)]+stepCost < safe[at(n)] && l.CanStep(p, n) {
				safe[at(n)] = safe[at(p)] + stepCost
				relaxed = append(relaxed, n)
			}
		}
	}
	fleeScratch.queue, fleeScratch.seeds, fleeScratch.relaxed = queue, seeds, relaxed
	best := h
	for _, d := range Dirs8 {
		if n := h.Add(d); g.passable(n) && l.CanStep(h, n) && safe[at(n)] < safe[at(best)] {
			best = n
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

var condKeys = [NumConds]string{"none", "hp", "mana", "visible", "adjacent",
	"in_range", "lethal", "boss_visible", "bag_full", "potions", "floor", "at_stairs", "loot"}

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
	if c.Kind == CondLoot {
		return lang.T(key, c.Op, Rarity(c.N).Name())
	}
	if c.Kind.HasN() {
		return lang.T(key, c.Op, c.N)
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

func NewGambit() Gambit { return Gambit{If: [2]Cond{{CondHP, OpLess, 50}}, Do: ActPotion} }

// SetGambits replaces the rules; abilities unlocked after level since get their default rule back,
// so a rule added by a level up while the list was being edited is not lost.
func (g *Game) SetGambits(gs []Gambit, since int) {
	g.Config.Gambits = slices.Clone(gs)
	g.unlockGambits(since, &round{})
	g.TacticsChanged()
}

func (g *Game) ResetGambits() {
	g.Config.Gambits = heroGambits(g.Hero)
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

// DefaultCond is the starting comparison when a condition kind is picked in the editor.
func (k CondKind) DefaultCond() Cond {
	switch k {
	case CondHP, CondMana:
		return Cond{k, OpLess, 50}
	case CondVisible:
		return Cond{k, OpAtLeast, 2}
	case CondAdjacent:
		return Cond{k, OpAtLeast, 1}
	case CondPotions:
		return Cond{k, OpAtMost, 0}
	case CondFloor, CondAtStairs:
		return Cond{k, OpAtLeast, 10}
	case CondLoot:
		return Cond{k, OpAtLeast, int(Rare)}
	}
	return Cond{Kind: k}
}

// StepN changes a condition number by d steps within its range.
func (c *Cond) StepN(d int) {
	lo, hi, by := 0, 100, 5
	switch c.Kind {
	case CondVisible, CondAdjacent:
		hi, by = 20, 1
	case CondPotions:
		hi, by = MaxPotions, 1
	case CondFloor, CondAtStairs:
		lo, hi, by = 1, 999, 1
	case CondLoot:
		hi, by = int(NumRarities)-1, 1
	}
	c.N = max(lo, min(hi, c.N+d*by))
}
