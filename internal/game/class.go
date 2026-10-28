package game

import "clidle/internal/lang"

type Class int

const (
	Warrior Class = iota
	Mage
	Rogue
	Cleric
	NumClasses
)

var classKeys = [NumClasses]string{"warrior", "mage", "rogue", "cleric"}

func (c Class) Name() string { return lang.T("class." + classKeys[c]) }

func (c Class) Desc() string { return lang.T("class." + classKeys[c] + ".desc") }

// DamageAttr is the attribute added to the hero's attack.
func (c Class) DamageAttr() Trait { return [NumClasses]Trait{Str, Int, Dex, Str}[c] }

// ChooseClass picks the hero's class for a brand new game.
func (g *Game) ChooseClass(c Class) {
	if g.ExpNum == 0 {
		g.Hero = NewHero(c)
	}
}
