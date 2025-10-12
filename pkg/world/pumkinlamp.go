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
	cauldronSprite        rl.Texture2D
	cauldronSrc           rl.Rectangle
	cauldronDest          rl.Rectangle
	cauldronMaxFrame      int = 20
)

func InitPumpkinLamps() {
	pumpkinLampsSprite = rl.LoadTexture("assets/world/pumpkinlamp.png")
	cauldronSprite = rl.LoadTexture("assets/world/Cauldron.png")
	PumpkinLampDest = rl.NewRectangle(500, 350, 16, 16)

	cauldronDest = rl.NewRectangle(800, 255, 16, 16)
	cauldronSrc = rl.NewRectangle(0, 0, 16, 16)

	pumpkinLampSrc = rl.NewRectangle(0, 0, 16, 16)

	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 608, Y: 447})
	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 544, Y: 399})
	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 176, Y: 481})
	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 848, Y: 272})
	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 784, Y: 303})
	Pumpkins = append(Pumpkins, Tile{Id: "pumpkinlamp", X: 864, Y: 528})
}

func LightPumpkinLamps() {
	frameCountPumpkinLamp++

	if frameCountPumpkinLamp >= pumpkinMaxFrame {
		frameCountPumpkinLamp = 0
		framePumpkinLamp++
	}

	framePumpkinLamp = framePumpkinLamp % pumpkinMaxFrame

	pumpkinLampSrc.X = float32(framePumpkinLamp * 16)

	cauldronSrc.X = float32(framePumpkinLamp * 16)
}

func DrawPumpkinLamp() {
	for i := range Pumpkins {
		PumpkinLampDest.X = float32(Pumpkins[i].X)
		PumpkinLampDest.Y = float32(Pumpkins[i].Y)
		rl.DrawTexturePro(pumpkinLampsSprite, pumpkinLampSrc, PumpkinLampDest, rl.NewVector2(0, 0), 0, rl.White)
	}
}

func DrawCauldron() {
	rl.DrawTexturePro(cauldronSprite, cauldronSrc, cauldronDest, rl.NewVector2(0, 0), 0, rl.White)
}

func UnloadPumpkinLamps() {
	rl.UnloadTexture(pumpkinLampsSprite)
	rl.UnloadTexture(cauldronSprite)
}
