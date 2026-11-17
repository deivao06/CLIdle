package game

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// referenceRun is the simulated player loop the dungeon simulation used before Runner, kept to check that Runner
// plays the same game; it stops after trips expeditions and returns the simulated time.
func referenceRun(g *Game, smart bool, trips int) int64 {
	if smart {
		g.Config.PotionsToBring = 5
	}
	var ms int64
	fails, start := 0, 0
	for range trips {
		deaths := g.Deaths
		list := g.ShortcutList()
		if smart {
			smartSpend(g)
			start = min(max(start, 0), len(list)-1)
			g.Config.StartFloor = list[start]
		} else {
			auditSpend(g)
			g.Config.StartFloor = slices.Max(list)
		}
		g.Depart()
		for g.Exp != nil {
			g.Advance(60000)
			ms += 60000
		}
		ms += TownWaitMs
		if g.Deaths > deaths {
			if fails++; fails >= 3 {
				start, fails = start-1, 0
			}
		} else {
			fails, start = 0, start+1
		}
	}
	return ms
}

func TestRunnerPlaysTheReferenceGameInAnySlices(t *testing.T) {
	for _, c := range []Class{Mage, Rogue} {
		for p := range NumProfiles {
			ref := New(4, c)
			ms := referenceRun(ref, p == ProfileCareful, 5)
			r := NewRunner(c, p, 4)
			for r.Ms < ms {
				r.Advance(min(7777, ms-r.Ms))
			}
			a, _ := json.Marshal(ref)
			b, _ := json.Marshal(r.Game)
			if r.Ms != ms || string(a) != string(b) {
				t.Fatalf("%v %v: runner at %d, reference at %d, same game %v", c, p.Key(), r.Ms, ms, string(a) == string(b))
			}
		}
	}
}

func TestReplayCode(t *testing.T) {
	for _, code := range []string{ReplayCode(Mage, ProfileRash, 1), " Cleric-Careful-42 "} {
		c, p, seed, ok := ParseReplayCode(code)
		if !ok || ReplayCode(c, p, seed) != strings.ToLower(strings.TrimSpace(code)) {
			t.Fatalf("%q: %v %v %d %v", code, c, p, seed, ok)
		}
	}
	if c, p, seed, ok := ParseReplayCode("mage-afobado-1"); !ok || ReplayCode(c, p, seed) != "mage-rash-1" {
		t.Fatal("old profile keys still read")
	}
	for _, bad := range []string{"", "mage-1", "knight-rash-1", "mage-lazy-1", "mage-rash-x"} {
		if _, _, _, ok := ParseReplayCode(bad); ok {
			t.Errorf("accepted %q", bad)
		}
	}
}
