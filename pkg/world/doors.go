package world

import (
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	frameCountDoor  int
	frameDoor       int = 0
	doorsSprite     rl.Texture2D
	HouseDoorSrc    rl.Rectangle
	HouseDoorDest   rl.Rectangle
	DoorsMaxFrame   int = 5
	openSound       rl.Sound
	openSoundLoaded bool
)

func InitDoors() {
	doorsSprite = rl.LoadTexture("assets/world/door.png")
	HouseDoorSrc = rl.NewRectangle(0, 0, 32, 32)
	HouseDoorDest = rl.NewRectangle(504, 175, 32, 32)

	// Load door open sound if available
	if _, err := os.Stat("assets/audio/open.mp3"); err == nil {
		openSound = rl.LoadSound("assets/audio/open.mp3")
		rl.SetSoundVolume(openSound, 0.7)
		openSoundLoaded = true
	}
}

func OpenHouseDoor() {
	prev := frameDoor
	frameCountDoor++

	if frameCountDoor >= DoorsMaxFrame {
		frameCountDoor = 0
		frameDoor++
	}

	frameDoor = frameDoor % DoorsMaxFrame

	HouseDoorSrc.X = float32(frameDoor * 32)

	if openSoundLoaded && prev == 0 && frameDoor == 1 && !rl.IsSoundPlaying(openSound) {
		rl.PlaySound(openSound)
	}
}

func DrawDoors() {
	rl.DrawTexturePro(doorsSprite, HouseDoorSrc, HouseDoorDest, rl.NewVector2(0, 0), 0, rl.White)
}

func UnloadDoorsTextures() {
	rl.UnloadTexture(doorsSprite)
	if openSoundLoaded {
		rl.UnloadSound(openSound)
		openSoundLoaded = false
	}
}

func PlayDoorOpenSound() {
	if openSoundLoaded && !rl.IsSoundPlaying(openSound) {
		rl.PlaySound(openSound)
	}
}
