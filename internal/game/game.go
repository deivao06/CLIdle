package game

type Hero struct {
	Pos Pos
	Spd int
}

type Expedition struct {
	Num   int
	Seed  uint64
	RNG   *RNG
	Level *Level
}

type Game struct {
	RNG    *RNG
	Hero   *Hero
	Exp    *Expedition
	Log    []LogEntry
	ExpNum int
	AccMs  int64
}

func New(seed uint64) *Game {
	g := &Game{
		RNG:  NewRNG(seed),
		Hero: &Hero{Spd: HeroBaseSpd},
	}
	g.StartExpedition()
	return g
}

func (g *Game) StepMs() int64 {
	return int64(1000 / StepsPerSec(g.Hero.Spd))
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
	g.logf(LogInfo, "Expedição #%d parte rumo à dungeon.", g.ExpNum)
	g.enterLevel(1)
}

func (g *Game) enterLevel(depth int) {
	e := g.Exp
	e.Level = GenerateLevel(e.RNG, depth)
	g.Hero.Pos = e.Level.Up
	e.Level.Reveal(g.Hero.Pos)
	g.logf(LogInfo, "Você chega ao andar %d.", depth)
}

func (g *Game) step() {
	l := g.Exp.Level
	h := g.Hero
	if next, ok := g.chooseStep(); ok {
		h.Pos = next
		l.Reveal(h.Pos)
	}
	if h.Pos == l.Down && g.explored() {
		g.enterLevel(l.Depth + 1)
	}
}

func (g *Game) known(p Pos) bool {
	l := g.Exp.Level
	return l.IsSeen(p) && l.At(p).Walkable()
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
	if next, ok := l.pathStep(g.Hero.Pos, g.known, g.frontier); ok {
		return next, true
	}
	return l.pathStep(g.Hero.Pos, g.known, func(p Pos) bool { return p == l.Down })
}
