package dungeon

import (
	"math"
	"math/rand"
	"os"

	"spooknloot/pkg/player"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	potionTexture    rl.Texture2D
	drinkSound       rl.Sound
	drinkSoundLoaded bool
)

type Potion struct {
	Position rl.Vector2
	Active   bool
	Sprite   rl.Texture2D
	Src      rl.Rectangle
	Dest     rl.Rectangle
	Dir      int
	Frame    int
}

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
	potions = nil
}

func SpawnPotion() {
	positions := GetRandomFloorPositions(1)
	if len(positions) == 0 {
		resetPotion()
		return
	}
	pos := positions[0]
	potions = []Potion{{
		Position: pos,
		Active:   true,
	}}
}

var potions []Potion

func SpawnPotions(amount int, tiles []world.Tile, tileSizePx int) {
	if amount <= 0 || len(tiles) == 0 || tileSizePx <= 0 {
		return
	}

	// Build a set of already occupied positions to avoid duplicates
	occupied := make(map[[2]int]struct{}, len(potions))
	for i := 0; i < len(potions); i++ {
		key := [2]int{int(potions[i].Position.X), int(potions[i].Position.Y)}
		occupied[key] = struct{}{}
	}

	// Spawn until we reach the requested amount, picking random tiles from the provided list.
	for len(potions) < amount {
		idx := rand.Intn(len(tiles))
		t := tiles[idx]
		x := float32(t.X * tileSizePx)
		y := float32(t.Y * tileSizePx)
		key := [2]int{int(x), int(y)}
		if _, exists := occupied[key]; exists {
			continue
		}
		potions = append(potions, Potion{
			Position: rl.NewVector2(x, y),
			Active:   true,
		})
		occupied[key] = struct{}{}
	}
}

func DrawPotion() {
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
		if !p.Active {
			continue
		}
		dst := rl.NewRectangle(p.Position.X, p.Position.Y, tileSize, tileSize)
		rl.DrawTexturePro(potionTexture, src, dst, rl.NewVector2(0, 0), 0, rl.White)
	}
}

func UpdatePotionPickup(playerHitbox rl.Rectangle) {
	if len(potions) == 0 {
		return
	}

	for i := 0; i < len(potions); {
		p := potions[i]
		pRect := rl.NewRectangle(p.Position.X, p.Position.Y, tileSize, tileSize)

		collides := playerHitbox.X < pRect.X+pRect.Width &&
			playerHitbox.X+playerHitbox.Width > pRect.X &&
			playerHitbox.Y < pRect.Y+pRect.Height &&
			playerHitbox.Y+playerHitbox.Height > pRect.Y

		if collides {
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

			potions = append(potions[:i], potions[i+1:]...)
		} else {
			i++
		}
	}
}
