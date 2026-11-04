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
	Undead   bool
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
	Boss   bool
	Asleep int
	Slowed int
}

func (m *Monster) Info() MonsterKind {
	if m.Boss {
		return BossKinds[m.Kind]
	}
	return MonsterKinds[m.Kind]
}

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
	s := MonsterScale(depth, k.MinDepth)
	hp := scaled(k.HP, s)
	return &Monster{
		Kind:  kind,
		HP:    hp,
		MaxHP: hp,
		Atk:   scaled(k.Atk, s*MonsterAtkMult),
		Def:   scaled(k.Def, s),
		Spd:   k.Spd,
		XP:    scaled(k.XP, s),
	}
}

// spawnBoss builds a boss from the strongest monster of the depth.
func spawnBoss(rng *RNG, depth int) *Monster {
	var base *Monster
	for range 20 {
		if m := spawnMonster(rng, depth); base == nil || m.MaxHP > base.MaxHP {
			base = m
		}
	}
	hp := int(float64(base.MaxHP) * BossHPMult)
	return &Monster{
		Kind:  (depth/BossEvery - 1) % len(BossKinds),
		Boss:  true,
		HP:    hp,
		MaxHP: hp,
		Atk:   int(float64(base.Atk) * BossAtkMult),
		Def:   int(float64(base.Def) * BossDefMult),
		Spd:   max(base.Spd, BossMinSpd),
		XP:    base.XP * BossXPMult,
	}
}

func scaled(v int, s float64) int { return int(float64(v)*s + 0.5) }

type Hero struct {
	Class    Class
	Pos      Pos
	HP       int
	MP       int
	Level    int
	XP       int
	Potions  int
	Equip    [NumSlots]*Item
	Training [NumTraits]int
}

func NewHero(c Class) *Hero {
	h := &Hero{Class: c, Level: 1, Potions: StartPotions}
	kind := ClassGear[c].Weapon[0]
	h.Equip[Weapon] = &Item{ID: 1, Slot: Weapon, Kind: kind, Name: ItemNames[Weapon][kind][0].name, Base: int(ItemBase(Weapon, kind, 1, Common) + 0.5)}
	h.HP = h.MaxHP()
	return h
}

func (h *Hero) MaxHP() int {
	c := ClassHP[h.Class]
	return (c.Base + (h.Level-1)*c.PerLevel + h.Attr(Con)*TrainHP) * (100 + h.Effect(Vigor)) / 100
}

func (h *Hero) Effect(k Effect) int {
	n := 0
	for _, it := range h.Equip {
		if it == nil {
			continue
		}
		for _, f := range it.Effects {
			if f.Kind == k {
				n += f.Value
			}
		}
	}
	return n
}

func boosted(n, pct int) int { return n * (100 + pct) / 100 }

func (h *Hero) MaxMP() int {
	if ClassMana[h.Class] == 0 {
		return 0
	}
	return ClassMana[h.Class] + h.Attr(Int)*ManaPerInt
}

func (h *Hero) Attr(t Trait) int {
	s := ClassStats[h.Class]
	return s.Start[t] + s.Growth[t]*(h.Level-1)/2 + h.Training[t]
}

func (h *Hero) Atk() int {
	return HeroBaseAtk + h.bonus(Weapon) + h.Attr(h.Class.DamageAttr())
}
func (h *Hero) Def() int {
	return HeroBaseDef + (h.Level-1)/2 + h.bonus(Armor) + h.bonus(Helmet) + h.bonus(Shield)
}
func (h *Hero) Spd() int { return HeroBaseSpd + h.bonus(Boots) + h.Attr(Dex) }

func (h *Hero) bonus(s Slot) int {
	if h.Equip[s] == nil {
		return 0
	}
	return h.Equip[s].Bonus()
}

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
