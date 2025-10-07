package world

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	frameCountDoor int
	frameDoor      int = 0
	doorsSprite    rl.Texture2D
	HouseDoorSrc   rl.Rectangle
	HouseDoorDest  rl.Rectangle
	DoorsMaxFrame  int = 5
)

func InitDoors() {
	doorsSprite = rl.LoadTexture("assets/world/dooranimationsprites.png")
	HouseDoorSrc = rl.NewRectangle(80, 0, 16, 16)
	HouseDoorDest = rl.NewRectangle(512, 190, 16, 16)
}

func OpenHouseDoor() {
	frameCountDoor++

	if frameCountDoor >= DoorsMaxFrame {
		frameCountDoor = 0
		frameDoor++
	}

	HouseDoorSrc.X = 16

	frameDoor = frameDoor % DoorsMaxFrame
}

func DrawDoors() {
	rl.DrawTexturePro(doorsSprite, HouseDoorSrc, HouseDoorDest, rl.NewVector2(0, 0), 0, rl.White)
}

func UnloadDoorsTextures() {
	rl.UnloadTexture(doorsSprite)
}
