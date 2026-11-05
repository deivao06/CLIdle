package game

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"clidle/internal/lang"
)

type Expedition struct {
	Num        int
	Seed       uint64
	RNG        *RNG
	Level      *Level
	Gold       int
	Bag        []*Item
	StartEquip [NumSlots]*Item
	Retreating bool
	RetreatWhy string

	Attacks        int
	SecondWindUsed bool
	ManaAcc        int
	Blessed        int
	Fighting       bool
}

type ExploreMode int

const (
	ExploreAll ExploreMode = iota
	DescendFast
)

type Config struct {
	Mode           ExploreMode
	PotionAtPct    int
	RetreatHPPct   int
	RetreatBagFull bool
	LastFloor      int
	PotionsToBring int
	StartFloor     int
	DeathLimit     int
	ManualEquip    bool
	KeepLoot       bool
	MinRarity      Rarity
	AnyClass       bool
}

func DefaultConfig() Config {
	return Config{Mode: ExploreAll, PotionAtPct: 40, RetreatHPPct: 30, RetreatBagFull: true, PotionsToBring: 3}
}

type Game struct {
	RNG        *RNG
	Hero       *Hero
	Exp        *Expedition
	Config     Config
	Log        []LogEntry
	Gold       int
	ExpNum     int
	AccMs      int64
	MaxDepth   int
	Deaths     int
	Returns    int
	Auto       bool
	Shop       []*Item
	Shortcuts  []int
	Kills      int
	BossKills  int
	AutoDeaths int
	Stash      []*Item
	Bell       bool
	CRT        bool
	Nerd       bool
	Effects    Effects
	Lang       string
	BagLevel   int
	LastDeath  []LogEntry
	DeathScene *DeathScene `json:",omitempty"`

	Events []Event `json:"-"`
	strong bool
}

func New(seed uint64, c Class) *Game {
	g := &Game{
		RNG:    NewRNG(seed),
		Hero:   NewHero(c),
		Config: DefaultConfig(),
	}
	g.refreshShop()
	g.logf(LogInfo, "%s", lang.T("log.welcome"))
	return g
}

func (g *Game) inCombat() bool {
	return g.Exp != nil && (g.adjacentMonster() != nil || g.ready(MagicMissile) && g.castTarget() != nil)
}

// StepMs is the duration of the next step: combat rounds have a fixed pace after the first, exploration follows speed.
func (g *Game) StepMs() int64 {
	if g.inCombat() && g.Exp.Fighting {
		return CombatRoundMs
	}
	return int64(1000 / StepsPerSec(g.Hero.Spd()-g.Hero.Attr(Dex)/ExploreDexDiv))
}

// Advance runs the simulation for ms milliseconds of game time.
func (g *Game) Advance(ms int64) {
	g.AccMs += ms
	for {
		if g.Exp == nil {
			if !g.Auto {
				g.AccMs = 0
				return
			}
			if g.AccMs < TownWaitMs {
				return
			}
			g.AccMs -= TownWaitMs
			g.StartExpedition()
			continue
		}
		step := g.StepMs()
		if g.AccMs < step {
			return
		}
		g.AccMs -= step
		g.step()
	}
}

func (g *Game) StartExpedition() {
	g.ExpNum++
	seed := g.RNG.Uint64()
	g.Exp = &Expedition{Num: g.ExpNum, Seed: seed, RNG: NewRNG(seed), StartEquip: g.Hero.Equip}
	g.Hero.HP = g.Hero.MaxHP()
	g.Hero.MP = g.Hero.MaxMP()
	g.buyPotions()
	g.logf(LogInfo, "%s", lang.T("log.depart", g.ExpNum))
	g.enterLevel(g.startFloor())
}

func (g *Game) enterLevel(depth int) {
	e := g.Exp
	e.Level = GenerateLevel(e.RNG, depth)
	e.SecondWindUsed = false
	l := e.Level
	startRoom := l.RoomAt(l.Up)
	for range MonstersPerLevel(depth) {
		room := e.RNG.Intn(len(l.Rooms) - 1)
		if room >= startRoom {
			room++
		}
		m := spawnMonster(e.RNG, depth)
		m.Pos = l.RandomFloor(e.RNG, room)
		l.Mons = append(l.Mons, m)
	}
	l.placeLoot(e.RNG, startRoom, g.Hero.Has(TreasureNose))
	if depth%BossEvery == 0 {
		b := spawnBoss(e.RNG, depth)
		b.Pos = l.RandomFloor(e.RNG, l.RoomAt(l.Down))
		l.Mons = append(l.Mons, b)
	}
	g.MaxDepth = max(g.MaxDepth, depth)
	g.Hero.Pos = l.Up
	l.Reveal(g.Hero.Pos)
	g.logf(LogInfo, "%s", lang.T("log.arrive_floor", depth))
	if b := l.boss(); b != nil {
		g.logf(LogBad, "%s", lang.T("log.boss_guard", lang.T(b.Info().Name)))
	}
}

func (g *Game) step() {
	l := g.Exp.Level
	h := g.Hero
	r := &round{}
	fighting := g.inCombat()
	g.regenMana()
	if g.Exp.Blessed > 0 {
		g.Exp.Blessed--
	}
	g.monstersAct(r, true)
	if g.Exp != nil {
		g.checkRetreat()
		m := g.adjacentMonster()
		switch {
		case g.secondWind(r):
		case g.lowHP() && g.healSpell(r):
		case g.needsPotion():
			heal := min(g.potionHeal(), h.MaxHP()-h.HP)
			h.HP += heal
			h.Potions--
			r.hero = lang.T("log.drink", heal)
		case !g.Exp.Retreating && g.cast(r):
		case m != nil && !g.Exp.Retreating && g.bless(r):
		case m != nil && !g.Exp.Retreating:
			g.heroAttack(m, r)
		default:
			if next, ok := g.chooseStep(); ok {
				h.Pos = next
				l.Reveal(h.Pos)
				g.pickup(r)
			} else if m != nil {
				g.heroAttack(m, r)
			}
		}
		g.monstersAct(r, false)
	}
	lines := g.flush(r)
	if g.Exp == nil {
		g.LastDeath = append(g.LastDeath, lines...)
		return
	}
	g.Exp.Fighting = fighting
	switch {
	case g.Exp.Retreating && h.Pos == l.Up:
		g.returnToTown()
	case !g.Exp.Retreating && h.Pos == l.Down && g.readyToDescend():
		if c := g.Config.LastFloor; c > 0 && l.Depth >= c {
			g.startRetreat(lang.T("retreat.floor_done", l.Depth))
			return
		}
		g.enterLevel(l.Depth + 1)
	}
}

func (g *Game) needsPotion() bool {
	h := g.Hero
	if h.Potions == 0 {
		return false
	}
	if g.lowHP() {
		return true
	}
	worst, avg := g.incoming()
	heal := g.potionHeal()
	return (h.MaxHP()-h.HP)*2 >= heal && worst >= h.HP && avg < heal
}

func (g *Game) potionHeal() int { return g.Hero.PotionHeal() }

// incoming is the worst and the average damage the adjacent awake monsters deal before the hero acts again.
func (g *Game) incoming() (worst, avg int) {
	h := g.Hero
	var w, a float64
	for _, m := range g.adjacentMonsters() {
		if m.Asleep > 0 {
			continue
		}
		rate := float64(m.Spd) / float64(h.Spd())
		w += max(1, float64(m.Atk)*1.2-float64(h.Def())*0.5) * math.Ceil(rate)
		a += max(1, float64(m.Atk)-float64(h.Def())*0.5) * rate
	}
	return int(w + 0.5), int(a + 0.5)
}

func (g *Game) lowHP() bool {
	h := g.Hero
	return h.HP*100 < h.MaxHP()*g.Config.PotionAtPct
}

func (g *Game) checkRetreat() {
	e := g.Exp
	h := g.Hero
	switch {
	case e.Retreating:
	case g.Config.RetreatBagFull && len(e.Bag) >= g.BagCap():
		g.startRetreat(lang.T("retreat.bag_full"))
	case h.Potions == 0 && h.HP*100 < h.MaxHP()*g.Config.RetreatHPPct:
		g.startRetreat(lang.T("retreat.low_hp"))
	}
}

// Retreat orders the hero back to the exit and keeps him in town afterwards.
func (g *Game) Retreat() {
	if g.Exp == nil || g.Exp.Retreating {
		return
	}
	g.startRetreat(lang.T("retreat.order"))
	if g.Auto {
		g.ToggleAuto()
	}
}

func (g *Game) startRetreat(why string) {
	g.Exp.Retreating = true
	g.Exp.RetreatWhy = why
	g.logf(LogInfo, "%s", lang.T("log.retreat", why))
}

func (g *Game) returnToTown() {
	e := g.Exp
	sold, kept := 0, 0
	for _, it := range e.Bag {
		if it.Rarity == Legendary || g.Config.KeepLoot && len(g.Stash) < StashSize {
			g.Stash = append(g.Stash, it)
			kept++
			continue
		}
		sold += it.SellPrice()
	}
	g.Gold += e.Gold + sold
	g.Returns++
	if len(e.Bag) == kept {
		g.logf(LogGood, "%s", lang.T("log.return", e.Num, e.RetreatWhy, e.Gold))
	} else {
		g.logf(LogGood, "%s", lang.T("log.return_sold", e.Num, e.RetreatWhy, e.Gold+sold, e.Gold, len(e.Bag)-kept, sold))
	}
	if kept > 0 {
		g.logf(LogLoot, "%s", lang.T("log.chest_kept", kept))
	}
	g.Exp = nil
	g.arriveInTown()
}

func (g *Game) wants(f *FloorItem) bool {
	if f.Item == nil {
		return true
	}
	c := g.Config
	return g.better(f.Item) || f.Item.Rarity >= c.MinRarity && (c.AnyClass || g.Hero.Class.CanUse(f.Item)) && len(g.Exp.Bag) < g.BagCap()
}

func (g *Game) better(it *Item) bool {
	if g.Exp != nil && g.Config.ManualEquip {
		return false
	}
	cur := g.Hero.Equip[it.Slot]
	return g.Hero.Class.CanUse(it) && (cur == nil || it.Bonus() > cur.Bonus())
}

func (g *Game) wantedItem(p Pos) bool {
	l := g.Exp.Level
	i := l.ItemAt(p)
	return i >= 0 && l.IsSeen(p) && g.wants(l.Items[i])
}

func (l *Level) boss() *Monster {
	for _, m := range l.Mons {
		if m.Boss {
			return m
		}
	}
	return nil
}

func (g *Game) readyToDescend() bool {
	if g.Exp.Level.boss() != nil {
		return false
	}
	if _, ok := g.Exp.Level.pathStep(g.Hero.Pos, g.passable, g.wantedItem); ok {
		return false
	}
	return g.Config.Mode == DescendFast || g.explored()
}

func (g *Game) pickup(r *round) {
	e := g.Exp
	l := e.Level
	h := g.Hero
	i := l.ItemAt(h.Pos)
	if i < 0 || !g.wants(l.Items[i]) {
		return
	}
	f := l.Items[i]
	l.Items = append(l.Items[:i], l.Items[i+1:]...)
	add := func(kind LogKind, format string, args ...any) {
		r.after = append(r.after, LogEntry{Text: lang.T(format, args...), Kind: kind})
	}
	switch {
	case f.Gold > 0:
		gold := boosted(f.Gold, h.Effect(GoldFind))
		e.Gold += gold
		add(LogLoot, "log.pick_gold", gold)
	case f.Potion:
		h.Potions++
		add(LogLoot, "log.pick_potion")
	case g.better(f.Item):
		it := f.Item
		g.emit(Event{Kind: EventLoot, Pos: h.Pos, Value: int(it.Rarity)})
		old := h.Equip[it.Slot]
		h.Equip[it.Slot] = it
		h.HP = min(h.HP, h.MaxHP())
		add(it.Rarity.LogKind(), "log.equip", it.Title(), it.Rarity.Name(), it.StatLabel())
		if old != nil && len(e.Bag) < g.BagCap() {
			e.Bag = append(e.Bag, old)
		}
	default:
		g.emit(Event{Kind: EventLoot, Pos: h.Pos, Value: int(f.Item.Rarity)})
		e.Bag = append(e.Bag, f.Item)
		add(f.Item.Rarity.LogKind(), "log.stash", f.Item.Title(), f.Item.Rarity.Name())
	}
}

// round collects everything that happens in one step so it is logged as a single line.
type round struct {
	hero  string
	mons  []*monsterRound
	hurt  bool
	after []LogEntry
}

type monsterRound struct {
	name   string
	hits   []int
	misses int
}

func (r *round) of(m *Monster) *monsterRound {
	name := lang.T(m.Info().Name)
	for _, mr := range r.mons {
		if mr.name == name {
			return mr
		}
	}
	mr := &monsterRound{name: name}
	r.mons = append(r.mons, mr)
	return mr
}

func (mr *monsterRound) String() string {
	if len(mr.hits) == 0 {
		if mr.misses > 1 {
			return lang.T("combat.mon_miss_n", mr.name, mr.misses)
		}
		return lang.T("combat.mon_miss", mr.name)
	}
	var dmgs []string
	for _, d := range mr.hits {
		dmgs = append(dmgs, strconv.Itoa(d))
	}
	s := lang.T("combat.mon_hit", mr.name, strings.Join(dmgs, ", "))
	if len(mr.hits) > 1 {
		s = lang.T("combat.mon_hit_n", mr.name, len(mr.hits), strings.Join(dmgs, ", "))
	}
	if mr.misses > 0 {
		s += lang.T("combat.and_miss")
	}
	return s
}

func (g *Game) flush(r *round) []LogEntry {
	var out []LogEntry
	var parts []string
	if r.hero != "" {
		parts = append(parts, r.hero)
	}
	for _, mr := range r.mons {
		parts = append(parts, mr.String())
	}
	if len(parts) > 0 {
		line := strings.Join(parts, " · ")
		kind := LogInfo
		if r.hurt {
			kind = LogBad
		}
		out = append(out, LogEntry{strings.ToUpper(line[:1]) + line[1:], kind, 1})
	}
	out = append(out, r.after...)
	for _, e := range out {
		g.logf(e.Kind, "%s", e.Text)
	}
	return out
}

func (g *Game) adjacentMonster() *Monster {
	l := g.Exp.Level
	var best *Monster
	for _, m := range l.Mons {
		if m.Pos.Dist(g.Hero.Pos) == 1 && l.CanStep(g.Hero.Pos, m.Pos) && (best == nil || m.HP < best.HP) {
			best = m
		}
	}
	return best
}

func (g *Game) heroAttack(m *Monster, r *round) {
	e := g.Exp
	h := g.Hero
	name := lang.T(m.Info().Name)
	sneak := h.Has(SneakAttack) && (!m.Awake || m.HP == m.MaxHP)
	m.Awake = true
	miss := h.MissChance()
	if e.Blessed > 0 {
		miss /= 2
	}
	if e.RNG.Chance(miss) {
		r.hero = lang.T("combat.hero_miss", name)
		return
	}
	e.Attacks++
	if h.Has(PowerStrike) && e.Attacks%PowerStrikeEvery == 0 {
		g.powerStrike(m, r)
		return
	}
	d := damage(e.RNG, h.Atk(), m.Def)
	key := "combat.hero_hit"
	if c := h.CritChance(); c > 0 && e.RNG.Chance(c) {
		d *= 2
		key = "combat.hero_crit"
	}
	if sneak {
		d *= SneakMult
		key = "ability.sneak_attack.log"
	}
	if m.Info().Undead && g.ready(TurnUndead) {
		g.useMana(TurnUndead)
		d *= TurnUndeadMult
		key = "spell.turn_undead.log"
	}
	g.strong = key != "combat.hero_hit"
	died := g.hurt(m, d, r)
	g.strong = false
	if !died || key != "combat.hero_hit" {
		r.hero = lang.T(key, name, d)
	}
}

// hurt applies the hero's damage to m and reports whether it died.
func (g *Game) hurt(m *Monster, d int, r *round) bool {
	e := g.Exp
	h := g.Hero
	name := lang.T(m.Info().Name)
	g.emit(Event{Kind: EventHit, Pos: m.Pos, Value: d, Big: g.strong})
	m.HP -= d
	m.Asleep = 0
	h.HP = min(h.MaxHP(), h.HP+d*h.Effect(LifeSteal)/100)
	if m.HP > 0 {
		return false
	}
	gold := boosted(GoldPerKill(m.XP), h.Effect(GoldFind))
	xp := boosted(m.XP, h.Effect(XPBoost))
	e.Gold += gold
	g.Kills++
	r.after = append(r.after, LogEntry{Text: lang.T("log.kill", name, xp, gold), Kind: LogGood})
	if m.Boss {
		g.killBoss(m, r)
	}
	g.emit(Event{Kind: EventKill, Pos: m.Pos, Big: m.Boss})
	e.Level.removeMonster(m)
	if h.gainXP(xp) {
		g.emit(Event{Kind: EventLevelUp, Pos: h.Pos})
		r.after = append(r.after, LogEntry{Text: lang.T("log.level_up", h.Level), Kind: LogGood})
	}
	return true
}

func (g *Game) killBoss(m *Monster, r *round) {
	e := g.Exp
	g.BossKills++
	depth := e.Level.Depth
	drop := newItemOf(e.RNG, depth, Epic)
	e.Level.Items = append(e.Level.Items, &FloorItem{Pos: m.Pos, Item: drop})
	text := lang.T("log.boss_dead", lang.T(m.Info().Name), drop.Title())
	if !slices.Contains(g.ShortcutList(), depth+1) {
		g.Shortcuts = append(g.ShortcutList(), depth+1)
		text += lang.T("log.shortcut", depth+1)
	}
	r.after = append(r.after, LogEntry{Text: text, Kind: LogEpic})
}

// ShortcutList is the set of floors an expedition can start from.
func (g *Game) ShortcutList() []int {
	if len(g.Shortcuts) == 0 {
		return []int{1}
	}
	return g.Shortcuts
}

func (g *Game) startFloor() int {
	if slices.Contains(g.ShortcutList(), g.Config.StartFloor) {
		return g.Config.StartFloor
	}
	return 1
}

func (l *Level) removeMonster(m *Monster) {
	for i, o := range l.Mons {
		if o == m {
			l.Mons = append(l.Mons[:i], l.Mons[i+1:]...)
			return
		}
	}
}

// monstersAct moves awake monsters; fast monsters act before the hero, the rest after.
func (g *Game) monstersAct(r *round, fast bool) {
	e := g.Exp
	l := e.Level
	h := g.Hero
	var dist []int
	for _, m := range l.Mons {
		if (m.Spd > h.Spd()) != fast {
			continue
		}
		if m.Asleep > 0 {
			m.Asleep--
			continue
		}
		if !m.Awake {
			if !l.Visible(h.Pos, m.Pos) || m.Boss && m.Pos.Dist(h.Pos) > BossWakeDist {
				continue
			}
			m.Awake = true
		}
		gain := float64(m.Spd) / float64(h.Spd())
		if m.Slowed > 0 {
			m.Slowed--
			gain /= 2
		}
		m.Energy += gain
		for m.Energy >= 1 {
			m.Energy--
			if m.Pos.Dist(h.Pos) == 1 && l.CanStep(m.Pos, h.Pos) {
				g.monsterAttack(m, r)
				if g.Exp == nil {
					return
				}
				continue
			}
			if dist == nil {
				dist = l.distMap(h.Pos)
			}
			m.moveToward(l, dist, h.Pos)
		}
	}
}

func (m *Monster) moveToward(l *Level, dist []int, hero Pos) {
	best, bestD := m.Pos, dist[m.Pos.Y*MapW+m.Pos.X]
	if bestD < 0 {
		return
	}
	for _, d := range Dirs8 {
		n := m.Pos.Add(d)
		if !l.In(n) || n == hero || !l.CanStep(m.Pos, n) || l.MonsterAt(n) != nil {
			continue
		}
		if nd := dist[n.Y*MapW+n.X]; nd >= 0 && nd < bestD {
			best, bestD = n, nd
		}
	}
	m.Pos = best
}

func (g *Game) monsterAttack(m *Monster, r *round) {
	e := g.Exp
	h := g.Hero
	mr := r.of(m)
	if e.RNG.Chance(MissChance(m.Dex)) || h.Has(Dodge) && e.RNG.Chance(h.DodgeChance()) {
		mr.misses++
		return
	}
	def := h.Def()
	if e.Blessed > 0 {
		def += BlessDef
	}
	d := damage(e.RNG, m.Atk, def)
	if c := CritChance(m.Dex); c > 0 && e.RNG.Chance(c) {
		d *= 2
	}
	g.emit(Event{Kind: EventHurt, Pos: h.Pos, Value: d, Big: d*BigHitDiv >= h.MaxHP()})
	h.HP -= d
	mr.hits = append(mr.hits, d)
	r.hurt = true
	if h.HP <= 0 {
		h.HP = 0
		r.after = append(r.after, LogEntry{Text: lang.T("log.death",
			mr.name, e.Level.Depth, e.Gold-e.deathKeep(), len(e.Bag)+g.foundEquipped(), e.deathKeep()), Kind: LogBad})
		g.die(m)
	}
}

func (g *Game) foundEquipped() int {
	n := 0
	for s, it := range g.Hero.Equip {
		if start := g.Exp.StartEquip[s]; it != nil && (start == nil || it.ID != start.ID) {
			n++
		}
	}
	return n
}

func (e *Expedition) deathKeep() int { return e.Gold * DeathKeepGoldPct / 100 }

func (g *Game) die(killer *Monster) {
	g.LastDeath = slices.Clone(g.Log[max(0, len(g.Log)-DeathLogLines):])
	g.DeathScene = g.deathScene(killer)
	g.Hero.Equip = g.Exp.StartEquip
	g.Gold += g.Exp.deathKeep()
	g.Exp = nil
	g.Deaths++
	if g.Auto && g.Config.DeathLimit > 0 {
		g.AutoDeaths++
		if g.AutoDeaths >= g.Config.DeathLimit {
			g.Auto = false
			g.logf(LogBad, "%s", lang.T("log.auto_death_limit", g.AutoDeaths))
		}
	}
	g.arriveInTown()
}

func (g *Game) known(p Pos) bool {
	l := g.Exp.Level
	return l.IsSeen(p) && l.At(p).Walkable()
}

func (g *Game) passable(p Pos) bool {
	return g.known(p) && g.Exp.Level.MonsterAt(p) == nil
}

// frontier reports a known walkable tile that touches unexplored space.
func (g *Game) frontier(p Pos) bool {
	l := g.Exp.Level
	if !g.known(p) {
		return false
	}
	for _, d := range Dirs8 {
		n := p.Add(d)
		if l.In(n) && !l.IsSeen(n) {
			return true
		}
	}
	return false
}

func (g *Game) explored() bool {
	_, ok := g.Exp.Level.pathStep(g.Hero.Pos, g.known, g.frontier)
	return !ok
}

func (g *Game) chooseStep() (Pos, bool) {
	l := g.Exp.Level
	if g.Exp.Retreating {
		return l.pathStep(g.Hero.Pos, g.passable, func(p Pos) bool { return p == l.Up })
	}
	next, i := g.plan()
	return next, i >= 0
}

const (
	goalMonster = iota
	goalItem
	goalRush
	goalFrontier
	goalLast
)

// plan returns the hero's next step and which goal it leads to.
func (g *Game) plan() (Pos, int) {
	l := g.Exp.Level
	h := g.Hero
	to := func(target Pos) func(Pos) bool { return func(p Pos) bool { return p == target } }
	boss := l.boss()
	seenMonster := func(p Pos) bool { return l.MonsterAt(p) != nil && l.Visible(h.Pos, p) }
	rush := func(p Pos) bool {
		return g.Config.Mode == DescendFast && boss == nil && l.IsSeen(l.Down) && p == l.Down
	}
	last := to(l.Down)
	if boss != nil {
		last = to(boss.Pos)
	}
	return l.search(h.Pos, g.passable, seenMonster, g.wantedItem, rush, g.frontier, last)
}

// Doing describes what the hero is up to, for the status line.
func (g *Game) Doing() string {
	if g.Exp == nil {
		return ""
	}
	e := g.Exp
	if e.Retreating {
		return lang.T("doing.retreat", e.RetreatWhy)
	}
	if m := g.adjacentMonster(); m != nil {
		return lang.T("doing.fight", lang.T(m.Info().Name))
	}
	_, i := g.plan()
	switch {
	case i == goalMonster:
		return lang.T("doing.hunt")
	case i == goalItem:
		return lang.T("doing.loot")
	case i == goalFrontier:
		return lang.T("doing.explore", e.Level.Depth)
	case i == goalLast && e.Level.boss() != nil:
		return lang.T("doing.boss", lang.T(e.Level.boss().Info().Name))
	}
	return lang.T("doing.descend", e.Level.Depth+1)
}
