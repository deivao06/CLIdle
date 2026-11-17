package game

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

// simSeeds are the fixed seeds of the dungeon simulation, so a run can be replayed in the playground.
var simSeeds = []uint64{58212943, 71904426, 83351607}

// clearRun plays one hero with a Runner until it clears the first dungeon or 480 simulated hours pass.
func clearRun(rec *simRecorder, c Class, seed uint64, smart bool) string {
	p := ProfileRash
	if smart {
		p = ProfileCareful
	}
	r := NewRunner(c, p, seed)
	g := r.Game
	ser := rec.series(c, p.Key(), seed)
	ser.Replay = ReplayCode(c, p, seed)
	rec.add(ser, 0, g)
	const capMs, sampleMs = 480 * 3600 * 1000, 15 * 60 * 1000
	lv100 := 0
	for r.Ms < capMs && g.Dungeon == 0 {
		r.Advance(min(sampleMs, capMs-r.Ms))
		ser.spent = r.Spent
		rec.add(ser, r.Ms, g)
		if lv100 == 0 && g.Exp != nil && g.Floor(g.Exp.Level.Depth) == DungeonFloors {
			lv100 = g.Hero.Level
		}
	}
	rec.done(ser)
	return fmt.Sprintf("%-8s %-9s seed %d: cleared=%-5v hours=%3.0f level=%2d lv@100=%2d deepest=%3d deaths %d",
		c.Name(), p.Key(), seed, g.Dungeon > 0, float64(r.Ms)/3.6e6, g.Hero.Level, lv100, g.MaxDepth, g.Deaths)
}

// TestSimDungeon runs the player profiles for one class (SIM_CLASS); SIM_PROFILE (rash or careful) runs just
// one of them, so each class and profile can run in its own process.
func TestSimDungeon(t *testing.T) {
	cs := os.Getenv("SIM_CLASS")
	if cs == "" {
		t.Skip()
	}
	ci, _ := strconv.Atoi(cs)
	profiles := []bool{false, true}
	if key := os.Getenv("SIM_PROFILE"); key != "" {
		p, ok := ParseProfile(key)
		if !ok {
			t.Fatal("SIM_PROFILE must be rash or careful")
		}
		profiles = []bool{p == ProfileCareful}
	}
	rec := newSimRecorder()
	for _, smart := range profiles {
		for _, seed := range simSeeds {
			fmt.Println("RESULT " + clearRun(rec, Class(ci), seed, smart))
		}
	}
}
