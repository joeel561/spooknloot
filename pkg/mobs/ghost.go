package mobs

import (
	"fmt"
	"math/rand"
	"spooknloot/pkg/world"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const screenHeight = 1080
const screenWidth = 1920

var (
	ghostSprite        rl.Texture2D
	ghosts             []Mob
	spawnTimer         int = 0
	spawnInterval      int = 500 // 500ms at 60 FPS
	ghostFrameCount    int
	globalFrameCount   int
	ghostHitBoxYOffset float32 = 3
)

func InitGhost() {
	ghostSprite = rl.LoadTexture("assets/mobs/ghost-spritesheet.png")

	rand.Seed(time.Now().UnixNano())

	SpawnGhost()
}

func SpawnGhost() {
	spawnTiles := world.Spawn

	if len(spawnTiles) == 0 {
		return
	}

	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		randomIndex := rand.Intn(len(spawnTiles))
		selectedTile := spawnTiles[randomIndex]

		x := float32(selectedTile.X * world.WorldMap.TileSize)
		y := float32(selectedTile.Y * world.WorldMap.TileSize)

		newGhost := Mob{
			Sprite:       ghostSprite,
			Src:          rl.NewRectangle(0, 0, 16, 30),
			Dest:         rl.NewRectangle(x, y, 16, 30),
			Dir:          5,
			Frame:        0,
			HitBox:       rl.NewRectangle(0, 0, 10, 10),
			FrameCount:   0,
			LastAttack:   0,
			IsAttacking:  false,
			AttackTimer:  0,
			MaxHealth:    5.0,
			Health:       5.0,
			HealthbarDir: 0,
			IsDead:       false,
			DeathTimer:   0,
		}

		ghosts = append(ghosts, newGhost)
		return
	}
}

func UpdateGhostSpawning() {
	spawnTimer++
	if spawnTimer >= spawnInterval {
		SpawnGhost()
		spawnTimer = 0
	}
}

func DrawGhosts() {
	for i := range ghosts {
		rl.DrawTexturePro(ghosts[i].Sprite, ghosts[i].Src, ghosts[i].Dest, rl.NewVector2(0, 0), 0, rl.White)
	}
}

func GhostMoving(playerPos rl.Vector2) {
	globalFrameCount++

	for i := range ghosts {
		if ghosts[i].IsDead {
			continue
		}

		ghosts[i].OldX, ghosts[i].OldY = ghosts[i].Dest.X, ghosts[i].Dest.Y
		ghosts[i].Src.X = ghosts[i].Src.Width * float32(ghosts[i].Frame)

		fmt.Println(ghosts[i].Frame)

		if ghosts[i].FrameCount%10 == 1 {
			ghosts[i].Frame++
		}

		if ghosts[i].Frame >= 4 {
			ghosts[i].Frame = 0
		}

		ghosts[i].FrameCount++

		if !ghosts[i].IsDead {
			dist := rl.Vector2Distance(rl.NewVector2(ghosts[i].Dest.X, ghosts[i].Dest.Y), playerPos)

			if !ghosts[i].IsAttacking && dist < 150 && dist > 5 {
				directionX := playerPos.X - ghosts[i].Dest.X
				directionY := playerPos.Y - ghosts[i].Dest.Y

				length := rl.Vector2Length(rl.NewVector2(directionX, directionY))
				if length > 0 {
					directionX /= length
					directionY /= length
				}

				moveSpeed := float32(0.8)

				ghosts[i].Dest.X += directionX * moveSpeed
				ghosts[i].Dest.Y += directionY * moveSpeed
			}
		}

		ghosts[i].HitBox.X = ghosts[i].Dest.X + (ghosts[i].Dest.Width / 2) - ghosts[i].HitBox.Width/2
		ghosts[i].HitBox.Y = ghosts[i].Dest.Y + (ghosts[i].Dest.Height / 2) + ghostHitBoxYOffset

		GhostCollision(i, world.Buildings)
		GhostCollision(i, world.Fence)
		GhostCollision(i, world.Markets)
		GhostCollision(i, world.Out)
		GhostCollision(i, world.Trees)
		GhostCollision(i, world.Bushes)
	}
}

func GhostCollision(ghostIndex int, tiles []world.Tile) {
	var jsonMap = world.WorldMap

	for i := 0; i < len(tiles); i++ {
		if ghosts[ghostIndex].HitBox.X < float32(tiles[i].X*jsonMap.TileSize+jsonMap.TileSize) &&
			ghosts[ghostIndex].HitBox.X+ghosts[ghostIndex].HitBox.Width > float32(tiles[i].X*jsonMap.TileSize) &&
			ghosts[ghostIndex].HitBox.Y < float32(tiles[i].Y*jsonMap.TileSize+jsonMap.TileSize) &&
			ghosts[ghostIndex].HitBox.Y+ghosts[ghostIndex].HitBox.Height > float32(tiles[i].Y*jsonMap.TileSize) {

			ghosts[ghostIndex].Dest.X = ghosts[ghostIndex].OldX
			ghosts[ghostIndex].Dest.Y = ghosts[ghostIndex].OldY
		}
	}
}

func ResetGhosts() {
	ghosts = []Mob{}
	spawnTimer = 0
	ghostFrameCount = 0
}

func UnloadGhostTexture() {
	rl.UnloadTexture(ghostSprite)
}
