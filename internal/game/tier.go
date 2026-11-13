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

// Deepest is the depth of the deepest floor reached in the current dungeon.
func (g *Game) Deepest() int { return max(1, g.Offset()+g.Reached) }

// PrestigeGain is what a prestige adds to every attribute for good: more for each tier cleared since the last one.
func (g *Game) PrestigeGain() int { return PrestigePerTier * g.Dungeon * (g.Dungeon + 1) / 2 }

func (g *Game) CanPrestige() bool {
	return g.InTown() && g.Dungeon > 0 && g.Hero.Level >= PrestigeLevel
}

// Prestige starts over with a new hero of class c in the first tier; only the name, the chest and the legacy bonus stay.
func (g *Game) Prestige(c Class) {
	if !g.CanPrestige() {
		return
	}
	old, gain := g.Hero, g.PrestigeGain()
	g.Hero = NewHero(c)
	g.Hero.Name, g.Hero.Legacy = old.Name, old.Legacy+gain
	if c != old.Class {
		g.Config.Gambits = heroGambits(g.Hero)
	}
	g.Gold, g.BagLevel = 0, 0
	g.Dungeon, g.Reached, g.Shortcuts = 0, 0, nil
	g.Config.StartFloor = 1
	g.Prestiges++
	g.rollDungeonName(g.RNG)
	g.refreshShop()
	g.logf(LogEpic, "%s", lang.T("log.prestige", g.Prestiges, gain, g.DungeonName()))
}
