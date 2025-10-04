package player

import (
	"fmt"
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
	playerMoving                                                             bool
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

	frameCount int

	playerSpeed float32 = 1.4

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
)

func InitPlayer() {
	playerSprite = rl.LoadTexture("assets/char/char-sprites.png")

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
			playerMoving = true
			playerUp = true
		}
	}

	if rl.IsKeyDown(rl.KeyS) || rl.IsKeyDown(rl.KeyDown) {
		if !playerMoveTool {
			playerMoving = true
			playerDown = true
		}
	}

	if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {

		if !playerMoveTool {
			playerMoving = true
			playerLeft = true
		}
	}

	if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
		if !playerMoveTool {
			playerMoving = true
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
	/* 	if activeItem.Name == "Hoe" && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		playerHoe = true
		playerMoveTool = true
	} */
}

func PlayerMoving() {
	oldX, oldY = PlayerDest.X, PlayerDest.Y
	playerSrc.X = playerSrc.Width * float32(playerFrame)

	if playerAttack {
		if playerDir == DirUp {
			playerDir = DirAttackUp
		}

		if playerDir == DirDown {
			playerDir = DirAttackDown
		}

		if playerDir == DirLeft {
			playerDir = DirAttackLeft
		}

		if playerDir == DirRight {
			playerDir = DirAttackRight
		}

		if frameCountAttack%4 == 0 {
			playerFrameAttack++

			fmt.Println("attack frame", playerFrameAttack)
		}

	}

	if playerFrameAttack > 4 {
		playerFrameAttack = 0
	}

	frameCountAttack++

	if playerMoving {
		if playerUp {
			playerDir = DirUp
			PlayerDest.Y -= playerSpeed
			playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}

			if playerJumping {
				PlayerDest.Y -= playerSpeed / 2
				PlayerDest.X += playerSpeed / 2
			}
		}
		if playerDown {
			playerDir = DirDown
			PlayerDest.Y += playerSpeed
			playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}
		}
		if playerLeft {
			playerDir = DirLeft
			PlayerDest.X -= playerSpeed
			playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}
		}

		if playerRight {
			playerDir = DirRight
			PlayerDest.X += playerSpeed
			playerSrc.X = float32(144) + playerSrc.Width*float32(playerFrame)

			if playerSpeed == 2 {
				playerSrc.X = float32(336) + playerSrc.Width*float32(playerFrame)
			}
		}

		if frameCount%8 == 1 {
			playerFrame++
		}

		/*PlayerOpenHouseDoor()
		PlayerOpenBarnDoor() */
	} else if frameCount%45 == 1 {
		playerFrame++
	}

	frameCount++
	if playerFrame >= 4 {
		playerFrame = 0
	}

	playerSrc.Y = playerSrc.Height * float32(playerDir)

	if !playerMoving && playerFrame > 4 {
		playerFrame = 0
	}

	if !playerAttack && playerFrameAttack > 4 {
		playerFrameAttack = 0
	}

	playerSrc.Y = playerSrc.Height * float32(playerDir)

	PlayerHitBox.X = PlayerDest.X + (PlayerDest.Width / 2) - PlayerHitBox.Width/2
	PlayerHitBox.Y = PlayerDest.Y + (PlayerDest.Height / 2) + playerHitBoxYOffset

	PlayerCollision(world.Out)

	Cam.Target = rl.NewVector2(float32(PlayerDest.X-(PlayerDest.Width/2)), float32(PlayerDest.Y-(PlayerDest.Height/2)))

	playerMoving, playerJumping, playerAttack = false, false, false
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

func UnloadPlayerTexture() {
	rl.UnloadTexture(playerSprite)
}
