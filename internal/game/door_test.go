package game

import "testing"

// doorLevel is a bare level with two rooms joined by a closed door at (10, 5).
func doorLevel() (*Game, *Level) {
	g := departed(1)
	l := g.Exp.Level
	for i := range l.Tiles {
		l.Tiles[i] = byte(VWall)
	}
	for x := 1; x < 20; x++ {
		l.Set(Pos{x, 5}, Floor)
	}
	l.Set(Pos{10, 5}, Door)
	l.Mons, l.Items = nil, nil
	return g, l
}

func TestClosedDoorBlocksSight(t *testing.T) {
	_, l := doorLevel()
	if l.Visible(Pos{8, 5}, Pos{12, 5}) {
		t.Fatal("seen through a closed door")
	}
	l.Set(Pos{10, 5}, OpenDoor)
	if !l.Visible(Pos{8, 5}, Pos{12, 5}) {
		t.Fatal("open door should let light through")
	}
}

func TestBeastsStopAtClosedDoors(t *testing.T) {
	_, l := doorLevel()
	dist := l.distMap(Pos{5, 5})
	wolf := &Monster{Kind: 4, Pos: Pos{11, 5}}
	orc := &Monster{Kind: 6, Pos: Pos{11, 5}}
	if !MonsterKinds[wolf.Kind].Beast || MonsterKinds[orc.Kind].Beast {
		t.Fatal("kinds changed")
	}
	l.Mons = []*Monster{wolf}
	wolf.moveToward(l, dist, Pos{5, 5})
	if wolf.Pos != (Pos{11, 5}) || l.At(Pos{10, 5}) != Door {
		t.Fatalf("wolf at %v, door %v", wolf.Pos, l.At(Pos{10, 5}))
	}
	l.Mons = []*Monster{orc}
	orc.moveToward(l, dist, Pos{5, 5})
	if orc.Pos != (Pos{11, 5}) || l.At(Pos{10, 5}) != OpenDoor {
		t.Fatalf("orc should spend its move opening: at %v, door %v", orc.Pos, l.At(Pos{10, 5}))
	}
	orc.moveToward(l, dist, Pos{5, 5})
	if orc.Pos != (Pos{10, 5}) {
		t.Fatalf("orc should walk through: at %v", orc.Pos)
	}
}

func TestRetreatingHeroOpensAndClosesDoors(t *testing.T) {
	g, l := doorLevel()
	for i := range l.Seen {
		l.Seen[i] = 1
	}
	l.Up = Pos{3, 5}
	l.Set(l.Up, StairsUp)
	g.Hero.Pos = Pos{11, 5}
	g.Exp.Retreating = true
	none := len(g.Config.Gambits)
	g.perform(none, &round{})
	if g.Hero.Pos != (Pos{11, 5}) || l.At(Pos{10, 5}) != OpenDoor {
		t.Fatalf("opening should take the step: hero %v door %v", g.Hero.Pos, l.At(Pos{10, 5}))
	}
	g.perform(none, &round{})
	g.perform(none, &round{})
	if g.Hero.Pos != (Pos{9, 5}) || l.At(Pos{10, 5}) != Door {
		t.Fatalf("door should close behind: hero %v door %v", g.Hero.Pos, l.At(Pos{10, 5}))
	}
}
