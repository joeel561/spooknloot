package player

import (
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1920
	screenHeight = 1080
)

var (
	playerSprite rl.Texture2D
	oldX, oldY   float32

	playerSrc                                                                rl.Rectangle
	PlayerDest                                                               rl.Rectangle
	PlayerMove                                                               bool
	playerDir                                                                Direction
	playerUp, playerDown, playerLeft, playerRight, playerAttack, playerBlock bool
	playerFrame                                                              int
	PlayerHitBox                                                             rl.Rectangle
	playerHitBoxYOffset                                                      float32 = 3
	playerMoveTool                                                           bool
	PlayerToolHitBox                                                         rl.Rectangle
	playerToolFrame                                                          int
	playerJumping                                                            bool
	playerJumpTimer                                                          int
	playerFrameAttack                                                        int
	frameCountAttack                                                         int
	attackActive                                                             bool
	baseFacing                                                               Direction
	PlayerRadius                                                             rl.Rectangle

	frameCount int

	playerSpeed float32 = 1.4

	healthBarTexture rl.Texture2D
	maxHealth        float32 = 10.0
	currentHealth    float32 = 10.0
	healthbarDir     int     = 0
	healthBarSrc     rl.Rectangle

	lastAttackTime int
	attackCooldown int     = 30
	attackRange    float32 = 40
	isAttacking    bool
	attackDuration int = 15
	attackTimer    int
	attackPressed  bool

	healthRegenTimer    int = 0
	healthRegenInterval int = 120

	Cam rl.Camera2D
)

type Direction int

const (
	DirDown = Direction(iota)
	DirLeft
	DirRight
	DirUp
	DirAttackDown
	DirAttackLeft
	DirAttackRight
	DirAttackUp
	DirJumpDown
	DirJumpLeft
	DirJumpRight
	DirJumpUp
	DirDamageDown
	DirDamageLeft
	DirDamageRight
	DirDamageUp
)

func InitPlayer() {
	playerSprite = rl.LoadTexture("assets/char/char-sprites.png")

	healthBarTexture = rl.LoadTexture("assets/char/healthbar.png")
	rl.SetTextureFilter(healthBarTexture, rl.FilterPoint)

	healthBarSrc = rl.NewRectangle(0, 0, 64, 16)

	playerSrc = rl.NewRectangle(0, 0, 48, 48)

	PlayerDest = rl.NewRectangle(495, 344, 48, 48)
	PlayerHitBox = rl.NewRectangle(0, 0, 6, 6)

	Cam = rl.NewCamera2D(rl.NewVector2(float32(screenWidth/2), float32(screenHeight/2)),
		rl.NewVector2(float32(PlayerDest.X-(PlayerDest.Width/2)), float32(PlayerDest.Y-(PlayerDest.Height/2))), 0, 4)
}

func DrawPlayerTexture() {
	rl.DrawTexturePro(playerSprite, playerSrc, PlayerDest, rl.NewVector2(0, 0), 0, rl.White)
}

func PlayerInput() {
	/* 	activeItem := userinterface.PlayerActiveItem */
	if rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeyUp) {
		if !playerMoveTool {
			PlayerMove = true
			playerUp = true
		}
	}

	if rl.IsKeyDown(rl.KeyS) || rl.IsKeyDown(rl.KeyDown) {
		if !playerMoveTool {
			PlayerMove = true
			playerDown = true
		}
	}

	if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
		if !playerMoveTool {
			PlayerMove = true
			playerLeft = true
		}
	}

	if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
		if !playerMoveTool {
			PlayerMove = true
			playerRight = true
		}
	}

	if rl.IsKeyDown(rl.KeySpace) {
		playerJumping = true
		playerJumpTimer = 1
	}

	if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) || playerJumping {
		playerSpeed = 2
	} else {
		playerSpeed = 1.4
	}

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		playerAttack = true
	}
}

func TryAttack(targetPos rl.Vector2, attackFunc func(float32)) bool {
	if attackPressed && frameCount-lastAttackTime >= attackCooldown && !attackActive {
		playerPos := rl.NewVector2(PlayerDest.X, PlayerDest.Y)
		dist := rl.Vector2Distance(playerPos, targetPos)

		if dist <= attackRange {
			attackFunc(1.2)
			lastAttackTime = frameCount
			attackActive = true
			attackTimer = attackDuration
			attackPressed = false
			return true
		}
	}
	attackPressed = false
	return false
}

func PlayerMoving() {
	oldX, oldY = PlayerDest.X, PlayerDest.Y
	playerSrc.X = playerSrc.Width * float32(playerFrame)

	if playerAttack && !attackActive {
		attackActive = true
		playerFrameAttack = 0
		frameCountAttack = 0

		playerDirections()

		playerAttack = false
	}

	if attackActive {
		frameCountAttack++
		if frameCountAttack%4 == 0 {
			playerFrameAttack++
		}
		if playerFrameAttack >= 5 {
			attackActive = false
			playerFrameAttack = 0

			switch baseFacing {
			case DirUp:
				playerDir = DirUp
			case DirDown:
				playerDir = DirDown
			case DirLeft:
				playerDir = DirLeft
			case DirRight:
				playerDir = DirRight
			default:
				playerDir = DirDown
			}
		}
	}

	RegenerateHealth()

	if PlayerMove {
		if playerUp {
			if !attackActive {
				playerDir = DirUp
				baseFacing = DirUp
			}
			PlayerDest.Y -= playerSpeed
			if !attackActive {
				playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)
			}

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}

			if playerJumping {
				PlayerDest.Y -= playerSpeed / 2
				PlayerDest.X += playerSpeed / 2
			}
		}
		if playerDown {
			if !attackActive {
				playerDir = DirDown
				baseFacing = DirDown
			}
			PlayerDest.Y += playerSpeed
			if !attackActive {
				playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)
			}

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}
		}
		if playerLeft {
			if !attackActive {
				playerDir = DirLeft
				baseFacing = DirLeft
			}
			PlayerDest.X -= playerSpeed
			if !attackActive {
				playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)
			}

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}
		}

		if playerRight {
			if !attackActive {
				playerDir = DirRight
				baseFacing = DirRight
			}
			PlayerDest.X += playerSpeed
			if !attackActive {
				playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)
			}

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}
		}

		if frameCount%8 == 1 {
			if !attackActive {
				playerFrame++
			}
		}

		PlayerOpenHouseDoor()

	} else if frameCount%45 == 1 {
		playerFrame++
	}

	frameCount++
	if playerFrame >= 4 {
		playerFrame = 0
	}

	playerSrc.Y = playerSrc.Height * float32(playerDir)

	if !PlayerMove && playerFrame > 4 {
		playerFrame = 0
	}

	playerSrc.Y = playerSrc.Height * float32(playerDir)

	if attackActive {
		playerSrc.X = playerSrc.Width * float32(playerFrameAttack)
	}

	PlayerHitBox.X = PlayerDest.X + (PlayerDest.Width / 2) - PlayerHitBox.Width/2
	PlayerHitBox.Y = PlayerDest.Y + (PlayerDest.Height / 2) + playerHitBoxYOffset
	PlayerRadius.X = PlayerDest.X + (PlayerDest.Width / 2) - (PlayerRadius.Width / 2)
	PlayerRadius.Y = PlayerDest.Y + (PlayerDest.Height / 2) - (PlayerRadius.Height / 2)
	PlayerRadius.Width = PlayerDest.Width + 200
	PlayerRadius.Height = PlayerDest.Height + 200

	PlayerCollision(world.Out)
	PlayerCollision(world.Fence)
	PlayerCollision(world.Buildings)
	PlayerCollision(world.Trees)
	PlayerCollision(world.Bushes)
	PlayerCollision(world.Markets)
	PlayerCollisionLamps()

	Cam.Target = rl.NewVector2(float32(PlayerDest.X-(PlayerDest.Width/2)), float32(PlayerDest.Y-(PlayerDest.Height/2)))

	PlayerMove, playerJumping = false, false
	playerUp, playerDown, playerLeft, playerRight = false, false, false, false
}

func PlayerCollision(tiles []world.Tile) {
	var jsonMap = world.WorldMap

	for i := 0; i < len(tiles); i++ {
		if PlayerHitBox.X < float32(tiles[i].X*jsonMap.TileSize+jsonMap.TileSize) &&
			PlayerHitBox.X+PlayerHitBox.Width > float32(tiles[i].X*jsonMap.TileSize) &&
			PlayerHitBox.Y < float32(tiles[i].Y*jsonMap.TileSize+jsonMap.TileSize) &&
			PlayerHitBox.Y+PlayerHitBox.Height > float32(tiles[i].Y*jsonMap.TileSize) {

			PlayerDest.X = oldX
			PlayerDest.Y = oldY
		}
	}
}

func PlayerCollisionLamps() {
	const lampBaseW, lampBaseH = 16, 16

	for i := 0; i < len(world.Lamps); i++ {
		lamp := world.Lamps[i]
		lampRectX := float32(lamp.X)
		lampRectY := float32(lamp.Y)

		if PlayerHitBox.X < lampRectX+float32(lampBaseW) &&
			PlayerHitBox.X+PlayerHitBox.Width > lampRectX &&
			PlayerHitBox.Y < lampRectY+float32(lampBaseH) &&
			PlayerHitBox.Y+PlayerHitBox.Height > lampRectY {
			PlayerDest.X = oldX
			PlayerDest.Y = oldY
		}
	}
}

func PlayerOpenHouseDoor() {
	world.HouseDoorSrc.X = 0

	if PlayerHitBox.X < float32(world.HouseDoorDest.X+world.HouseDoorDest.Width) &&
		PlayerHitBox.X+PlayerHitBox.Width > float32(world.HouseDoorDest.X) &&
		PlayerHitBox.Y < float32(world.HouseDoorDest.Y+world.HouseDoorDest.Height) &&
		PlayerHitBox.Y+PlayerHitBox.Height > float32(world.HouseDoorDest.Y) {
		world.OpenHouseDoor()
	}
}

func RegenerateHealth() {
	healthRegenTimer++

	if healthRegenTimer >= healthRegenInterval {
		if currentHealth < maxHealth {
			currentHealth += 1.0
			if currentHealth > maxHealth {
				currentHealth = maxHealth
			}

			UpdateHealthBar()
		}
		healthRegenTimer = 0
	}
}

func UpdateHealthBar() {
	healthPercentage := currentHealth / maxHealth
	if healthPercentage > 0.8 {
		healthbarDir = 0
	} else if healthPercentage > 0.6 {
		healthbarDir = 1
	} else if healthPercentage > 0.4 {
		healthbarDir = 2
	} else if healthPercentage > 0.2 {
		healthbarDir = 3
	} else {
		healthbarDir = 4
	}
}

func SetPlayerDamageState() {
	playerDir = DirDamageDown
}

func TakeDamage(damage float32) {
	currentHealth -= damage
	if currentHealth < 0 {
		currentHealth = 0
	}

	UpdateHealthBar()
}

func DrawHealthBar() {
	healthBarSrc.Y = healthBarSrc.Height * float32(healthbarDir)

	margin := float32(16)
	barW, barH := float32(64)*2, float32(16)*2

	healthBarX := margin
	healthBarY := barH + margin

	healthBarDest := rl.NewRectangle(healthBarX, healthBarY, barW, barH)

	rl.DrawTexturePro(healthBarTexture, healthBarSrc, healthBarDest, rl.NewVector2(0, 0), 0, rl.White)
}

func GetCurrentHealth() float32 {
	return currentHealth
}

func GetMaxHealth() float32 {
	return maxHealth
}

func IsPlayerDead() bool {
	return currentHealth <= 0
}

func ResetPlayer() {
	currentHealth = maxHealth
	PlayerDest.X = 495
	PlayerDest.Y = 344
	playerDir = 1
	playerFrame = 0
	PlayerMove = false
	playerUp, playerDown, playerLeft, playerRight = false, false, false, false
	isAttacking = false
	attackTimer = 0
	attackPressed = false
	frameCount = 0
	lastAttackTime = 0
	healthRegenTimer = 0

	Cam.Target = rl.NewVector2(float32(PlayerDest.X-(PlayerDest.Width/2)), float32(PlayerDest.Y-(PlayerDest.Height/2)))

	UpdateHealthBar()
}

func playerDirections() {
	switch playerDir {
	case DirUp:
		playerDir = DirAttackUp
		baseFacing = DirUp
	case DirDown:
		playerDir = DirAttackDown
		baseFacing = DirDown
	case DirLeft:
		playerDir = DirAttackLeft
		baseFacing = DirLeft
	case DirRight:
		playerDir = DirAttackRight
		baseFacing = DirRight
	case DirAttackUp:
		baseFacing = DirUp
	case DirAttackDown:
		baseFacing = DirDown
	case DirAttackLeft:
		baseFacing = DirLeft
	case DirAttackRight:
		baseFacing = DirRight
	case DirJumpUp:
		playerDir = DirJumpUp
		baseFacing = DirUp
	case DirJumpDown:
		playerDir = DirJumpDown
		baseFacing = DirDown
	case DirJumpLeft:
		playerDir = DirJumpLeft
		baseFacing = DirLeft
	case DirJumpRight:
		playerDir = DirJumpRight
		baseFacing = DirRight
	case DirDamageUp:
		playerDir = DirDamageUp
		baseFacing = DirUp
	case DirDamageDown:
		playerDir = DirDamageDown
		baseFacing = DirDown
	case DirDamageLeft:
		playerDir = DirDamageLeft
		baseFacing = DirLeft
	case DirDamageRight:
		playerDir = DirDamageRight
		baseFacing = DirRight
	default:
		playerDir = DirAttackDown
		baseFacing = DirDown
	}
}

func UnloadPlayerTexture() {
	rl.UnloadTexture(playerSprite)
	rl.UnloadTexture(healthBarTexture)
}
