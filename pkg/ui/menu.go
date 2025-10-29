package ui

import (
	"os"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	menuTex          rl.Texture2D
	menuTexLoaded    bool
	menuFont         rl.Font
	menuFontLoaded   bool
	menuTitleOffsetY float32
)

func InitMenu(texturePath string) {
	menuTexLoaded = false
	menuFontLoaded = false
	if _, err := os.Stat(texturePath); err == nil {
		t := rl.LoadTexture(texturePath)
		if t.ID != 0 {
			menuTex = t
			menuTexLoaded = true
		}
	}

	const defaultFontPath = "assets/ui/FantasyRPGtext.ttf"
	if _, err := os.Stat(defaultFontPath); err == nil {
		f := rl.LoadFontEx(defaultFontPath, 64, nil, 0)
		if f.BaseSize != 0 {
			menuFont = f
			menuFontLoaded = true
		}
	}
}

func UnloadMenu() {
	if menuTexLoaded && menuTex.ID != 0 {
		rl.UnloadTexture(menuTex)
		menuTexLoaded = false
	}
	if menuFontLoaded && menuFont.BaseSize != 0 {
		rl.UnloadFont(menuFont)
		menuFontLoaded = false
	}
}

func SetMenuTitleOffsetY(offset float32) {
	menuTitleOffsetY = offset
}

func DrawMenuOverlay() {
	w := rl.GetScreenWidth()
	h := rl.GetScreenHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rl.NewColor(0, 0, 0, 180))

	SetMenuTitleOffsetY(60)

	if menuTexLoaded {
		src := rl.NewRectangle(0, 0, float32(menuTex.Width), float32(menuTex.Height))

		maxW := float32(w) * 0.7
		maxH := float32(h) * 0.7
		scaleW := maxW / float32(menuTex.Width)
		scaleH := maxH / float32(menuTex.Height)
		scale := scaleW
		if scaleH < scale {
			scale = scaleH
		}
		dw := float32(menuTex.Width) * scale
		dh := float32(menuTex.Height) * scale
		dst := rl.NewRectangle(float32(w)/2-dw/2, float32(h)/2-dh/2-60, dw, dh)
		origin := rl.NewVector2(0, 0)
		rl.DrawTexturePro(menuTex, src, dst, origin, 0, rl.RayWhite)
	}

	title := "SPOOK 'N LOOT"
	description := "A game by joeel56\nYour goal is to kill all enemies and reach the exit\n of the dungeon.\nYou have 20 levels and every level gets harder\ntill you reach the boss.\nIf you die you start from the beginning."
	instructions := "Press ESC to open the menu\nYou can walk with WASD or arrow keys\nAttack the enemies with left click\nYou can pause the music with F7\nF10 to toggle fullscreen"
	smallTextBottom := "Assets by franuka.art"
	smallTextBottomSize := float32(16)
	smallTextBottomLines := strings.Split(smallTextBottom, "\n")
	smallTextBottomHeight := float32(0)
	if len(smallTextBottomLines) > 0 {
		smallTextBottomHeight = float32(len(smallTextBottomLines))*smallTextBottomSize + float32(max(0, len(smallTextBottomLines)-1))*smallTextBottomSize
	}
	// Layout settings
	titleSize := float32(64)
	bodySize := float32(22)
	lineGap := float32(6)
	blockGap := float32(18)

	// Split into lines
	descLines := strings.Split(description, "\n")
	instLines := strings.Split(instructions, "\n")
	cx := float32(w) / 2
	cy := float32(h) / 2

	// Compute total height for vertical centering
	descHeight := float32(0)
	if len(descLines) > 0 {
		descHeight = float32(len(descLines))*bodySize + float32(max(0, len(descLines)-1))*lineGap
	}
	instHeight := float32(0)
	if len(instLines) > 0 {
		instHeight = float32(len(instLines))*bodySize + float32(max(0, len(instLines)-1))*lineGap
	}
	totalHeight := titleSize + blockGap + descHeight + blockGap + instHeight + smallTextBottomHeight
	startY := cy - (totalHeight / 2)

	textColor := rl.NewColor(134, 87, 87, 255)

	if menuFontLoaded {
		spacing := float32(0)

		tw := rl.MeasureTextEx(menuFont, title, titleSize, spacing)
		titleY := startY - menuTitleOffsetY
		rl.DrawTextEx(menuFont, title, rl.NewVector2(cx-(tw.X/2), titleY), titleSize, spacing, rl.RayWhite)

		y := startY + titleSize + blockGap - menuTitleOffsetY
		for _, line := range descLines {
			lw := rl.MeasureText(line, int32(bodySize))
			rl.DrawText(line, int32(cx)-int32(lw/2), int32(y), int32(bodySize), textColor)
			y += bodySize + lineGap
		}

		y += blockGap - lineGap
		for _, line := range instLines {
			lw := rl.MeasureText(line, int32(bodySize))
			rl.DrawText(line, int32(cx)-int32(lw/2), int32(y), int32(bodySize), textColor)
			y += bodySize + lineGap
		}
		y += blockGap - lineGap
		for _, line := range smallTextBottomLines {
			lw := rl.MeasureText(line, int32(smallTextBottomSize))
			rl.DrawText(line, int32(cx)-int32(lw/2), int32(y), int32(smallTextBottomSize), textColor)
			y += smallTextBottomSize + lineGap
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
