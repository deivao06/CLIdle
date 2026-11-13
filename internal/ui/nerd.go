package ui

import (
	"fmt"

	"clidle/internal/game"
	"clidle/internal/lang"
)

func num(v float64) string { return fmt.Sprintf("%.4g", v) }

func nerdPct(v float64) string { return num(v*100) + "%" }

func nerdRow(section, label, value, desc string) prepRow {
	return prepRow{section: section, text: fmt.Sprintf("%-24s %s", label, value), style: sPlain, desc: desc}
}

// nerdRows lists the balance numbers straight from the game constants, so the tab never goes stale.
func (m Model) nerdRows() []prepRow {
	var rows []prepRow
	sec := ""
	add := func(key, value string) {
		rows = append(rows, nerdRow(sec, lang.T("nerd."+key), value, lang.T("nerd."+key+".desc")))
	}

	sec = lang.T("nerd.section.hero")
	add("hero_atk", fmt.Sprint(game.HeroBaseAtk))
	add("hero_def", fmt.Sprint(game.HeroBaseDef))
	add("hero_spd", fmt.Sprint(game.HeroBaseSpd))
	add("steps", lang.T("nerd.steps.value", num(game.StepsPerSec(0)), num(game.StepsPerSec(1)-game.StepsPerSec(0))))
	add("spd_level", "1/"+fmt.Sprint(game.SpdLevelDiv))
	add("spd_dex", "1/"+fmt.Sprint(game.SpdDexDiv))
	add("xp_next", lang.T("nerd.growth.value", fmt.Sprint(game.XPToNext(1)), ratio(float64(game.XPToNext(21)), float64(game.XPToNext(20)))))
	add("con_hp", fmt.Sprintf("+%d HP", game.TrainHP))
	add("armor_hp", fmt.Sprintf("%d%%", game.ArmorHPPct))
	add("start_potions", fmt.Sprint(game.StartPotions))
	add("potion_heal", fmt.Sprintf("%d%%", game.PotionHealPct))
	add("max_potions", fmt.Sprint(game.MaxPotions))
	add("death_gold", fmt.Sprintf("%d%%", game.DeathKeepGoldPct))
	add("town_wait", fmt.Sprintf("%d s", game.TownWaitMs/1000))
	add("offline_cap", fmt.Sprintf("%d h", int(game.OfflineCap.Hours())))

	sec = lang.T("nerd.section.combat")
	add("damage", lang.T("nerd.damage.value"))
	add("miss_even", nerdPct(game.MissEven))
	add("crit_top", nerdPct(game.CritTop))
	add("dex_soften", fmt.Sprint(game.DexSoften))
	add("combat_round", fmt.Sprintf("%d ms", game.CombatRoundMs))

	sec = lang.T("nerd.section.monsters")
	add("monster_growth", "+"+nerdPct(game.MonsterGrowthLin))
	add("monster_atk", "×"+num(game.MonsterAtkMult))
	add("monster_dex", fmt.Sprint(game.MonsterDexEvery))
	add("monster_window", fmt.Sprint(game.MonsterWindow))
	add("monster_count", fmt.Sprintf("%d–%d", game.MonstersPerLevel(1), game.MonstersPerLevel(1000)))
	for _, k := range game.MonsterKinds {
		desc := lang.T("nerd.monster.desc")
		if k.Undead {
			desc += " " + lang.T("nerd.monster.undead")
		}
		rows = append(rows, nerdRow(sec, string(k.Glyph)+" "+lang.T(k.Name),
			lang.T("nerd.monster.value", k.HP, k.Atk, k.Def, k.Spd, k.XP, k.Dex, k.MinDepth), desc))
	}

	sec = lang.T("nerd.section.bosses")
	add("boss_every", fmt.Sprint(game.BossEvery))
	add("boss_hp", lang.T("nerd.per_boss", "×"+num(game.BossHPMult), "+"+num(game.BossHPPerBoss)))
	add("boss_atk", lang.T("nerd.per_boss", "×"+num(game.BossAtkMult), "+"+num(game.BossAtkPerBoss)))
	add("boss_def", "×"+num(game.BossDefMult))
	add("boss_xp", "×"+num(game.BossXPMult))
	add("boss_spd", fmt.Sprint(game.BossMinSpd))
	add("boss_dex", fmt.Sprintf("+%d", game.BossDexBonus))
	add("boss_wake", fmt.Sprint(game.BossWakeDist))
	add("boss_epic", nerdPct(game.BossEpicChance))
	add("boss_legendary", nerdPct(game.BossLegendaryChance))

	sec = lang.T("nerd.section.dungeons")
	add("dungeon_floors", fmt.Sprint(game.DungeonFloors))
	add("dungeon_step", fmt.Sprint(game.DungeonStep))
	add("dungeon_bonus", fmt.Sprintf("+%d", game.DungeonAttrBonus))
	add("dungeon_gold", fmt.Sprint(game.DungeonGoldPiles))

	sec = lang.T("nerd.section.loot")
	add("gold_piles", fmt.Sprintf("%d–%d", game.GoldPilesMin, game.GoldPilesMax))
	add("gold_pile", lang.T("nerd.growth.value", num(game.GoldPerPile(1)), ratio(game.GoldPerPile(11), game.GoldPerPile(10))))
	add("gold_kill", lang.T("nerd.gold_kill.value", game.GoldKillDiv))
	add("items", fmt.Sprintf("%d–%d", game.ItemsMin, game.ItemsMax))
	add("potion_chance", nerdPct(game.PotionChance))
	for r := game.Magic; r < game.NumRarities; r++ {
		rows = append(rows, nerdRow(sec, lang.T("nerd.rarity_chance", r.Name()), nerdPct(game.RarityChance[r]), lang.T("nerd.rarity_chance.desc")))
	}

	sec = lang.T("nerd.section.items")
	for _, sl := range []game.Slot{game.Weapon, game.Armor, game.Helmet, game.Boots} {
		rows = append(rows, nerdRow(sec, lang.T("nerd.item_base", lang.T(slotNames[sl])), lang.T("nerd.floors.value",
			itemBase(sl, 1), itemBase(sl, 10), itemBase(sl, 30)), lang.T("nerd.item_base."+[...]string{"weapon", "armor", "boots", "helmet"}[sl]+".desc")))
	}
	for r := game.Common; r < game.NumRarities; r++ {
		rows = append(rows, nerdRow(sec, lang.T("nerd.rarity_mult", r.Name()), "×"+num(game.ItemBase(game.Weapon, 0, 0, r)/game.ItemBase(game.Weapon, 0, 0, game.Common)), lang.T("nerd.rarity_mult.desc")))
	}
	for i, w := range []string{"light", "medium", "heavy"} {
		rows = append(rows, nerdRow(sec, lang.T("nerd.weight."+w), "×"+num(game.ItemBase(game.Armor, i, 0, 0)), lang.T("nerd.weight.desc")))
	}
	add("item_spread", "±"+nerdPct(game.ItemSpread))
	add("effects", lang.T("nerd.effects.value", game.EffectCount[game.Rare], game.EffectCount[game.Epic], game.EffectCount[game.Legendary]))
	for e := range game.NumEffects {
		r := game.EffectRange[e]
		rows = append(rows, nerdRow(sec, lang.T("nerd.effect."+effectKeys[e]), lang.T("nerd.effect.range", r[0], r[1], game.EffectCap[e]), lang.T("nerd.effect."+effectKeys[e]+".desc")))
	}

	sec = lang.T("nerd.section.town")
	add("bag", fmt.Sprint(game.BagSize))
	add("bag_slots", fmt.Sprintf("+%d × %d", game.BagSlotsPerLevel, game.BagMaxLevel))
	add("bag_cost", lang.T("nerd.growth.value", fmt.Sprint(game.BagCost(0)), ratio(float64(game.BagCost(1)), float64(game.BagCost(0)))))
	add("shop_size", fmt.Sprint(game.ShopSize))
	add("shop_markup", fmt.Sprintf("×%d", game.ShopMarkup))
	add("item_value", fmt.Sprint(game.ItemValuePiles))
	add("effect_value", fmt.Sprint(game.EffectValuePiles))
	add("upgrade_refund", fmt.Sprintf("%d%%", game.UpgradeRefundPct))
	add("potion_price", fmt.Sprint(game.PotionPrice))
	add("potion_sell", fmt.Sprint(game.PotionSellPrice))
	add("upgrade", "1/"+fmt.Sprint(game.UpgradeDivisor))
	add("upgrade_max", fmt.Sprintf("+%d", game.UpgradeMax))
	big := &game.Item{Base: 1000}
	add("upgrade_cost", lang.T("nerd.upgrade_cost.value", ratio(float64(game.UpgradeCost(big, 1)), float64(game.UpgradeCost(big, 0)))))
	add("max_gold", fmt.Sprint(game.MaxGold))
	add("train", lang.T("nerd.growth.value", fmt.Sprint(game.TrainCost(0)), ratio(float64(game.TrainCost(10)), float64(game.TrainCost(9)))))

	sec = lang.T("nerd.section.classes")
	for c := range game.NumClasses {
		hp := game.ClassHP[c]
		rows = append(rows, nerdRow(sec, c.Name(), lang.T("nerd.class.value", hp.Base, hp.PerLevel)+classStats(c), lang.T("nerd.class.desc")))
		if game.ClassMana[c] > 0 {
			rows = append(rows, nerdRow(sec, lang.T("nerd.mana", c.Name()), lang.T("nerd.mana.value", game.ClassMana[c], game.ClassManaPerInt[c],
				num(float64(game.ClassManaRegenPerInt[c])/game.ManaRegenDiv)), lang.T("nerd.mana.desc")))
		}
	}
	add("mana_regen", num(float64(game.ManaRegenPerStep)/game.ManaRegenDiv))

	sec = lang.T("nerd.section.abilities")
	for a := range game.NumAbilities {
		info := game.AbilityInfo[a]
		v := lang.T("nerd.ability.value", info.Class.Name(), info.Level)
		if info.Mana > 0 {
			v += lang.T("nerd.ability.mana", info.Mana)
		}
		rows = append(rows, nerdRow(sec, a.Name(), v, a.Desc()))
	}

	sec = lang.T("nerd.section.ability_numbers")
	add("power_strike", fmt.Sprint(game.PowerStrikeEvery))
	add("second_wind", fmt.Sprintf("%d%%", game.SecondWindPct))
	add("second_wind_heal", fmt.Sprintf("%d%%", game.SecondWindHealPct))
	add("sneak", fmt.Sprintf("×%d", game.SneakMult))
	add("dodge", nerdPct(game.DodgePerDex))
	add("dodge_cap", nerdPct(game.DodgeCap))
	add("nose_gold", "×"+num(game.NoseGoldMult))
	add("nose_item", nerdPct(game.NoseItemChance))
	add("spell_range", fmt.Sprint(game.SpellRange))
	for _, a := range []game.Ability{game.MagicMissile, game.Fireball, game.FrostRay} {
		rows = append(rows, nerdRow(sec, lang.T("nerd.spell_power", a.Name()), fmt.Sprint(game.SpellPower[a]), lang.T("nerd.spell_power.desc")))
	}
	add("sleep_min", fmt.Sprintf("%d%%", game.SleepMinHPPct))
	add("sleep", fmt.Sprint(game.SleepRounds))
	add("sleep_boss", fmt.Sprint(game.SleepBossRounds))
	add("frost", fmt.Sprint(game.FrostSlowRounds))
	add("blink", fmt.Sprintf("%d%%", game.BlinkHPPct))
	add("heal", fmt.Sprintf("%d%%", game.HealPct))
	add("bless", fmt.Sprint(game.BlessRounds))
	add("bless_def", fmt.Sprintf("+%d", game.BlessDef))
	add("turn_undead", fmt.Sprintf("×%d", game.TurnUndeadMult))

	sec = lang.T("nerd.section.luck")
	h := m.g.Hero
	add("luck_miss", nerdPct(h.MissChance(m.typicalDex())))
	add("luck_dodge", nerdPct(h.DodgeChance()))
	add("luck_min", fmt.Sprint(game.LuckStreakMin))
	add("luck_hero", streakValue(m.g.BestHeroMisses))
	add("luck_mon", streakValue(m.g.BestMonMisses))
	return rows
}

func streakValue(s game.Streak) string {
	if s.Run == 0 {
		return "—"
	}
	return lang.T("nerd.luck.streak", s.Run, s.Odds())
}

func ratio(next, cur float64) string { return "×" + num(next/cur) }

var effectKeys = [game.NumEffects]string{"life_steal", "crit", "vigor", "gold_find", "potion_boost", "xp_boost"}

func itemBase(sl game.Slot, depth int) string {
	return fmt.Sprint(game.ItemRoll(sl, int(game.Medium), depth, game.Common, 0.5))
}

func classStats(c game.Class) string {
	var s string
	st := game.ClassStats[c]
	for t := range game.NumTraits {
		if st.Start[t] > 0 {
			s += lang.T("nerd.class.start", statName(t), st.Start[t])
		}
		if st.Growth[t] > 0 {
			s += lang.T("nerd.class.growth", statName(t), st.Growth[t])
		}
	}
	return s
}

func (m Model) viewNerd(h int) string {
	return m.viewRows([]string{sMuted.Render(m.keysHint("nerd.keys"))}, m.nerdRows(), m.w, h)
}

func statName(t game.Trait) string {
	return lang.T([...]string{"stat.str", "stat.dex", "stat.int", "stat.con"}[t])
}
