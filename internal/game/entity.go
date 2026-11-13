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
	Dex      int
}

type Monster struct {
	Kind   int
	Pos    Pos
	HP     int
	MaxHP  int
	Atk    int
	Def    int
	Spd    int
	Dex    int
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
	eligible := eligibleKinds(depth)
	return newMonster(eligible[rng.Intn(len(eligible))], depth)
}

func eligibleKinds(depth int) []int {
	var eligible []int
	for i, k := range MonsterKinds {
		if k.MinDepth <= depth {
			eligible = append(eligible, i)
		}
	}
	return eligible[max(0, len(eligible)-MonsterWindow):]
}

// TypicalDex is the average DEX of the monsters that spawn on a floor.
func TypicalDex(depth int) int {
	kinds := eligibleKinds(depth)
	sum := 0
	for _, k := range kinds {
		sum += newMonster(k, depth).Dex
	}
	return (sum + len(kinds)/2) / len(kinds)
}

func newMonster(kind, depth int) *Monster {
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
		Dex:   k.Dex + (depth-k.MinDepth)/MonsterDexEvery,
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
	k := float64(depth/BossEvery - 1)
	hp := int(float64(base.MaxHP) * (BossHPMult + BossHPPerBoss*k))
	return &Monster{
		Kind:  (depth/BossEvery - 1) % len(BossKinds),
		Boss:  true,
		HP:    hp,
		MaxHP: hp,
		Atk:   int(float64(base.Atk) * (BossAtkMult + BossAtkPerBoss*k)),
		Def:   int(float64(base.Def) * BossDefMult),
		Spd:   max(base.Spd, BossMinSpd),
		Dex:   base.Dex + BossDexBonus,
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
	armor := h.bonus(Armor) + h.bonus(Helmet) + h.bonus(Shield)
	return (c.Base + (h.Level-1)*c.PerLevel + h.Attr(Con)*TrainHP + armor*ArmorHPPct/100) * (100 + h.Effect(Vigor)) / 100
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
	return min(n, EffectCap[k])
}

func boosted(n, pct int) int { return pctOf(n, 100+pct) }

func (h *Hero) MaxMP() int {
	if ClassMana[h.Class] == 0 {
		return 0
	}
	return ClassMana[h.Class] + h.Attr(Int)*ClassManaPerInt[h.Class]
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
func (h *Hero) Spd() int {
	return HeroBaseSpd + h.bonus(Boots) + h.Level/SpdLevelDiv + h.Attr(Dex)/SpdDexDiv
}

func (h *Hero) MissChance(def int) float64 { return MissChance(h.Attr(Dex), def) }
func (h *Hero) CritChance(def int) float64 {
	return CritChance(h.Attr(Dex), def) + float64(h.Effect(Crit))/100
}

func (h *Hero) DodgeChance() float64 {
	if !h.Has(Dodge) {
		return 0
	}
	return min(DodgeCap, float64(h.Attr(Dex))*DodgePerDex)
}

func (h *Hero) PotionHeal() int { return h.MaxHP() * (PotionHealPct + h.Effect(PotionBoost)) / 100 }

// ManaPerStep is the mana recovered per step, for display.
func (h *Hero) ManaPerStep() float64 {
	if h.MaxMP() == 0 {
		return 0
	}
	return float64(ManaRegenPerStep+h.Attr(Int)*ClassManaRegenPerInt[h.Class]) / ManaRegenDiv
}

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
