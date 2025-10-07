package world

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	frameCountLamp int
	frameLamp      int = 0
	lampsSprite    rl.Texture2D
	LampSrcTop     rl.Rectangle
	LampSrcBottom  rl.Rectangle
	LampDestTop    rl.Rectangle
	LampDestBottom rl.Rectangle
	LampsMaxFrame  int = 14
	Lamps          []Tile
)

func InitLamps() {
	lampsSprite = rl.LoadTexture("assets/world/lamppost.png")
	LampSrcTop = rl.NewRectangle(80, 0, 16, 16)
	LampSrcBottom = rl.NewRectangle(80, 16, 16, 16)
	LampDestBottom = rl.NewRectangle(500, 350, 16, 16)
	LampDestTop = rl.NewRectangle(500, 334, 16, 16)

	Lamps = append(Lamps, Tile{Id: "lamp", X: 547, Y: 332})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 477, Y: 332})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 349, Y: 332})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 237, Y: 332})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 659, Y: 332})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 787, Y: 332})

	Lamps = append(Lamps, Tile{Id: "lamp", X: 349, Y: 394})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 237, Y: 394})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 659, Y: 394})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 787, Y: 394})

	Lamps = append(Lamps, Tile{Id: "lamp", X: 547, Y: 426})
	Lamps = append(Lamps, Tile{Id: "lamp", X: 477, Y: 426})
}

func LightLamps() {
	frameCountLamp++

	if frameCountLamp >= LampsMaxFrame {
		frameCountLamp = 0
		frameLamp++
	}

	frameLamp = frameLamp % LampsMaxFrame

	LampSrcTop.X = float32(frameLamp * 16)
	LampSrcBottom.X = float32(frameLamp * 16)

}

func DrawTopLamp() {
	for i := range Lamps {
		LampDestTop.X = float32(Lamps[i].X)
		LampDestTop.Y = float32(Lamps[i].Y - 16)
		rl.DrawTexturePro(lampsSprite, LampSrcTop, LampDestTop, rl.NewVector2(0, 0), 0, rl.White)
	}
}

func DrawBottomLamp() {
	for i := range Lamps {
		LampDestBottom.X = float32(Lamps[i].X)
		LampDestBottom.Y = float32(Lamps[i].Y)
		rl.DrawTexturePro(lampsSprite, LampSrcBottom, LampDestBottom, rl.NewVector2(0, 0), 0, rl.White)
	}
}
