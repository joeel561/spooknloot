package main

import (
	"fmt"
	"time"

	"spooknloot/pkg/coop"
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

// collectVisibleRemotes picks the players to draw this frame: everyone in
// the same area (there is only one dungeon run at a time).
func collectVisibleRemotes() {
	visibleRemotes = visibleRemotes[:0]
	if gameLobby == nil || currentScene != scenePlaying {
		return
	}
	for _, s := range gameLobby.RemoteStates(remoteInterpolationDelay) {
		if s.Area == currentArea() {
			visibleRemotes = append(visibleRemotes, s)
		}
	}
}

// drawRemotePlayers draws the other characters in world space.
func drawRemotePlayers() {
	for _, s := range visibleRemotes {
		tint := rl.White
		switch {
		case session.Statuses[s.ID].State == coop.BledOut:
			tint = rl.DarkGray
		case s.Health == 0:
			tint = rl.Gray
		}
		player.DrawCharacter(s.X, s.Y, player.Direction(s.Dir), int(s.Frame), tint)
	}
}

func remoteCenter(s lobby.PlayerState) rl.Vector2 { return rl.NewVector2(s.X+24, s.Y+30) }

// reviveCandidate returns the closest downed teammate within reach.
func reviveCandidate() (netcode.PeerID, bool) {
	if gameLobby == nil || currentArea() == lobby.AreaWorld || player.IsPlayerDead() {
		return 0, false
	}
	best, bestDist := netcode.PeerID(0), float32(coop.ReviveReach)
	for _, s := range gameLobby.RemoteStates(remoteInterpolationDelay) {
		if s.Area != currentArea() || session.Statuses[s.ID].State != coop.Downed {
			continue
		}
		if d := rl.Vector2Distance(playerCenter(), remoteCenter(s)); d <= bestDist {
			best, bestDist = s.ID, d
		}
	}
	return best, best != 0
}

// updateReviving revives the closest downed teammate while E is held.
func updateReviving() {
	target, ok := reviveCandidate()
	if ok && rl.IsKeyDown(rl.KeyE) {
		session.SetReviving(target)
	} else {
		session.SetReviving(0)
	}
}

// drawRemoteNames draws name tags in screen space, so the text stays sharp
// regardless of the camera zoom. Downed players also get revive info.
func drawRemoteNames() {
	if len(visibleRemotes) == 0 {
		return
	}
	candidate, _ := reviveCandidate()
	const size = 18
	for _, s := range visibleRemotes {
		name := gameLobby.PlayerName(s.ID)
		head := rl.GetWorldToScreen2D(rl.NewVector2(s.X+24, s.Y+12), player.Cam)
		w := rl.MeasureText(name, size)
		x, y := int32(head.X)-w/2, int32(head.Y)-size-4
		rl.DrawRectangle(x-4, y-2, w+8, size+4, rl.NewColor(0, 0, 0, 140))
		rl.DrawText(name, x, y, size, rl.RayWhite)

		st := session.Statuses[s.ID]
		below := int32(head.Y) + 70
		switch {
		case st.State == coop.BledOut:
			drawTag("bled out", int32(head.X), below, rl.LightGray)
		case st.State == coop.Downed && st.Reviver != 0:
			progress := float32(time.Since(st.ReviveStart).Seconds()) / (coop.ReviveTime / 60.0)
			drawProgressBar(int32(head.X), below, min(progress, 1))
		case st.State == coop.Downed && s.ID == candidate:
			drawTag("Hold E to revive", int32(head.X), below, rl.NewColor(231, 152, 50, 255))
		case st.State == coop.Downed:
			drawTag("down!", int32(head.X), below, rl.NewColor(230, 100, 100, 255))
		}
	}
}

func drawTag(text string, cx, y int32, color rl.Color) {
	const size = 18
	w := rl.MeasureText(text, size)
	rl.DrawRectangle(cx-w/2-4, y-2, w+8, size+4, rl.NewColor(0, 0, 0, 160))
	rl.DrawText(text, cx-w/2, y, size, color)
}

func drawProgressBar(cx, y int32, progress float32) {
	const w, h = 80, 10
	rl.DrawRectangle(cx-w/2-2, y-2, w+4, h+4, rl.NewColor(0, 0, 0, 180))
	rl.DrawRectangle(cx-w/2, y, int32(w*progress), h, rl.NewColor(120, 190, 110, 255))
}

// drawDownedBanner tells the local player that they are down or bled out.
func drawDownedBanner() {
	if gameLobby == nil || session == nil {
		return
	}
	st := session.LocalStatus()
	var text string
	switch st.State {
	case coop.Downed:
		if st.Reviver != 0 {
			text = gameLobby.PlayerName(st.Reviver) + " is reviving you..."
		} else {
			left := coop.BleedOutTime/60 - int(time.Since(st.Since).Seconds())
			text = fmt.Sprintf("You are down! A teammate can revive you (%ds)", max(left, 0))
		}
	case coop.BledOut:
		text = "You bled out. You'll be back on the next level."
	default:
		return
	}
	const size = 26
	w := rl.MeasureText(text, size)
	x := int32(rl.GetScreenWidth())/2 - w/2
	y := int32(rl.GetScreenHeight()) - 90
	rl.DrawRectangle(x-16, y-10, w+32, size+20, rl.NewColor(0, 0, 0, 190))
	rl.DrawText(text, x, y, size, rl.RayWhite)
}
