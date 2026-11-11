package ui

import (
	"fmt"
	"slices"
	"strings"

	"clidle/internal/game"
	"clidle/internal/lang"
)

func sign(d int) int {
	if d < 0 {
		return -1
	}
	return 1
}

func labRow(section, label, value, desc string, adjust func(int)) prepRow {
	return prepRow{section: section, text: fmt.Sprintf("%-24s %s", label, value), label: label, value: value,
		style: sPlain, adjust: adjust, enter: func() { adjust(1) }, desc: desc}
}

func (l Lab) heroRows() []prepRow {
	sc := l.sc
	h := l.g.Hero
	sec := lang.T("lab.section.hero")
	rows := []prepRow{
		labRow(sec, lang.T("lab.class"), sc.Class.Name(), sc.Class.Desc(), func(d int) {
			c := sc.Class
			cycle(&c, sign(d), game.NumClasses)
			sc.SetClass(c)
		}),
		labRow(sec, lang.T("lab.level"), fmt.Sprint(sc.Level), lang.T("lab.level.desc"), func(d int) {
			sc.SetLevel(clamp(sc.Level+d, 1, labMaxLevel))
		}),
		labRow(sec, lang.T("lab.potions"), fmt.Sprint(sc.Potions), lang.T("lab.potions.desc"), func(d int) {
			step(&sc.Potions, d, 1, 0, game.MaxPotions)
		}),
	}
	sec = lang.T("lab.section.train")
	for t := range game.NumTraits {
		rows = append(rows, labRow(sec, t.Name(), lang.T("lab.train.value", sc.Train[t], h.Attr(t)), lang.T("lab.train.desc"),
			func(d int) { step(&sc.Train[t], d, 1, 0, labMaxTrain) }))
	}
	return rows
}

// labKinds is what the class can wear in a slot, as indexes of game.ItemNames.
func labKinds(c game.Class, s game.Slot) []int {
	gear := game.ClassGear[c]
	switch s {
	case game.Weapon:
		return gear.Weapon
	case game.Armor:
		return gear.Armor
	case game.Shield:
		if !gear.Shield {
			return nil
		}
	}
	return []int{0}
}

func (l Lab) gearRows() []prepRow {
	sc := l.sc
	var rows []prepRow
	for s := range game.NumSlots {
		lg := &sc.Gear[s]
		sec := lang.T(slotNames[s])
		sec = strings.ToUpper(sec[:1]) + sec[1:]
		opts := append([]int{-1}, labKinds(sc.Class, s)...)
		at := 0
		if lg.On {
			at = max(0, slices.Index(opts, lg.Kind))
		}
		name, desc := lang.T("lab.none"), lang.T("lab.item.desc")
		if it := lg.Item(s, 0); it != nil {
			name, desc = it.Label(), it.StatLabel()
		}
		row := labRow(sec, lang.T("lab.item"), name, desc, func(d int) {
			i := at
			cycle(&i, sign(d), len(opts))
			lg.On, lg.Kind, lg.Depth = opts[i] >= 0, max(0, opts[i]), max(1, lg.Depth)
		})
		if lg.On {
			row.style = rarityStyle(lg.Rarity)
		}
		rows = append(rows, row)
		if !lg.On {
			continue
		}
		rows = append(rows,
			labRow(sec, lang.T("lab.rarity"), lg.Rarity.Name(), desc, func(d int) {
				cycle(&lg.Rarity, sign(d), game.NumRarities)
				lg.Effects = lg.Effects[:min(len(lg.Effects), game.EffectCount[lg.Rarity])]
			}),
			labRow(sec, lang.T("lab.item_depth"), fmt.Sprint(lg.Depth), lang.T("lab.item_depth.desc"), func(d int) {
				step(&lg.Depth, d, 1, 1, labMaxDepth)
			}),
			labRow(sec, lang.T("lab.upgrade"), fmt.Sprintf("+%d", lg.Upgrade), desc, func(d int) {
				step(&lg.Upgrade, d, 1, 0, game.UpgradeMax)
			}))
		for i := range min(len(lg.Effects)+1, game.EffectCount[lg.Rarity]) {
			rows = append(rows, l.effectRows(sec, lg, i)...)
		}
	}
	return rows
}

// effectRows edits effect i of an item: the kind cycles through the ones it does not have yet.
func (l Lab) effectRows(sec string, lg *game.LabGear, i int) []prepRow {
	opts := []game.Effect{-1}
	for k := range game.NumEffects {
		if !slices.ContainsFunc(lg.Effects, func(f game.ItemEffect) bool { return f.Kind == k }) || i < len(lg.Effects) && lg.Effects[i].Kind == k {
			opts = append(opts, k)
		}
	}
	at, value := 0, lang.T("lab.none")
	if i < len(lg.Effects) {
		at, value = slices.Index(opts, lg.Effects[i].Kind), lg.Effects[i].Label()
	}
	desc := lang.T("lab.effect.desc")
	rows := []prepRow{labRow(sec, lang.T("lab.effect", i+1), value, desc, func(d int) {
		j := at
		cycle(&j, sign(d), len(opts))
		switch k := opts[j]; {
		case k < 0:
			lg.Effects = slices.Delete(lg.Effects, i, i+1)
		case i < len(lg.Effects):
			lg.Effects[i] = game.ItemEffect{Kind: k, Value: game.EffectRange[k][1]}
		default:
			lg.Effects = append(lg.Effects, game.ItemEffect{Kind: k, Value: game.EffectRange[k][1]})
		}
	})}
	if i < len(lg.Effects) {
		f := &lg.Effects[i]
		r := game.EffectRange[f.Kind]
		rows = append(rows, labRow(sec, lang.T("lab.effect_value"), fmt.Sprint(f.Value), desc, func(d int) {
			step(&f.Value, d, 1, r[0], r[1])
		}))
	}
	return rows
}

// labGridRow is the first of the three rows of the room grid on the floor page.
const labGridRow = 3

func (l Lab) floorRows() []prepRow {
	sc, st := l.sc, l.st
	f := sc.Floor(st.floor)
	sec := lang.T("lab.section.floor")
	value := fmt.Sprint(st.floor)
	if f == nil {
		value += lang.T("lab.floor.missing")
	}
	var floors []string
	for _, fl := range sc.Floors {
		if fl.Exists() {
			floors = append(floors, fmt.Sprint(fl.Depth))
		}
	}
	list := lang.T("lab.none")
	if len(floors) > 0 {
		list = strings.Join(floors, ", ")
	}
	seed := fmt.Sprint(sc.Seed)
	if st.typing {
		seed = st.buf + "▏"
	}
	rows := []prepRow{
		labRow(sec, lang.T("lab.depth"), value, lang.T("lab.depth.desc", list), func(d int) {
			st.floor = clamp(st.floor+d, 1, labMaxDepth)
		}),
		cfgRow(sec, "prep.mode", lang.T([...]string{"prep.mode.explore", "prep.mode.descend"}[sc.Mode]), func(int) {
			sc.Mode = 1 - sc.Mode
		}),
		{section: sec, text: fmt.Sprintf("%-24s %s", lang.T("lab.seed"), seed), style: sPlain, desc: lang.T("lab.seed.desc"),
			act: func(*Model) { st.typing, st.buf = true, fmt.Sprint(sc.Seed) }},
	}
	if st.typing {
		rows[2].style = sTitle
	}
	if st.inRoom {
		return l.roomRows(f)
	}
	grid := lang.T("lab.section.rooms")
	for y := range 3 {
		rows = append(rows, prepRow{section: grid, text: l.gridLine(y), style: sPlain, desc: lang.T("lab.grid.desc"),
			adjust: func(d int) {
				st.col = clamp(st.col+sign(d), 0, 2)
				st.room = y*3 + st.col
			},
			act: func(m *Model) { st.inRoom, m.cursor = true, 0 }})
	}
	return rows
}

// roomRows edits the selected room; the grid stays above them, and esc or the last row goes back to it.
func (l Lab) roomRows(f *game.LabFloor) []prepRow {
	st := l.st
	r := st.room
	var rows []prepRow
	sec := lang.T("lab.section.room", r+1, lang.T(fmt.Sprintf("lab.pos.%d", r)))
	on := f != nil && f.Rooms[r]
	rows = append(rows, labRow(sec, lang.T("lab.room_on"), onOff(on), lang.T("lab.room_on.desc"), func(int) { l.toggleRoom(r) }))
	if on {
		rows = append(rows,
			labRow(sec, lang.T("lab.up"), onOff(f.Up == r), lang.T("lab.up.desc"), func(int) { f.Up = r }),
			labRow(sec, lang.T("lab.down"), onOff(f.Down == r), lang.T("lab.down.desc"), func(int) { f.Down = r }))
		for j := range f.Mons[r] {
			rows = append(rows, l.monsterRows(sec, f, r, j)...)
		}
		rows = append(rows, prepRow{section: sec, text: lang.T("lab.add"), style: sTitle, desc: lang.T("lab.add.desc"),
			act: func(*Model) { f.Mons[r] = append(f.Mons[r], game.LabMon{Depth: st.floor, Count: 1}) }})
	}
	return append(rows, prepRow{section: sec, text: lang.T("lab.back_grid"), style: sTitle, desc: lang.T("lab.back_grid.desc"),
		act: st.leaveRoom})
}

// leaveRoom goes back to the grid, with the cursor on the room's row.
func (st *labState) leaveRoom(m *Model) {
	st.inRoom, m.cursor = false, labGridRow+st.room/3
}

// toggleRoom turns a room of the edited floor on or off; a floor left without rooms stops existing.
func (l Lab) toggleRoom(r int) {
	if !l.sc.EditFloor(l.st.floor).Toggle(r) {
		l.st.warn = lang.T("lab.warn.linked")
	}
	l.sc.Prune()
}

// gridLine draws one row of the 3x3 room grid: stairs and monsters of each room, the selected one highlighted.
func (l Lab) gridLine(y int) string {
	f := l.sc.Floor(l.st.floor)
	focus := l.st.inRoom || l.cursor >= labGridRow && l.cursor < labGridRow+3
	var cells []string
	for x := range 3 {
		i := y*3 + x
		text, st := strings.Repeat(" ", labCellW/2)+"·"+strings.Repeat(" ", labCellW-labCellW/2-1), sDim
		if f != nil && f.Rooms[i] {
			var marks []string
			if f.Up == i {
				marks = append(marks, "<")
			}
			if f.Down == i {
				marks = append(marks, ">")
			}
			for _, mg := range f.Mons[i] {
				marks = append(marks, fmt.Sprintf("%d%c", mg.Count, labMonsterKind(mg).Glyph))
			}
			text, st = "["+fmt.Sprintf("%-*s", labCellW-2, truncate(strings.Join(marks, " "), labCellW-2))+"]", sValue
		}
		switch {
		case i == l.st.room && focus:
			st = sTabOn
		case i == l.st.room:
			st = sTitle
		}
		cells = append(cells, st.Render(text))
	}
	return strings.Join(cells, " ")
}

func labMonsterKind(m game.LabMon) game.MonsterKind {
	if m.Boss {
		return game.BossKinds[m.Kind]
	}
	return game.MonsterKinds[m.Kind]
}

func (l Lab) monsterRows(sec string, f *game.LabFloor, r, j int) []prepRow {
	mg := &f.Mons[r][j]
	k := labMonsterKind(*mg)
	name := lang.T(k.Name)
	if mg.Boss {
		name += lang.T("lab.boss_suffix")
	}
	desc := lang.T("lab.boss.desc")
	if m := mg.Monster(); m != nil {
		desc = lang.T("lab.mon.desc", m.MaxHP, m.Atk, m.Def, m.Spd, m.Dex, m.XP)
	}
	kinds := len(game.MonsterKinds)
	kind := func(d int) {
		i := mg.Kind
		if mg.Boss {
			i += kinds
		}
		cycle(&i, sign(d), kinds+len(game.BossKinds))
		mg.Kind, mg.Boss = i%kinds, i >= kinds
		if mg.Boss {
			mg.Kind, mg.Count = i-kinds, 1
		}
		mg.Depth = max(mg.Depth, mg.MinDepth())
		if mg.Boss {
			mg.Depth -= mg.Depth % game.BossEvery
		}
	}
	strength := func(d int) {
		by := 1
		if mg.Boss {
			by = game.BossEvery
		}
		step(&mg.Depth, d, by, mg.MinDepth(), labMaxDepth)
	}
	remove := func() { f.Mons[r] = slices.Delete(f.Mons[r], j, j+1) }
	row := labRow(sec, lang.T("lab.monster", j+1), name, desc+" "+lang.T("lab.remove.desc"), kind)
	row.style, row.sell = *monsterStyle(k), remove
	count := labRow(sec, lang.T("lab.count"), fmt.Sprint(mg.Count), desc, func(d int) { step(&mg.Count, d, 1, 1, labMaxCount) })
	depth := labRow(sec, lang.T("lab.mon_depth"), lang.T("lab.mon_depth.value", mg.Depth), desc, strength)
	count.sell, depth.sell = remove, remove
	return []prepRow{row, count, depth}
}

func (l Lab) configRows() []prepRow {
	g := l.g
	sec := lang.T("lab.section.config")
	return []prepRow{
		cfgRow(sec, "prep.language", lang.T("lang.name"), func(int) { lang.Set(lang.Next(lang.Current())) }),
		cfgRow(sec, "prep.effects", lang.T([...]string{"prep.effects_full", "prep.effects_subtle", "prep.effects_off"}[g.Effects]), func(d int) {
			g.Effects = (g.Effects + game.Effects(sign(d)) + game.NumEffectLevels) % game.NumEffectLevels
		}),
	}
}
