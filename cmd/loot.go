package main

import (
	"fmt"
	"math"
	"time"

	"spooknloot/pkg/coop"
	"spooknloot/pkg/dungeon"
	"spooknloot/pkg/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const toastDuration = 2 * time.Second

var (
	inventoryOpen bool
	toasts        []toast
)

type toast struct {
	text  string
	until time.Time
}

func addLootToast(item coop.ItemKind, count int) {
	text := fmt.Sprintf("+%d gold", count)
	if item == coop.ItemPotion {
		text = fmt.Sprintf("+%d potion", count)
		if count > 1 {
			text += "s"
		}
	}
	toasts = append(toasts, toast{text: text, until: time.Now().Add(toastDuration)})
	if len(toasts) > 3 {
		toasts = toasts[len(toasts)-3:]
	}
}

// drawDrops draws loot on the floor in world space; coins bob gently.
func drawDrops() {
	if session == nil {
		return
	}
	for _, d := range session.Drops {
		bob := float32(math.Sin(rl.GetTime()*4+float64(d.ID))) * 1.5
		switch d.Kind {
		case coop.ItemPotion:
			dungeon.DrawPotionIcon(rl.NewRectangle(d.Pos.X-6, d.Pos.Y-6+bob, 12, 12))
		default:
			ui.DrawCoin(d.Pos.X, d.Pos.Y+bob, 3)
		}
	}
}

// drawLootHUD draws the inventory bar, pickup messages and, if open, the
// inventory window.
func drawLootHUD() {
	ui.DrawInventoryHUD(session.Gold, session.PotionCount)

	now := time.Now()
	kept := toasts[:0]
	for _, t := range toasts {
		if now.Before(t.until) {
			kept = append(kept, t)
		}
	}
	toasts = kept
	if len(toasts) > 0 {
		t := toasts[len(toasts)-1]
		ui.DrawToast(t.text, float32(t.until.Sub(now))/float32(500*time.Millisecond))
	}

	if inventoryOpen && ui.DrawInventory(session.Gold, session.PotionCount, coop.MaxPotions) {
		session.UsePotion()
	}
}
