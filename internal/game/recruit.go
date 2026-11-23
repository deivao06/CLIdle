package game

import (
	"errors"
	"slices"
)

// Recruit is a hero the guild can hire: the name and class are fixed, the points are the player's to spend.
type Recruit struct {
	Name  string
	Class Class
}

var recruitNames = []string{"Aldric", "Brenna", "Cedric", "Dagny", "Edda", "Falk", "Gisla", "Halvard", "Ingrid", "Joran",
	"Kaelen", "Liesel", "Magnus", "Nessa", "Osric", "Petra", "Quill", "Rowan", "Sigrun", "Tamsin", "Ulric", "Vala",
	"Wendel", "Ysolde", "Bram", "Corwin", "Elowen", "Garrick", "Hilde", "Isolde", "Leif", "Mira", "Odo", "Rurik", "Sabine", "Theron"}

func (gd *Guild) rollRecruits() {
	taken := func(name string) bool {
		return slices.ContainsFunc(gd.Heroes, func(h *Game) bool { return h.Hero.Name == name }) ||
			slices.ContainsFunc(gd.Recruits, func(r Recruit) bool { return r.Name == name })
	}
	gd.Recruits = nil
	for range RecruitCount {
		name := recruitNames[gd.RNG.Intn(len(recruitNames))]
		for taken(name) {
			name = recruitNames[gd.RNG.Intn(len(recruitNames))]
		}
		gd.Recruits = append(gd.Recruits, Recruit{name, Class(gd.RNG.Intn(int(NumClasses)))})
	}
}

// RerollRecruits replaces the recruits on offer for a price.
func (gd *Guild) RerollRecruits() bool {
	if gd.Gold < RecruitReroll {
		return false
	}
	gd.Gold -= RecruitReroll
	gd.rollRecruits()
	return true
}

func (gd *Guild) Beds() int { return GuildBeds }

func (gd *Guild) HirePrice() int { return HireCost(len(gd.Heroes)) }

var (
	ErrNoBed  = errors.New("no bed")
	ErrNoGold = errors.New("no gold")
)

// CanHire says why a new hero cannot join the guild now, or nil.
func (gd *Guild) CanHire() error {
	switch {
	case len(gd.Heroes) >= gd.Beds():
		return ErrNoBed
	case gd.Gold < gd.HirePrice():
		return ErrNoGold
	}
	return nil
}

// Hire adds a hero to the guild with the points spent; recruit is the index of a recruit on offer, or -1 for
// a hero made by hand.
func (gd *Guild) Hire(recruit int, name string, c Class, pts [NumTraits]int) (*Game, error) {
	if err := gd.CanHire(); err != nil {
		return nil, err
	}
	if recruit >= 0 {
		name, c = gd.Recruits[recruit].Name, gd.Recruits[recruit].Class
	}
	g := New(gd.RNG.Uint64(), c)
	g.Hero.Name = name
	if err := g.Hero.Spend(pts); err != nil {
		return nil, err
	}
	gd.Gold -= gd.HirePrice()
	g.Guild = gd
	gd.Heroes = append(gd.Heroes, g)
	if recruit >= 0 {
		gd.Recruits = slices.Delete(gd.Recruits, recruit, recruit+1)
	}
	return g, nil
}

// Spend puts the hero's free points into the attributes; all of them must be spent.
func (h *Hero) Spend(pts [NumTraits]int) error {
	sum := 0
	for _, p := range pts {
		if p < 0 {
			return errors.New("negative points")
		}
		sum += p
	}
	if sum != h.FreePoints {
		return errors.New("points left")
	}
	for t, p := range pts {
		h.Points[t] += p
	}
	h.FreePoints = 0
	h.HP = h.MaxHP()
	return nil
}
