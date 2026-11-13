package game

// DeathScene is the part of the map the hero saw when it died, kept for the hero tab.
// Glyphs holds what is drawn; Marks says how to color each cell (see the Mark constants).
type DeathScene struct {
	Depth  int
	Glyphs []string
	Marks  []string
}

const (
	MarkBlank   = ' '
	MarkFloor   = '.'
	MarkWall    = 'w'
	MarkDoor    = '+'
	MarkStairs  = 's'
	MarkDim     = 'd'
	MarkHero    = '@'
	MarkMonster = 'm'
	MarkKiller  = 'k'
	MarkGold    = '$'
	MarkPotion  = '!'
)

// MarkItem is the mark of an item of rarity r: '0' for common up to '4' for legendary.
func MarkItem(r Rarity) byte { return byte('0' + r) }

const (
	DeathSceneW = 31
	DeathSceneH = 11
	CorpseGlyph = '@'
)

func (g *Game) deathScene(killer *Monster) *DeathScene {
	l := g.Exp.Level
	h := g.Hero.Pos
	x0 := min(max(0, h.X-DeathSceneW/2), MapW-DeathSceneW)
	y0 := min(max(0, h.Y-DeathSceneH/2), MapH-DeathSceneH)
	s := &DeathScene{Depth: g.Floor(l.Depth)}
	for y := y0; y < y0+DeathSceneH; y++ {
		glyphs, marks := make([]rune, DeathSceneW), make([]byte, DeathSceneW)
		for x := x0; x < x0+DeathSceneW; x++ {
			glyphs[x-x0], marks[x-x0] = g.sceneCell(l, Pos{x, y}, killer)
		}
		s.Glyphs = append(s.Glyphs, string(glyphs))
		s.Marks = append(s.Marks, string(marks))
	}
	return s
}

func (g *Game) sceneCell(l *Level, p Pos, killer *Monster) (rune, byte) {
	h := g.Hero.Pos
	if p == h {
		return CorpseGlyph, MarkHero
	}
	if !l.IsSeen(p) {
		return ' ', MarkBlank
	}
	visible := l.Visible(h, p)
	if m := l.MonsterAt(p); m != nil && visible {
		if m == killer {
			return m.Info().Glyph, MarkKiller
		}
		return m.Info().Glyph, MarkMonster
	}
	if i := l.ItemAt(p); i >= 0 {
		f := l.Items[i]
		switch {
		case !visible:
			return f.Glyph(), MarkDim
		case f.Gold > 0:
			return f.Glyph(), MarkGold
		case f.Potion:
			return f.Glyph(), MarkPotion
		}
		return f.Glyph(), MarkItem(f.Item.Rarity)
	}
	t := l.At(p)
	if !visible {
		return t.Glyph(), MarkDim
	}
	switch t {
	case HWall, VWall:
		return t.Glyph(), MarkWall
	case Door:
		return t.Glyph(), MarkDoor
	case StairsDown, StairsUp:
		return t.Glyph(), MarkStairs
	case Floor, Corridor:
		return t.Glyph(), MarkFloor
	}
	return t.Glyph(), MarkBlank
}
