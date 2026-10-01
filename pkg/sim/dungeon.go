package sim

import "math/rand"

const (
	DungeonW     = 40
	DungeonH     = 25
	maxRooms     = 8
	minRoomSize  = 4
	maxRoomSize  = 8
	TileFloor    = 0
	TileExit     = 9
	TileNone     = -1
	maxTorchRoom = 3
)

// Tile values: 0 floor, 1 top, 2 bottom, 3 left, 4 right, 5 TL, 6 TR,
// 7 BL, 8 BR (walls), 9 exit, -1 nothing (solid rock, not drawn).

type room struct{ X, Y, W, H int }

func (r room) center() (int, int) { return r.X + r.W/2, r.Y + r.H/2 }

func (r room) intersects(o room) bool {
	return r.X <= o.X+o.W && r.X+r.W >= o.X && r.Y <= o.Y+o.H && r.Y+r.H >= o.Y
}

// FloorDecal is a decoration sprite on a floor tile.
type FloorDecal struct{ X, Y, Sprite int }

// Torch is a wall torch on a tile.
type Torch struct{ X, Y int }

// DungeonLayout is one generated dungeon level. The same seed always
// produces the same layout, so only the seed is sent over the network.
type DungeonLayout struct {
	Tiles     [][]int
	Spawn     Vec2
	Exit      Rect
	Walls     []Rect
	Decals    []FloorDecal
	Torches   []Torch
	rng       *rand.Rand
	floorList []Vec2
}

func GenerateDungeon(seed int64) *DungeonLayout {
	rng := rand.New(rand.NewSource(seed))
	l := &DungeonLayout{rng: rng}
	l.Tiles = make([][]int, DungeonH)
	for y := range l.Tiles {
		l.Tiles[y] = make([]int, DungeonW)
		for x := range l.Tiles[y] {
			l.Tiles[y][x] = 1
		}
	}

	var rooms []room
	for i := 0; i < maxRooms; i++ {
		w := rng.Intn(maxRoomSize-minRoomSize+1) + minRoomSize
		h := rng.Intn(maxRoomSize-minRoomSize+1) + minRoomSize
		r := room{X: rng.Intn(DungeonW-w-2) + 1, Y: rng.Intn(DungeonH-h-2) + 1, W: w, H: h}

		overlap := false
		for _, o := range rooms {
			if r.intersects(o) {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				l.Tiles[y][x] = TileFloor
			}
		}
		if len(rooms) > 0 {
			x1, y1 := rooms[len(rooms)-1].center()
			x2, y2 := r.center()
			if rng.Intn(2) == 0 {
				l.carveCorridor(x1, y1, x2, y1)
				l.carveCorridor(x2, y1, x2, y2)
			} else {
				l.carveCorridor(x1, y1, x1, y2)
				l.carveCorridor(x1, y2, x2, y2)
			}
		}
		rooms = append(rooms, r)
	}

	// The first room always fits, so there is at least one.
	sx, sy := rooms[0].center()
	l.Spawn = Vec2{float32(sx * TileSize), float32(sy * TileSize)}
	ex, ey := rooms[len(rooms)-1].center()
	l.Exit = Rect{float32(ex * TileSize), float32(ey * TileSize), TileSize, TileSize}
	l.Tiles[ey][ex] = TileExit

	l.classifyWalls()
	for y := 0; y < DungeonH; y++ {
		for x := 0; x < DungeonW; x++ {
			if t := l.Tiles[y][x]; t > 0 && t != TileExit {
				l.Walls = append(l.Walls, Rect{float32(x * TileSize), float32(y * TileSize), TileSize, TileSize})
			}
		}
	}
	for y := 0; y < DungeonH; y++ {
		for x := 0; x < DungeonW; x++ {
			if l.Tiles[y][x] == TileFloor {
				l.floorList = append(l.floorList, Vec2{float32(x * TileSize), float32(y * TileSize)})
			}
		}
	}
	l.placeDecals(rooms)
	l.placeTorches(rooms)
	return l
}

// RandomFloorPositions picks n distinct floor tiles. It keeps using the
// layout's own random source, so the host gets reproducible results too.
func (l *DungeonLayout) RandomFloorPositions(n int) []Vec2 {
	floor := append([]Vec2(nil), l.floorList...)
	l.rng.Shuffle(len(floor), func(i, j int) { floor[i], floor[j] = floor[j], floor[i] })
	return floor[:min(n, len(floor))]
}

func (l *DungeonLayout) carveCorridor(x1, y1, x2, y2 int) {
	set := func(x, y int) {
		if x >= 0 && y >= 0 && x < DungeonW && y < DungeonH {
			l.Tiles[y][x] = TileFloor
		}
	}
	// Corridors are two tiles wide.
	if x1 == x2 {
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for y := y1; y <= y2; y++ {
			set(x1, y)
			set(x1+1, y)
		}
	} else if y1 == y2 {
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		for x := x1; x <= x2; x++ {
			set(x, y1)
			set(x, y1+1)
		}
	}
}

// classifyWalls turns solid tiles next to floor into edge and corner walls.
func (l *DungeonLayout) classifyWalls() {
	orig := make([][]int, DungeonH)
	for y := range orig {
		orig[y] = append([]int(nil), l.Tiles[y]...)
	}
	floor := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < DungeonW && y < DungeonH && orig[y][x] == TileFloor
	}
	for y := 0; y < DungeonH; y++ {
		for x := 0; x < DungeonW; x++ {
			if orig[y][x] == TileFloor || orig[y][x] == TileExit {
				continue
			}
			up, down, left, right := floor(x, y-1), floor(x, y+1), floor(x-1, y), floor(x+1, y)
			ul, ur, dl, dr := floor(x-1, y-1), floor(x+1, y-1), floor(x-1, y+1), floor(x+1, y+1)
			t := &l.Tiles[y][x]
			switch {
			case !(up || down || left || right || ul || ur || dl || dr):
				*t = TileNone
			case dr && !right && !down:
				*t = 5
			case dl && !left && !down:
				*t = 6
			case ur && !right && !up:
				*t = 7
			case ul && !left && !up:
				*t = 8
			case y == 0 && down:
				*t = 1
			case down && !up:
				*t = 1
			case up && !down:
				*t = 2
			case right && !left:
				*t = 3
			case left && !right:
				*t = 4
			default:
				*t = 1
			}
		}
	}
}

func (l *DungeonLayout) placeDecals(rooms []room) {
	spawnX, spawnY := int(l.Spawn.X)/TileSize, int(l.Spawn.Y)/TileSize
	for _, r := range rooms {
		var cand [][2]int
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if l.Tiles[y][x] == TileFloor && !(x == spawnX && y == spawnY) {
					cand = append(cand, [2]int{x, y})
				}
			}
		}
		if len(cand) == 0 {
			continue
		}
		l.rng.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })

		// Cycle through the 4 decal sprites in shuffled order for variety.
		order := []int{0, 1, 2, 3}
		l.rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		idx := 0
		for i := 0; i < min(7, len(cand)); i++ {
			l.Decals = append(l.Decals, FloorDecal{X: cand[i][0], Y: cand[i][1], Sprite: order[idx]})
			idx++
			if idx == len(order) {
				prev := order[len(order)-1]
				l.rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
				if order[0] == prev {
					order[0], order[1] = order[1], order[0]
				}
				idx = 0
			}
		}
	}
}

func (l *DungeonLayout) placeTorches(rooms []room) {
	for _, r := range rooms {
		var cand []Torch
		// Top walls directly above the room's floor get front torches.
		for x := r.X; x < r.X+r.W; x++ {
			y := r.Y - 1
			if y >= 0 && l.Tiles[y][x] == 1 && l.Tiles[y+1][x] == TileFloor {
				cand = append(cand, Torch{x, y})
			}
		}
		if len(cand) == 0 {
			continue
		}
		l.rng.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
		count := l.rng.Intn(min(maxTorchRoom, len(cand))) + 1
		l.Torches = append(l.Torches, cand[:count]...)
	}
}
