package game

import "time"

const catchUpChunkMs = 60 * 1000

// Summary is what happened to the guild while the game was closed.
type Summary struct {
	Away   time.Duration
	Capped bool
	Gold   int
	Heroes []HeroSummary
}

// HeroSummary is one hero's share of the time away.
type HeroSummary struct {
	Name        string
	Expeditions int
	Returns     int
	Deaths      int
	Kills       int
	BossKills   int
	LevelFrom   int
	LevelTo     int
	DepthFrom   int
	DepthTo     int
	Dungeons    int
}

// Eventful reports whether anything worth showing happened while away.
func (s Summary) Eventful() bool {
	n := 0
	for _, h := range s.Heroes {
		n += h.Expeditions + h.Returns + h.Deaths + h.Kills
	}
	return n > 0 || s.Gold != 0
}

// CatchUp simulates the time the game was closed, up to OfflineCap, reporting progress
// from 0 to 1 after each chunk.
func (gd *Guild) CatchUp(away time.Duration, progress func(float64)) Summary {
	s := Summary{Away: away, Capped: away > OfflineCap, Gold: -gd.Gold}
	for _, g := range gd.Heroes {
		s.Heroes = append(s.Heroes, HeroSummary{
			Name:        g.Hero.Name,
			Expeditions: -g.ExpNum,
			Returns:     -g.Returns,
			Deaths:      -g.Deaths,
			Kills:       -g.Kills,
			BossKills:   -g.BossKills,
			LevelFrom:   g.Hero.Level,
			DepthFrom:   g.Reached,
			Dungeons:    -g.Dungeon,
		})
	}
	total := min(away, OfflineCap).Milliseconds()
	for done := int64(0); done < total; {
		chunk := min(catchUpChunkMs, total-done)
		gd.Advance(chunk)
		done += chunk
		if progress != nil {
			progress(float64(done) / float64(total))
		}
	}
	s.Gold += gd.Gold
	for i, g := range gd.Heroes[:len(s.Heroes)] {
		h := &s.Heroes[i]
		h.Expeditions += g.ExpNum
		h.Returns += g.Returns
		h.Deaths += g.Deaths
		h.Kills += g.Kills
		h.BossKills += g.BossKills
		h.LevelTo = g.Hero.Level
		h.DepthTo = g.Reached
		h.Dungeons += g.Dungeon
	}
	return s
}
