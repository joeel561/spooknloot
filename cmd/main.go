package main

import (
	"os"
	"spooknloot/pkg/boss"
	"spooknloot/pkg/debug"
	"spooknloot/pkg/dungeon"
	"spooknloot/pkg/mobs"
	"spooknloot/pkg/player"
	"spooknloot/pkg/ui"
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

	musicPaused  bool
	worldMusic   rl.Music
	dungeonMusic rl.Music
	bossMusic    rl.Music
	currentMusic string
	printDebug   bool

	menuOpen        bool = true
	menuPausedMusic bool

	inDungeon           bool
	inBoss              bool
	dungeonsCleared     int
	savedWorldPos       rl.Vector2
	exitCooldownFrames  int
	dungeonSpawnCount   int
	dungeonSpawnBaseMin int = 5
	dungeonSpawnBaseMax int = 10
	exitSoundPlayed     bool
	mobsClearFrames     int
)

func drawScene() {
	if inBoss {
		boss.Draw()
	} else if inDungeon {
		dungeon.Draw()
		mobs.DrawMobs()
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
		mobs.SpawnMobs(8, "random")

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

	rl.InitAudioDevice()

	if _, err := os.Stat("assets/audio/world.mp3"); err == nil {
		worldMusic = rl.LoadMusicStream("assets/audio/world.mp3")
		rl.SetMusicVolume(worldMusic, 0.2)
	}
	if _, err := os.Stat("assets/audio/dungeon.mp3"); err == nil {
		dungeonMusic = rl.LoadMusicStream("assets/audio/dungeon.mp3")
		rl.SetMusicVolume(dungeonMusic, 0.6)
	}
	if _, err := os.Stat("assets/audio/boss.mp3"); err == nil {
		bossMusic = rl.LoadMusicStream("assets/audio/boss.mp3")
		rl.SetMusicVolume(bossMusic, 0.6)
	}

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

	playTrack("world")

	ui.InitMenu("assets/ui/map.png")
}

func input() {
	if rl.IsKeyPressed(rl.KeyF10) {
		rl.ToggleBorderlessWindowed()
	}

	if rl.IsKeyPressed(rl.KeyF7) {
		musicPaused = !musicPaused
		if musicPaused {
			pauseCurrentMusic()
		} else {
			resumeCurrentMusic()
		}
	}

	if menuOpen {
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyEscape) {
			menuOpen = false
			if menuPausedMusic {
				resumeCurrentMusic()
				menuPausedMusic = false
			}
		}
	} else {
		if rl.IsKeyPressed(rl.KeyEscape) {
			menuOpen = true
			if !musicPaused {
				pauseCurrentMusic()
				menuPausedMusic = true
			}
		}
	}

	if !menuOpen {
		player.PlayerInput()
	}

	if rl.IsKeyPressed(rl.KeyF3) {
		printDebug = !printDebug
	}

	if !menuOpen && inDungeon && rl.IsKeyPressed(rl.KeyBackspace) {
		exitDungeon()
	}
	if !menuOpen && inBoss && rl.IsKeyPressed(rl.KeyBackspace) {
		exitBoss()
	}
}

func update() {
	running = !rl.WindowShouldClose()

	updateCurrentMusic()

	if menuOpen {
		return
	}

	if !inDungeon && !inBoss {
		world.LightLamps()
		world.LightPumpkinLamps()
	}

	if player.IsPlayerDead() {
		player.PlayerMoving()
		if player.HasPlayerDeathAnimationFinished() {
			if inDungeon {
				exitDungeon()
			} else if inBoss {
				exitBoss()
			}
			dungeonsCleared = 0
			dungeonSpawnCount = 0

			mobs.ResetMobs()
			player.ResetPlayer()
		}
		return
	}

	player.PlayerMoving()

	playerPos := rl.NewVector2(player.PlayerHitBox.X, player.PlayerHitBox.Y)
	attackPlayerFunc := func() {
		player.SetPlayerDamageState()
		player.TakeDamage(0.1)
	}
	if inDungeon {
		mobs.MobMoving(playerPos, attackPlayerFunc)
	} else if !inBoss {
		mobs.MobMoving(playerPos, attackPlayerFunc)
	}

	if mobs.IsMobAlive() {
		closestMobIndex := mobs.GetClosestMobIndex(playerPos)
		if closestMobIndex != -1 {
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

		if mobs.IsMobAlive() {
			dungeon.HideExit()
			exitSoundPlayed = false
			mobsClearFrames = 0
		} else {
			if mobsClearFrames < 6 {
				mobsClearFrames++
			}
			if mobsClearFrames >= 6 {
				dungeon.ShowExit()
				if !exitSoundPlayed {
					world.PlayDoorOpenSound()
					exitSoundPlayed = true
				}
			}
		}

		if exitCooldownFrames <= 0 && !mobs.IsMobAlive() && dungeon.IsPlayerAtExit(player.PlayerHitBox) {
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

	if menuOpen {
		ui.DrawMenuOverlay()
	}

	rl.EndDrawing()
}

func quit() {
	stopAllTracks()
	if worldMusic.CtxType != 0 {
		rl.UnloadMusicStream(worldMusic)
	}
	if dungeonMusic.CtxType != 0 {
		rl.UnloadMusicStream(dungeonMusic)
	}
	if bossMusic.CtxType != 0 {
		rl.UnloadMusicStream(bossMusic)
	}
	rl.CloseAudioDevice()
	player.UnloadPlayerTexture()
	world.UnloadWorldTexture()
	world.UnloadDoorsTextures()
	world.UnloadPumpkinLamps()
	mobs.UnloadMobsTexture()
	dungeon.Unload()
	ui.UnloadMenu()

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

func playTrack(which string) {
	if currentMusic == which {
		return
	}
	stopAllTracks()
	switch which {
	case "dungeon":
		if dungeonMusic.CtxType != 0 {
			rl.PlayMusicStream(dungeonMusic)
			if musicPaused {
				rl.PauseMusicStream(dungeonMusic)
			}
		}
	case "boss":
		if bossMusic.CtxType != 0 {
			rl.PlayMusicStream(bossMusic)
			if musicPaused {
				rl.PauseMusicStream(bossMusic)
			}
		}
	default:
		if worldMusic.CtxType != 0 {
			rl.PlayMusicStream(worldMusic)
			if musicPaused {
				rl.PauseMusicStream(worldMusic)
			}
		}
		which = "world"
	}
	currentMusic = which
}

func stopAllTracks() {
	if worldMusic.CtxType != 0 {
		rl.StopMusicStream(worldMusic)
	}
	if dungeonMusic.CtxType != 0 {
		rl.StopMusicStream(dungeonMusic)
	}
	if bossMusic.CtxType != 0 {
		rl.StopMusicStream(bossMusic)
	}
}

func updateCurrentMusic() {
	switch currentMusic {
	case "dungeon":
		if dungeonMusic.CtxType != 0 {
			rl.UpdateMusicStream(dungeonMusic)
		}
	case "boss":
		if bossMusic.CtxType != 0 {
			rl.UpdateMusicStream(bossMusic)
		}
	case "world":
		if worldMusic.CtxType != 0 {
			rl.UpdateMusicStream(worldMusic)
		}
	}
}

func pauseCurrentMusic() {
	switch currentMusic {
	case "dungeon":
		if dungeonMusic.CtxType != 0 {
			rl.PauseMusicStream(dungeonMusic)
		}
	case "boss":
		if bossMusic.CtxType != 0 {
			rl.PauseMusicStream(bossMusic)
		}
	case "world":
		if worldMusic.CtxType != 0 {
			rl.PauseMusicStream(worldMusic)
		}
	}
}

func resumeCurrentMusic() {
	switch currentMusic {
	case "dungeon":
		if dungeonMusic.CtxType != 0 {
			rl.ResumeMusicStream(dungeonMusic)
		}
	case "boss":
		if bossMusic.CtxType != 0 {
			rl.ResumeMusicStream(bossMusic)
		}
	case "world":
		if worldMusic.CtxType != 0 {
			rl.ResumeMusicStream(worldMusic)
		}
	}
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
	mobs.SetExternalColliders(dungeon.GetColliders())
	spawn := dungeon.GetSpawnPosition()
	player.SetPosition(spawn.X, spawn.Y)
	exitCooldownFrames = exitCooldownFramesDefault

	baseMin, baseMax := dungeonSpawnBaseMin, dungeonSpawnBaseMax
	if dungeonSpawnCount == 0 {
		dungeonSpawnCount = baseMin + int(rl.GetRandomValue(0, int32(baseMax-baseMin)))
	} else {
		inc := int(rl.GetRandomValue(2, 5))
		dungeonSpawnCount += inc
		if dungeonSpawnCount > baseMax+dungeonSpawnBaseMax {
			dungeonSpawnCount = baseMax + dungeonSpawnBaseMax
		}
	}

	positions := dungeon.GetRandomFloorPositions(dungeonSpawnCount)
	mobs.ResetMobs()
	mobs.SpawnMobsAtPositions(positions, "random")

	exitSoundPlayed = false
	mobsClearFrames = 0

	playTrack("dungeon")
}

func exitDungeon() {
	if !inDungeon {
		return
	}
	inDungeon = false
	player.ClearExternalColliders()
	mobs.ClearExternalColliders()
	player.SetPosition(savedWorldPos.X, savedWorldPos.Y)
	mobs.ResetMobs()
	playTrack("world")
}

func nextDungeon() {
	dungeon.Generate()
	player.SetExternalColliders(dungeon.GetColliders())
	mobs.SetExternalColliders(dungeon.GetColliders())
	spawn := dungeon.GetSpawnPosition()
	player.SetPosition(spawn.X, spawn.Y)
	exitCooldownFrames = exitCooldownFramesDefault

	baseMin, baseMax := dungeonSpawnBaseMin, dungeonSpawnBaseMax
	if dungeonSpawnCount == 0 {
		dungeonSpawnCount = baseMin + int(rl.GetRandomValue(0, int32(baseMax-baseMin)))
	} else {
		inc := int(rl.GetRandomValue(2, 5))
		dungeonSpawnCount += inc
		if dungeonSpawnCount > baseMax+dungeonSpawnBaseMax {
			dungeonSpawnCount = baseMax + dungeonSpawnBaseMax
		}
	}

	positions := dungeon.GetRandomFloorPositions(dungeonSpawnCount)
	mobs.ResetMobs()
	mobs.SpawnMobsAtPositions(positions, "random")

	exitSoundPlayed = false
	mobsClearFrames = 0

	playTrack("dungeon")
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

	playTrack("boss")
}

func exitBoss() {
	if !inBoss {
		return
	}
	inBoss = false

	player.SetPosition(savedWorldPos.X, savedWorldPos.Y)
	playTrack("world")
}
