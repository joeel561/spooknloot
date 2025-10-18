package mobs

import (
	"fmt"
	"math"

	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Mob struct {
	Sprite       rl.Texture2D
	OldX, OldY   float32
	Src          rl.Rectangle
	Dest         rl.Rectangle
	Dir          int
	Frame        int
	HitBox       rl.Rectangle
	FrameCount   int
	LastAttack   int
	IsAttacking  bool
	AttackTimer  int
	MaxHealth    float32
	Health       float32
	HealthbarDir int
	IsDead       bool
	DeathTimer   int
}

var (
	ghost Mob
)

type Direction int

const (
	DirIdleDown = Direction(iota)
	DirIdleLeft
	DirIdleRight
	DirIdleUp
	DirMoveDown
	DirMoveLeft
	DirMoveRight
	DirMoveUp
	DirAttackDown
	DirAttackLeft
	DirAttackRight
	DirAttackUp
	DirDead
	DirDamageDown
	DirDamageLeft
	DirDamageRight
	DirDamageUp
)

func InitMobs() {
	ghost = Mob{
		Sprite:       ghostSprite,
		Src:          rl.NewRectangle(0, 0, 16, 16),
		Dest:         rl.NewRectangle(495, 526, 16, 16),
		Dir:          int(DirIdleDown),
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
}

func MobMoving(playerPos rl.Vector2) {
	globalFrameCount++

	ghost.OldX, ghost.OldY = ghost.Dest.X, ghost.Dest.Y
	ghost.Src.X = ghost.Src.Width * float32(ghost.Frame)

	if ghost.FrameCount%10 == 1 {
		ghost.Frame++
	}

	if ghost.Frame >= 4 {
		ghost.Frame = 0
	}

	ghost.FrameCount++

	if !ghost.IsDead {
		dist := rl.Vector2Distance(rl.NewVector2(ghost.Dest.X, ghost.Dest.Y), playerPos)

		if !ghost.IsAttacking && dist < 180 && dist > 5 {
			directionX := playerPos.X - ghost.Dest.X
			directionY := playerPos.Y - ghost.Dest.Y

			length := rl.Vector2Length(rl.NewVector2(directionX, directionY))
			if length > 0 {
				directionX /= length
				directionY /= length
			}

			if float32(math.Abs(float64(directionX))) > float32(math.Abs(float64(directionY))) {
				if directionX > 0 {
					ghost.Dir = int(DirMoveRight)
				} else {
					ghost.Dir = int(DirMoveLeft)
				}
			} else {
				if directionY > 0 {
					ghost.Dir = int(DirMoveDown)
				} else {
					ghost.Dir = int(DirMoveUp)
				}
			}

			moveSpeed := float32(1.0)

			ghost.Dest.X += directionX * moveSpeed
			ghost.Dest.Y += directionY * moveSpeed
		}
	}

	fmt.Println(ghost.Dir)

	ghost.HitBox.X = ghost.Dest.X + (ghost.Dest.Width / 2) - ghost.HitBox.Width/2
	ghost.HitBox.Y = ghost.Dest.Y + (ghost.Dest.Height / 2) + ghostHitBoxYOffset

	ghost.Src.Y = ghost.Src.Height * float32(ghost.Dir)

	MobCollision(world.Bushes)
	MobCollision(world.Out)
	MobCollision(world.Markets)
	MobCollision(world.Buildings)
}

func MobCollision(tiles []world.Tile) {
	var jsonMap = world.WorldMap

	for i := 0; i < len(tiles); i++ {
		if ghost.HitBox.X < float32(tiles[i].X*jsonMap.TileSize+jsonMap.TileSize) &&
			ghost.HitBox.X+ghost.HitBox.Width > float32(tiles[i].X*jsonMap.TileSize) &&
			ghost.HitBox.Y < float32(tiles[i].Y*jsonMap.TileSize+jsonMap.TileSize) &&
			ghost.HitBox.Y+ghost.HitBox.Height > float32(tiles[i].Y*jsonMap.TileSize) {

			ghost.Dest.X = ghost.OldX
			ghost.Dest.Y = ghost.OldY
		}
	}
}

func DrawMobs() {
	rl.DrawTexturePro(ghost.Sprite, ghost.Src, ghost.Dest, rl.NewVector2(0, 0), 0, rl.White)
}
