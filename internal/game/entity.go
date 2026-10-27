package game

type MonsterKind struct {
	Glyph    rune
	Name     string
	Color    string
	HP       int
	Atk      int
	Def      int
	Spd      int
	XP       int
	MinDepth int
}

type Monster struct {
	Kind   int
	Pos    Pos
	HP     int
	MaxHP  int
	Atk    int
	Def    int
	Spd    int
	XP     int
	Awake  bool
	Energy float64
}

func (m *Monster) Info() MonsterKind { return MonsterKinds[m.Kind] }

func spawnMonster(rng *RNG, depth int) *Monster {
	var eligible []int
	for i, k := range MonsterKinds {
		if k.MinDepth <= depth {
			eligible = append(eligible, i)
		}
	}
	eligible = eligible[max(0, len(eligible)-MonsterWindow):]
	kind := eligible[rng.Intn(len(eligible))]
	k := MonsterKinds[kind]
	s := MonsterScale(depth - k.MinDepth)
	hp := scaled(k.HP, s)
	return &Monster{
		Kind:  kind,
		HP:    hp,
		MaxHP: hp,
		Atk:   scaled(k.Atk, s),
		Def:   scaled(k.Def, s),
		Spd:   k.Spd,
		XP:    scaled(k.XP, s),
	}
}

func scaled(v int, s float64) int { return int(float64(v)*s + 0.5) }

type FloorItem struct {
	Pos Pos
}

type Hero struct {
	Pos   Pos
	HP    int
	Level int
	XP    int
}

func NewHero() *Hero {
	h := &Hero{Level: 1}
	h.HP = h.MaxHP()
	return h
}

func (h *Hero) MaxHP() int { return HeroBaseHP + (h.Level-1)*HeroHPPerLevel }
func (h *Hero) Atk() int   { return HeroBaseAtk + (h.Level-1)*HeroAtkPerLevel/2 }
func (h *Hero) Def() int   { return HeroBaseDef + (h.Level-1)/2 }
func (h *Hero) Spd() int   { return HeroBaseSpd }

func (h *Hero) gainXP(xp int) (leveled bool) {
	h.XP += xp
	for h.XP >= XPToNext(h.Level) {
		h.XP -= XPToNext(h.Level)
		old := h.MaxHP()
		h.Level++
		h.HP += h.MaxHP() - old
		leveled = true
	}
	return leveled
}

func damage(rng *RNG, atk, def int) int {
	d := float64(atk)*(0.8+0.4*rng.Float()) - float64(def)*0.5
	return max(1, int(d+0.5))
}
