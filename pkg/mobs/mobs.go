package mobs

import (
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
	Damage       bool
}

var (
	ghost               Mob
	deathDuration       int     = 120
	attackRange         float32 = 25
	attackDuration      int     = 20
	mobHealthBarTexture rl.Texture2D
	mobHealthBarSrc     rl.Rectangle
	attackCooldown      int = 60
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
		Damage:       false,
	}

	mobHealthBarTexture = rl.LoadTexture("assets/spooknloot/char/healthbar.png")
	mobHealthBarSrc = rl.NewRectangle(0, 0, 64, 16)
}

func MobMoving(playerPos rl.Vector2, attackPlayerFunc func()) {
	globalFrameCount++

	ghost.OldX, ghost.OldY = ghost.Dest.X, ghost.Dest.Y
	ghost.Src.X = ghost.Src.Width * float32(ghost.Frame)

	if ghost.Health <= 0 && !ghost.IsDead {
		ghost.Dir = int(DirDead)
		ghost.IsDead = true
		ghost.Frame = 0
	}

	if ghost.FrameCount%10 == 1 {
		ghost.Frame++
	}

	if ghost.IsDead {
		if ghost.Frame >= 1 {
			ghost.Frame = 0
		}
	} else {
		if ghost.Frame >= 4 {
			ghost.Frame = 0
		}
	}

	if ghost.Damage {
		if ghost.Frame >= 2 {
			ghost.Frame = 0
			ghost.Damage = false
		}
	} else {
		if ghost.Frame >= 4 {
			ghost.Frame = 0
		}
	}

	if ghost.Frame >= 4 {
		ghost.Frame = 0
	}

	ghost.FrameCount++

	if ghost.IsDead {
		ghost.Dir = int(DirDead)
		ghost.DeathTimer++
		if ghost.DeathTimer >= deathDuration {
			ghost.IsDead = false
			ghost.DeathTimer = 0
		}
	}

	if !ghost.IsDead {
		dist := rl.Vector2Distance(rl.NewVector2(ghost.Dest.X, ghost.Dest.Y), playerPos)

		if dist <= attackRange && globalFrameCount-ghost.LastAttack >= attackCooldown && !ghost.IsAttacking {
			ghost.LastAttack = globalFrameCount
			ghost.IsAttacking = true
			ghost.AttackTimer = attackDuration
		}

		if ghost.IsAttacking {
			ghost.Dir = int(DirAttackDown)
			ghost.AttackTimer--

			if ghost.AttackTimer <= attackDuration-3 && ghost.AttackTimer > attackDuration-6 {
				attackPlayerFunc()
			}
			if ghost.AttackTimer <= 0 {
				ghost.IsAttacking = false
			}
		}

		if !ghost.IsAttacking && dist < 180 && dist > 2 {
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

					if ghost.IsAttacking {
						ghost.Dir = int(DirAttackRight)
					}
				} else {
					ghost.Dir = int(DirMoveLeft)
					if ghost.IsAttacking {
						ghost.Dir = int(DirAttackLeft)
					}
				}
			} else {
				if directionY > 0 {
					ghost.Dir = int(DirMoveDown)
					if ghost.IsAttacking {
						ghost.Dir = int(DirAttackDown)
					}
				} else {
					ghost.Dir = int(DirMoveUp)
				}
			}

			moveSpeed := float32(1.0)

			ghost.Dest.X += directionX * moveSpeed
			ghost.Dest.Y += directionY * moveSpeed
		}
	}

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

func DrawGhostHealthBar() {
	if ghost.Health <= 0 {
		return
	}

	mobHealthBarSrc.Y = mobHealthBarSrc.Height * float32(ghost.HealthbarDir)

	healthBarX := ghost.Dest.X + (ghost.Dest.Width / 2)
	healthBarY := ghost.Dest.Y

	mobHealthBarDest := rl.NewRectangle(healthBarX, healthBarY, float32(64), float32(16))

	rl.DrawTexturePro(mobHealthBarTexture, mobHealthBarSrc, mobHealthBarDest, rl.NewVector2(0, 0), 0, rl.White)
}

func GetMobPosition() rl.Vector2 {
	return rl.NewVector2(ghost.Dest.X, ghost.Dest.Y)
}

func IsMobAlive() bool {
	return ghost.Health > 0 && !ghost.IsDead
}

func GetMobPositionByIndex(index int) rl.Vector2 {
	if index < 0 || index >= 1 || ghost.Health <= 0 || ghost.IsDead {
		//distance := rl.Vector2Distance(rl.NewVector2(ghost.Dest.X, ghost.Dest.Y), playerPos)

		return rl.Vector2{}
	}

	return rl.NewVector2(ghost.Dest.X, ghost.Dest.Y)
}

func GetClosestMobIndex(playerPos rl.Vector2) int {
	if ghost.Health <= 0 || ghost.IsDead {
		return -1
	}
	distance := rl.Vector2Distance(rl.NewVector2(ghost.Dest.X, ghost.Dest.Y), playerPos)

	if distance <= 1000 {
		return 0
	}
	return -1
}

func DrawMobs() {
	rl.DrawTexturePro(ghost.Sprite, ghost.Src, ghost.Dest, rl.NewVector2(0, 0), 0, rl.White)
}

func DamageMob(mobIndex int, damage float32) {
	if ghost.Health <= 0 || ghost.IsDead {
		return
	}

	wasAlive := ghost.Health > 0

	ghost.Health -= damage
	if ghost.Health < 0 {
		ghost.Health = 0
	}

	if wasAlive && ghost.Health <= 0 {
		ghost.IsDead = true
		ghost.DeathTimer = 0
	}

	healthPercentage := ghost.Health / ghost.MaxHealth
	if healthPercentage > 0.8 {
		ghost.HealthbarDir = 0
	} else if healthPercentage > 0.6 {
		ghost.HealthbarDir = 1
	} else if healthPercentage > 0.4 {
		ghost.HealthbarDir = 2
	} else if healthPercentage > 0.2 {
		ghost.HealthbarDir = 3
	} else {
		ghost.HealthbarDir = 4
	}
}

func ResetMobs() {
	ghost.Dest.X = 495
	ghost.Dest.Y = 526
	ghost.Health = ghost.MaxHealth
	ghost.IsDead = false
	ghost.DeathTimer = 0
	globalFrameCount = 0
}
