package dungeon

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Wall torches ---

type wallTorch struct {
	x          int
	y          int
	frame      int // 0-based frame index in spritesheet row
	frameCount int // total frames available in texture
}

var wallTorches []wallTorch

func generateRoomWallTorches(rooms []Room) {
	wallTorches = wallTorches[:0]
	if len(rooms) == 0 {
		return
	}

	// Helper to check floor
	isFloor := func(x, y int) bool {
		if x < 0 || y < 0 || x >= mapW || y >= mapH {
			return false
		}
		return tiles[y][x] == 0
	}

	for _, r := range rooms {
		candidates := make([]wallTorch, 0, r.W)

		// Top walls in this room: tile index 1 and floor directly below → front torch (frame 0)
		for x := r.X; x < r.X+r.W; x++ {
			y := r.Y - 1
			if y >= 0 && tiles[y][x] == 1 && isFloor(x, y+1) {
				candidates = append(candidates, wallTorch{x: x, y: y, frame: 0, frameCount: 4})
			}
		}

		if len(candidates) == 0 {
			continue
		}

		rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
		maxPerRoom := 3
		if maxPerRoom > len(candidates) {
			maxPerRoom = len(candidates)
		}
		if maxPerRoom == 0 {
			continue
		}
		// Place only a few torches per room (1..maxPerRoom)
		count := rand.Intn(maxPerRoom) + 1
		wallTorches = append(wallTorches, candidates[:count]...)
	}
}

func drawWallTorches() {
	if len(wallTorches) == 0 {
		return
	}
	for _, t := range wallTorches {
		tileDest.X = float32(t.x * tileSize)
		tileDest.Y = float32(t.y * tileSize)
		tex := torchFrontTexture
		if tex.ID == 0 {
			continue
		}
		cols := tex.Width / int32(tileSize)
		// Animate frames based on time; default to frame 0 if no animation desired
		currentFrame := 0
		if t.frameCount > 1 {
			// 8 frames per second; adjust as desired
			currentFrame = int(rl.GetTime()*8.0) % t.frameCount
		}
		fx := float32(tileSize) * float32((currentFrame)%int(cols))
		fy := float32(tileSize) * float32((currentFrame)/int(cols))
		rl.DrawTexturePro(tex, rl.NewRectangle(fx, fy, tileSize, tileSize), tileDest, rl.NewVector2(0, 0), 0, rl.White)
	}
}
