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

func SpawnPotions(amount int, tiles []world.Tile) {
	if amount <= 0 || len(tiles) == 0 {
		potions = nil
		return
	}

	potions = make([]Potion, 0, amount)

	seen := make(map[int]struct{}, len(tiles))
	for len(potions) < amount && len(seen) < len(tiles) {
		idx := rand.Intn(len(tiles))
		if _, ok := seen[idx]; ok {
			continue
		}
		seen[idx] = struct{}{}
		t := tiles[idx]
		x := float32(t.X * world.WorldMap.TileSize)
		y := float32(t.Y * world.WorldMap.TileSize)
		potions = append(potions, Potion{
			Position: rl.NewVector2(x, y),
			Active:   true,
		})
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
