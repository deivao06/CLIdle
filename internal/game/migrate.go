package game

// migrate brings an older save up to saveVersion, one version at a time.
func migrate(f *saveFile) {
	if f.Version < 2 {
		migrateV2(f.Game)
	}
	f.Version = saveVersion
}

// migrateV2 maps the v1 training order (Vigor, Força, Resistência, Agilidade) onto the attributes.
func migrateV2(g *Game) {
	old := g.Hero.Training
	g.Hero.Training = [NumTraits]int{Str: old[1], Dex: old[3], Con: old[0] + old[2]}
}
