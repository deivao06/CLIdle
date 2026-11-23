package game

import "slices"

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
	// AccMs is the time not yet given to the heroes; Ticket numbers the heroes waiting for the mission table.
	AccMs  int64 `json:",omitempty"`
	Ticket int64 `json:",omitempty"`
	// Buildings holds the level of each building.
	Buildings [NumBuildings]int `json:",omitempty"`
}

// GuildTickMs is the slice of time every hero runs before the next one, so heroes that share the gold and the
// mission table come out the same however the time is cut.
const GuildTickMs = 100

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
	gd.AccMs += ms
	for gd.AccMs >= GuildTickMs {
		gd.AccMs -= GuildTickMs
		for _, h := range gd.Heroes {
			h.Advance(GuildTickMs)
		}
	}
	gd.RecruitMs += ms
	for gd.RecruitMs >= RecruitRefreshMs {
		gd.RecruitMs -= RecruitRefreshMs
		gd.rollRecruits()
	}
}

func (gd *Guild) Slots() int { return MissionSlots + gd.Buildings[MissionTable] }

// Running counts the heroes out on an expedition.
func (gd *Guild) Running() int {
	n := 0
	for _, h := range gd.Heroes {
		if h.Exp != nil {
			n++
		}
	}
	return n
}

// mayStart lets g leave when the mission table has room and no hero has waited longer; otherwise g takes a
// turn in the queue.
func (gd *Guild) mayStart(g *Game) bool {
	ahead := slices.ContainsFunc(gd.Heroes, func(o *Game) bool {
		return o != g && o.Exp == nil && o.Queue != 0 && (g.Queue == 0 || o.Queue < g.Queue)
	})
	if gd.Running() < gd.Slots() && !ahead {
		return true
	}
	if g.Queue == 0 {
		gd.Ticket++
		g.Queue = gd.Ticket
	}
	return false
}

// note adds an important event of a hero to the guild log.
func (g *Game) note(kind LogKind, text string) {
	gd := g.Guild
	gd.Log = append(gd.Log, LogEntry{Text: g.Hero.Name + ": " + text, Kind: kind})
	if n := len(gd.Log) - GuildLogSize; n > 0 {
		gd.Log = gd.Log[n:]
	}
}
