package main

import (
	"os"
	assetpack "spooknloot"
	"spooknloot/pkg/boss"
	"spooknloot/pkg/coop"
	"spooknloot/pkg/debug"
	"spooknloot/pkg/dungeon"
	"spooknloot/pkg/lobby"
	"spooknloot/pkg/mobs"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/player"
	"spooknloot/pkg/ui"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1500
	screenHeight = 900
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

	menuOpen        bool
	menuPausedMusic bool

	currentScene  = sceneTitle
	nameInput     = ui.TextInput{MaxLen: lobby.MaxNameLength}
	addressInput  = ui.TextInput{MaxLen: 64}
	statusMessage string
	lanBrowser    *netcode.Browser
	gameLobby     *lobby.Lobby
	session       *coop.Session

	bossWinOpen bool
)

type scene int

const (
	sceneTitle scene = iota
	sceneJoin
	sceneLobby
	scenePlaying
)

func drawScene() {
	var mobViews []coop.MobView
	if session != nil {
		mobViews = session.Mobs()
	}

	switch currentArea() {
	case lobby.AreaBoss:
		boss.Draw()
		mobs.DrawMobs(mobViews)
		dungeon.DrawPotions(session.Potions)
	case lobby.AreaDungeon:
		dungeon.SetExitVisible(session.ExitOpen)
		dungeon.Draw()
		dungeon.DrawPotions(session.Potions)
		mobs.DrawMobs(mobViews)
	default:
		world.DrawWorld()
		world.DrawBottomLamp()
		world.DrawDoors()
		mobs.DrawMobs(mobViews)
		world.DrawPumpkinLamp()
	}

	drawRemotePlayers()
	player.DrawPlayerTexture()
	drawBolts()

	if currentArea() == lobby.AreaWorld {
		world.DrawWheat()
		world.DrawTopLamp()
		world.DrawCauldron()
	}

	if printDebug {
		debug.DrawPlayerOutlines()
	}
}

func init() {

	// Prepare embedded assets (for single-file distribution). This extracts
	// assets to a temporary directory and switches CWD so existing relative
	// file paths continue to work unchanged.
	_, _, _ = assetpack.Prepare()

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

	nameInput.Text = lobby.SanitizeName(os.Getenv("USERNAME"))

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

	// Title, join and lobby screens handle their own input while drawing.
	if currentScene != scenePlaying {
		return
	}

	if rl.IsKeyPressed(rl.KeyB) {
		session.DebugBoss()
	}

	if menuOpen {
		if rl.IsKeyPressed(rl.KeyQ) {
			returnToTitle("")
			return
		}
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyEscape) {
			menuOpen = false
			if menuPausedMusic {
				resumeCurrentMusic()
				menuPausedMusic = false
			}
		}
	} else if !bossWinOpen {
		if rl.IsKeyPressed(rl.KeyEscape) {
			menuOpen = true
			if !musicPaused {
				pauseCurrentMusic()
				menuPausedMusic = true
			}
		}
	}

	if bossWinOpen {
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			mp := rl.GetMousePosition()
			b := ui.GetBossWinButtonRect()
			if mp.X >= b.X && mp.X <= b.X+b.Width && mp.Y >= b.Y && mp.Y <= b.Y+b.Height {
				bossWinOpen = false
			}
		}
		return
	}

	if !menuOpen {
		player.PlayerInput()
		if rl.IsKeyPressed(rl.KeyJ) && !player.IsPlayerDead() {
			session.AcceptInvite()
		}
	}

	if rl.IsKeyPressed(rl.KeyF3) {
		printDebug = !printDebug
	}
}

func update() {
	running = !rl.WindowShouldClose()

	updateCurrentMusic()
	updateLobby()

	if currentScene != scenePlaying {
		return
	}
	updateGame()
}

func render() {
	var cam = player.Cam

	rl.BeginDrawing()
	if currentArea() == lobby.AreaWorld {
		rl.ClearBackground(worldBgColor)
	} else {
		rl.ClearBackground(dungeonBgColor)
	}
	collectVisibleRemotes()
	rl.BeginMode2D(cam)

	drawScene()
	rl.EndMode2D()

	drawRemoteNames()

	if currentScene == scenePlaying {
		player.DrawHealthBar()
		if gameLobby != nil {
			ui.DrawMultiplayerHUD(gameLobby)
		}
		if inv := session.Invite; inv != nil && currentArea() == lobby.AreaWorld {
			ui.DrawInvite(inv.From, inv.Until)
		}
		drawDownedBanner()
		if currentArea() == lobby.AreaDungeon {
			ui.DrawRunProgress(session.Level, coop.DungeonLevels)
		}
		rows := scoreRows()
		ui.DrawLiveRanking(rows)
		if rl.IsKeyDown(rl.KeyTab) && !menuOpen {
			ui.DrawScoreboard(rows)
		}
	}

	if bossWinOpen {
		ui.DrawBossWinOverlay()
	}

	if session != nil && currentArea() == lobby.AreaBoss {
		if health, ok := session.Boss(); ok && health > 0 {
			drawBossHealthBar(float32(health) / 100)
		}
	}

	if printDebug {
		debug.DrawDebug(debug.DebugText())
	}

	if menuOpen {
		ui.DrawMenuOverlay()
	}

	drawMenuScreens()

	if devFrameHook != nil {
		devFrameHook()
	}

	rl.EndDrawing()
}

// devFrameHook runs at the end of every rendered frame when set. Local test
// scripts use it to drive the game and take screenshots.
var devFrameHook func()

func drawBossHealthBar(percent float32) {
	barW := float32(480)
	barH := float32(20)
	barX := float32(rl.GetScreenWidth())/2 - barW/2
	barY := float32(16)
	bg := rl.NewRectangle(barX, barY, barW, barH)
	fg := rl.NewRectangle(barX+2, barY+2, (barW-4)*percent, barH-4)
	rl.DrawRectangleRec(bg, rl.NewColor(0, 0, 0, 200))
	color := rl.Color{R: 190, G: 75, B: 75, A: 255}
	if percent <= 0.2 {
		color = rl.Color{R: 57, G: 108, B: 60, A: 255}
	} else if percent <= 0.5 {
		color = rl.Color{R: 231, G: 152, B: 50, A: 255}
	}
	rl.DrawRectangleRec(fg, color)
}

func quit() {
	if gameLobby != nil {
		gameLobby.Leave()
	}
	stopLANBrowser()
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
