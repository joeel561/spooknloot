package dungeon

import (
	"math"
	"os"

	"spooknloot/pkg/sim"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	potionTexture    rl.Texture2D
	drinkSound       rl.Sound
	drinkSoundLoaded bool
)

func initPotion() {
	if potionTexture.ID != 0 {
		return
	}
	potionTexture = rl.LoadTexture("assets/dungeon/red_portion.png")
	rl.SetTextureFilter(potionTexture, rl.FilterPoint)

	if _, err := os.Stat("assets/audio/drink.mp3"); err == nil {
		drinkSound = rl.LoadSound("assets/audio/drink.mp3")
		rl.SetSoundVolume(drinkSound, 0.7)
		drinkSoundLoaded = true
	}
}

func unloadPotion() {
	if potionTexture.ID != 0 {
		rl.UnloadTexture(potionTexture)
		potionTexture = rl.Texture2D{}
	}
	if drinkSoundLoaded {
		rl.UnloadSound(drinkSound)
		drinkSoundLoaded = false
	}
}

// DrawPotions draws the potions the host placed in the current area.
func DrawPotions(potions []sim.Vec2) {
	if potionTexture.ID == 0 || len(potions) == 0 {
		return
	}
	cols := int32(potionTexture.Width) / int32(tileSize)
	if cols <= 0 {
		return
	}

	frame := int(math.Mod(rl.GetTime()*8.0, 4))
	sx := float32(tileSize) * float32((frame)%int(cols))
	sy := float32(tileSize) * float32((frame)/int(cols))
	src := rl.NewRectangle(sx, sy, tileSize, tileSize)

	for _, p := range potions {
		dst := rl.NewRectangle(p.X, p.Y, tileSize, tileSize)
		rl.DrawTexturePro(potionTexture, src, dst, rl.NewVector2(0, 0), 0, rl.White)
	}
}

func PlayDrinkSound() {
	if drinkSoundLoaded {
		rl.PlaySound(drinkSound)
	}
}
