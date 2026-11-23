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
	RNG      *RNG      `json:",omitempty"`
	Recruits []Recruit `json:",omitempty"`
	// RecruitMs counts the time since the recruits were last replaced.
	RecruitMs int64 `json:",omitempty"`
}

// GuildLogSize is how many entries the guild log keeps.
const GuildLogSize = 50

// guildSeed keeps the guild's random numbers apart from its first hero's.
const guildSeed = 0x6a09e667f3bcc909

// link points every hero back at the guild after a load.
func (gd *Guild) link() {
	for _, h := range gd.Heroes {
		h.Guild = gd
	}
	if gd.RNG == nil {
		gd.RNG = NewRNG(uint64(gd.Gold) ^ guildSeed)
	}
	if len(gd.Recruits) == 0 {
		gd.rollRecruits()
	}
}

// Advance runs every hero for ms milliseconds of game time.
func (gd *Guild) Advance(ms int64) {
	for _, h := range gd.Heroes {
		h.Advance(ms)
	}
	gd.RecruitMs += ms
	for gd.RecruitMs >= RecruitRefreshMs {
		gd.RecruitMs -= RecruitRefreshMs
		gd.rollRecruits()
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
