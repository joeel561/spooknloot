package dungeon

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

const torchFrames = 4

func drawWallTorches() {
	tex := torchFrontTexture
	if tex.ID == 0 {
		return
	}
	cols := tex.Width / int32(tileSize)
	// 8 frames per second
	frame := int(rl.GetTime()*8.0) % torchFrames
	fx := float32(tileSize) * float32(frame%int(cols))
	fy := float32(tileSize) * float32(frame/int(cols))
	for _, t := range layout.Torches {
		tileDest.X = float32(t.X * tileSize)
		tileDest.Y = float32(t.Y * tileSize)
		rl.DrawTexturePro(tex, rl.NewRectangle(fx, fy, tileSize, tileSize), tileDest, rl.NewVector2(0, 0), 0, rl.White)
	}
}
