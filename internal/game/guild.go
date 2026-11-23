package game

// Guild is what the heroes share: the gold, the chest and the game options.
type Guild struct {
	Gold     int
	Stash    []*Item
	Bell     bool
	CRT      bool
	Nerd     bool
	Effects  Effects
	Lang     string
	SeenNews string `json:",omitempty"`
	Log      []LogEntry
	Heroes   []*Game
}

// GuildLogSize is how many entries the guild log keeps.
const GuildLogSize = 50

// link points every hero back at the guild after a load.
func (gd *Guild) link() {
	for _, h := range gd.Heroes {
		h.Guild = gd
	}
}

// Advance runs every hero for ms milliseconds of game time.
func (gd *Guild) Advance(ms int64) {
	for _, h := range gd.Heroes {
		h.Advance(ms)
	}
}

// note adds an important event of a hero to the guild log.
func (g *Game) note(kind LogKind, text string) {
	gd := g.Guild
	gd.Log = append(gd.Log, LogEntry{Text: g.Hero.Name + ": " + text, Kind: kind})
	if n := len(gd.Log) - GuildLogSize; n > 0 {
		gd.Log = gd.Log[n:]
	}
}
