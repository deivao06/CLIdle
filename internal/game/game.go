package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
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
	Config     Config
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
}

func DefaultConfig() Config {
	return Config{Mode: ExploreAll, PotionAtPct: 40, RetreatHPPct: 30, RetreatBagFull: true, PotionsToBring: 3}
}

type Game struct {
	RNG       *RNG
	Hero      *Hero
	Exp       *Expedition
	Config    Config
	Log       []LogEntry
	Gold      int
	ExpNum    int
	AccMs     int64
	MaxDepth  int
	Deaths    int
	Returns   int
	Auto      bool
	Shop      []*Item
	Pending   []*Item
	Upgrades  [NumSlots]int
	Shortcuts []int
	Kills     int
	BossKills int
	Bell      bool
}

func New(seed uint64) *Game {
	g := &Game{
		RNG:    NewRNG(seed),
		Hero:   NewHero(),
		Config: DefaultConfig(),
	}
	g.refreshShop()
	g.logf(LogInfo, "Bem-vindo à cidade. Prepare a expedição na aba 2 e aperte e para partir.")
	return g
}

// StepMs is the duration of the next step: combat rounds have a fixed pace, exploration follows speed.
func (g *Game) StepMs() int64 {
	if g.Exp != nil && g.adjacentMonster() != nil {
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
	g.Exp = &Expedition{Num: g.ExpNum, Seed: seed, RNG: NewRNG(seed), StartEquip: g.Hero.Equip, Config: g.Config}
	g.Hero.HP = g.Hero.MaxHP()
	g.buyPotions()
	g.logf(LogInfo, "Expedição #%d parte rumo à dungeon.", g.ExpNum)
	g.enterLevel(g.startFloor())
}

func (g *Game) enterLevel(depth int) {
	e := g.Exp
	e.Level = GenerateLevel(e.RNG, depth)
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
	l.placeLoot(e.RNG, startRoom)
	if depth%BossEvery == 0 {
		b := spawnBoss(e.RNG, depth)
		b.Pos = l.RandomFloor(e.RNG, l.RoomAt(l.Down))
		l.Mons = append(l.Mons, b)
	}
	g.MaxDepth = max(g.MaxDepth, depth)
	g.Hero.Pos = l.Up
	l.Reveal(g.Hero.Pos)
	g.logf(LogInfo, "Você chega ao andar %d.", depth)
	if b := l.boss(); b != nil {
		g.logf(LogBad, "Uma presença terrível guarda a escada: %s.", b.Info().Name)
	}
}

func (g *Game) step() {
	l := g.Exp.Level
	h := g.Hero
	r := &round{}
	g.monstersAct(r, true)
	if g.Exp != nil {
		g.checkRetreat()
		if m := g.adjacentMonster(); g.needsPotion() {
			heal := min(h.MaxHP()*PotionHealPct/100, h.MaxHP()-h.HP)
			h.HP += heal
			h.Potions--
			r.hero = fmt.Sprintf("você bebe uma poção (+%d HP)", heal)
		} else if m != nil {
			g.heroAttack(m, r)
		} else if next, ok := g.chooseStep(); ok {
			h.Pos = next
			l.Reveal(h.Pos)
			g.pickup(r)
		}
		g.monstersAct(r, false)
	}
	g.flush(r)
	if g.Exp == nil {
		return
	}
	switch {
	case g.Exp.Retreating && h.Pos == l.Up:
		g.returnToTown()
	case !g.Exp.Retreating && h.Pos == l.Down && g.readyToDescend():
		if c := g.Exp.Config.LastFloor; c > 0 && l.Depth >= c {
			g.startRetreat(fmt.Sprintf("andar %d concluído", l.Depth))
			return
		}
		g.enterLevel(l.Depth + 1)
	}
}

func (g *Game) needsPotion() bool {
	h := g.Hero
	return h.Potions > 0 && h.HP*100 < h.MaxHP()*g.Exp.Config.PotionAtPct
}

func (g *Game) checkRetreat() {
	e := g.Exp
	h := g.Hero
	switch {
	case e.Retreating:
	case g.Exp.Config.RetreatBagFull && len(e.Bag) >= BagSize:
		g.startRetreat("mochila cheia")
	case h.Potions == 0 && h.HP*100 < h.MaxHP()*g.Exp.Config.RetreatHPPct:
		g.startRetreat("HP baixo e sem poções")
	}
}

// Retreat orders the hero back to the exit.
func (g *Game) Retreat() {
	if g.Exp != nil && !g.Exp.Retreating {
		g.startRetreat("ordem sua")
	}
}

func (g *Game) startRetreat(why string) {
	g.Exp.Retreating = true
	g.logf(LogInfo, "Retirada (%s): voltando para a saída.", why)
}

func (g *Game) returnToTown() {
	e := g.Exp
	sold := 0
	for _, it := range e.Bag {
		sold += it.Value()
	}
	g.Gold += e.Gold + sold
	g.Returns++
	if len(e.Bag) == 0 {
		g.logf(LogGood, "Expedição #%d volta à cidade: +%d ouro.", e.Num, e.Gold)
	} else {
		g.logf(LogGood, "Expedição #%d volta à cidade: +%d ouro (%d achado + %d itens vendidos por %d).",
			e.Num, e.Gold+sold, e.Gold, len(e.Bag), sold)
	}
	g.Exp = nil
	g.arriveInTown()
}

func (g *Game) wants(f *FloorItem) bool {
	if f.Item == nil {
		return true
	}
	return g.better(f.Item) || len(g.Exp.Bag) < BagSize
}

func (g *Game) better(it *Item) bool {
	cur := g.Hero.Equip[it.Slot]
	return cur == nil || it.Bonus() > cur.Bonus()
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
	if g.Exp.Config.Mode == DescendFast {
		return true
	}
	if _, ok := g.Exp.Level.pathStep(g.Hero.Pos, g.passable, g.wantedItem); ok {
		return false
	}
	return g.explored()
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
		r.after = append(r.after, LogEntry{Text: fmt.Sprintf(format, args...), Kind: kind})
	}
	switch {
	case f.Gold > 0:
		e.Gold += f.Gold
		add(LogLoot, "Você pega %d de ouro.", f.Gold)
	case f.Potion:
		h.Potions++
		add(LogLoot, "Você pega uma poção.")
	case g.better(f.Item):
		it := f.Item
		old := h.Equip[it.Slot]
		h.Equip[it.Slot] = it
		add(it.Rarity.LogKind(), "Você equipa %s (%s, %s).", it.Title(), it.Rarity.Name(), it.StatLabel())
		if old != nil && len(e.Bag) < BagSize {
			e.Bag = append(e.Bag, old)
		}
	default:
		e.Bag = append(e.Bag, f.Item)
		add(f.Item.Rarity.LogKind(), "Você guarda %s (%s) na mochila.", f.Item.Title(), f.Item.Rarity.Name())
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
	name := m.Info().Name
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
			return fmt.Sprintf("o %s erra %dx", mr.name, mr.misses)
		}
		return "o " + mr.name + " erra"
	}
	var dmgs []string
	for _, d := range mr.hits {
		dmgs = append(dmgs, strconv.Itoa(d))
	}
	s := fmt.Sprintf("o %s te acerta (%s)", mr.name, strings.Join(dmgs, ", "))
	if len(mr.hits) > 1 {
		s = fmt.Sprintf("o %s te acerta %dx (%s)", mr.name, len(mr.hits), strings.Join(dmgs, ", "))
	}
	if mr.misses > 0 {
		s += ", erra"
	}
	return s
}

func (g *Game) flush(r *round) {
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
		g.logf(kind, "%s", strings.ToUpper(line[:1])+line[1:])
	}
	for _, e := range r.after {
		g.logf(e.Kind, "%s", e.Text)
	}
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
	name := m.Info().Name
	m.Awake = true
	if e.RNG.Chance(MissChance) {
		r.hero = "você erra o " + name
		return
	}
	d := damage(e.RNG, h.Atk(), m.Def)
	m.HP -= d
	if m.HP > 0 {
		r.hero = fmt.Sprintf("você acerta o %s (%d)", name, d)
		return
	}
	gold := GoldPerKill(m.XP)
	e.Gold += gold
	g.Kills++
	r.after = append(r.after, LogEntry{Text: fmt.Sprintf("Você mata o %s! +%d XP, +%d ouro", name, m.XP, gold), Kind: LogGood})
	if m.Boss {
		g.killBoss(m, r)
	}
	e.Level.removeMonster(m)
	if h.gainXP(m.XP) {
		r.after = append(r.after, LogEntry{Text: fmt.Sprintf("Você sobe para o nível %d!", h.Level), Kind: LogGood})
	}
}

func (g *Game) killBoss(m *Monster, r *round) {
	e := g.Exp
	g.BossKills++
	depth := e.Level.Depth
	drop := newItemOf(e.RNG, depth, Epic)
	e.Level.Items = append(e.Level.Items, &FloorItem{Pos: m.Pos, Item: drop})
	text := fmt.Sprintf("%s foi derrotado! Deixou cair %s.", m.Info().Name, drop.Title())
	if !slices.Contains(g.ShortcutList(), depth+1) {
		g.Shortcuts = append(g.ShortcutList(), depth+1)
		text += fmt.Sprintf(" Atalho para o andar %d liberado.", depth+1)
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
		if !m.Awake {
			if !l.Visible(h.Pos, m.Pos) {
				continue
			}
			m.Awake = true
		}
		m.Energy += float64(m.Spd) / float64(h.Spd())
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
	if e.RNG.Chance(MissChance) {
		mr.misses++
		return
	}
	d := damage(e.RNG, m.Atk, h.Def())
	h.HP -= d
	mr.hits = append(mr.hits, d)
	r.hurt = true
	if h.HP <= 0 {
		h.HP = 0
		r.after = append(r.after, LogEntry{Text: fmt.Sprintf("Você foi morto por um %s no andar %d. Perdeu %d de ouro e %d itens.",
			mr.name, e.Level.Depth, e.Gold, len(e.Bag)+g.foundEquipped()), Kind: LogBad})
		g.die()
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

func (g *Game) die() {
	g.Hero.Equip = g.Exp.StartEquip
	g.Exp = nil
	g.Deaths++
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
	h := g.Hero
	to := func(target Pos) func(Pos) bool { return func(p Pos) bool { return p == target } }
	if g.Exp.Retreating {
		return l.pathStep(h.Pos, g.passable, to(l.Up))
	}
	seenMonster := func(p Pos) bool {
		m := l.MonsterAt(p)
		return m != nil && l.Visible(h.Pos, p)
	}
	boss := l.boss()
	rush := func(p Pos) bool {
		return g.Exp.Config.Mode == DescendFast && boss == nil && l.IsSeen(l.Down) && p == l.Down
	}
	last := to(l.Down)
	if boss != nil {
		last = to(boss.Pos)
	}
	next, i := l.search(h.Pos, g.passable, seenMonster, g.wantedItem, rush, g.frontier, last)
	return next, i >= 0
}
