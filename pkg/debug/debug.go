package debug

import (
	"fmt"
	"spooknloot/pkg/player"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	playerSrc                                     rl.Rectangle
	playerMoving                                  bool
	playerDir                                     int
	playerUp, playerDown, playerLeft, playerRight bool
	playerFrame                                   int
	playerHitBox                                  rl.Rectangle
	playerSpeed                                   float32 = 1.4
	musicPaused                                   bool
)

func rectToString(rec rl.Rectangle) string {
	return fmt.Sprintf("X:%v, Y:%v, H:%v, W:%v", rec.X, rec.Y, rec.Height, rec.Width)
}

func vec2ToString(vec rl.Vector2) string {
	return fmt.Sprintf("X:%v, Y:%v", vec.X, vec.Y)
}

func DebugText() []string {
	var cam = player.Cam

	return []string{
		fmt.Sprintf("FPS: %v", rl.GetFPS()),
		fmt.Sprintf("Cam Target %v", vec2ToString(cam.Target)),
		fmt.Sprintf("Player Direction: %v   U:%v, D:%v, L:%v, R:%v", playerDir, playerUp, playerDown, playerLeft, playerRight),
		fmt.Sprintf("Player Speed: %v", playerSpeed),
		fmt.Sprintf("Player Frame: %v", playerFrame),
		fmt.Sprintf("Player Moving: %v", playerMoving),
		fmt.Sprintf("Player Src %v", rectToString(playerSrc)),
		fmt.Sprintf("Player Dest %v", rectToString(player.PlayerDest)),
		fmt.Sprintf("Player Hitbox %v", rectToString(player.PlayerHitBox)),
	}
}

func DrawDebug(DebugText []string) {
	textSize := 10
	lineSpace := 15

	offsetX := 10
	offsetY := 10

	for i, line := range DebugText {
		rl.DrawText(line, int32(offsetX), int32(offsetY+lineSpace*i), int32(textSize), rl.Black)
	}
}

func DrawPlayerOutlines() {
	// Draw cetner map cross
	rl.DrawLineEx(rl.NewVector2(0, 0), rl.NewVector2(-20, 0), 1, rl.Gray)
	rl.DrawLineEx(rl.NewVector2(0, 0), rl.NewVector2(20, 0), 1, rl.Red)
	rl.DrawTriangle(rl.NewVector2(16, 2), rl.NewVector2(20, 0), rl.NewVector2(16, -2), rl.Red)
	rl.DrawText("X", int32(22), int32(-5), int32(10), rl.Black)
	rl.DrawLineEx(rl.NewVector2(0, 0), rl.NewVector2(0, -20), 1, rl.Gray)
	rl.DrawLineEx(rl.NewVector2(0, 0), rl.NewVector2(0, 20), 1, rl.Blue)
	rl.DrawTriangle(rl.NewVector2(-2, 16), rl.NewVector2(0, 20), rl.NewVector2(2, 16), rl.Blue)
	rl.DrawText("Y", int32(-2), int32(22), int32(10), rl.Black)

	// Draw collision rectangle
	rl.DrawRectangleLinesEx(player.PlayerHitBox, 1, rl.Green)
	rl.DrawRectangleLinesEx(player.PlayerDest, 1, rl.Purple)
}
