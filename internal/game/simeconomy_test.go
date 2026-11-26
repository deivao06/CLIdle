package game

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"testing"
)

// economyRun plays one hero for SIM_HOURS simulated hours (72 when unset), through every dungeon it clears,
// and records how much gold it earns and where the gold goes. SIM_MARKET starts the guild with that Market level.
// The idle profile is a hero on auto-expedition that never buys anything for itself: all its gold is left
// for the guild.
func economyRun(rec *simRecorder, c Class, label string, seed uint64) string {
	const sampleMs, perHour = 15 * 60 * 1000, 4
	hours, _ := strconv.Atoi(os.Getenv("SIM_HOURS"))
	capMs := int64(cmp.Or(hours, 72)) * 3600 * 1000
	var r *Runner
	g := New(seed, c)
	g.Auto = true
	if p, ok := ParseProfile(label); ok {
		r = NewRunner(c, p, seed)
		r.Endless = true
		g = r.Game
	}
	if lvl, _ := strconv.Atoi(os.Getenv("SIM_MARKET")); lvl > 0 {
		g.Guild.Buildings[Market] = lvl
		g.Guild.refreshShop()
	}
	ser := rec.series(c, label, seed)
	if r != nil {
		ser.Replay = ReplayCode(c, r.Profile, seed)
	}
	start := g.Guild.Gold
	var earned []int
	for ms := int64(0); ms < capMs; {
		step := min(sampleMs, capMs-ms)
		var on [NumSpends]int
		inExp, dry, mana := 0, 0, 0
		for done := int64(0); done < step; done += 1000 {
			if r != nil {
				r.Advance(1000)
			} else {
				g.Advance(1000)
			}
			if g.Exp != nil && g.Hero.MaxMP() > 0 {
				inExp++
				mana += g.Hero.MP * 100 / g.Hero.MaxMP()
				if g.Hero.MP < firstSpellCost(c) {
					dry++
				}
			}
		}
		if r != nil {
			ser.spent, on = r.Spent, r.SpentOn
		}
		ms += step
		e := g.Guild.Gold - start + ser.spent
		earned = append(earned, e)
		rate := e
		if len(earned) > perHour {
			rate = e - earned[len(earned)-1-perHour]
		}
		rec.add(ser, ms, g)
		pt := &ser.Points[len(ser.Points)-1]
		pt.Earned, pt.Rate = e, rate
		if inExp > 0 {
			pt.Dry, pt.Mana = dry*100/inExp, mana/inExp
		}
		pt.Smith, pt.TrainGold, pt.Bag, pt.Shop, pt.Potions = on[SpendSmith], on[SpendTrain], on[SpendBag], on[SpendShop], on[SpendPotions]
	}
	rec.done(ser)
	last := ser.Points[len(ser.Points)-1]
	return fmt.Sprintf("%-8s %-8s seed %d: earned=%8d last hour=%6d dungeon=%d deepest=%3d level=%2d spent=%8d (smith %d, train %d, bag %d, shop %d, potions %d)",
		c.Name(), label, seed, last.Earned, last.Rate, g.Dungeon+1, g.MaxDepth, g.Hero.Level, last.Spent, last.Smith, last.TrainGold, last.Bag, last.Shop, last.Potions)
}

// TestSimEconomy measures the gold income of one class (SIM_CLASS) for each player profile (SIM_PROFILE: rash,
// careful or idle; all three when unset).
func TestSimEconomy(t *testing.T) {
	cs := os.Getenv("SIM_CLASS")
	if cs == "" {
		t.Skip()
	}
	ci, _ := strconv.Atoi(cs)
	labels := []string{"idle", ProfileRash.Key(), ProfileCareful.Key()}
	if l := os.Getenv("SIM_PROFILE"); l != "" {
		labels = []string{l}
	}
	rec := newSimRecorder()
	for _, l := range labels {
		for _, seed := range simSeeds {
			fmt.Println("RESULT " + economyRun(rec, Class(ci), l, seed))
		}
	}
}

func firstSpellCost(c Class) int {
	for _, a := range AbilityInfo {
		if a.Class == c && a.Level == 1 && a.Mana > 0 {
			return a.Mana
		}
	}
	return 0
}
