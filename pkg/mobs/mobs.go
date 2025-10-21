package mobs

import (
	"fmt"
	"math"
	"math/rand"

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
	ghost            Mob
	deathDuration    int     = 120
	attackRange      float32 = 25
	attackDuration   int     = 20
	attackCooldown   int     = 60
	mobs             []Mob
	mobTexture       rl.Texture2D
	batSprite        rl.Texture2D
	globalFrameCount int
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
	batSprite = rl.LoadTexture("assets/mobs/bat-spritesheet.png")

}

func SpawnMobs(amount int, mobType string) int {
	if len(world.Spawn) == 0 || amount <= 0 {
		return len(mobs)
	}

	if mobType == "bat" {
		mobTexture = batSprite
	}

	for len(mobs) < amount {
		randomIndex := rand.Intn(len(world.Spawn))
		selectedTile := world.Spawn[randomIndex]
		x := float32(selectedTile.X * world.WorldMap.TileSize)
		y := float32(selectedTile.Y * world.WorldMap.TileSize)

		newMob := Mob{
			Sprite:       mobTexture,
			Src:          rl.NewRectangle(0, 0, 16, 16),
			Dest:         rl.NewRectangle(x, y, 16, 16),
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

		mobs = append(mobs, newMob)
	}

	return len(mobs)
}

func MobMoving(playerPos rl.Vector2, attackPlayerFunc func()) {
	globalFrameCount++

	for i := range mobs {

		mobs[i].OldX, mobs[i].OldY = mobs[i].Dest.X, mobs[i].Dest.Y
		mobs[i].Src.X = mobs[i].Src.Width * float32(mobs[i].Frame)

		if mobs[i].Health <= 0 && !mobs[i].IsDead {
			continue
		}

		if mobs[i].FrameCount%10 == 1 {
			mobs[i].Frame++
		}

		if mobs[i].IsDead {
			if mobs[i].Frame >= 1 {
				mobs[i].Frame = 0
			}
		} else {
			if mobs[i].Frame >= 4 {
				mobs[i].Frame = 0
			}
		}

		if mobs[i].Damage {
			if mobs[i].Frame >= 2 {
				mobs[i].Frame = 0
				mobs[i].Damage = false
			}
		} else {
			if mobs[i].Frame >= 4 {
				mobs[i].Frame = 0
			}
		}

		if mobs[i].Frame >= 4 {
			mobs[i].Frame = 0
		}

		mobs[i].FrameCount++

		if mobs[i].IsDead {
			mobs[i].Dir = int(DirDead)
			mobs[i].DeathTimer++
			if mobs[i].DeathTimer >= deathDuration {
				mobs[i].IsDead = false
				mobs[i].DeathTimer = 0
			}
		}

		if !mobs[i].IsDead {
			dist := rl.Vector2Distance(rl.NewVector2(mobs[i].Dest.X, mobs[i].Dest.Y), playerPos)

			if dist <= attackRange && globalFrameCount-mobs[i].LastAttack >= attackCooldown && !mobs[i].IsAttacking {
				mobs[i].LastAttack = globalFrameCount
				mobs[i].IsAttacking = true
				mobs[i].AttackTimer = attackDuration
			}

			if mobs[i].IsAttacking {
				mobs[i].Dir = int(DirAttackDown)
				mobs[i].AttackTimer--

				if mobs[i].AttackTimer <= attackDuration-3 && mobs[i].AttackTimer > attackDuration-6 {
					attackPlayerFunc()
				}
				if mobs[i].AttackTimer <= 0 {
					mobs[i].IsAttacking = false
				}
			}

			if !mobs[i].IsAttacking && dist < 180 && dist > 8 {
				directionX := playerPos.X - mobs[i].Dest.X
				directionY := playerPos.Y - mobs[i].Dest.Y

				length := rl.Vector2Length(rl.NewVector2(directionX, directionY))
				if length > 0 {
					directionX /= length
					directionY /= length
				}

				if float32(math.Abs(float64(directionX))) > float32(math.Abs(float64(directionY))) {
					if directionX > 0 {
						mobs[i].Dir = int(DirMoveRight)

						if mobs[i].IsAttacking {
							mobs[i].Dir = int(DirAttackRight)
						}
					} else {
						mobs[i].Dir = int(DirMoveLeft)
						if mobs[i].IsAttacking {
							mobs[i].Dir = int(DirAttackLeft)
						}
					}
				} else {
					if directionY > 0 {
						mobs[i].Dir = int(DirMoveDown)
						if mobs[i].IsAttacking {
							mobs[i].Dir = int(DirAttackDown)
						}
					} else {
						mobs[i].Dir = int(DirMoveUp)
					}
				}

				moveSpeed := float32(1.0)

				mobs[i].Dest.X += directionX * moveSpeed
				mobs[i].Dest.Y += directionY * moveSpeed
			}
		}

		mobs[i].HitBox.X = mobs[i].Dest.X + (mobs[i].Dest.Width / 2) - mobs[i].HitBox.Width/2
		mobs[i].HitBox.Y = mobs[i].Dest.Y + (mobs[i].Dest.Height / 2)

		mobs[i].Src.Y = mobs[i].Src.Height * float32(mobs[i].Dir)

		MobCollision(i, world.Bushes)
		MobCollision(i, world.Out)
		MobCollision(i, world.Markets)
		MobCollision(i, world.Buildings)
	}
}

func MobCollision(mobIndex int, tiles []world.Tile) {
	var jsonMap = world.WorldMap

	for i := 0; i < len(tiles); i++ {
		if mobs[mobIndex].HitBox.X < float32(tiles[i].X*jsonMap.TileSize+jsonMap.TileSize) &&
			mobs[mobIndex].HitBox.X+mobs[mobIndex].HitBox.Width > float32(tiles[i].X*jsonMap.TileSize) &&
			mobs[mobIndex].HitBox.Y < float32(tiles[i].Y*jsonMap.TileSize+jsonMap.TileSize) &&
			mobs[mobIndex].HitBox.Y+mobs[mobIndex].HitBox.Height > float32(tiles[i].Y*jsonMap.TileSize) {

			mobs[mobIndex].Dest.X = mobs[mobIndex].OldX
			mobs[mobIndex].Dest.Y = mobs[mobIndex].OldY
		}
	}
}

func DrawMobsHealthBar(mobIndex int) {
	if mobs[mobIndex].Health <= 0 {
		return
	}

	maxBarWidth := mobs[mobIndex].Dest.Width
	barHeight := float32(1)
	padding := float32(0)

	barX := mobs[mobIndex].Dest.X + padding
	barY := mobs[mobIndex].Dest.Y - barHeight - 2

	healthPercent := mobs[mobIndex].Health / mobs[mobIndex].MaxHealth
	if healthPercent < 0 {
		healthPercent = 0
	} else if healthPercent > 1 {
		healthPercent = 1
	}

	fmt.Println(healthPercent)

	currentWidth := maxBarWidth * healthPercent

	bgRect := rl.NewRectangle(barX, barY, maxBarWidth, barHeight)
	rl.DrawRectangleRec(bgRect, rl.NewColor(0, 0, 0, 160))

	fgColor := rl.Color{R: 190, G: 75, B: 75, A: 255}
	if healthPercent <= 0.2 {
		fgColor = rl.Color{R: 57, G: 108, B: 60, A: 255}
	} else if healthPercent <= 0.5 {
		fgColor = rl.Color{R: 231, G: 152, B: 50, A: 255}
	}

	fgRect := rl.NewRectangle(barX, barY, currentWidth, barHeight)
	rl.DrawRectangleRec(fgRect, fgColor)
}

func GetMobPosition() rl.Vector2 {
	for i := range mobs {
		if mobs[i].Health > 0 && !mobs[i].IsDead {
			return rl.NewVector2(mobs[i].Dest.X, mobs[i].Dest.Y)
		}
	}

	return rl.NewVector2(0, 0)
}

func IsMobAlive() bool {
	for i := range mobs {
		if mobs[i].Health > 0 && !mobs[i].IsDead {
			return true
		}
	}
	return false
}

func GetMobPositionByIndex(index int) rl.Vector2 {
	if index < 0 || index >= len(mobs) || mobs[index].Health <= 0 || mobs[index].IsDead {
		return rl.NewVector2(0, 0)
	}

	return rl.NewVector2(mobs[index].Dest.X, mobs[index].Dest.Y)
}

func GetMobHitboxCenterByIndex(index int) rl.Vector2 {
	if index < 0 || index >= len(mobs) || mobs[index].Health <= 0 || mobs[index].IsDead {
		return rl.NewVector2(0, 0)
	}

	cx := mobs[index].HitBox.X + (mobs[index].HitBox.Width / 2)
	cy := mobs[index].HitBox.Y + (mobs[index].HitBox.Height / 2)
	return rl.NewVector2(cx, cy)
}

func GetClosestMobIndex(playerPos rl.Vector2) int {
	closestIndex := -1
	closestIndexDistance := float32(999999)

	for i := range mobs {
		if mobs[i].Health > 0 && !mobs[i].IsDead {
			distance := rl.Vector2Distance(playerPos, rl.NewVector2(mobs[i].Dest.X, mobs[i].Dest.Y))
			if distance < closestIndexDistance {
				closestIndexDistance = distance
				closestIndex = i
			}
		}
	}

	return closestIndex
}

func DrawMobs() {
	for i := range mobs {
		if mobs[i].Health > 0 || mobs[i].IsDead {
			rl.DrawTexturePro(mobs[i].Sprite, mobs[i].Src, mobs[i].Dest, rl.NewVector2(0, 0), 0, rl.White)
			if mobs[i].Health > 0 && !mobs[i].IsDead {
				DrawMobsHealthBar(i)
			}
		}
	}
}

func DamageMob(mobIndex int, damage float32) {
	if mobIndex < 0 || mobIndex >= len(mobs) {
		return
	}

	wasAlive := mobs[mobIndex].Health > 0

	mobs[mobIndex].Health -= damage
	if mobs[mobIndex].Health < 0 {
		mobs[mobIndex].Health = 0
	}

	if wasAlive && mobs[mobIndex].Health <= 0 {
		mobs[mobIndex].IsDead = true
		mobs[mobIndex].DeathTimer = 0
	}

	healthPercentage := mobs[mobIndex].Health / mobs[mobIndex].MaxHealth
	if healthPercentage > 0.8 {
		mobs[mobIndex].HealthbarDir = 0
	} else if healthPercentage > 0.6 {
		mobs[mobIndex].HealthbarDir = 1
	} else if healthPercentage > 0.4 {
		mobs[mobIndex].HealthbarDir = 2
	} else if healthPercentage > 0.2 {
		mobs[mobIndex].HealthbarDir = 3
	} else {
		mobs[mobIndex].HealthbarDir = 4
	}
}

func ResetMobs() {
	mobs = []Mob{}
	globalFrameCount = 0
}

func UnloadMobsTexture() {
	rl.UnloadTexture(mobTexture)
}
