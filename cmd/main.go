package main

import (
	"spooknloot/pkg/debug"
	"spooknloot/pkg/mobs"
	"spooknloot/pkg/player"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1920
	screenHeight = 1080
)

var (
	running = true
	bgColor = rl.NewColor(143, 77, 87, 1)

	musicPaused bool
	music       rl.Music
	printDebug  bool
)

func drawScene() {
	world.DrawWorld()
	world.DrawBottomLamp()
	world.DrawDoors()
	mobs.DrawMobs()

	player.DrawPlayerTexture()
	mobs.SpawnMobs(5, "bat")

	world.DrawWheat()
	world.DrawPumpkinLamp()
	world.DrawTopLamp()
	world.DrawCauldron()

	if printDebug {
		debug.DrawPlayerOutlines()
	}
}

func init() {
	rl.InitWindow(screenWidth, screenHeight, "spook 'n loot - a game by joeel56")
	rl.SetExitKey(0)
	rl.SetTargetFPS(60)

	world.InitWorld()
	world.InitDoors()
	world.InitLamps()
	world.InitPumpkinLamps()

	world.LoadMap("pkg/world/map.json")

	player.InitPlayer()
	mobs.InitMobs()

	printDebug = false
}

func input() {
	if rl.IsKeyPressed(rl.KeyF10) {
		rl.ToggleBorderlessWindowed()
	}

	player.PlayerInput()

	if rl.IsKeyPressed(rl.KeyF3) {
		printDebug = !printDebug
	}

	if rl.IsKeyPressed(rl.KeyEscape) {
		running = false
	}
}

func update() {
	running = !rl.WindowShouldClose()

	world.LightLamps()
	world.LightPumpkinLamps()

	if player.IsPlayerDead() {
		// Keep updating to progress the death animation
		player.PlayerMoving()
		if player.HasPlayerDeathAnimationFinished() {
			player.ResetPlayer()
			mobs.ResetMobs()
		}
		return
	}

	player.PlayerMoving()

	playerPos := rl.NewVector2(player.PlayerHitBox.X, player.PlayerHitBox.Y)
	attackPlayerFunc := func() {
		player.SetPlayerDamageState()
		player.TakeDamage(0.5)
	}
	mobs.MobMoving(playerPos, attackPlayerFunc)

	if mobs.IsMobAlive() {
		closestMobIndex := mobs.GetClosestMobIndex(playerPos)
		if closestMobIndex != -1 {
			// Use mob hitbox center for reliable melee range checks
			mobCenter := mobs.GetMobHitboxCenterByIndex(closestMobIndex)
			player.TryAttack(mobCenter, func(damage float32) {
				mobs.DamageMob(closestMobIndex, damage)
			})
		}
	}
}

func render() {
	var cam = player.Cam

	rl.BeginDrawing()
	rl.ClearBackground(bgColor)
	rl.BeginMode2D(cam)

	drawScene()
	rl.EndMode2D()

	player.DrawHealthBar()

	if printDebug {
		debug.DrawDebug(debug.DebugText())
	}

	rl.EndDrawing()
}

func quit() {
	player.UnloadPlayerTexture()
	world.UnloadWorldTexture()
	world.UnloadDoorsTextures()
	world.UnloadPumpkinLamps()
	mobs.UnloadMobsTexture()

	rl.CloseWindow()
}

func main() {
	for running {
		input()
		update()
		render()
	}

	quit()
}
