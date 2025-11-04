package dungeon

import (
	"math"
	"os"

	"spooknloot/pkg/player"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	potionTexture    rl.Texture2D
	potionActive     bool
	potionRect       rl.Rectangle
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
	resetPotion()
}

func resetPotion() {
	potionActive = false
	potionRect = rl.NewRectangle(0, 0, 0, 0)
}

func SpawnPotion() {
	positions := GetRandomFloorPositions(1)
	if len(positions) == 0 {
		resetPotion()
		return
	}
	pos := positions[0]
	potionRect = rl.NewRectangle(pos.X, pos.Y, tileSize, tileSize)
	potionActive = true
}

func drawPotion() {
	if !potionActive || potionTexture.ID == 0 {
		return
	}
	cols := int32(potionTexture.Width) / int32(tileSize)
	if cols <= 0 {
		return
	}

	frame := int(math.Mod(rl.GetTime()*8.0, 4))
	sx := float32(tileSize) * float32((frame)%int(cols))
	sy := float32(tileSize) * float32((frame)/int(cols))
	rl.DrawTexturePro(potionTexture, rl.NewRectangle(sx, sy, tileSize, tileSize), potionRect, rl.NewVector2(0, 0), 0, rl.White)
}

func UpdatePotionPickup(playerHitbox rl.Rectangle) {
	if !potionActive {
		return
	}
	if playerHitbox.X < potionRect.X+potionRect.Width &&
		playerHitbox.X+playerHitbox.Width > potionRect.X &&
		playerHitbox.Y < potionRect.Y+potionRect.Height &&
		playerHitbox.Y+playerHitbox.Height > potionRect.Y {

		maxH := player.GetMaxHealth()
		curH := player.GetCurrentHealth()
		heal := 0.6 * maxH
		missing := maxH - curH
		if missing < 0 {
			missing = 0
		}
		if heal > missing {
			heal = missing
		}
		if heal > 0 {
			player.TakeDamage(-heal)
		}
		if drinkSoundLoaded {
			rl.PlaySound(drinkSound)
		}
		potionActive = false
	}
}
