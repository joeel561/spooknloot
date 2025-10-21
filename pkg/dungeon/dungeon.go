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
	tiles          [][]int
	dungeonTexture rl.Texture2D
	tileSrc        rl.Rectangle
	tileDest       rl.Rectangle
	spawnPx        rl.Vector2
	exitPx         rl.Rectangle
	colliders      []rl.Rectangle
	initialized    bool
)

// Tile indices mapping for spritesheet:
// 0 floor, 1 wall, 2 corner TL, 3 corner TR, 4 corner BL, 5 corner BR

func Init() {
	if initialized {
		return
	}
	dungeonTexture = rl.LoadTexture("assets/world/spritesheet.png")
	rl.SetTextureFilter(dungeonTexture, rl.FilterPoint)
	tileSrc = rl.NewRectangle(0, 0, tileSize, tileSize)
	tileDest = rl.NewRectangle(0, 0, tileSize, tileSize)
	initialized = true
}

func Unload() {
	if initialized {
		rl.UnloadTexture(dungeonTexture)
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
		x := rand.Intn(mapW - w - 1)
		y := rand.Intn(mapH - h - 1)

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

	// Determine spawn at first room center, exit at last room center
	if len(rooms) > 0 {
		sx, sy := rooms[0].Center()
		spawnPx = rl.NewVector2(float32(sx*tileSize), float32(sy*tileSize))

		ex, ey := rooms[len(rooms)-1].Center()
		exitPx = rl.NewRectangle(float32(ex*tileSize), float32(ey*tileSize), tileSize, tileSize)
	} else {
		spawnPx = rl.NewVector2(float32(2*tileSize), float32(2*tileSize))
		exitPx = rl.NewRectangle(float32((mapW-3)*tileSize), float32((mapH-3)*tileSize), tileSize, tileSize)
	}

	// Classify wall corners 2..5 for nicer visuals
	classifyCorners()

	// Build colliders for any non-floor tiles (1..5)
	colliders = colliders[:0]
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			t := tiles[y][x]
			if t != 0 {
				r := rl.NewRectangle(float32(x*tileSize), float32(y*tileSize), tileSize, tileSize)
				colliders = append(colliders, r)
			}
		}
	}
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
	// Copy of tiles to read neighbors without interference
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

	isWall := func(x, y int) bool {
		if x < 0 || y < 0 || x >= mapW || y >= mapH {
			return true
		}
		return original[y][x] != 0
	}

	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			if original[y][x] == 0 {
				continue
			}

			up := isFloor(x, y-1)
			down := isFloor(x, y+1)
			left := isFloor(x-1, y)
			right := isFloor(x+1, y)

			// Corners where two adjacent sides are floor and the other sides are walls
			// TL corner: floor right and floor down
			if right && down && isWall(x-1, y) && isWall(x, y-1) {
				tiles[y][x] = 2
				continue
			}
			// TR corner: floor left and floor down
			if left && down && isWall(x+1, y) && isWall(x, y-1) {
				tiles[y][x] = 3
				continue
			}
			// BL corner: floor right and floor up
			if right && up && isWall(x-1, y) && isWall(x, y+1) {
				tiles[y][x] = 4
				continue
			}
			// BR corner: floor left and floor up
			if left && up && isWall(x+1, y) && isWall(x, y+1) {
				tiles[y][x] = 5
				continue
			}
			// Otherwise keep as generic wall (1)
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
			tileSrc.X = float32(tileSize) * float32((t)%int(texColumns))
			tileSrc.Y = float32(tileSize) * float32((t)/int(texColumns))
			tileDest.X = float32(x * tileSize)
			tileDest.Y = float32(y * tileSize)

			// Draw floor under walls for completeness
			if t != 0 {
				floorSrcX := float32(tileSize) * float32((0)%int(texColumns))
				floorSrcY := float32(tileSize) * float32((0)/int(texColumns))
				rl.DrawTexturePro(tex, rl.NewRectangle(floorSrcX, floorSrcY, tileSize, tileSize), tileDest, rl.NewVector2(0, 0), 0, rl.White)
			}
			rl.DrawTexturePro(tex, tileSrc, tileDest, rl.NewVector2(0, 0), 0, rl.White)
		}
	}
}

func GetColliders() []rl.Rectangle {
	return colliders
}

func GetSpawnPosition() rl.Vector2 {
	return spawnPx
}

func IsPlayerAtExit(playerHitbox rl.Rectangle) bool {
	return playerHitbox.X < exitPx.X+exitPx.Width &&
		playerHitbox.X+playerHitbox.Width > exitPx.X &&
		playerHitbox.Y < exitPx.Y+exitPx.Height &&
		playerHitbox.Y+playerHitbox.Height > exitPx.Y
}
