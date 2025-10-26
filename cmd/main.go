package main

import (
	"spooknloot/pkg/boss"
	"spooknloot/pkg/debug"
	"spooknloot/pkg/dungeon"
	"spooknloot/pkg/mobs"
	"spooknloot/pkg/player"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth               = 1500
	screenHeight              = 900
	exitCooldownFramesDefault = 20
)

var (
	running        = true
	worldBgColor   = rl.NewColor(143, 77, 87, 1)
	dungeonBgColor = rl.NewColor(41, 29, 43, 1)

	musicPaused bool
	music       rl.Music
	printDebug  bool

	// game mode
	inDungeon          bool
	inBoss             bool
	dungeonsCleared    int
	savedWorldPos      rl.Vector2
	exitCooldownFrames int
)

func drawScene() {
	if inBoss {
		boss.Draw()
	} else if inDungeon {
		dungeon.Draw()
	} else {
		world.DrawWorld()
		world.DrawBottomLamp()
		world.DrawDoors()
		mobs.DrawMobs()
		world.DrawPumpkinLamp()
	}

	player.DrawPlayerTexture()

	if !inDungeon {
		world.DrawWheat()
		world.DrawTopLamp()
		world.DrawCauldron()
		mobs.SpawnMobs(5, "bat")

	}

	if printDebug {
		debug.DrawPlayerOutlines()
	}
}

func init() {

	monitor := rl.GetCurrentMonitor()
	monW := rl.GetMonitorWidth(monitor)
	monH := rl.GetMonitorHeight(monitor)

	winW := int32(screenWidth)
	winH := int32(screenHeight)
	if monW > 0 && int32(monW) < winW {
		winW = int32(monW)
	}
	if monH > 0 && int32(monH) < winH {
		winH = int32(monH)
	}

	rl.InitWindow(winW, winH, "spook 'n loot - a game by joeel56")
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
	boss.Init()
	boss.LoadMap("pkg/boss/map.json")

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

	if inDungeon && rl.IsKeyPressed(rl.KeyBackspace) {
		exitDungeon()
	}
	if inBoss && rl.IsKeyPressed(rl.KeyBackspace) {
		exitBoss()
	}
}

func update() {
	running = !rl.WindowShouldClose()

	if !inDungeon && !inBoss {
		world.LightLamps()
		world.LightPumpkinLamps()
	}

	if player.IsPlayerDead() {
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
	if !inDungeon && !inBoss {
		mobs.MobMoving(playerPos, attackPlayerFunc)
	}

	if !inDungeon && !inBoss && mobs.IsMobAlive() {
		closestMobIndex := mobs.GetClosestMobIndex(playerPos)
		if closestMobIndex != -1 {
			// Use mob hitbox center for reliable melee range checks
			mobCenter := mobs.GetMobHitboxCenterByIndex(closestMobIndex)
			player.TryAttack(mobCenter, func(damage float32) {
				mobs.DamageMob(closestMobIndex, damage)
			})
		}
	}

	if !inDungeon && !inBoss {
		checkEnterDungeon()
	} else if inDungeon {
		if exitCooldownFrames > 0 {
			exitCooldownFrames--
		}
		// exit when hitting dungeon exit tile
		if exitCooldownFrames <= 0 && dungeon.IsPlayerAtExit(player.PlayerHitBox) {
			// increment cleared counter and decide next state
			dungeonsCleared++
			if dungeonsCleared >= 5 {
				enterBoss()
			} else {
				nextDungeon()
			}
		}
	} else if inBoss {
		// boss room update hooks could go here
	}
}

func render() {
	var cam = player.Cam

	rl.BeginDrawing()
	if inDungeon || inBoss {
		rl.ClearBackground(dungeonBgColor)
	} else {
		rl.ClearBackground(worldBgColor)
	}
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
	exitCooldownFrames = exitCooldownFramesDefault
}

func exitDungeon() {
	if !inDungeon {
		return
	}
	inDungeon = false
	player.ClearExternalColliders()
	player.SetPosition(savedWorldPos.X, savedWorldPos.Y)
}

func nextDungeon() {
	// Keep the player in dungeon mode and generate a new layout
	// Reset colliders and move player to the new spawn
	dungeon.Generate()
	player.SetExternalColliders(dungeon.GetColliders())
	spawn := dungeon.GetSpawnPosition()
	player.SetPosition(spawn.X, spawn.Y)
	exitCooldownFrames = exitCooldownFramesDefault
}

func enterBoss() {
	if inBoss {
		return
	}
	inDungeon = false
	inBoss = true

	player.ClearExternalColliders()
	spawn := boss.GetSpawnPosition()
	player.SetPosition(spawn.X, spawn.Y)
}

func exitBoss() {
	if !inBoss {
		return
	}
	inBoss = false

	player.SetPosition(savedWorldPos.X, savedWorldPos.Y)
}
