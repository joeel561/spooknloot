package main

import (
	"time"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/player"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Remote players are shown this far in the past so there are always two
// snapshots to interpolate between.
const remoteInterpolationDelay = 100 * time.Millisecond

// visibleRemotes holds the other players drawn this frame.
var visibleRemotes []lobby.PlayerState

func currentArea() lobby.Area {
	switch {
	case inBoss:
		return lobby.AreaBoss
	case inDungeon:
		return lobby.AreaDungeon
	default:
		return lobby.AreaWorld
	}
}

func sendLocalState() {
	x, y, dir, frame := player.Appearance()
	health := player.GetCurrentHealth() / player.GetMaxHealth() * 100
	gameLobby.SetLocalState(lobby.PlayerState{
		X:      x,
		Y:      y,
		Dir:    uint8(dir),
		Frame:  uint8(frame),
		Area:   currentArea(),
		Health: uint8(health),
	})
}

// collectVisibleRemotes picks the players to draw this frame. Dungeons and
// the boss room are still separate per player, so others are only shown
// in the world for now.
func collectVisibleRemotes() {
	visibleRemotes = visibleRemotes[:0]
	if gameLobby == nil || currentScene != scenePlaying || currentArea() != lobby.AreaWorld {
		return
	}
	for _, s := range gameLobby.RemoteStates(remoteInterpolationDelay) {
		if s.Area == lobby.AreaWorld {
			visibleRemotes = append(visibleRemotes, s)
		}
	}
}

// drawRemotePlayers draws the other characters in world space.
func drawRemotePlayers() {
	for _, s := range visibleRemotes {
		tint := rl.White
		if s.Health == 0 {
			tint = rl.Gray
		}
		player.DrawCharacter(s.X, s.Y, player.Direction(s.Dir), int(s.Frame), tint)
	}
}

// drawRemoteNames draws name tags in screen space, so the text stays sharp
// regardless of the camera zoom.
func drawRemoteNames() {
	if len(visibleRemotes) == 0 {
		return
	}
	names := make(map[netcode.PeerID]string, len(gameLobby.Players))
	for _, p := range gameLobby.Players {
		names[p.ID] = p.Name
	}
	const size = 18
	for _, s := range visibleRemotes {
		name := names[s.ID]
		if name == "" {
			continue
		}
		head := rl.GetWorldToScreen2D(rl.NewVector2(s.X+24, s.Y+12), player.Cam)
		w := rl.MeasureText(name, size)
		x, y := int32(head.X)-w/2, int32(head.Y)-size-4
		rl.DrawRectangle(x-4, y-2, w+8, size+4, rl.NewColor(0, 0, 0, 140))
		rl.DrawText(name, x, y, size, rl.RayWhite)
	}
}
