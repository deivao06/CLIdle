package game

import "testing"

func TestBossGuardsTenthFloor(t *testing.T) {
	g := departed(1)
	g.enterLevel(10)
	l := g.Exp.Level
	b := l.boss()
	if b == nil || l.RoomAt(b.Pos) != l.RoomAt(l.Down) {
		t.Fatalf("boss missing or away from stairs: %+v", b)
	}
	g.Hero.Pos = l.Down
	if g.readyToDescend() {
		t.Fatal("cannot descend while the boss lives")
	}
}

func TestKillingBossUnlocksShortcut(t *testing.T) {
	g := departed(2)
	g.enterLevel(10)
	b := g.Exp.Level.boss()
	b.HP = 1
	b.Def = 0
	g.Hero.Pos = b.Pos.Add(Pos{-1, 0})
	for g.Exp.Level.boss() != nil {
		g.heroAttack(b, &round{})
	}
	if g.BossKills != 1 || len(g.ShortcutList()) != 2 || g.ShortcutList()[1] != 11 {
		t.Fatalf("kills %d shortcuts %v", g.BossKills, g.ShortcutList())
	}
	if i := g.Exp.Level.ItemAt(b.Pos); i < 0 || g.Exp.Level.Items[i].Item.Rarity != Epic {
		t.Fatal("boss should drop an epic item")
	}
	g.returnToTown()
	g.Config.StartFloor = 11
	g.Depart()
	if g.Exp.Level.Depth != 11 {
		t.Fatalf("started on %d", g.Exp.Level.Depth)
	}
}

func TestLockedStartFloorFallsBackToFirst(t *testing.T) {
	g := New(3)
	g.Config.StartFloor = 31
	g.Depart()
	if g.Exp.Level.Depth != 1 {
		t.Fatalf("started on %d", g.Exp.Level.Depth)
	}
}
