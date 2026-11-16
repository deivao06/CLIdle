package game

// GenerateCave grows a cave with a cellular automaton and keeps its largest open area.
// Rooms are the 3x3 grid zones that hold enough floor, so spawns, loot and stairs work as in a dungeon.
func GenerateCave(rng *RNG, depth int) *Level {
	for {
		if l := carveCave(rng, depth); l != nil {
			placeStairs(l, rng)
			return l
		}
	}
}

func carveCave(rng *RNG, depth int) *Level {
	l := &Level{Depth: depth, Tiles: make([]byte, MapW*MapH), Seen: make([]byte, MapW*MapH)}
	border := func(x, y int) bool { return x == 0 || y == 0 || x == MapW-1 || y == MapH-1 }
	for y := range MapH {
		for x := range MapW {
			if !border(x, y) && !rng.Chance(CaveRockPct) {
				l.Set(Pos{x, y}, Floor)
			}
		}
	}
	next := make([]byte, len(l.Tiles))
	for range CavePasses {
		for y := range MapH {
			for x := range MapW {
				rock := 0
				for _, d := range Dirs8 {
					if l.At(Pos{x + d.X, y + d.Y}) == Rock {
						rock++
					}
				}
				if l.At(Pos{x, y}) == Rock {
					rock++
				}
				t := Floor
				if border(x, y) || rock >= 5 {
					t = Rock
				}
				next[y*MapW+x] = byte(t)
			}
		}
		copy(l.Tiles, next)
	}
	if !l.keepLargestArea(CaveMinFloorPct * (MapW - 2) * (MapH - 2) / 100) {
		return nil
	}
	cw, ch := MapW/gridCols, MapH/gridRows
	for gy := range gridRows {
		for gx := range gridCols {
			z := Room{gx * cw, gy * ch, cw, ch}
			n := 0
			for y := z.Y + 1; y <= z.Y+z.H-2; y++ {
				for x := z.X + 1; x <= z.X+z.W-2; x++ {
					if l.At(Pos{x, y}) == Floor {
						n++
					}
				}
			}
			if n >= CaveZoneMinFloor {
				l.Rooms = append(l.Rooms, z)
			}
		}
	}
	if len(l.Rooms) < 2 {
		return nil
	}
	for y := range MapH {
		for x := range MapW {
			if p := (Pos{x, y}); l.At(p) == Rock && l.touchesFloor(p) {
				l.Set(p, CaveWall)
			}
		}
	}
	return l
}

// keepLargestArea fills every open area but the largest with rock; false when that area is smaller than minSize.
func (l *Level) keepLargestArea(minSize int) bool {
	area := make([]int, len(l.Tiles))
	var best, bestSize int
	var stack []Pos
	for i := range l.Tiles {
		if Tile(l.Tiles[i]) != Floor || area[i] != 0 {
			continue
		}
		id, size := i+1, 0
		area[i] = id
		stack = append(stack[:0], Pos{i % MapW, i / MapW})
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++
			for _, d := range Dirs8 {
				if n := p.Add(d); l.At(n) == Floor && area[n.Y*MapW+n.X] == 0 {
					area[n.Y*MapW+n.X] = id
					stack = append(stack, n)
				}
			}
		}
		if size > bestSize {
			best, bestSize = id, size
		}
	}
	for i, a := range area {
		if a != best {
			l.Tiles[i] = byte(Rock)
		}
	}
	return bestSize >= minSize
}

func (l *Level) touchesFloor(p Pos) bool {
	for _, d := range Dirs8 {
		if l.At(p.Add(d)) == Floor {
			return true
		}
	}
	return false
}
