package dungeon

import (
	"math/rand"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	tileSize = 16
	mapW     = 40
	mapH     = 25
	maxRooms = 8
	minSize  = 4
	maxSize  = 8
)

type Room struct {
	X, Y, W, H int
}

func (r Room) Center() (int, int) {
	return r.X + r.W/2, r.Y + r.H/2
}

func (r Room) Intersects(o Room) bool {
	return r.X <= o.X+o.W && r.X+r.W >= o.X &&
		r.Y <= o.Y+o.H && r.Y+r.H >= o.Y
}

var (
	tiles             [][]int
	dungeonTexture    rl.Texture2D
	dungeonAddTexture rl.Texture2D
	torchFrontTexture rl.Texture2D
	tileSrc           rl.Rectangle
	tileDest          rl.Rectangle
	spawnPx           rl.Vector2
	exitPx            rl.Rectangle
	colliders         []rl.Rectangle
	initialized       bool
	exitVisible       bool
)

// Tile indices mapping for spritesheet:
// 0 floor, 1 top, 2 bottom, 3 left, 4 right, 5 TL, 6 TR, 7 BL, 8 BR, 9 exit.

func Init() {
	if initialized {
		return
	}
	dungeonTexture = rl.LoadTexture("assets/dungeon/spritesheet.png")
	rl.SetTextureFilter(dungeonTexture, rl.FilterPoint)
	dungeonAddTexture = rl.LoadTexture("assets/dungeon/dungeon_add.png")
	rl.SetTextureFilter(dungeonAddTexture, rl.FilterPoint)
	torchFrontTexture = rl.LoadTexture("assets/dungeon/torch_front.png")
	rl.SetTextureFilter(torchFrontTexture, rl.FilterPoint)
	initPotion()
	tileSrc = rl.NewRectangle(0, 0, tileSize, tileSize)
	tileDest = rl.NewRectangle(0, 0, tileSize, tileSize)
	initialized = true
}

func Unload() {
	if initialized {
		rl.UnloadTexture(dungeonTexture)
		rl.UnloadTexture(dungeonAddTexture)
		rl.UnloadTexture(torchFrontTexture)
		unloadPotion()
		initialized = false
	}
}

func Generate() {
	rand.Seed(time.Now().UnixNano())

	// Start with solid walls (1)
	tiles = make([][]int, mapH)
	for y := range tiles {
		tiles[y] = make([]int, mapW)
		for x := range tiles[y] {
			tiles[y][x] = 1
		}
	}

	rooms := []Room{}
	for i := 0; i < maxRooms; i++ {
		w := rand.Intn(maxSize-minSize+1) + minSize
		h := rand.Intn(maxSize-minSize+1) + minSize
		x := rand.Intn(mapW-w-2) + 1
		y := rand.Intn(mapH-h-2) + 1

		newRoom := Room{X: x, Y: y, W: w, H: h}

		overlap := false
		for _, other := range rooms {
			if newRoom.Intersects(other) {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}

		// Carve room to floor (0)
		for ry := y; ry < y+h; ry++ {
			for rx := x; rx < x+w; rx++ {
				tiles[ry][rx] = 0
			}
		}

		// Connect rooms via corridors
		if len(rooms) > 0 {
			prev := rooms[len(rooms)-1]
			cx1, cy1 := prev.Center()
			cx2, cy2 := newRoom.Center()
			if rand.Intn(2) == 0 {
				carveCorridor(cx1, cy1, cx2, cy1)
				carveCorridor(cx2, cy1, cx2, cy2)
			} else {
				carveCorridor(cx1, cy1, cx1, cy2)
				carveCorridor(cx1, cy2, cx2, cy2)
			}
		}

		rooms = append(rooms, newRoom)
	}

	if len(rooms) > 0 {
		sx, sy := rooms[0].Center()
		spawnPx = rl.NewVector2(float32(sx*tileSize), float32(sy*tileSize))

		ex, ey := rooms[len(rooms)-1].Center()
		exitPx = rl.NewRectangle(float32(ex*tileSize), float32(ey*tileSize), tileSize, tileSize)
		tiles[ey][ex] = 9
	} else {
		spawnPx = rl.NewVector2(float32(2*tileSize), float32(2*tileSize))
		exitPx = rl.NewRectangle(float32((mapW-3)*tileSize), float32((mapH-3)*tileSize), tileSize, tileSize)
	}

	exitVisible = false

	classifyCorners()

	colliders = colliders[:0]
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			t := tiles[y][x]
			if t > 0 && t != 9 { // 1..8 are walls
				r := rl.NewRectangle(float32(x*tileSize), float32(y*tileSize), tileSize, tileSize)
				colliders = append(colliders, r)
			}
		}
	}

	generateRoomFloorOverlays(rooms)
	generateRoomWallTorches(rooms)
	resetPotion()
	spawnPotion()
}

func carveCorridor(x1, y1, x2, y2 int) {
	if x1 == x2 {
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for y := y1; y <= y2; y++ {
			tiles[y][x1] = 0
		}
	} else if y1 == y2 {
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		for x := x1; x <= x2; x++ {
			tiles[y1][x] = 0
		}
	}
}

func classifyCorners() {
	original := make([][]int, mapH)
	for y := 0; y < mapH; y++ {
		original[y] = make([]int, mapW)
		copy(original[y], tiles[y])
	}

	isFloor := func(x, y int) bool {
		if x < 0 || y < 0 || x >= mapW || y >= mapH {
			return false
		}
		return original[y][x] == 0
	}

	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {

			if original[y][x] == 0 || original[y][x] == 9 {
				continue
			}

			up := isFloor(x, y-1)
			down := isFloor(x, y+1)
			left := isFloor(x-1, y)
			right := isFloor(x+1, y)
			diagUL := isFloor(x-1, y-1)
			diagUR := isFloor(x+1, y-1)
			diagDL := isFloor(x-1, y+1)
			diagDR := isFloor(x+1, y+1)

			nearFloor := up || down || left || right || diagUL || diagUR || diagDL || diagDR
			if !nearFloor {

				tiles[y][x] = -1
				continue
			}

			if diagDR && !right && !down {
				// TL corner: floor at down-right
				tiles[y][x] = 5
				continue
			}
			if diagDL && !left && !down {
				// TR corner: floor at down-left
				tiles[y][x] = 6
				continue
			}
			if diagUR && !right && !up {
				// BL corner: floor at up-right
				tiles[y][x] = 7
				continue
			}
			if diagUL && !left && !up {
				// BR corner: floor at up-left
				tiles[y][x] = 8
				continue
			}

			if y == 0 && down {
				tiles[y][x] = 1
				continue
			}

			// Edges (1..4)
			if down && !up {
				// Top edge
				tiles[y][x] = 1
				continue
			}
			if up && !down {
				// Bottom edge
				tiles[y][x] = 2
				continue
			}
			if right && !left {
				// Left edge
				tiles[y][x] = 3
				continue
			}
			if left && !right {
				// Right edge
				tiles[y][x] = 4
				continue
			}

			// Default to top edge when multiple floors around
			tiles[y][x] = 1
		}
	}
}

func Draw() {
	if !initialized || len(tiles) == 0 {
		return
	}
	tex := dungeonTexture
	texColumns := tex.Width / int32(tileSize)

	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			t := tiles[y][x]
			if t < 0 {
				continue
			}
			tileDest.X = float32(x * tileSize)
			tileDest.Y = float32(y * tileSize)

			if t != 0 {
				floorSrcX := float32(tileSize) * float32((0)%int(texColumns))
				floorSrcY := float32(tileSize) * float32((0)/int(texColumns))
				rl.DrawTexturePro(tex, rl.NewRectangle(floorSrcX, floorSrcY, tileSize, tileSize), tileDest, rl.NewVector2(0, 0), 0, rl.White)
			}

			if t == 9 && !exitVisible {
				continue
			}

			tileSrc.X = float32(tileSize) * float32((t)%int(texColumns))
			tileSrc.Y = float32(tileSize) * float32((t)/int(texColumns))
			rl.DrawTexturePro(tex, tileSrc, tileDest, rl.NewVector2(0, 0), 0, rl.White)
		}
	}

	drawFloorOverlays()

	drawPotion()

	drawWallTorches()
}

func GetColliders() []rl.Rectangle {
	return colliders
}

func GetSpawnPosition() rl.Vector2 {
	return spawnPx
}

func IsPlayerAtExit(playerHitbox rl.Rectangle) bool {
	if !exitVisible {
		return false
	}
	return playerHitbox.X < exitPx.X+exitPx.Width &&
		playerHitbox.X+playerHitbox.Width > exitPx.X &&
		playerHitbox.Y < exitPx.Y+exitPx.Height &&
		playerHitbox.Y+playerHitbox.Height > exitPx.Y
}

func GetRandomFloorPositions(n int) []rl.Vector2 {
	if len(tiles) == 0 || n <= 0 {
		return nil
	}

	floorTiles := make([]rl.Vector2, 0, mapW*mapH)
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			if tiles[y][x] == 0 { // floor
				px := float32(x * tileSize)
				py := float32(y * tileSize)
				floorTiles = append(floorTiles, rl.NewVector2(px, py))
			}
		}
	}

	if len(floorTiles) == 0 {
		return nil
	}

	rand.Shuffle(len(floorTiles), func(i, j int) { floorTiles[i], floorTiles[j] = floorTiles[j], floorTiles[i] })
	if n > len(floorTiles) {
		n = len(floorTiles)
	}
	return append([]rl.Vector2(nil), floorTiles[:n]...)
}

func ShowExit() {
	exitVisible = true
}

func HideExit() {
	exitVisible = false
}

func IsExitVisible() bool {
	return exitVisible
}

type floorOverlay struct {
	x    int
	y    int
	tile int // 0..3 index into dungeon_add sprites
}

var floorOverlays []floorOverlay

func generateRoomFloorOverlays(rooms []Room) {
	floorOverlays = floorOverlays[:0]
	if len(rooms) == 0 {
		return
	}

	spawnTileX := int(spawnPx.X) / tileSize
	spawnTileY := int(spawnPx.Y) / tileSize

	for _, r := range rooms {
		// Collect candidate floor tiles inside this room
		candidates := make([][2]int, 0, r.W*r.H)
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if tiles[y][x] == 0 { // floor only
					if x == spawnTileX && y == spawnTileY {
						continue // avoid spawn tile
					}
					candidates = append(candidates, [2]int{x, y})
				}
			}
		}
		if len(candidates) == 0 {
			continue
		}

		rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
		count := 7
		if count > len(candidates) {
			count = len(candidates)
		}

		// Prepare shuffled tile order (0..3), reshuffle when exhausted to keep variety
		base := []int{0, 1, 2, 3}
		rand.Shuffle(len(base), func(i, j int) { base[i], base[j] = base[j], base[i] })
		idx := 0

		for i := 0; i < count; i++ {
			pos := candidates[i]

			// Pick tile ensuring variety across placements
			tile := base[idx]
			idx++
			if idx == len(base) {
				prev := base[len(base)-1]
				rand.Shuffle(len(base), func(i, j int) { base[i], base[j] = base[j], base[i] })
				if len(base) > 1 && base[0] == prev {
					base[0], base[1] = base[1], base[0]
				}
				idx = 0
			}

			floorOverlays = append(floorOverlays, floorOverlay{x: pos[0], y: pos[1], tile: tile})
		}
	}
}

func drawFloorOverlays() {
	if len(floorOverlays) == 0 {
		return
	}
	tex := dungeonAddTexture
	if tex.ID == 0 {
		return
	}
	cols := tex.Width / int32(tileSize)
	for _, ov := range floorOverlays {
		tileDest.X = float32(ov.x * tileSize)
		tileDest.Y = float32(ov.y * tileSize)
		sx := float32(tileSize) * float32((ov.tile)%int(cols))
		sy := float32(tileSize) * float32((ov.tile)/int(cols))
		rl.DrawTexturePro(tex, rl.NewRectangle(sx, sy, tileSize, tileSize), tileDest, rl.NewVector2(0, 0), 0, rl.White)
	}
}
