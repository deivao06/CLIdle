package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// simPoint is one simulated hour of a run, for tools/simview.
type simPoint struct {
	H       float64 `json:"h"`
	Floor   int     `json:"floor"`
	Level   int     `json:"level"`
	Deaths  int     `json:"deaths"`
	Dungeon int     `json:"dungeon"`
	Gold    int     `json:"gold"`
}

type simSeries struct {
	Class  string     `json:"class"`
	Label  string     `json:"label"`
	Seed   uint64     `json:"seed"`
	Done   bool       `json:"done"`
	Points []simPoint `json:"points"`
}

type simFile struct {
	Batch   string       `json:"batch"`
	Title   string       `json:"title"`
	Started time.Time    `json:"started"`
	Updated time.Time    `json:"updated"`
	Series  []*simSeries `json:"series"`
}

// simRecorder writes a run's hourly points where tools/simview reads them; it does nothing unless SIMVIEW_BATCH is set.
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
	return &simRecorder{path: filepath.Join(dir, name+".json"), file: simFile{Batch: batch, Title: os.Getenv("SIMVIEW_TITLE"), Started: time.Now()}}
}

func (r *simRecorder) series(class Class, label string, seed uint64) *simSeries {
	s := &simSeries{Class: class.Name(), Label: label, Seed: seed}
	r.file.Series = append(r.file.Series, s)
	return s
}

func (r *simRecorder) add(s *simSeries, ms int64, g *Game) {
	s.Points = append(s.Points, simPoint{float64(ms) / 3.6e6, g.MaxDepth, g.Hero.Level, g.Deaths, g.Dungeon, g.Gold})
	if time.Since(r.last) > 2*time.Second {
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
