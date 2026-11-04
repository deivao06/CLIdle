package game

// Event tells the UI about something worth animating. Events never affect the simulation and are not saved.
type Event struct {
	Kind  EventKind
	Pos   Pos
	Value int
	Big   bool
}

type EventKind int

const (
	EventHit EventKind = iota
	EventHurt
	EventKill
	EventLoot
	EventLevelUp
)

const maxEvents = 32

func (g *Game) emit(e Event) {
	if len(g.Events) >= maxEvents {
		g.Events = g.Events[1:]
	}
	g.Events = append(g.Events, e)
}

// Effects is how much map animation the player wants; the zero value shows everything.
type Effects int

const (
	EffectsFull Effects = iota
	EffectsSubtle
	EffectsOff
	NumEffectLevels
)
