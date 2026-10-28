package game

import "testing"

func TestWarriorKeepsTodaysCurve(t *testing.T) {
	h := NewHero(Warrior)
	h.Level = 9
	h.Training[Con], h.Training[Str], h.Training[Dex] = 2, 3, 1
	if h.MaxHP() != 30+8*8+2*5 || h.Atk() != 5+12+3 || h.Def() != 1+4 || h.Spd() != 4+1 {
		t.Fatalf("hp %d atk %d def %d spd %d", h.MaxHP(), h.Atk(), h.Def(), h.Spd())
	}
}

func TestMainAttributeGrowsMost(t *testing.T) {
	for c := range NumClasses {
		h := NewHero(c)
		h.Level = 21
		main := c.DamageAttr()
		for tr := range NumTraits {
			if tr != main && tr != Con && h.Attr(tr) > h.Attr(main) {
				t.Fatalf("%v: %v grows past %v", c, tr, main)
			}
		}
	}
}
