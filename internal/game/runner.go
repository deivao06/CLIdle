package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"clidle/internal/lang"
)

// Profile is how a simulated player spends gold and picks the starting floor between expeditions.
type Profile int

const (
	ProfileRash Profile = iota
	ProfileCareful
	NumProfiles
)

var profileKeys = [NumProfiles]string{"rash", "careful"}

// profileOldKeys are the keys simulations and replay codes used before 2026-11-17.
var profileOldKeys = [NumProfiles]string{"afobado", "cuidadoso"}

func (p Profile) Key() string  { return profileKeys[p] }
func (p Profile) Name() string { return lang.T("profile." + profileKeys[p]) }

// ParseProfile reads a profile key, the current one or the old one.
func ParseProfile(key string) (Profile, bool) {
	i := slices.Index(profileKeys[:], key)
	if i < 0 {
		i = slices.Index(profileOldKeys[:], key)
	}
	return Profile(max(i, 0)), i >= 0
}

// ReplayCode names a simulated run by class, profile and seed, like mage-careful-58212943.
func ReplayCode(c Class, p Profile, seed uint64) string {
	return fmt.Sprintf("%s-%s-%d", classKeys[c], p.Key(), seed)
}

// ParseReplayCode reads a code made by ReplayCode.
func ParseReplayCode(code string) (Class, Profile, uint64, bool) {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(code)), "-")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	c := slices.Index(classKeys[:], parts[0])
	p, ok := ParseProfile(parts[1])
	seed, err := strconv.ParseUint(parts[2], 10, 64)
	return Class(max(c, 0)), p, seed, c >= 0 && ok && err == nil
}

// RunnerChunkMs is how often the simulated player checks whether the hero is back in town.
const RunnerChunkMs = 60000

// Runner plays a whole game the way a simulated player would: it spends in town, picks the starting floor
// and departs again, checking for the hero's return once a minute. Advance gives the same game whatever the
// slicing, so a replay can stop anywhere.
type Runner struct {
	Game    *Game
	Profile Profile
	Ms      int64
	Spent   int
	InExp   bool
	Chunk   int64
	Fails   int
	Start   int
	Deaths  int
}

// NewRunner starts a new hero of class c for the profile, before its first trip to town.
func NewRunner(c Class, p Profile, seed uint64) *Runner {
	r := &Runner{Game: New(seed, c), Profile: p}
	if p == ProfileCareful {
		r.Game.Config.PotionsToBring = 5
	}
	return r
}

// Advance plays ms of simulated time; the town visit between expeditions is instant and is followed by a
// TownWaitMs pause that only the clock sees. It stops once the first dungeon is cleared.
func (r *Runner) Advance(ms int64) {
	g := r.Game
	end := r.Ms + ms
	for r.Ms < end && g.Dungeon == 0 {
		if !r.InExp {
			r.depart()
		}
		step := min(RunnerChunkMs-r.Chunk, end-r.Ms)
		g.Advance(step)
		r.Ms += step
		if r.Chunk += step; r.Chunk == RunnerChunkMs {
			r.Chunk = 0
			if g.Exp == nil {
				r.back()
			}
		}
	}
}

func (r *Runner) depart() {
	g := r.Game
	r.Deaths = g.Deaths
	before := g.Guild.Gold
	list := g.ShortcutList()
	if r.Profile == ProfileCareful {
		smartSpend(g)
		r.Start = min(max(r.Start, 0), len(list)-1)
		g.Config.StartFloor = list[r.Start]
	} else {
		auditSpend(g)
		g.Config.StartFloor = slices.Max(list)
	}
	g.Depart()
	r.Spent += before - g.Guild.Gold
	r.InExp = true
}

// back counts the trip that just ended: the careful profile steps back a shortcut after three deaths in a row
// and forward after each return.
func (r *Runner) back() {
	r.InExp = false
	r.Ms += TownWaitMs
	if r.Game.Deaths > r.Deaths {
		if r.Fails++; r.Fails >= 3 {
			r.Start, r.Fails = r.Start-1, 0
		}
	} else {
		r.Fails, r.Start = 0, r.Start+1
	}
}

var auditTraits = [NumClasses][]Trait{
	Warrior: {Str, Con},
	Mage:    {Int, Con},
	Rogue:   {Dex},
	Cleric:  {Int, Con},
}

// auditSpend is the same policy as spendClass (sim50_test.go): buy the cheapest of smith/train/bag while
// keeping gold for the potions.
func auditSpend(g *Game) {
	reserve := g.Config.PotionsToBring * PotionPrice
	for {
		best, cost := auditCheapest(g)
		if best < 0 || g.Guild.Gold-cost < reserve {
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

func auditCheapest(g *Game) (int, int) {
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
	for _, t := range auditTraits[g.Hero.Class] {
		try(100+int(t), TrainCost(g.Hero.Training[t]))
	}
	if _, c, ok := g.NextBag(); ok {
		try(200, c)
	}
	return best, cost
}

// smartSpend buys shop upgrades first, then the cheapest of blacksmith, training (main attribute and CON) and bag,
// keeping gold for five potions.
func smartSpend(g *Game) {
	for {
		bought := false
		for i, it := range g.Shop {
			cur := g.Hero.Equip[it.Slot]
			if g.Hero.Class.CanUse(it) && (cur == nil || it.Bonus() > cur.Bonus()) && g.Guild.Gold-ShopPrice(it) >= 5*PotionPrice {
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
		if best < 0 || g.Guild.Gold-cost < reserve {
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
