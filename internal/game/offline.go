package game

import "time"

const catchUpChunkMs = 60 * 1000

type Summary struct {
	Away        time.Duration
	Capped      bool
	Expeditions int
	Returns     int
	Deaths      int
	Kills       int
	BossKills   int
	Gold        int
	LevelFrom   int
	LevelTo     int
	DepthFrom   int
	DepthTo     int
	Dungeons    int
}

// Eventful reports whether anything worth showing happened while away.
func (s Summary) Eventful() bool {
	return s.Expeditions+s.Returns+s.Deaths+s.Kills > 0 || s.Gold != 0
}

// CatchUp simulates the time the game was closed, up to OfflineCap, reporting progress
// from 0 to 1 after each chunk.
func (g *Game) CatchUp(away time.Duration, progress func(float64)) Summary {
	s := Summary{
		Away:        away,
		Capped:      away > OfflineCap,
		Expeditions: -g.ExpNum,
		Returns:     -g.Returns,
		Deaths:      -g.Deaths,
		Kills:       -g.Kills,
		BossKills:   -g.BossKills,
		Gold:        -g.Gold,
		LevelFrom:   g.Hero.Level,
		DepthFrom:   g.Reached,
		Dungeons:    -g.Dungeon,
	}
	total := min(away, OfflineCap).Milliseconds()
	for done := int64(0); done < total; {
		chunk := min(catchUpChunkMs, total-done)
		g.Advance(chunk)
		done += chunk
		if progress != nil {
			progress(float64(done) / float64(total))
		}
	}
	s.Expeditions += g.ExpNum
	s.Returns += g.Returns
	s.Deaths += g.Deaths
	s.Kills += g.Kills
	s.BossKills += g.BossKills
	s.Gold += g.Gold
	s.LevelTo = g.Hero.Level
	s.DepthTo = g.Reached
	s.Dungeons += g.Dungeon
	return s
}
