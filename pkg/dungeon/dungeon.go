package dungeon

import (
	"spooknloot/pkg/sim"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const tileSize = sim.TileSize

var (
	layout            *sim.DungeonLayout
	dungeonTexture    rl.Texture2D
	dungeonAddTexture rl.Texture2D
	torchFrontTexture rl.Texture2D
	tileSrc           rl.Rectangle
	tileDest          rl.Rectangle
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

// SetLayout switches to the dungeon level the host generated (the layout
// is rebuilt from the same seed on every machine).
func SetLayout(l *sim.DungeonLayout) {
	layout = l
	exitVisible = false
	colliders = colliders[:0]
	for _, w := range l.Walls {
		colliders = append(colliders, rl.NewRectangle(w.X, w.Y, w.W, w.H))
	}
}

func Draw() {
	if !initialized || layout == nil {
		return
	}
	tex := dungeonTexture
	texColumns := tex.Width / int32(tileSize)

	for y, row := range layout.Tiles {
		for x, t := range row {
			if t < 0 {
				continue
			}
			tileDest.X = float32(x * tileSize)
			tileDest.Y = float32(y * tileSize)

			if t != 0 {
				// Walls and the exit are drawn on top of a floor tile.
				rl.DrawTexturePro(tex, rl.NewRectangle(0, 0, tileSize, tileSize), tileDest, rl.NewVector2(0, 0), 0, rl.White)
			}

			if t == sim.TileExit && !exitVisible {
				continue
			}

			tileSrc.X = float32(tileSize) * float32((t)%int(texColumns))
			tileSrc.Y = float32(tileSize) * float32((t)/int(texColumns))
			rl.DrawTexturePro(tex, tileSrc, tileDest, rl.NewVector2(0, 0), 0, rl.White)
		}
	}

	drawFloorDecals()
	drawWallTorches()
}

func GetColliders() []rl.Rectangle {
	return colliders
}

func IsPlayerAtExit(playerHitbox rl.Rectangle) bool {
	if !exitVisible || layout == nil {
		return false
	}
	e := layout.Exit
	return playerHitbox.X < e.X+e.W &&
		playerHitbox.X+playerHitbox.Width > e.X &&
		playerHitbox.Y < e.Y+e.H &&
		playerHitbox.Y+playerHitbox.Height > e.Y
}

func SetExitVisible(visible bool) {
	exitVisible = visible
}

func drawFloorDecals() {
	tex := dungeonAddTexture
	if tex.ID == 0 {
		return
	}
	cols := tex.Width / int32(tileSize)
	for _, d := range layout.Decals {
		tileDest.X = float32(d.X * tileSize)
		tileDest.Y = float32(d.Y * tileSize)
		sx := float32(tileSize) * float32(d.Sprite%int(cols))
		sy := float32(tileSize) * float32(d.Sprite/int(cols))
		rl.DrawTexturePro(tex, rl.NewRectangle(sx, sy, tileSize, tileSize), tileDest, rl.NewVector2(0, 0), 0, rl.White)
	}
}
