package game

import (
	"errors"
	"testing"
)

func TestHireSpendsGoldBedsAndPoints(t *testing.T) {
	g := New(27182818, Warrior)
	gd := g.Guild
	if len(gd.Recruits) != RecruitCount {
		t.Fatalf("%d recruits on offer", len(gd.Recruits))
	}
	if _, err := gd.Hire(0, "", 0, [NumTraits]int{Str: CreationPoints}); !errors.Is(err, ErrNoGold) {
		t.Fatalf("hired without gold: %v", err)
	}
	gd.Gold = 10000
	r := gd.Recruits[0]
	h, err := gd.Hire(0, "", 0, [NumTraits]int{Str: 2, Con: 3})
	if err != nil {
		t.Fatal(err)
	}
	if h.Hero.Name != r.Name || h.Hero.Class != r.Class || h.Guild != gd || len(gd.Heroes) != 2 || len(gd.Recruits) != RecruitCount-1 {
		t.Fatalf("hired %+v", h.Hero)
	}
	if gd.Gold != 10000-HireCost(1) {
		t.Fatalf("gold %d", gd.Gold)
	}
	plain := NewHero(r.Class)
	if h.Hero.Attr(Str) != plain.Attr(Str)+2*TrainGain(Str) || h.Hero.Attr(Con) != plain.Attr(Con)+3 || h.Hero.FreePoints != 0 || h.Hero.HP != h.Hero.MaxHP() {
		t.Fatalf("points not spent: FOR %d CON %d", h.Hero.Attr(Str), h.Hero.Attr(Con))
	}
	if _, err := gd.Hire(-1, "Ana", Mage, [NumTraits]int{Int: CreationPoints - 1}); err == nil {
		t.Fatal("hired with a point left")
	}
	if _, err := gd.Hire(-1, "Ana", Mage, [NumTraits]int{Int: CreationPoints}); err != nil {
		t.Fatal(err)
	}
	if _, err := gd.Hire(0, "", 0, [NumTraits]int{Int: CreationPoints}); !errors.Is(err, ErrNoBed) {
		t.Fatalf("hired past the beds: %v", err)
	}
}

func TestRecruitsChangeEveryHour(t *testing.T) {
	g := New(16180339, Rogue)
	gd := g.Guild
	first := append([]Recruit(nil), gd.Recruits...)
	gd.Advance(RecruitRefreshMs - 1)
	if gd.Recruits[0] != first[0] {
		t.Fatal("recruits changed before an hour")
	}
	gd.Advance(1)
	same := 0
	for i := range first {
		if gd.Recruits[i] == first[i] {
			same++
		}
	}
	if same == len(first) {
		t.Fatal("recruits did not change after an hour")
	}
	gd.Gold = RecruitReroll
	before := append([]Recruit(nil), gd.Recruits...)
	if !gd.RerollRecruits() || gd.Gold != 0 || gd.Recruits[0] == before[0] && gd.Recruits[1] == before[1] && gd.Recruits[2] == before[2] {
		t.Fatal("paying did not reroll the recruits")
	}
}
