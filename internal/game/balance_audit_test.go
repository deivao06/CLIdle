package game

// Balance audit simulator. Skipped unless BALANCE_AUDIT is set, so `go test ./...` stays fast.
//
//	BALANCE_AUDIT=1 AUDIT_OUT=/some/dir go test -run TestBalanceAudit -count=1 -timeout 0 ./internal/game/
//
// Env vars: AUDIT_OUT (output dir), AUDIT_LABEL (model name in the files), AUDIT_HOURS (game hours per
// full run, default 24), AUDIT_SEEDS (comma list, default 1,2,3), AUDIT_FIGHTS (duels per cell, default 400),
// AUDIT_CLASSES (comma list of class indexes, default all), AUDIT_MAXDEPTH (stop a full run at this floor, default 100).
//
// Layer A runs the real game (New + Advance) with an owner-like spending policy (copy of spendClass in
// sim50_test.go) and StartFloor = deepest shortcut. Layer B runs isolated duels with the real step():
// one hero against one monster (or boss) in an empty lit room, using real spawnMonster/spawnBoss.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

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

func auditEnvInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func auditEnvList(k, def string) []int {
	s := os.Getenv(k)
	if s == "" {
		s = def
	}
	var out []int
	for _, f := range strings.Split(s, ",") {
		if v, err := strconv.Atoi(strings.TrimSpace(f)); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func auditOut(t *testing.T) string {
	dir := os.Getenv("AUDIT_OUT")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "clidle-balance-audit")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func auditLabel() string {
	if s := os.Getenv("AUDIT_LABEL"); s != "" {
		return s
	}
	return "current"
}

func auditCloneHero(h *Hero) *Hero {
	c := *h
	for s, it := range h.Equip {
		if it != nil {
			cp := *it
			cp.Effects = slices.Clone(it.Effects)
			c.Equip[s] = &cp
		}
	}
	return &c
}

func auditTotalXP(h *Hero) int {
	n := h.XP
	for l := 1; l < h.Level; l++ {
		n += XPToNext(l)
	}
	return n
}

// ---------- layer A: full game ----------

type auditFloor struct {
	Sec, Kills, Deaths, XP, Gold int
}

type auditHour struct {
	Hour, Level, MaxDepth, Deaths, Exps, HP, Atk, Def, Spd, Gold, Income, Kills int
	NextCost                                                                    int
}

type auditRun struct {
	Class  Class
	Seed   uint64
	Hours  int
	Reach  map[int]int // floor -> second of first arrival
	Floors map[int]*auditFloor
	Hourly []auditHour
	Snap   map[int]*Hero // hero when it first entered the floor
	Final  auditHour
}

// auditMaxDepth stops a full run early: past it gold and stats overflow in soft models.
var auditMaxDepth = auditEnvInt("AUDIT_MAXDEPTH", 100)

func auditFullRun(c Class, seed uint64, hours int) *auditRun {
	g := New(seed, c)
	g.Auto = true
	r := &auditRun{Class: c, Seed: seed, Hours: hours, Reach: map[int]int{}, Floors: map[int]*auditFloor{}, Snap: map[int]*Hero{}}
	floor := func(d int) *auditFloor {
		if r.Floors[d] == nil {
			r.Floors[d] = &auditFloor{}
		}
		return r.Floors[d]
	}
	income := 0
	snap := func(sec int) auditHour {
		h := g.Hero
		_, cost := auditCheapest(g)
		return auditHour{sec / 3600, h.Level, g.MaxDepth, g.Deaths, g.ExpNum, h.MaxHP(), h.Atk(), h.Def(), h.Spd(), g.Gold, income, g.Kills, cost}
	}
	for sec := 0; sec < hours*3600 && g.MaxDepth < auditMaxDepth; sec++ {
		depth, expGold, gold, deaths, kills0, xp := 0, 0, g.Gold, g.Deaths, g.Kills, auditTotalXP(g.Hero)
		if g.Exp != nil {
			depth, expGold = g.Exp.Level.Depth, g.Exp.Gold
		}
		g.Advance(1000)
		if d := g.Gold - gold; d > 0 {
			income += d
		}
		if depth > 0 {
			f := floor(depth)
			f.Sec++
			f.Kills += g.Kills - kills0
			f.Deaths += g.Deaths - deaths
			f.XP += auditTotalXP(g.Hero) - xp
			if g.Exp != nil && g.Exp.Level.Depth == depth {
				f.Gold += g.Exp.Gold - expGold
			}
		}
		if g.Exp != nil {
			d := g.Exp.Level.Depth
			if _, ok := r.Reach[d]; !ok {
				r.Reach[d] = sec
				r.Snap[d] = auditCloneHero(g.Hero)
			}
		}
		if g.InTown() {
			auditSpend(g)
			g.Config.StartFloor = slices.Max(g.ShortcutList())
		}
		if (sec+1)%3600 == 0 {
			r.Hourly = append(r.Hourly, snap(sec+1))
			income = 0
		}
	}
	r.Final = snap(len(r.Hourly) * 3600)
	return r
}

// ---------- layer B: duels ----------

type auditDuel struct {
	Win, Lose, Flee   int
	Rounds            []int // combat rounds of won fights (steps that start in combat)
	Ms                []int // game ms of won fights, approach included
	HPLost            []float64
	Potions, Fights   int
	Hits              []int // hero landed attacks in won fights
	DmgDealt, DmgTook int
	RoundsAll         int
	XP, Gold          int
	MonHP, MonAtk     int
}

func auditArena(depth int) *Level {
	l := &Level{Depth: depth, Tiles: make([]byte, MapW*MapH), Seen: make([]byte, MapW*MapH)}
	l.carveRoom(Room{10, 3, 30, 13})
	l.Up = Pos{11, 4}
	l.Set(l.Up, StairsUp)
	l.Down = Pos{38, 14}
	l.Set(l.Down, StairsDown)
	return l
}

// auditFight runs one fight with the real step(). raw: no potions and no retreat (pure TTK/TTD).
func auditFight(h *Hero, ms []*Monster, depth int, seed uint64, raw bool, d *auditDuel) {
	hero := auditCloneHero(h)
	hero.HP = hero.MaxHP()
	hero.MP = hero.MaxMP()
	cfg := DefaultConfig()
	if raw {
		hero.Potions = 0
		cfg.Gambits = gambitsV5(configV5{PotionAtPct: 40, RetreatBagFull: true})
	} else {
		hero.Potions = cfg.PotionsToBring
	}
	g := &Game{RNG: NewRNG(seed), Hero: hero, Config: cfg}
	l := auditArena(depth)
	g.Exp = &Expedition{Num: 1, Seed: seed, RNG: NewRNG(seed ^ 0x9e3779b97f4a7c15), Level: l, StartEquip: hero.Equip}
	hero.Pos = Pos{20, 9}
	d.Fights++
	for i, m := range ms {
		mon := *m
		mon.Pos = Pos{24, 9 + []int{0, -2, 2, -4, 4}[i]}
		mon.Awake = !m.Boss
		l.Mons = append(l.Mons, &mon)
		d.MonHP += m.MaxHP
		d.MonAtk += m.Atk
	}
	l.Reveal(hero.Pos)
	monHP := func() int {
		n := 0
		for _, m := range l.Mons {
			n += max(0, m.HP)
		}
		return n
	}
	startXP := auditTotalXP(hero)
	pots := hero.Potions
	rounds, elapsed := 0, 0
	for range 1000 {
		hp, mhp := hero.HP, monHP()
		elapsed += int(g.StepMs())
		if g.inCombat() {
			rounds++
			d.RoundsAll++
		}
		g.step()
		if g.Exp == nil {
			if g.Deaths > 0 {
				d.Lose++
				d.DmgTook += hp
			} else {
				d.Flee++
			}
			d.Potions += pots - hero.Potions
			return
		}
		d.DmgDealt += mhp - monHP()
		if hero.HP < hp {
			d.DmgTook += hp - hero.HP
		}
		if len(l.Mons) == 0 {
			d.Win++
			d.Rounds = append(d.Rounds, rounds)
			d.Ms = append(d.Ms, elapsed)
			d.HPLost = append(d.HPLost, float64(hero.MaxHP()-hero.HP)/float64(hero.MaxHP()))
			d.Hits = append(d.Hits, g.Exp.Attacks)
			d.Potions += pots - hero.Potions
			d.XP += auditTotalXP(hero) - startXP
			d.Gold += g.Exp.Gold
			return
		}
	}
	d.Flee++
}

func auditPct(xs []int, p float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := slices.Clone(xs)
	sort.Ints(s)
	return float64(s[min(len(s)-1, int(p*float64(len(s))))])
}

func auditMean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	t := 0.0
	for _, x := range xs {
		t += x
	}
	return t / float64(len(xs))
}

// auditDuels fights the hero n times against `pack` monsters spawned for depth (real spawn mix) or the boss.
func auditDuels(h *Hero, depth, n int, boss, raw bool, pack int) *auditDuel {
	d := &auditDuel{}
	rng := NewRNG(uint64(depth)*1000003 + uint64(pack)*31 + 7)
	for i := range n {
		var ms []*Monster
		for range pack {
			if boss {
				ms = append(ms, spawnBoss(rng, depth))
			} else {
				ms = append(ms, spawnMonster(rng, depth))
			}
		}
		auditFight(h, ms, depth, uint64(i+1)*7919+uint64(depth), raw, d)
	}
	return d
}

func auditDuelRow(d *auditDuel, maxHP int) []string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
	n := float64(d.Fights)
	ttd := math.Inf(1)
	if d.DmgTook > 0 {
		ttd = float64(maxHP) * float64(d.RoundsAll) / float64(d.DmgTook)
	}
	return []string{
		f(float64(d.Win) / n), f(float64(d.Lose) / n), f(float64(d.Flee) / n),
		f(auditPct(d.Rounds, 0.1)), f(auditPct(d.Rounds, 0.5)), f(auditPct(d.Rounds, 0.9)),
		f(auditPct(d.Ms, 0.5) / 1000), f(auditPct(d.Hits, 0.5)), f(auditMean(d.HPLost)), f(float64(d.Potions) / n),
		f(float64(d.DmgDealt) / float64(d.RoundsAll)), f(float64(d.DmgTook) / float64(d.RoundsAll)), f(ttd),
		f(float64(d.XP) / max(1, float64(d.Win))), f(float64(d.Gold) / max(1, float64(d.Win))),
		f(float64(d.MonHP) / n), f(float64(d.MonAtk) / n),
	}
}

var auditDuelHeader = []string{"win", "lose", "flee", "ttk_p10", "ttk_p50", "ttk_p90", "sec_p50", "hits_p50", "hp_lost", "potions",
	"dps_dealt", "dps_taken", "ttd_rounds", "xp_win", "gold_win", "mon_hp", "mon_atk"}

// ---------- synthetic (hypothetical) heroes ----------

// auditSynthHero builds a hypothetical hero: level lvl, best class gear of itemDepth and rarity r on every
// slot the class can use, `up` smith levels and the given training. Item rolls use the midpoint.
func auditSynthHero(c Class, lvl, itemDepth int, r Rarity, up int, train [NumTraits]int) *Hero {
	h := NewHero(c)
	h.Level = lvl
	h.Training = train
	itemDepth = max(1, itemDepth)
	for s := range NumSlots {
		kind := 0
		switch s {
		case Weapon:
			kind = ClassGear[c].Weapon[0]
		case Armor:
			kind = slices.Max(ClassGear[c].Armor)
		case Shield:
			if !ClassGear[c].Shield {
				h.Equip[s] = nil
				continue
			}
		}
		h.Equip[s] = &Item{ID: uint64(s + 1), Slot: s, Kind: kind, Rarity: r, Base: ItemRoll(s, kind, itemDepth, r, 0.5), Upgrade: up}
	}
	h.HP = h.MaxHP()
	return h
}

// ---------- the test ----------

func auditWrite(t *testing.T, dir, name string, header []string, rows [][]string) {
	var b strings.Builder
	b.WriteString(strings.Join(header, ",") + "\n")
	for _, r := range rows {
		b.WriteString(strings.Join(r, ",") + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBalanceAudit(t *testing.T) {
	if os.Getenv("BALANCE_AUDIT") == "" {
		t.Skip("set BALANCE_AUDIT=1")
	}
	dir, label := auditOut(t), auditLabel()
	suffix := label + "_c" + strings.ReplaceAll(os.Getenv("AUDIT_CLASSES"), ",", "")
	hours := auditEnvInt("AUDIT_HOURS", 24)
	fights := auditEnvInt("AUDIT_FIGHTS", 400)
	var seeds []uint64
	for _, s := range auditEnvList("AUDIT_SEEDS", "1,2,3") {
		seeds = append(seeds, uint64(s))
	}
	classes := auditEnvList("AUDIT_CLASSES", "0,1,2,3")

	// Layer A.
	type job struct {
		c    Class
		seed uint64
	}
	var jobs []job
	for _, c := range classes {
		for _, s := range seeds {
			jobs = append(jobs, job{Class(c), s})
		}
	}
	runs := make([]*auditRun, len(jobs))
	auditParallel(len(jobs), func(i int) { runs[i] = auditFullRun(jobs[i].c, jobs[i].seed, hours) })

	marks := []int{5, 10, 11, 15, 20, 21, 25, 30, 31, 35, 40, 41, 45, 50}
	var reach, hourly, floors [][]string
	for _, r := range runs {
		row := []string{label, r.Class.Name(), fmt.Sprint(r.Seed)}
		for _, f := range marks {
			if s, ok := r.Reach[f]; ok {
				row = append(row, fmt.Sprintf("%.2f", float64(s)/3600))
			} else {
				row = append(row, "")
			}
		}
		fh := r.Final
		row = append(row, fmt.Sprint(fh.MaxDepth), fmt.Sprint(fh.Level), fmt.Sprint(fh.Deaths), fmt.Sprint(fh.Exps), fmt.Sprint(fh.HP), fmt.Sprint(fh.Atk), fmt.Sprint(fh.Def))
		reach = append(reach, row)
		for _, h := range r.Hourly {
			hourly = append(hourly, []string{label, r.Class.Name(), fmt.Sprint(r.Seed), fmt.Sprint(h.Hour), fmt.Sprint(h.Level), fmt.Sprint(h.MaxDepth),
				fmt.Sprint(h.Deaths), fmt.Sprint(h.Exps), fmt.Sprint(h.HP), fmt.Sprint(h.Atk), fmt.Sprint(h.Def), fmt.Sprint(h.Spd),
				fmt.Sprint(h.Gold), fmt.Sprint(h.Income), fmt.Sprint(h.Kills), fmt.Sprint(h.NextCost)})
		}
		var ds []int
		for d := range r.Floors {
			ds = append(ds, d)
		}
		sort.Ints(ds)
		for _, d := range ds {
			f := r.Floors[d]
			floors = append(floors, []string{label, r.Class.Name(), fmt.Sprint(r.Seed), fmt.Sprint(d), fmt.Sprint(f.Sec), fmt.Sprint(f.Kills),
				fmt.Sprint(f.Deaths), fmt.Sprint(f.XP), fmt.Sprint(f.Gold)})
		}
	}
	var reachAll [][]string
	for _, r := range runs {
		for f := 1; f <= 80; f++ {
			if s, ok := r.Reach[f]; ok {
				h := r.Snap[f]
				reachAll = append(reachAll, []string{label, r.Class.Name(), fmt.Sprint(r.Seed), fmt.Sprint(f), fmt.Sprint(s), fmt.Sprint(h.Level), fmt.Sprint(h.MaxHP()), fmt.Sprint(h.Atk()), fmt.Sprint(h.Def()), fmt.Sprint(h.SpellAtk())})
			}
		}
	}
	auditWrite(t, dir, suffix+"_reachall.csv", []string{"model", "class", "seed", "floor", "sec", "lvl", "hp", "atk", "def", "spell_atk"}, reachAll)
	reachHeader := []string{"model", "class", "seed"}
	for _, f := range marks {
		reachHeader = append(reachHeader, fmt.Sprintf("h_f%d", f))
	}
	reachHeader = append(reachHeader, "max_depth", "level", "deaths", "expeditions", "hp", "atk", "def")
	auditWrite(t, dir, suffix+"_reach.csv", reachHeader, reach)
	auditWrite(t, dir, suffix+"_hourly.csv", []string{"model", "class", "seed", "hour", "level", "max_depth", "deaths", "exps", "hp", "atk", "def", "spd", "gold", "income_h", "kills", "next_cost"}, hourly)
	auditWrite(t, dir, suffix+"_floors.csv", []string{"model", "class", "seed", "floor", "sec", "kills", "deaths", "xp", "gold"}, floors)

	// Layer B: duels with the real heroes captured on arrival (seed = first seed), floor f and f+k.
	type cell struct {
		r             *auditRun
		snapF, fightF int
		boss, raw     bool
		pack          int
	}
	var cells []cell
	for _, r := range runs {
		if r.Seed != seeds[0] {
			continue
		}
		for f := 1; f <= 60; f++ {
			if r.Snap[f] == nil {
				continue
			}
			for _, raw := range []bool{true, false} {
				cells = append(cells, cell{r, f, f, false, raw, 1}, cell{r, f, f, false, raw, 2}, cell{r, f, f, false, raw, 3})
				if f%5 == 1 || f%10 == 0 {
					for k := 1; k <= 9; k++ {
						cells = append(cells, cell{r, f, f + k, false, raw, 1})
					}
				}
				if f%BossEvery == 0 {
					cells = append(cells, cell{r, f, f, true, raw, 1})
					if r.Snap[f+1] != nil {
						cells = append(cells, cell{r, f + 1, f, true, raw, 1})
					}
				}
			}
		}
	}
	rows := make([][]string, len(cells))
	auditParallel(len(cells), func(i int) {
		c := cells[i]
		h := c.r.Snap[c.snapF]
		d := auditDuels(h, c.fightF, fights, c.boss, c.raw, c.pack)
		mode := "raw"
		if !c.raw {
			mode = "kit"
		}
		rows[i] = append([]string{label, c.r.Class.Name(), fmt.Sprint(c.r.Seed), fmt.Sprint(c.snapF), fmt.Sprint(c.fightF), fmt.Sprint(c.boss), mode, fmt.Sprint(c.pack),
			fmt.Sprint(h.Level), fmt.Sprint(h.MaxHP()), fmt.Sprint(h.Atk()), fmt.Sprint(h.Def()), fmt.Sprint(h.Spd())}, auditDuelRow(d, h.MaxHP())...)
	})
	auditWrite(t, dir, suffix+"_duels.csv", append([]string{"model", "class", "seed", "hero_floor", "fight_floor", "boss", "mode", "pack", "lvl", "hp", "atk", "def", "spd"}, auditDuelHeader...), rows)

	// Layer B': hypothetical synthetic profiles (labelled), every floor 1..60, raw mode.
	type prof struct {
		name   string
		offset int
		r      Rarity
		up     int
	}
	profs := []prof{{"under", -5, Common, 0}, {"adequate", 0, Common, 2}, {"over", 5, Rare, 4}, {"offense", 0, Common, 2}, {"defense", 0, Common, 2}}
	type scell struct {
		c Class
		p prof
		f int
	}
	var scells []scell
	for _, c := range classes {
		for _, p := range profs {
			for f := 1; f <= 60; f++ {
				scells = append(scells, scell{Class(c), p, f})
			}
		}
	}
	srows := make([][]string, len(scells))
	auditParallel(len(scells), func(i int) {
		s := scells[i]
		lvl := auditLevelAt(runs, s.c, s.f)
		h := auditSynthHero(s.c, lvl, s.f+s.p.offset, s.p.r, s.p.up, [NumTraits]int{})
		switch s.p.name {
		case "offense":
			h.Equip[Weapon] = &Item{ID: 9, Slot: Weapon, Kind: h.Equip[Weapon].Kind, Rarity: Epic, Base: ItemRoll(Weapon, h.Equip[Weapon].Kind, s.f, Epic, 0.5), Upgrade: 4}
			h.Equip[Armor].Base = ItemRoll(Armor, h.Equip[Armor].Kind, max(1, s.f-5), Common, 0.5)
		case "defense":
			for _, sl := range []Slot{Armor, Helmet, Shield} {
				if it := h.Equip[sl]; it != nil {
					it.Rarity, it.Base, it.Upgrade = Epic, ItemRoll(sl, it.Kind, s.f, Epic, 0.5), 4
				}
			}
			h.Equip[Weapon].Base = ItemRoll(Weapon, h.Equip[Weapon].Kind, max(1, s.f-5), Common, 0.5)
		}
		d := auditDuels(h, s.f, fights/2, false, true, 1)
		srows[i] = append([]string{label, s.c.Name(), s.p.name, fmt.Sprint(s.f), fmt.Sprint(lvl), fmt.Sprint(h.MaxHP()), fmt.Sprint(h.Atk()), fmt.Sprint(h.Def())}, auditDuelRow(d, h.MaxHP())...)
	})
	auditWrite(t, dir, suffix+"_synth.csv", append([]string{"model", "class", "profile", "floor", "lvl", "hp", "atk", "def"}, auditDuelHeader...), srows)

	// Snapshots for inspection.
	snaps := map[string]map[int]*Hero{}
	for _, r := range runs {
		if r.Seed == seeds[0] {
			snaps[r.Class.Name()] = r.Snap
		}
	}
	b, _ := json.Marshal(snaps)
	_ = os.WriteFile(filepath.Join(dir, suffix+"_snaps.json"), b, 0o644)
}

// auditLevelAt is the median hero level on arrival at floor f over the full runs of class c (fallback: last known).
func auditLevelAt(runs []*auditRun, c Class, f int) int {
	var lv []int
	for _, r := range runs {
		if r.Class != c {
			continue
		}
		best := 1
		for d := 1; d <= f; d++ {
			if h := r.Snap[d]; h != nil {
				best = h.Level
			}
		}
		lv = append(lv, best)
	}
	if len(lv) == 0 {
		return 1
	}
	return int(auditPct(lv, 0.5))
}

// auditParallel runs fn(0..n-1) in order. path.go keeps a global scratch buffer, so the game cannot run on
// several goroutines; parallelize by starting one process per class (AUDIT_CLASSES) instead.
func auditParallel(n int, fn func(int)) {
	for i := range n {
		fn(i)
	}
}
