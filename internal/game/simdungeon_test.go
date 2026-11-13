package game

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"testing"
)

// smartSpend buys shop upgrades first, then the cheapest of blacksmith, training (main attribute and CON) and bag,
// keeping gold for five potions.
func smartSpend(g *Game) {
	for {
		bought := false
		for i, it := range g.Shop {
			cur := g.Hero.Equip[it.Slot]
			if g.Hero.Class.CanUse(it) && (cur == nil || it.Bonus() > cur.Bonus()) && g.Gold-ShopPrice(it) >= 5*PotionPrice {
				g.BuyShop(i)
				bought = true
				break
			}
		}
		if !bought {
			break
		}
	}
	reserve := 5 * PotionPrice
	for {
		best, cost := -1, 1<<62
		try := func(id, c int) {
			if c > 0 && c < cost {
				best, cost = id, c
			}
		}
		for s := range NumSlots {
			if it := g.Hero.Equip[s]; it != nil && it.Upgrade < UpgradeMax {
				try(int(s), g.UpgradeCost(s))
			}
		}
		for _, t := range append(slices.Clone(auditTraits[g.Hero.Class]), Con) {
			try(100+int(t), TrainCost(g.Hero.Training[t]))
		}
		if _, c, ok := g.NextBag(); ok {
			try(200, c)
		}
		if best < 0 || g.Gold-cost < reserve {
			return
		}
		switch {
		case best < 100:
			g.Upgrade(Slot(best))
		case best < 200:
			g.Train(Trait(best - 100))
		default:
			g.UpgradeBag()
		}
	}
}

// clearRun plays one hero until it clears the first dungeon or 240 simulated hours pass. The smart policy also steps
// back a shortcut after three deaths in a row and forward after each return.
func clearRun(rec *simRecorder, c Class, seed uint64, smart bool) string {
	g := New(seed, c)
	label := "old"
	if smart {
		label = "smart"
	}
	ser := rec.series(c, label, seed)
	nextH := int64(0)
	rec.add(ser, 0, g)
	if smart {
		g.Config.PotionsToBring = 5
	}
	var ms int64
	fails, start := 0, 0
	const capMs = 240 * 3600 * 1000
	lv100 := 0
	for ms < capMs && g.Dungeon == 0 {
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
		for g.Exp != nil && ms < capMs {
			g.Advance(60000)
			ms += 60000
			if ms >= nextH {
				rec.add(ser, ms, g)
				nextH = ms + 3600*1000
			}
			if lv100 == 0 && g.Exp != nil && g.Floor(g.Exp.Level.Depth) == DungeonFloors {
				lv100 = g.Hero.Level
			}
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
	rec.add(ser, ms, g)
	rec.done(ser)
	return fmt.Sprintf("%-8s smart=%-5v seed %d: cleared=%-5v hours=%3.0f level=%2d lv@100=%2d deepest=%3d deaths %d",
		c.Name(), smart, seed, g.Dungeon > 0, float64(ms)/3.6e6, g.Hero.Level, lv100, g.MaxDepth, g.Deaths)
}

// TestSimDungeon runs both spending policies for one class (SIM_CLASS); run one process per class.
func TestSimDungeon(t *testing.T) {
	cs := os.Getenv("SIM_CLASS")
	if cs == "" {
		t.Skip()
	}
	ci, _ := strconv.Atoi(cs)
	rec := newSimRecorder()
	for _, smart := range []bool{false, true} {
		for seed := uint64(1); seed <= 3; seed++ {
			fmt.Println("RESULT " + clearRun(rec, Class(ci), seed, smart))
		}
	}
}
