package world

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	frameCountPumpkinLamp int
	framePumpkinLamp      int = 0
	pumpkinLampsSprite    rl.Texture2D
	pumpkinLampSrc        rl.Rectangle
	PumpkinLampDest       rl.Rectangle
	pumpkinMaxFrame       int = 20
	Pumpkins              []Tile
)

func InitPumpkinLamps() {
	pumpkinLampsSprite = rl.LoadTexture("assets/world/pumpkinlamp.png")
	PumpkinLampDest = rl.NewRectangle(500, 350, 16, 16)

	pumpkinLampSrc = rl.NewRectangle(0, 0, 16, 16)

	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 550, Y: 350})
	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 477, Y: 360})

}

func LightPumpkinLamps() {
	frameCountPumpkinLamp++

	if frameCountPumpkinLamp >= pumpkinMaxFrame {
		frameCountPumpkinLamp = 0
		framePumpkinLamp++
	}

	framePumpkinLamp = framePumpkinLamp % pumpkinMaxFrame

	pumpkinLampSrc.X = float32(framePumpkinLamp * 16)

}

func DrawPumpkinLamp() {
	for i := range Pumpkins {
		PumpkinLampDest.X = float32(Pumpkins[i].X)
		PumpkinLampDest.Y = float32(Pumpkins[i].Y)
		rl.DrawTexturePro(pumpkinLampsSprite, pumpkinLampSrc, PumpkinLampDest, rl.NewVector2(0, 0), 0, rl.White)
	}
}

func UnloadPumpkinLamps() {
	rl.UnloadTexture(pumpkinLampsSprite)
}
