package main

import (
	"math"
	"time"

	"spooknloot/pkg/boss"
	"spooknloot/pkg/coop"
	"spooknloot/pkg/dungeon"
	"spooknloot/pkg/lobby"
	"spooknloot/pkg/mobs"
	"spooknloot/pkg/player"
	"spooknloot/pkg/sim"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	worldSpawnX, worldSpawnY = 495, 344
)

var (
	bossPlayerSpawn = sim.Vec2{X: 548, Y: 285}
	bossMobSpawn    = sim.Vec2{X: 548, Y: 200}
)

func currentArea() lobby.Area {
	if session == nil {
		return lobby.AreaWorld
	}
	return session.Area
}

// startSession begins the shared game: over the lobby connection in
// multiplayer, or as a local host in singleplayer.
func startSession() {
	if gameLobby != nil {
		session = coop.NewSession(coop.LobbyNet{L: gameLobby}, coopConfig(), gameLobby.Seed)
	} else {
		session = coop.NewSession(coop.LocalNet{PlayerClass: selectedClass}, coopConfig(), time.Now().UnixNano())
	}
}

// coopConfig collects the static map data the host simulation needs.
func coopConfig() coop.Config {
	var cfg coop.Config
	tileRects := func(tiles []world.Tile, size int) []sim.Rect {
		out := make([]sim.Rect, len(tiles))
		for i, t := range tiles {
			out[i] = sim.Rect{X: float32(t.X * size), Y: float32(t.Y * size), W: float32(size), H: float32(size)}
		}
		return out
	}

	ts := world.WorldMap.TileSize
	for _, l := range world.WorldMap.Layers {
		switch l.Name {
		case "out", "fence", "buildings", "trees", "bushes", "market":
			cfg.WorldColliders = append(cfg.WorldColliders, tileRects(l.Tiles, ts)...)
		case "spawn":
			for _, t := range l.Tiles {
				cfg.WorldMobSpawns = append(cfg.WorldMobSpawns, sim.Vec2{X: float32(t.X * ts), Y: float32(t.Y * ts)})
			}
		}
	}
	// Lamps are placed in pixels, with a 16x16 base like the player collision.
	for _, l := range world.Lamps {
		cfg.WorldColliders = append(cfg.WorldColliders, sim.Rect{X: float32(l.X), Y: float32(l.Y), W: 16, H: 16})
	}

	for _, r := range boss.GetColliders() {
		cfg.BossWalls = append(cfg.BossWalls, sim.Rect{X: r.X, Y: r.Y, W: r.Width, H: r.Height})
	}
	bts := boss.BossMap.TileSize
	for _, l := range boss.BossMap.Layers {
		if l.Name == "Floor" {
			for _, t := range l.Tiles {
				cfg.BossFloor = append(cfg.BossFloor, sim.Vec2{X: float32(t.X * bts), Y: float32(t.Y * bts)})
			}
		}
	}
	cfg.BossPlayerSpawn = bossPlayerSpawn
	cfg.BossSpawn = bossMobSpawn
	return cfg
}

func localPlayerState() lobby.PlayerState {
	x, y, dir, frame := player.Appearance()
	return lobby.PlayerState{
		X:     x,
		Y:     y,
		Dir:   uint8(dir),
		Frame: uint8(frame),
		Area:  currentArea(),
		// Round up: 0 must mean "down", not "almost dead".
		Health: uint8(math.Ceil(float64(player.GetCurrentHealth() / player.GetMaxHealth() * 100))),
	}
}

func playerCenter() rl.Vector2 {
	return rl.NewVector2(player.PlayerHitBox.X+player.PlayerHitBox.Width/2, player.PlayerHitBox.Y+player.PlayerHitBox.Height/2)
}

func updateGame() {
	session.SetLocalState(localPlayerState())
	// Singleplayer pauses while the menu is open; with others the world
	// keeps running for everyone.
	if !(menuOpen && gameLobby == nil) {
		session.Update()
	}
	handleSessionEvents()
	updateBolts()

	if menuOpen {
		return
	}

	if currentArea() == lobby.AreaWorld {
		world.LightLamps()
		world.LightPumpkinLamps()
	}

	if player.IsPlayerDead() {
		player.PlayerMoving()
		session.SetReviving(0)
		// In a dungeon run you stay down until a teammate revives you, you
		// bleed out, or the whole group is down (the host decides).
		if currentArea() == lobby.AreaWorld && player.HasPlayerDeathAnimationFinished() {
			enterWorld()
			player.ResetPlayer()
		}
		return
	}

	player.PlayerMoving()
	updateReviving()

	if m, ok := mobs.Closest(session.Mobs(), playerCenter()); ok {
		c := m.Center()
		player.TryAttack(rl.NewVector2(c.X, c.Y), func(float32) {
			session.Attack(m.ID)
			if localClass() == sim.ClassMage {
				fireBolt(rl.NewVector2(c.X, c.Y))
			}
		})
	}

	switch currentArea() {
	case lobby.AreaWorld:
		if rl.CheckCollisionRecs(player.PlayerHitBox, world.HouseDoorDest) {
			session.EnterDungeon()
		}
	case lobby.AreaDungeon:
		if dungeon.IsPlayerAtExit(player.PlayerHitBox) {
			session.ReachedExit()
		}
	}
}

func handleSessionEvents() {
	for _, ev := range session.TakeEvents() {
		switch ev.Kind {
		case coop.EventEnterArea:
			// Players who were down come back on the next level.
			if player.IsPlayerDead() {
				player.Revive(coop.RespawnHealth * player.GetMaxHealth())
			}
			enterArea(ev.Spawn)
		case coop.EventRevived:
			player.Revive(ev.Amount * player.GetMaxHealth())
		case coop.EventLoot:
			addLootToast(ev.Item, ev.Count)
		case coop.EventWipe:
			enterWorld()
			player.ResetPlayer()
		case coop.EventDamage:
			if !player.IsPlayerDead() {
				player.SetPlayerDamageState()
				player.TakeDamage(ev.Amount)
			}
		case coop.EventHeal:
			missing := player.GetMaxHealth() - player.GetCurrentHealth()
			if heal := min(ev.Amount*player.GetMaxHealth(), missing); heal > 0 {
				player.TakeDamage(-heal)
			}
			if !ev.Quiet {
				dungeon.PlayDrinkSound()
			}
		case coop.EventExitOpened:
			world.PlayDoorOpenSound()
		case coop.EventRunComplete:
			enterWorld()
			bossWinOpen = true
		}
	}
}

// enterArea moves the local player into the area the session switched to.
func enterArea(spawn sim.Vec2) {
	switch session.Area {
	case lobby.AreaDungeon:
		dungeon.SetLayout(session.Layout)
		player.SetExternalColliders(dungeon.GetColliders())
		playTrack("dungeon")
	case lobby.AreaBoss:
		player.SetExternalColliders(boss.GetColliders())
		playTrack("boss")
	}
	player.SetPosition(spawn.X, spawn.Y)
}

func enterWorld() {
	player.ClearExternalColliders()
	player.SetPosition(worldSpawnX, worldSpawnY)
	playTrack("world")
}
