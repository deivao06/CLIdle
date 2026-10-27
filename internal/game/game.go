package game

import (
	"fmt"
	"strconv"
	"strings"
)

type Expedition struct {
	Num   int
	Seed  uint64
	RNG   *RNG
	Level *Level
}

type Game struct {
	RNG      *RNG
	Hero     *Hero
	Exp      *Expedition
	Log      []LogEntry
	ExpNum   int
	AccMs    int64
	MaxDepth int
	Deaths   int
}

func New(seed uint64) *Game {
	g := &Game{
		RNG:  NewRNG(seed),
		Hero: NewHero(),
	}
	g.StartExpedition()
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
	g.Exp = &Expedition{Num: g.ExpNum, Seed: seed, RNG: NewRNG(seed)}
	g.Hero.HP = g.Hero.MaxHP()
	g.logf(LogInfo, "Expedição #%d parte rumo à dungeon.", g.ExpNum)
	g.enterLevel(1)
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
	g.MaxDepth = max(g.MaxDepth, depth)
	g.Hero.Pos = l.Up
	l.Reveal(g.Hero.Pos)
	g.logf(LogInfo, "Você chega ao andar %d.", depth)
}

func (g *Game) step() {
	l := g.Exp.Level
	h := g.Hero
	r := &round{}
	g.monstersAct(r, true)
	if g.Exp != nil {
		if m := g.adjacentMonster(); m != nil {
			g.heroAttack(m, r)
		} else if next, ok := g.chooseStep(); ok {
			h.Pos = next
			l.Reveal(h.Pos)
		}
		g.monstersAct(r, false)
	}
	g.flush(r)
	if g.Exp == nil {
		return
	}
	if h.Pos == l.Down && g.explored() {
		g.enterLevel(l.Depth + 1)
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
	r.after = append(r.after, LogEntry{Text: fmt.Sprintf("Você mata o %s! +%d XP", name, m.XP), Kind: LogGood})
	e.Level.removeMonster(m)
	if h.gainXP(m.XP) {
		r.after = append(r.after, LogEntry{Text: fmt.Sprintf("Você sobe para o nível %d!", h.Level), Kind: LogGood})
	}
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
		r.after = append(r.after, LogEntry{Text: fmt.Sprintf("Você foi morto por um %s no andar %d.", mr.name, e.Level.Depth), Kind: LogBad})
		g.die()
	}
}

func (g *Game) die() {
	g.Exp = nil
	g.Deaths++
	g.Hero.HP = g.Hero.MaxHP()
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
	seenMonster := func(p Pos) bool {
		m := l.MonsterAt(p)
		return m != nil && l.Visible(h.Pos, p)
	}
	if next, ok := l.pathStep(h.Pos, g.passable, seenMonster); ok {
		return next, true
	}
	if next, ok := l.pathStep(h.Pos, g.passable, g.frontier); ok {
		return next, true
	}
	return l.pathStep(h.Pos, g.passable, func(p Pos) bool { return p == l.Down })
}
