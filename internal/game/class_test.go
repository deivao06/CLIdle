package game

import "testing"

func TestWarriorKeepsTodaysCurve(t *testing.T) {
	h := NewHero(Warrior)
	h.Equip[Weapon] = nil
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
		main := c.MainAttr()
		for tr := range NumTraits {
			if tr != main && tr != Con && h.Attr(tr) > h.Attr(main) {
				t.Fatalf("%v: %v grows past %v", c, tr, main)
			}
		}
	}
}

func TestChooseClassOnlyBeforeFirstExpedition(t *testing.T) {
	g := New(1, Warrior)
	g.ChooseClass(Mage)
	if g.Hero.Class != Mage || g.Hero.HP != g.Hero.MaxHP() {
		t.Fatalf("class %v hp %d/%d", g.Hero.Class, g.Hero.HP, g.Hero.MaxHP())
	}
	g.Depart()
	g.ChooseClass(Rogue)
	if g.Hero.Class != Mage {
		t.Fatal("class changed after departing")
	}
}

func TestMageMeleeIgnoresInt(t *testing.T) {
	h := NewHero(Mage)
	h.Equip[Weapon] = nil
	h.Level = 10
	if h.Atk() != HeroBaseAtk+h.Attr(Str) {
		t.Fatalf("mage melee %d uses INT", h.Atk())
	}
}

func TestRogueExploresAtMostTwiceWarriorPace(t *testing.T) {
	w, r := heroAt(Warrior, 17, 1), heroAt(Rogue, 17, 1)
	if r.StepMs()*2 < w.StepMs() {
		t.Fatalf("rogue %dms warrior %dms", r.StepMs(), w.StepMs())
	}
}

func TestHeroStartsWithClassWeapon(t *testing.T) {
	for c := range NumClasses {
		w := NewHero(c).Equip[Weapon]
		if w == nil || !c.CanUse(w) {
			t.Fatalf("%v starts with %v", c, w)
		}
	}
}

func TestWarriorAndClericShareWeapons(t *testing.T) {
	sword, mace := &Item{Slot: Weapon, Kind: Blade}, &Item{Slot: Weapon, Kind: Mace}
	if !Warrior.CanUse(mace) || !Cleric.CanUse(sword) || Mage.CanUse(sword) || Rogue.CanUse(mace) {
		t.Fatal("only warrior and cleric share blades and maces")
	}
}
