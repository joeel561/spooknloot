package main

import (
	"spooknloot/pkg/debug"
	"spooknloot/pkg/dungeon"
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

	// game mode
	inDungeon     bool
	savedWorldPos rl.Vector2
)

func drawScene() {
	if inDungeon {
		dungeon.Draw()
	} else {
		world.DrawWorld()
		world.DrawBottomLamp()
		world.DrawDoors()
		mobs.DrawMobs()
		world.DrawWheat()
		world.DrawPumpkinLamp()
		world.DrawTopLamp()
		world.DrawCauldron()
	}

	player.DrawPlayerTexture()

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

	dungeon.Init()

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

	// temporary dev toggle to exit dungeon
	if inDungeon && rl.IsKeyPressed(rl.KeyBackspace) {
		exitDungeon()
	}
}

func update() {
	running = !rl.WindowShouldClose()

	if !inDungeon {
		world.LightLamps()
		world.LightPumpkinLamps()
	}

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
	if !inDungeon {
		mobs.MobMoving(playerPos, attackPlayerFunc)
	}

	if !inDungeon && mobs.IsMobAlive() {
		closestMobIndex := mobs.GetClosestMobIndex(playerPos)
		if closestMobIndex != -1 {
			// Use mob hitbox center for reliable melee range checks
			mobCenter := mobs.GetMobHitboxCenterByIndex(closestMobIndex)
			player.TryAttack(mobCenter, func(damage float32) {
				mobs.DamageMob(closestMobIndex, damage)
			})
		}
	}

	if !inDungeon {
		checkEnterDungeon()
	} else {
		// exit when hitting dungeon exit tile
		if dungeon.IsPlayerAtExit(player.PlayerHitBox) {
			exitDungeon()
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
	dungeon.Unload()

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

func checkEnterDungeon() {
	// enter when overlapping the house door
	if player.PlayerHitBox.X < float32(world.HouseDoorDest.X+world.HouseDoorDest.Width) &&
		player.PlayerHitBox.X+player.PlayerHitBox.Width > float32(world.HouseDoorDest.X) &&
		player.PlayerHitBox.Y < float32(world.HouseDoorDest.Y+world.HouseDoorDest.Height) &&
		player.PlayerHitBox.Y+player.PlayerHitBox.Height > float32(world.HouseDoorDest.Y) {
		enterDungeon()
	}
}

func enterDungeon() {
	if inDungeon {
		return
	}
	inDungeon = true
	savedWorldPos = rl.NewVector2(player.PlayerDest.X, player.PlayerDest.Y)

	dungeon.Generate()
	player.SetExternalColliders(dungeon.GetColliders())
	spawn := dungeon.GetSpawnPosition()
	player.SetPosition(spawn.X, spawn.Y)
}

func exitDungeon() {
	if !inDungeon {
		return
	}
	inDungeon = false
	player.ClearExternalColliders()
	player.SetPosition(savedWorldPos.X, savedWorldPos.Y)
}
