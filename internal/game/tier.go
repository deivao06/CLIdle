package game

import "clidle/internal/lang"

// Offset is the difficulty the current dungeon adds to its floors: floor f is depth Offset()+f.
func (g *Game) Offset() int { return g.Dungeon * DungeonStep }

// Floor is the floor number inside the current dungeon for a depth.
func (g *Game) Floor(depth int) int { return depth - g.Offset() }

// DungeonName is the current dungeon's name and tier, like "Infernal Cave (Tier 2)".
func (g *Game) DungeonName() string {
	return lang.T("dungeon.name", lang.T(dungeonNouns[g.Noun%len(dungeonNouns)]), lang.T(dungeonAdjs[g.Adj%len(dungeonAdjs)]), g.Dungeon+1)
}

func (g *Game) rollDungeonName(rng *RNG) {
	g.Noun, g.Adj = rng.Intn(len(dungeonNouns)), rng.Intn(len(dungeonAdjs))
}

// completeDungeon rewards the kill of the last floor's boss; the hero heads home and the next dungeon opens on arrival.
func (g *Game) completeDungeon(r *round) {
	e := g.Exp
	e.DungeonDone = true
	it := newItemOf(e.RNG, e.Level.Depth, Legendary)
	g.Stash = append(g.Stash, it)
	gold := goldOf(GoldPerPile(e.Level.Depth) * DungeonGoldPiles)
	g.Gold = addGold(g.Gold, gold)
	g.Hero.Bonus += DungeonAttrBonus
	r.after = append(r.after, LogEntry{Text: lang.T("log.dungeon_done", g.DungeonName(), it.Title(), gold, DungeonAttrBonus), Kind: LogEpic})
	if !e.Retreating {
		g.startRetreat(lang.T("retreat.dungeon_done"))
	}
}

// nextDungeon opens the next dungeon once the hero who cleared the last one is out.
func (g *Game) nextDungeon() {
	g.Dungeon++
	g.Shortcuts, g.Reached = nil, 0
	g.Config.StartFloor = g.Offset() + 1
	g.rollDungeonName(g.RNG)
	g.logf(LogEpic, "%s", lang.T("log.dungeon_next", g.DungeonName()))
}

// dungeonFor is the highest dungeon whose floors include depth.
func dungeonFor(depth int) int { return max(0, (depth-1)/DungeonStep) }
