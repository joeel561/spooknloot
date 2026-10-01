package main

import (
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/player"
	"spooknloot/pkg/sim"
	"spooknloot/pkg/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// selectedClass is picked on the title screen.
var selectedClass = sim.ClassWarrior

// localClass is the class the local player plays this game.
func localClass() sim.Class {
	if gameLobby != nil {
		if p, ok := gameLobby.LocalPlayer(); ok {
			return p.Class
		}
	}
	return selectedClass
}

// classOf returns any player's class.
func classOf(id netcode.PeerID) sim.Class {
	if gameLobby != nil {
		for _, p := range gameLobby.Players {
			if p.ID == id {
				return p.Class
			}
		}
	}
	return selectedClass
}

func applyClass(c sim.Class) {
	st := c.Stats()
	player.SetClass(st.MaxHealth, st.Range, ui.ClassColor(c))
}

// Mages shoot a short magic bolt at the mob they hit.

const boltFrames = 10

type bolt struct {
	from, to rl.Vector2
	frames   int
}

var bolts []bolt

func fireBolt(to rl.Vector2) {
	bolts = append(bolts, bolt{from: playerCenter(), to: to, frames: boltFrames})
}

func updateBolts() {
	kept := bolts[:0]
	for _, b := range bolts {
		if b.frames--; b.frames > 0 {
			kept = append(kept, b)
		}
	}
	bolts = kept
}

// drawBolts draws the bolts in world space, fading out.
func drawBolts() {
	for _, b := range bolts {
		alpha := uint8(255 * b.frames / boltFrames)
		rl.DrawLineEx(b.from, b.to, 1.5, rl.NewColor(150, 190, 255, alpha))
		rl.DrawCircleV(b.to, 3, rl.NewColor(210, 230, 255, alpha))
	}
}
