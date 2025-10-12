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
	doorsSprite = rl.LoadTexture("assets/world/door.png")
	HouseDoorSrc = rl.NewRectangle(0, 0, 32, 32)
	HouseDoorDest = rl.NewRectangle(504, 175, 32, 32)
}

func OpenHouseDoor() {
	frameCountDoor++

	if frameCountDoor >= DoorsMaxFrame {
		frameCountDoor = 0
		frameDoor++
	}

	frameDoor = frameDoor % DoorsMaxFrame

	HouseDoorSrc.X = float32(frameDoor * 32)
}

func DrawDoors() {
	rl.DrawTexturePro(doorsSprite, HouseDoorSrc, HouseDoorDest, rl.NewVector2(0, 0), 0, rl.White)
}

func UnloadDoorsTextures() {
	rl.UnloadTexture(doorsSprite)
}
