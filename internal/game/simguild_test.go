package game

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// priceScheme is a set of guild prices to compare: the price of a building's next level and of the next hero.
type priceScheme struct {
	key      string
	building func(b Building, level int) int
	hire     func(heroes int) int
}

// marketPrice is scheme A with the Market at base × (level+1)².
func marketPrice(base int) func(Building, int) int {
	return func(b Building, l int) int {
		if b == Market {
			return base * (l + 1) * (l + 1)
		}
		return BuildingCost(b, l)
	}
}

func pow(base, exp int) int {
	n := 1
	for range exp {
		n *= base
	}
	return n
}

var priceSchemes = []priceScheme{
	{"v18", func(_ Building, l int) int { return 500 * (l + 1) * (l + 1) }, func(n int) int { return 200 * n }},
	{"A", BuildingCost, HireCost},
	{"M30", marketPrice(30000), HireCost},
	{"M60", marketPrice(60000), HireCost},
	{"B", func(b Building, l int) int {
		base := BuildingBase[b]
		if b == MissionTable {
			base = 125000
		}
		return base * pow(3, l)
	}, HireCost},
}

// guildRun plays a whole guild for SIM_DAYS days (21 when unset) or until everything is bought. Every hero
// plays like the rash profile; a share of the gold they bring (SIM_SHARE percent, 25 when unset) is saved for
// the guild, which buys the cheapest of its next hero (while the mission table has room) and the next
// building level.
func guildRun(rec *simRecorder, c Class, ps priceScheme, seed uint64) string {
	days, _ := strconv.Atoi(os.Getenv("SIM_DAYS"))
	share, _ := strconv.Atoi(os.Getenv("SIM_SHARE"))
	capMs := int64(cmp.Or(days, 21)) * 24 * 3600 * 1000
	share = cmp.Or(share, 25)
	const sampleMs = 3600 * 1000
	first := New(seed, c)
	gd := first.Guild
	served := map[*Game]bool{}
	join := func(h *Game) {
		h.Auto = true
		h.Config.DeathLimit = 0
	}
	join(first)
	ser := rec.series(c, ps.key, seed)
	fund, earned, last := 0, 0, gd.Gold
	var bought []string
	var rates []int
	maxed := func() bool {
		for b := range NumBuildings {
			if b != Market && gd.Buildings[b] < BuildingMax[b] {
				return false
			}
		}
		return len(gd.Heroes) >= gd.Slots() && os.Getenv("SIM_MARKET_RUN") == ""
	}
	buy := func(ms int64) {
		for {
			best, price := -1, 0
			if len(gd.Heroes) < gd.Slots() && len(gd.Heroes) < gd.Beds() && len(gd.Recruits) > 0 {
				best, price = int(NumBuildings), ps.hire(len(gd.Heroes))
			}
			for b := range NumBuildings {
				if gd.Buildings[b] < BuildingMax[b] && !(b == Market && gd.MarketLocked()) {
					if p := ps.building(b, gd.Buildings[b]); best < 0 || p < price {
						best, price = int(b), p
					}
				}
			}
			if best < 0 || fund < price || gd.Gold < price {
				return
			}
			h := fmt.Sprintf("%.1fh", float64(ms)/3.6e6)
			if best == int(NumBuildings) {
				gd.Gold -= price - gd.HirePrice()
				rc := gd.Recruits[0].Class
				var pts [NumTraits]int
				pts[auditTraits[rc][0]] = CreationPoints
				g, err := gd.Hire(0, "", rc, pts)
				if err != nil {
					panic(err)
				}
				join(g)
				bought = append(bought, fmt.Sprintf("hero%d@%s", len(gd.Heroes), h))
			} else {
				b := Building(best)
				cost, _ := gd.NextBuilding(b)
				gd.Gold -= price - cost
				gd.Build(b)
				bought = append(bought, fmt.Sprintf("%s%d@%s", b.Key(), gd.Buildings[b], h))
			}
			fund -= price
		}
	}
	var ms int64
	for ms < capMs && !maxed() {
		gd.Advance(GuildTickMs)
		ms += GuildTickMs
		if d := gd.Gold - last; d > 0 {
			earned += d
			fund += d * share / 100
		}
		fund = min(fund, gd.Gold)
		buy(ms)
		for _, h := range gd.Heroes {
			if h.Exp != nil {
				served[h] = false
				continue
			}
			if served[h] {
				continue
			}
			served[h] = true
			h.Auto = true
			gd.Gold -= fund
			smartSpend(h, nil)
			gd.Gold += fund
			h.Config.StartFloor = h.ShortcutList()[len(h.ShortcutList())-1]
		}
		last = gd.Gold
		if ms%sampleMs == 0 {
			rates = append(rates, earned)
			rate := earned
			if n := len(rates); n > 1 {
				rate = earned - rates[n-2]
			}
			rec.add(ser, ms, first)
			pt := &ser.Points[len(ser.Points)-1]
			pt.Earned, pt.Rate, pt.Heroes, pt.Mission = earned, rate, len(gd.Heroes), gd.Buildings[MissionTable]
			pt.Market, pt.ShopLvl, pt.Deepest, pt.Restock = gd.Buildings[Market], gd.ShopLevel(), gd.Deepest(), gd.RestockPrice()
			for b := range NumBuildings {
				pt.Built += gd.Buildings[b]
			}
		}
	}
	rec.done(ser)
	return fmt.Sprintf("%-8s %-4s seed %d: %.1f days earned=%d heroes=%d bought: %s",
		c.Name(), ps.key, seed, float64(ms)/8.64e7, earned, len(gd.Heroes), strings.Join(bought, " "))
}

// TestSimGuild compares the guild prices (SIM_SCHEME: v18, A or B; all when unset) for a guild that starts with
// a hero of class SIM_CLASS.
func TestSimGuild(t *testing.T) {
	cs := os.Getenv("SIM_CLASS")
	if cs == "" {
		t.Skip()
	}
	ci, _ := strconv.Atoi(cs)
	rec := newSimRecorder()
	for _, ps := range priceSchemes {
		if k := os.Getenv("SIM_SCHEME"); k != "" && k != ps.key {
			continue
		}
		fmt.Println("RESULT " + guildRun(rec, Class(ci), ps, simSeeds[0]))
	}
}
