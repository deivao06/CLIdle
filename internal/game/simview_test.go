package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// simPoint is one sample of a run (every 15 simulated minutes in TestSimDungeon), for tools/simview.
type simPoint struct {
	H       float64 `json:"h"`
	Floor   int     `json:"floor"`
	Level   int     `json:"level"`
	Deaths  int     `json:"deaths"`
	Dungeon int     `json:"dungeon"`
	Gold    int     `json:"gold"`
	Spent   int     `json:"spent"`
	Train   int     `json:"train"`
	// Economy (TestSimEconomy): all the gold earned so far, the gold earned in the last hour, and the gold spent on each thing.
	Earned    int `json:"earned,omitempty"`
	Rate      int `json:"rate,omitempty"`
	Smith     int `json:"smith,omitempty"`
	TrainGold int `json:"trainGold,omitempty"`
	Bag       int `json:"bag,omitempty"`
	Shop      int `json:"shop,omitempty"`
	Potions   int `json:"potions,omitempty"`
	// Guild (TestSimGuild): heroes hired, building levels bought in all and the mission table's level.
	Heroes  int `json:"heroes,omitempty"`
	Built   int `json:"built,omitempty"`
	Mission int `json:"mission,omitempty"`
	// Market (TestSimGuild): the Market's level, the floor the shop's items come from, the guild's deepest floor
	// and what a paid restock costs.
	Market  int `json:"market,omitempty"`
	ShopLvl int `json:"shopLvl,omitempty"`
	Deepest int `json:"deepest,omitempty"`
	Restock int `json:"restock,omitempty"`
	// Dry is the % of expedition time in the last sample with less mana than the class's first spell costs.
	// Mana is the average mana in % of the max over the same time.
	Dry  int `json:"dry,omitempty"`
	Mana int `json:"mana,omitempty"`
}

type simSeries struct {
	Class  string      `json:"class"`
	Label  string      `json:"label"`
	Seed   uint64      `json:"seed"`
	Replay string      `json:"replay,omitempty"`
	Done   bool        `json:"done"`
	Points []simPoint  `json:"points"`
	Fights []bossPoint `json:"fights,omitempty"`
	spent  int
}

// bossPoint is one boss of a boss check: how many of the fights the hero won and how much boss HP was left.
type bossPoint struct {
	Floor    int    `json:"floor"`
	Level    int    `json:"level"`
	Fights   int    `json:"fights"`
	Wins     int    `json:"wins"`
	Pct      int    `json:"pct"`
	BossLeft int    `json:"bossLeft"`
	Seed     uint64 `json:"seed"`
	Won      bool   `json:"won"`
	Code     string `json:"code"`
}

type simFile struct {
	Kind    string       `json:"kind,omitempty"`
	Batch   string       `json:"batch"`
	Title   string       `json:"title"`
	Note    string       `json:"note,omitempty"`
	Started time.Time    `json:"started"`
	Updated time.Time    `json:"updated"`
	Series  []*simSeries `json:"series"`
}

// simRecorder writes a run's points where tools/simview reads them; it does nothing unless SIMVIEW_BATCH is set.
// Each process writes its own file (SIMVIEW_FILE), since the game cannot run in parallel inside one process.
type simRecorder struct {
	path string
	file simFile
	last time.Time
}

func newSimRecorder() *simRecorder {
	batch := os.Getenv("SIMVIEW_BATCH")
	if batch == "" {
		return &simRecorder{}
	}
	name := os.Getenv("SIMVIEW_FILE")
	if name == "" {
		name = "run"
	}
	dir := filepath.Join("..", "..", "tools", "simview", "data", batch)
	os.MkdirAll(dir, 0o755)
	writeSimFormulas(batch)
	return &simRecorder{path: filepath.Join(dir, name+".json"), file: simFile{Batch: batch, Title: os.Getenv("SIMVIEW_TITLE"), Note: os.Getenv("SIMVIEW_NOTE"), Started: time.Now()}}
}

func (r *simRecorder) series(class Class, label string, seed uint64) *simSeries {
	s := &simSeries{Class: class.Name(), Label: label, Seed: seed}
	r.file.Series = append(r.file.Series, s)
	return s
}

func (r *simRecorder) add(s *simSeries, ms int64, g *Game) {
	train := 0
	for _, n := range g.Hero.Training {
		train += n
	}
	s.Points = append(s.Points, simPoint{H: float64(ms) / 3.6e6, Floor: g.MaxDepth, Level: g.Hero.Level, Deaths: g.Deaths, Dungeon: g.Dungeon, Gold: g.Guild.Gold, Spent: s.spent, Train: train})
	if time.Since(r.last) > 500*time.Millisecond {
		r.flush()
	}
}

func (r *simRecorder) done(s *simSeries) {
	s.Done = true
	r.flush()
}

func (r *simRecorder) flush() {
	if r.path == "" {
		return
	}
	r.last = time.Now()
	r.file.Updated = r.last
	data, _ := json.Marshal(r.file)
	tmp := r.path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		os.Rename(tmp, r.path)
	}
}
