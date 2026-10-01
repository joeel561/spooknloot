package ui

import (
	"fmt"

	"spooknloot/pkg/dungeon"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	coinColor     = rl.NewColor(240, 196, 64, 255)
	coinEdgeColor = rl.NewColor(160, 110, 30, 255)
)

// DrawCoin draws a gold coin centered at (cx, cy).
func DrawCoin(cx, cy, radius float32) {
	rl.DrawCircleV(rl.NewVector2(cx, cy), radius, coinEdgeColor)
	rl.DrawCircleV(rl.NewVector2(cx, cy), radius*0.75, coinColor)
}

// DrawInventoryHUD shows gold and potions in the bottom left corner.
func DrawInventoryHUD(gold, potions int) {
	x := float32(20)
	y := float32(rl.GetScreenHeight()) - 56
	rl.DrawRectangle(int32(x)-8, int32(y)-8, 250, 48, rl.NewColor(0, 0, 0, 140))
	DrawCoin(x+14, y+16, 11)
	rl.DrawText(fmt.Sprint(gold), int32(x)+32, int32(y)+6, 22, rl.RayWhite)
	dungeon.DrawPotionIcon(rl.NewRectangle(x+100, y-2, 36, 36))
	rl.DrawText(fmt.Sprintf("%d  [1] drink", potions), int32(x)+138, int32(y)+6, 18, rl.RayWhite)
}

const (
	invCols, invRows = 4, 2
	invSlot          = float32(72)
	invGap           = float32(10)
)

// DrawInventory draws the inventory window (I key). It returns true when
// the player clicked the potion slot to drink one.
func DrawInventory(gold, potions, maxPotions int) bool {
	w := invCols*invSlot + (invCols-1)*invGap + 60
	h := invRows*invSlot + (invRows-1)*invGap + 150
	cx, cy := screenCenter()
	panel := rl.NewRectangle(cx-w/2, cy-h/2, w, h)
	drawPanel(panel)
	drawLabelCentered("INVENTORY", cx, panel.Y+18, 34, rl.RayWhite)

	slot := func(i int) rl.Rectangle {
		col, row := float32(i%invCols), float32(i/invCols)
		return rl.NewRectangle(panel.X+30+col*(invSlot+invGap), panel.Y+70+row*(invSlot+invGap), invSlot, invSlot)
	}
	for i := 0; i < invCols*invRows; i++ {
		r := slot(i)
		rl.DrawRectangleRec(r, inputColor)
		rl.DrawRectangleLinesEx(r, 2, panelBorderColor)
	}

	// Slot 0: gold, slot 1: potions. More item kinds get the next slots.
	g := slot(0)
	DrawCoin(g.X+g.Width/2, g.Y+g.Height/2-6, 16)
	drawSlotCount(g, fmt.Sprint(gold))

	used := false
	if potions > 0 {
		p := slot(1)
		if hovered(p) {
			rl.DrawRectangleLinesEx(p, 2, accentColor)
			used = rl.IsMouseButtonPressed(rl.MouseLeftButton)
		}
		dungeon.DrawPotionIcon(rl.NewRectangle(p.X+12, p.Y+4, p.Width-24, p.Height-24))
		drawSlotCount(p, fmt.Sprintf("%d/%d", potions, maxPotions))
	}

	for i, hint := range []string{"Click a potion or press 1 to drink it", "Press I to close"} {
		hw := rl.MeasureText(hint, 16)
		rl.DrawText(hint, int32(cx)-hw/2, int32(panel.Y+panel.Height)-52+int32(i)*22, 16, mutedTextColor)
	}
	return used
}

func drawSlotCount(r rl.Rectangle, text string) {
	w := rl.MeasureText(text, 16)
	rl.DrawText(text, int32(r.X+r.Width)-w-6, int32(r.Y+r.Height)-20, 16, rl.RayWhite)
}

// DrawToast shows a short message above the inventory bar; alpha fades
// it out.
func DrawToast(text string, alpha float32) {
	if alpha < 0 {
		alpha = 0
	}
	a := uint8(255 * min(alpha, 1))
	x := int32(20)
	y := int32(rl.GetScreenHeight()) - 96
	w := rl.MeasureText(text, 20)
	rl.DrawRectangle(x-8, y-6, w+16, 32, rl.NewColor(0, 0, 0, a/2))
	rl.DrawText(text, x, y, 20, rl.NewColor(240, 196, 64, a))
}
