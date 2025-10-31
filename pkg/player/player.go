package player

import (
	"os"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth          = 1500
	screenHeight         = 900
	camYOffset   float32 = 1
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
	playerFrameDead                                                          int
	frameCountAttack                                                         int
	attackActive                                                             bool
	baseFacing                                                               Direction
	PlayerRadius                                                             rl.Rectangle

	frameCount int

	playerSpeed float32 = 1.4

	healthBarTexture rl.Texture2D
	healthBarScale   float32 = 4
	maxHealth        float32 = 20.0
	currentHealth    float32 = 20.0
	healthbarDir     int     = 0
	healthBarSrc     rl.Rectangle

	attackRange       float32 = 40
	isAttacking       bool
	attackDuration    int = 15
	attackTimer       int
	attackPressed     bool
	attackHasHit      bool
	playerDamageTimer int

	healthRegenTimer    int = 0
	healthRegenInterval int = 120

	takeDamage bool

	Cam rl.Camera2D

	// Death animation state
	deathAnimationComplete bool

	// External collision handling (e.g., dungeon)
	useExternalColliders bool
	externalColliders    []rl.Rectangle

	// Audio
	attackSound        rl.Sound
	attackSoundLoaded  bool
	damageSound        rl.Sound
	damageSoundLoaded  bool
	walkingSound       rl.Sound
	walkingSoundLoaded bool
	lastFootstepFrame  int
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
	DirJumpDown
	DirJumpLeft
	DirJumpRight
	DirJumpUp
	DirDamageDown
	DirDamageLeft
	DirDamageRight
	DirDamageUp
	DirDeadDown
	DirDeadLeft
	DirDeadRight
	DirDeadUp
	DirDashDown
	DirDashLeft
	DirDashRight
	DirDashUp
)

func InitPlayer() {
	playerSprite = rl.LoadTexture("assets/char/char-sheet.png")

	healthBarTexture = rl.LoadTexture("assets/char/healthbar.png")
	rl.SetTextureFilter(healthBarTexture, rl.FilterPoint)

	healthBarSrc = rl.NewRectangle(0, 0, 64, 16)

	playerSrc = rl.NewRectangle(0, 0, 48, 48)

	PlayerDest = rl.NewRectangle(495, 344, 48, 48)
	PlayerHitBox = rl.NewRectangle(0, 0, 6, 6)

	Cam = rl.NewCamera2D(
		rl.NewVector2(float32(rl.GetScreenWidth()/2), float32(rl.GetScreenHeight()/2)),
		rl.NewVector2(float32(PlayerDest.X+(PlayerDest.Width/2)), float32(PlayerDest.Y+(PlayerDest.Height/2)+camYOffset)),
		0,
		4,
	)

	if _, err := os.Stat("assets/audio/attack.mp3"); err == nil {
		attackSound = rl.LoadSound("assets/audio/attack.mp3")
		rl.SetSoundVolume(attackSound, 0.6)
		attackSoundLoaded = true
	}

	if _, err := os.Stat("assets/audio/damage.mp3"); err == nil {
		damageSound = rl.LoadSound("assets/audio/damage.mp3")
		rl.SetSoundVolume(damageSound, 0.6)
		damageSoundLoaded = true
	}

	if _, err := os.Stat("assets/audio/walking.mp3"); err == nil {
		walkingSound = rl.LoadSound("assets/audio/walking.mp3")
		rl.SetSoundVolume(walkingSound, 0.35)
		walkingSoundLoaded = true
		lastFootstepFrame = -10
	}
}

func DrawPlayerTexture() {
	rl.DrawTexturePro(playerSprite, playerSrc, PlayerDest, rl.NewVector2(0, 0), 0, rl.White)
}

func PlayerInput() {
	// Block input while dead so the player can't move/attack during death animation
	if IsPlayerDead() {
		return
	}
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
		if !(rl.IsKeyDown(rl.KeySpace) || rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)) {
			playerAttack = true
			attackPressed = true
		}
	}
}

func TryAttack(targetPos rl.Vector2, attackFunc func(float32)) bool {

	if attackActive && !attackHasHit {
		px := PlayerHitBox.X + (PlayerHitBox.Width / 2)
		py := PlayerHitBox.Y + (PlayerHitBox.Height / 2)
		playerCenter := rl.NewVector2(px, py)

		dist := rl.Vector2Distance(playerCenter, targetPos)
		if dist <= attackRange {
			attackFunc(2.5)
			attackHasHit = true
			attackTimer = attackDuration
			playerAttack = false
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

	if IsPlayerDead() {
		if attackSoundLoaded && rl.IsSoundPlaying(attackSound) {
			rl.StopSound(attackSound)
		}
		if damageSoundLoaded && rl.IsSoundPlaying(damageSound) {
			rl.StopSound(damageSound)
		}
		if walkingSoundLoaded && rl.IsSoundPlaying(walkingSound) {
			rl.StopSound(walkingSound)
		}
		attackActive = false
	}

	if playerAttack && !attackActive {
		attackActive = true
		playerFrameAttack = 0
		frameCountAttack = 0
		attackHasHit = false
		if attackSoundLoaded && !rl.IsSoundPlaying(attackSound) {
			rl.PlaySound(attackSound)
		}

		if walkingSoundLoaded && rl.IsSoundPlaying(walkingSound) {
			rl.StopSound(walkingSound)
		}

		playerDirections()

		playerAttack = false
	}

	if takeDamage {
		if playerFrame >= 2 {
			playerFrame = 0
		}
		switch baseFacing {
		case DirMoveDown:
			playerDir = DirDamageDown
		case DirMoveUp:
			playerDir = DirDamageUp
		case DirMoveLeft:
			playerDir = DirDamageLeft
		case DirMoveRight:
			playerDir = DirDamageRight
		default:
			playerDir = DirDamageDown
		}
		playerDamageTimer = 10
		takeDamage = false

		if damageSoundLoaded && !rl.IsSoundPlaying(damageSound) {
			rl.PlaySound(damageSound)
		}
		if walkingSoundLoaded && rl.IsSoundPlaying(walkingSound) {
			rl.StopSound(walkingSound)
		}
	}

	if attackActive {
		frameCountAttack++
		if frameCountAttack%4 == 0 {
			playerFrameAttack++
		}
		if attackSoundLoaded && !rl.IsSoundPlaying(attackSound) {
			rl.PlaySound(attackSound)
		}
		if playerFrameAttack >= 4 {
			attackActive = false
			playerFrameAttack = 0

			switch baseFacing {
			case DirMoveDown:
				playerDir = DirMoveDown
			case DirMoveUp:
				playerDir = DirMoveUp
			case DirMoveLeft:
				playerDir = DirMoveLeft
			case DirMoveRight:
				playerDir = DirMoveRight
			default:
				playerDir = DirIdleDown
			}
			if attackSoundLoaded && rl.IsSoundPlaying(attackSound) {
				rl.StopSound(attackSound)
			}
		}
	}

	RegenerateHealth()

	if PlayerMove {
		if playerUp {
			if !attackActive && playerDamageTimer == 0 {
				playerDir = DirMoveUp
				baseFacing = DirMoveUp
			}
			PlayerDest.Y -= playerSpeed

			if playerSpeed == 2 {
				playerDir = DirDashUp
			}

			if IsPlayerDead() {
				playerDir = DirDeadUp
			}
		}
		if playerDown {
			if !attackActive && playerDamageTimer == 0 {
				playerDir = DirMoveDown
				baseFacing = DirMoveDown
			}
			PlayerDest.Y += playerSpeed

			if playerSpeed == 2 {
				playerDir = DirDashDown
			}

			if IsPlayerDead() {
				playerDir = DirDeadDown
			}
		}
		if playerLeft {
			if !attackActive && playerDamageTimer == 0 {
				playerDir = DirMoveLeft
				baseFacing = DirMoveLeft
			}
			PlayerDest.X -= playerSpeed

			if playerSpeed == 2 {
				playerDir = DirDashLeft
			}

			if IsPlayerDead() {
				playerDir = DirDeadLeft
			}
		}

		if playerRight {
			if !attackActive && playerDamageTimer == 0 {
				playerDir = DirMoveRight
				baseFacing = DirMoveRight
			}
			PlayerDest.X += playerSpeed

			if playerSpeed == 2 {
				playerDir = DirDashRight
			}

			if IsPlayerDead() {
				playerDir = DirDeadRight
			}
		}

		if frameCount%8 == 1 {
			if !attackActive {
				playerFrame++
			}
			if IsPlayerDead() {
				playerFrameDead++
			}
			if playerDamageTimer > 0 && playerFrame >= 2 {
				playerFrame = 0
			}

			if walkingSoundLoaded && playerDamageTimer == 0 && !attackActive {
				if playerFrame == 0 || playerFrame == 2 {
					if frameCount-lastFootstepFrame >= 8 { // minimal gap between steps
						lastFootstepFrame = frameCount
						if !rl.IsSoundPlaying(walkingSound) {
							rl.PlaySound(walkingSound)
						} else {
							rl.StopSound(walkingSound)
							rl.PlaySound(walkingSound)
						}
					}
				}
			}
		}

		PlayerOpenHouseDoor()

	} else if frameCount%45 == 1 {
		playerFrame++
		if playerDamageTimer > 0 && playerFrame >= 2 {
			playerFrame = 0
		}
	}

	frameCount++
	if playerDamageTimer > 0 {
		playerDamageTimer--
		if damageSoundLoaded && !rl.IsSoundPlaying(damageSound) {
			rl.PlaySound(damageSound)
		}
	} else {
		if damageSoundLoaded && rl.IsSoundPlaying(damageSound) {
			rl.StopSound(damageSound)
		}
	}

	if IsPlayerDead() {
		switch baseFacing {
		case DirMoveUp:
			playerDir = DirDeadUp
		case DirMoveLeft:
			playerDir = DirDeadLeft
		case DirMoveRight:
			playerDir = DirDeadRight
		default:
			playerDir = DirDeadDown
		}

		if !deathAnimationComplete {
			if frameCount%8 == 1 {
				playerFrameDead++
			}
			if playerFrameDead >= 7 {
				playerFrameDead = 7
				deathAnimationComplete = true
			}
		}
	}
	if playerFrame >= 3 {
		playerFrame = 0
	}

	if !PlayerMove && playerFrame >= 3 {
		playerFrame = 0
	}

	playerSrc.Y = playerSrc.Height * float32(playerDir)

	if attackActive {
		playerSrc.X = playerSrc.Width * float32(playerFrameAttack)
	}

	if IsPlayerDead() {
		playerSrc.X = playerSrc.Width * float32(playerFrameDead)
	}

	PlayerHitBox.X = PlayerDest.X + (PlayerDest.Width / 2) - PlayerHitBox.Width/2
	PlayerHitBox.Y = PlayerDest.Y + (PlayerDest.Height / 2) + playerHitBoxYOffset
	PlayerRadius.X = PlayerDest.X + (PlayerDest.Width / 2) - (PlayerRadius.Width / 2)
	PlayerRadius.Y = PlayerDest.Y + (PlayerDest.Height / 2) - (PlayerRadius.Height / 2)
	PlayerRadius.Width = PlayerDest.Width + 200
	PlayerRadius.Height = PlayerDest.Height + 200

	if useExternalColliders {
		PlayerCollisionRects(externalColliders)
	} else {
		PlayerCollision(world.Out)
		PlayerCollision(world.Fence)
		PlayerCollision(world.Buildings)
		PlayerCollision(world.Trees)
		PlayerCollision(world.Bushes)
		PlayerCollision(world.Markets)
		PlayerCollisionLamps()
	}

	if !PlayerMove {
		if walkingSoundLoaded && rl.IsSoundPlaying(walkingSound) {
			rl.StopSound(walkingSound)
		}
	}

	// Keep camera offset centered on current window size every frame (important after fullscreen toggle)
	Cam.Offset = rl.NewVector2(float32(rl.GetScreenWidth()/2), float32(rl.GetScreenHeight()/2))
	Cam.Target = rl.NewVector2(float32(PlayerDest.X+(PlayerDest.Width/2)), float32(PlayerDest.Y+(PlayerDest.Height/2)+camYOffset))

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

func PlayerCollisionRects(rects []rl.Rectangle) {
	for i := 0; i < len(rects); i++ {
		if PlayerHitBox.X < rects[i].X+rects[i].Width &&
			PlayerHitBox.X+PlayerHitBox.Width > rects[i].X &&
			PlayerHitBox.Y < rects[i].Y+rects[i].Height &&
			PlayerHitBox.Y+PlayerHitBox.Height > rects[i].Y {

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
	if IsPlayerDead() {
		return
	}
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

func SetPlayerDamageState() bool {
	takeDamage = true
	return takeDamage
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

	margin := float32(10)
	barW, barH := float32(64)*healthBarScale, float32(16)*healthBarScale

	healthBarX := margin
	healthBarY := margin

	healthBarDest := rl.NewRectangle(healthBarX, healthBarY, barW, barH)

	rl.DrawTexturePro(healthBarTexture, healthBarSrc, healthBarDest, rl.NewVector2(0, 0), 0, rl.White)
}

func SetHealthBarScale(scale float32) {
	if scale < 1 {
		scale = 1
	}
	healthBarScale = scale
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

func HasPlayerDeathAnimationFinished() bool {
	return deathAnimationComplete
}

func ResetPlayer() {
	currentHealth = maxHealth
	PlayerDest.X = 495
	PlayerDest.Y = 344
	playerDir = 1
	playerFrame = 0
	playerFrameDead = 0
	PlayerMove = false
	playerUp, playerDown, playerLeft, playerRight = false, false, false, false
	isAttacking = false
	attackTimer = 0
	playerAttack = false
	frameCount = 0
	healthRegenTimer = 0
	deathAnimationComplete = false

	attackActive = false
	if attackSoundLoaded && rl.IsSoundPlaying(attackSound) {
		rl.StopSound(attackSound)
	}
	if damageSoundLoaded && rl.IsSoundPlaying(damageSound) {
		rl.StopSound(damageSound)
	}

	Cam.Offset = rl.NewVector2(float32(rl.GetScreenWidth()/2), float32(rl.GetScreenHeight()/2))
	Cam.Target = rl.NewVector2(float32(PlayerDest.X+(PlayerDest.Width/2)), float32(PlayerDest.Y+(PlayerDest.Height/2)+camYOffset))

	UpdateHealthBar()
}

func playerDirections() {
	switch playerDir {
	case DirMoveUp:
		playerDir = DirAttackUp
		baseFacing = DirMoveUp
	case DirMoveDown:
		playerDir = DirAttackDown
		baseFacing = DirMoveDown
	case DirMoveLeft:
		playerDir = DirAttackLeft
		baseFacing = DirMoveLeft
	case DirMoveRight:
		playerDir = DirAttackRight
		baseFacing = DirMoveRight
	case DirAttackUp:
		baseFacing = DirMoveUp
	case DirAttackDown:
		baseFacing = DirMoveDown
	case DirAttackLeft:
		baseFacing = DirMoveLeft
	case DirAttackRight:
		baseFacing = DirMoveRight
	default:
		playerDir = DirAttackDown
		baseFacing = DirMoveDown
	}
}

func UnloadPlayerTexture() {
	rl.UnloadTexture(playerSprite)
	rl.UnloadTexture(healthBarTexture)
	if attackSoundLoaded {
		rl.UnloadSound(attackSound)
		attackSoundLoaded = false
	}
	if damageSoundLoaded {
		rl.UnloadSound(damageSound)
		damageSoundLoaded = false
	}
	if walkingSoundLoaded {
		rl.UnloadSound(walkingSound)
		walkingSoundLoaded = false
	}
}

func SetExternalColliders(rects []rl.Rectangle) {
	externalColliders = rects
	useExternalColliders = true
}

func ClearExternalColliders() {
	externalColliders = nil
	useExternalColliders = false
}

func SetPosition(x, y float32) {
	PlayerDest.X = x
	PlayerDest.Y = y
	// Ensure camera follows player center consistently with Y offset
	Cam.Offset = rl.NewVector2(float32(rl.GetScreenWidth()/2), float32(rl.GetScreenHeight()/2))
	Cam.Target = rl.NewVector2(float32(PlayerDest.X+(PlayerDest.Width/2)), float32(PlayerDest.Y+(PlayerDest.Height/2)+camYOffset))
}
