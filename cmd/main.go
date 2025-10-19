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

	player.DrawPlayerTexture()
	mobs.DrawGhosts()
	mobs.DrawMobs()

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
	player.InitPlayer()

	world.InitWorld()
	world.InitDoors()
	world.InitLamps()
	world.InitPumpkinLamps()

	world.LoadMap("pkg/world/map.json")

	mobs.InitGhost()
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
			mobPos := mobs.GetMobPositionByIndex(closestMobIndex)
			player.TryAttack(mobPos, func(damage float32) {
				mobs.DamageMob(closestMobIndex, damage)
			})
		}
	}
	//mobs.UpdateGhostSpawning()
	/*
		player.PlayerUseTools()
		items.UpdateItems() */

	/* 	//rl.UpdateMusicStream(music)
	   	if musicPaused {
	   		rl.PauseMusicStream(music)
	   	} else {
	   		rl.ResumeMusicStream(music)
	   	} */
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

	/* 	userinterface.DrawUserInterface() */

	rl.EndDrawing()
}

func quit() {
	player.UnloadPlayerTexture()
	world.UnloadWorldTexture()
	world.UnloadDoorsTextures()
	world.UnloadPumpkinLamps()
	mobs.UnloadGhostTexture()
	/*
		userinterface.UnloadUserInterface() */
	//	rl.UnloadMusicStream(music)
	//	rl.CloseAudioDevice()
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
