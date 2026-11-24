package game

import (
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
	// DungeonDone marks the expedition that killed the last floor's boss.
	DungeonDone bool `json:",omitempty"`
	RetreatWhy  string

	Attacks        int
	SecondWindUsed bool
	ManaAcc        int
	Fighting       bool

	ActiveRule  int
	ActiveMs    int64
	RetreatRule int
	Disarmed    int
	// Acted is the rule (index+1) that acted this step; zero means none did.
	Acted int `json:",omitempty"`

	HeroMisses Streak `json:",omitempty"`
	MonMisses  Streak `json:",omitempty"`
}

type ExploreMode int

const (
	ExploreAll ExploreMode = iota
	DescendFast
)

type Config struct {
	Mode           ExploreMode
	Gambits        []Gambit
	PotionsToBring int
	StartFloor     int
	DeathLimit     int
	ManualEquip    bool
	KeepLoot       bool
	MinRarity      Rarity
	AnyClass       bool
}

func DefaultConfig() Config {
	return Config{Mode: ExploreAll, Gambits: defaultV5(), PotionsToBring: 3}
}

type Game struct {
	RNG      *RNG
	Hero     *Hero
	Exp      *Expedition
	Config   Config
	Log      []LogEntry
	ExpNum   int
	AccMs    int64
	MaxDepth int
	// Dungeon counts the dungeons cleared; Noun and Adj name the current one; Reached is its deepest floor.
	Dungeon    int `json:",omitempty"`
	Noun       int `json:",omitempty"`
	Adj        int `json:",omitempty"`
	Reached    int `json:",omitempty"`
	Prestiges  int `json:",omitempty"`
	Deaths     int
	Returns    int
	LastGold   int `json:",omitempty"`
	Auto       bool
	Shop       []*Item
	Shortcuts  []int
	Kills      int
	BossKills  int
	AutoDeaths int
	BagLevel   int
	LastDeath  []LogEntry
	DeathScene *DeathScene `json:",omitempty"`
	// BestHeroMisses and BestMonMisses are the least likely miss streaks ever seen.
	BestHeroMisses Streak `json:",omitempty"`
	BestMonMisses  Streak `json:",omitempty"`

	// Queue is the hero's turn for the mission table while all its expeditions are taken; zero when not waiting.
	Queue int64 `json:",omitempty"`

	Events   []Event `json:"-"`
	Guild    *Guild  `json:"-"`
	lab      *LabRun
	strong   bool
	atStairs bool
}

func New(seed uint64, c Class) *Game {
	g := &Game{
		RNG:    NewRNG(seed),
		Hero:   NewHero(c),
		Config: DefaultConfig(),
	}
	g.Config.Gambits = heroGambits(g.Hero)
	g.rollDungeonName(NewRNG(^seed))
	g.Guild = &Guild{Heroes: []*Game{g}, RNG: NewRNG(seed ^ guildSeed)}
	g.Guild.link()
	g.refreshShop()
	g.logf(LogInfo, "%s", lang.T("log.welcome"))
	return g
}

func (g *Game) inCombat() bool {
	return g.Exp != nil && (g.adjacentMonster() != nil || (g.ready(MagicMissile) || g.ready(HolyLight)) && g.castTarget() != nil)
}

// StepMs is the duration of the next step: combat rounds have a fixed pace after the first, exploration follows speed.
func (g *Game) StepMs() int64 {
	if g.inCombat() && g.Exp.Fighting {
		return CombatRoundMs
	}
	return int64(1000 / StepsPerSec(g.Hero.Spd()))
}

// Advance runs the simulation for ms milliseconds of game time.
func (g *Game) Advance(ms int64) {
	g.AccMs += ms
	for {
		if g.Exp == nil {
			if !g.Auto {
				g.AccMs, g.Queue = 0, 0
				return
			}
			if g.AccMs < TownWaitMs {
				return
			}
			if !g.Guild.mayStart(g) {
				g.AccMs = TownWaitMs
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
		if g.Exp.ActiveRule > 0 {
			g.Exp.ActiveMs += step
		}
		g.step()
	}
}

func (g *Game) StartExpedition() {
	g.Queue = 0
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
	if g.Cave() {
		e.Level = GenerateCave(e.RNG, depth)
	} else {
		e.Level = GenerateLevel(e.RNG, depth)
	}
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
	if g.Floor(depth)%BossEvery == 0 {
		b := spawnBoss(depth)
		b.Pos = l.RandomFloor(e.RNG, l.RoomAt(l.Down))
		l.Mons = append(l.Mons, b)
	}
	g.MaxDepth = max(g.MaxDepth, depth)
	g.Reached = max(g.Reached, g.Floor(depth))
	g.Hero.Pos = l.Up
	l.Reveal(g.Hero.Pos)
	g.logf(LogInfo, "%s", lang.T("log.arrive_floor", g.Floor(depth)))
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
	g.monstersAct(r, true)
	drive := ActExplore
	if g.Exp != nil {
		g.Exp.Acted = 0
		b := g.behavior()
		drive = g.driving(b)
		if !g.instant(b, r) {
			g.perform(b, r)
			g.markActed(b)
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
	case !g.Exp.Retreating && (drive == ActExplore || drive == ActDescend) && h.Pos == l.Down && g.readyToDescend(g.modeFor(drive)):
		if g.atStairsRule() {
			return
		}
		if g.lab != nil {
			g.labDescend()
			return
		}
		g.enterLevel(l.Depth + 1)
	}
}

// PotionHeal is the HP a potion gives the hero: a share of max HP raised by item effects and the Apothecary.
func (g *Game) PotionHeal() int {
	h := g.Hero
	return h.MaxHP() * (PotionHealPct + h.Effect(PotionBoost) + ApothecaryHeal(g.Guild.Buildings[Apothecary])) / 100
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
		if it.Rarity == Legendary || g.Config.KeepLoot {
			g.Guild.Stash = append(g.Guild.Stash, it)
			kept++
			continue
		}
		sold = addGold(sold, it.SellPrice())
	}
	g.LastGold = addGold(e.Gold, sold)
	g.Guild.Gold = addGold(g.Guild.Gold, g.LastGold)
	g.Returns++
	if len(e.Bag) == kept {
		g.logf(LogGood, "%s", lang.T("log.return", e.Num, e.RetreatWhy, Num(e.Gold)))
	} else {
		g.logf(LogGood, "%s", lang.T("log.return_sold", e.Num, e.RetreatWhy, Num(addGold(e.Gold, sold)), Num(e.Gold), len(e.Bag)-kept, Num(sold)))
	}
	if kept > 0 {
		g.logf(LogLoot, "%s", lang.T("log.chest_kept", kept))
	}
	g.note(LogGood, lang.T("guild.log.return", Num(addGold(e.Gold, sold)), e.RetreatWhy, g.DungeonName()))
	g.endExpedition()
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

func (g *Game) readyToDescend(mode ExploreMode) bool {
	if g.Exp.Level.boss() != nil {
		return false
	}
	if _, ok := g.Exp.Level.pathStep(g.Hero.Pos, g.passable, g.wantedItem); ok {
		return false
	}
	return mode == DescendFast || g.explored()
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
		e.Gold = addGold(e.Gold, gold)
		add(LogLoot, "log.pick_gold", Num(gold))
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
	hero    string
	mons    []*monsterRound
	hurt    bool
	monLuck bool
	after   []LogEntry
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
	if r.monLuck {
		parts[len(parts)-1] += " " + g.Exp.MonMisses.Odds()
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
	miss := h.MissChance(m.Dex)
	if e.RNG.Chance(miss) {
		g.emit(Event{Kind: EventMiss, Pos: m.Pos})
		r.hero = lang.T("combat.hero_miss", name)
		if e.HeroMisses.add(miss, &g.BestHeroMisses) {
			r.hero += " " + e.HeroMisses.Odds()
		}
		return
	}
	e.HeroMisses = Streak{}
	e.Attacks++
	if h.Has(PowerStrike) && e.Attacks%PowerStrikeEvery == 0 {
		g.powerStrike(m, r)
		return
	}
	d := damage(e.RNG, h.Atk(), m.Def)
	key := "combat.hero_hit"
	if c := h.CritChance(m.Dex); c > 0 && e.RNG.Chance(c) {
		d = crit(d, h.CritDamage())
		key = "combat.hero_crit"
	}
	if sneak {
		g.emit(Event{Kind: EventSkill, Pos: h.Pos, Value: int(SneakAttack)})
		d *= SneakMult
		key = "ability.sneak_attack.log"
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
	e.Gold = addGold(e.Gold, gold)
	g.Kills++
	r.after = append(r.after, LogEntry{Text: lang.T("log.kill", name, xp, Num(gold)), Kind: LogGood})
	if m.Boss {
		g.killBoss(m, r)
	}
	g.emit(Event{Kind: EventKill, Pos: m.Pos, Big: m.Boss})
	e.Level.removeMonster(m)
	if old := h.Level; h.gainXP(xp) {
		g.emit(Event{Kind: EventLevelUp, Pos: h.Pos})
		r.after = append(r.after, LogEntry{Text: lang.T("log.level_up", h.Level), Kind: LogGood})
		g.unlockGambits(old, r)
	}
	return true
}

func (g *Game) killBoss(m *Monster, r *round) {
	e := g.Exp
	g.BossKills++
	depth := e.Level.Depth
	rarity := Rare
	switch roll := e.RNG.Float(); {
	case roll < BossLegendaryChance:
		rarity = Legendary
	case roll < BossLegendaryChance+BossEpicChance:
		rarity = Epic
	}
	g.note(LogEpic, lang.T("guild.log.boss", lang.T(m.Info().Name), g.Floor(depth), g.DungeonName()))
	if g.lab == nil && g.Floor(depth) >= DungeonFloors {
		r.after = append(r.after, LogEntry{Text: lang.T("log.boss_dead_last", lang.T(m.Info().Name)), Kind: LogEpic})
		g.completeDungeon(r)
		return
	}
	drop := newItemOf(e.RNG, depth, rarity)
	e.Level.Items = append(e.Level.Items, &FloorItem{Pos: m.Pos, Item: drop})
	text := lang.T("log.boss_dead", lang.T(m.Info().Name), drop.Title())
	if !slices.Contains(g.ShortcutList(), depth+1) {
		g.Shortcuts = append(g.ShortcutList(), depth+1)
		text += lang.T("log.shortcut", g.Floor(depth+1))
	}
	r.after = append(r.after, LogEntry{Text: text, Kind: drop.Rarity.LogKind()})
}

// ShortcutList is the set of floors an expedition can start from.
func (g *Game) ShortcutList() []int {
	if len(g.Shortcuts) == 0 {
		return []int{g.Offset() + 1}
	}
	return g.Shortcuts
}

func (g *Game) startFloor() int {
	if slices.Contains(g.ShortcutList(), g.Config.StartFloor) {
		return g.Config.StartFloor
	}
	return g.Offset() + 1
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
		if m.SleepProof > 0 {
			m.SleepProof--
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
		if !l.In(n) || n == hero || !l.CanStep(m.Pos, n) || l.MonsterAt(n) != nil || l.At(n) == Door && !m.OpensDoors() {
			continue
		}
		if nd := dist[n.Y*MapW+n.X]; nd >= 0 && nd < bestD {
			best, bestD = n, nd
		}
	}
	if l.At(best) == Door {
		l.Set(best, OpenDoor)
		return
	}
	m.Pos = best
}

func (g *Game) monsterAttack(m *Monster, r *round) {
	e := g.Exp
	h := g.Hero
	mr := r.of(m)
	miss := e.RNG.Chance(MissChance(m.Dex, h.Attr(Dex)))
	if dodge := !miss && h.Has(Dodge) && e.RNG.Chance(h.DodgeChance()); miss || dodge {
		kind := EventMiss
		if dodge {
			kind = EventDodge
		}
		g.emit(Event{Kind: kind, Pos: h.Pos})
		mr.misses++
		r.monLuck = e.MonMisses.add(MonsterMissChance(m.Dex, h), &g.BestMonMisses)
		return
	}
	e.MonMisses, r.monLuck = Streak{}, false
	d := damage(e.RNG, m.Atk, h.Def())
	if c := CritChance(m.Dex, h.Attr(Dex)); c > 0 && e.RNG.Chance(c) {
		d = crit(d, CritDamageBase)
	}
	g.emit(Event{Kind: EventHurt, Pos: h.Pos, Value: d, Big: d*BigHitDiv >= h.MaxHP()})
	h.HP -= d
	mr.hits = append(mr.hits, d)
	r.hurt = true
	if h.HP <= 0 {
		h.HP = 0
		r.after = append(r.after, LogEntry{Text: lang.T("log.death",
			mr.name, g.Floor(e.Level.Depth), Num(e.Gold-e.deathKeep()), len(e.Bag), Num(e.deathKeep())), Kind: LogBad})
		g.note(LogBad, lang.T("guild.log.death", mr.name, g.Floor(e.Level.Depth), g.DungeonName()))
		g.die(m)
	}
}

func (e *Expedition) deathKeep() int { return pctOf(e.Gold, DeathKeepGoldPct) }

func (g *Game) die(killer *Monster) {
	g.LastDeath = slices.Clone(g.Log[max(0, len(g.Log)-DeathLogLines):])
	g.DeathScene = g.deathScene(killer)
	g.LastGold = g.Exp.deathKeep()
	g.Guild.Gold = addGold(g.Guild.Gold, g.LastGold)
	g.Deaths++
	if g.Auto && g.Config.DeathLimit > 0 {
		g.AutoDeaths++
		if g.AutoDeaths >= g.Config.DeathLimit {
			g.Auto = false
			g.logf(LogBad, "%s", lang.T("log.auto_death_limit", g.AutoDeaths))
		}
	}
	g.endExpedition()
}

func (g *Game) endExpedition() {
	done := g.Exp.DungeonDone
	g.Exp = nil
	if done {
		g.nextDungeon()
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

func (g *Game) chooseStep(mode ExploreMode) (Pos, bool) {
	l := g.Exp.Level
	if g.Exp.Retreating {
		return l.pathStep(g.Hero.Pos, g.passable, func(p Pos) bool { return p == l.Up })
	}
	next, i := g.plan(mode)
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
func (g *Game) plan(mode ExploreMode) (Pos, int) {
	l := g.Exp.Level
	h := g.Hero
	to := func(target Pos) func(Pos) bool { return func(p Pos) bool { return p == target } }
	boss := l.boss()
	seenMonster := func(p Pos) bool {
		m := l.MonsterAt(p)
		return m != nil && (m.Spotted || l.Visible(h.Pos, p))
	}
	rush := func(p Pos) bool {
		return mode == DescendFast && boss == nil && l.IsSeen(l.Down) && p == l.Down
	}
	last := to(l.Down)
	if boss != nil {
		last = to(boss.Pos)
	}
	// The hero only steers around the monsters it sees.
	pass := func(p Pos) bool { return g.known(p) && (l.MonsterAt(p) == nil || !l.Visible(h.Pos, p)) }
	return l.search(h.Pos, pass, seenMonster, g.wantedItem, rush, g.frontier, last)
}

// Doing describes what the hero is up to, for the status line.
func (g *Game) Doing() string {
	if g.Exp == nil {
		return ""
	}
	e := g.Exp
	drive, active := ActExplore, e.ActiveRule-1
	if active >= 0 && active < len(g.Config.Gambits) {
		drive = g.Config.Gambits[active].Do
	}
	if e.Retreating && (active < 0 || active >= g.retreatIndex()) {
		return lang.T("doing.retreat", e.RetreatWhy)
	}
	switch drive {
	case ActIdle:
		return lang.T("doing.idle")
	case ActFlee:
		return lang.T("doing.flee")
	}
	if m := g.adjacentMonster(); m != nil {
		return lang.T("doing.fight", lang.T(m.Info().Name))
	}
	if drive == ActHunt {
		return lang.T("doing.hunt")
	}
	_, i := g.plan(g.modeFor(drive))
	switch {
	case i == goalMonster:
		return lang.T("doing.hunt")
	case i == goalItem:
		return lang.T("doing.loot")
	case i == goalFrontier:
		return lang.T("doing.explore", g.Floor(e.Level.Depth))
	case i == goalLast && e.Level.boss() != nil:
		return lang.T("doing.boss", lang.T(e.Level.boss().Info().Name))
	}
	return lang.T("doing.descend", g.Floor(e.Level.Depth+1))
}
