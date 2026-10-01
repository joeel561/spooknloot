package ui

import (
	"unicode/utf8"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	panelColor       = rl.NewColor(25, 17, 27, 235)
	panelBorderColor = rl.NewColor(134, 87, 87, 255)
	buttonColor      = rl.NewColor(74, 48, 66, 255)
	buttonHoverColor = rl.NewColor(110, 66, 86, 255)
	buttonDisabled   = rl.NewColor(50, 42, 50, 255)
	accentColor      = rl.NewColor(231, 152, 50, 255)
	mutedTextColor   = rl.NewColor(170, 150, 160, 255)
	// MutedTextColor is for secondary text outside this package.
	MutedTextColor    = mutedTextColor
	errorTextColor    = rl.NewColor(230, 100, 100, 255)
	readyColor        = rl.NewColor(120, 190, 110, 255)
	inputColor        = rl.NewColor(15, 10, 17, 255)
	inputFocusedColor = rl.NewColor(231, 152, 50, 255)
)

// drawLabel draws text with the menu font when available.
func drawLabel(text string, x, y, size float32, color rl.Color) {
	if menuFontLoaded {
		rl.DrawTextEx(menuFont, text, rl.NewVector2(x, y), size, 0, color)
		return
	}
	rl.DrawText(text, int32(x), int32(y), int32(size), color)
}

func measureLabel(text string, size float32) float32 {
	if menuFontLoaded {
		return rl.MeasureTextEx(menuFont, text, size, 0).X
	}
	return float32(rl.MeasureText(text, int32(size)))
}

func drawLabelCentered(text string, cx, y, size float32, color rl.Color) {
	drawLabel(text, cx-measureLabel(text, size)/2, y, size, color)
}

func drawPanel(r rl.Rectangle) {
	rl.DrawRectangleRec(r, panelColor)
	rl.DrawRectangleLinesEx(r, 2, panelBorderColor)
}

func hovered(r rl.Rectangle) bool {
	return rl.CheckCollisionPointRec(rl.GetMousePosition(), r)
}

// button draws a clickable button and reports whether it was clicked this frame.
func button(r rl.Rectangle, label string, enabled bool) bool {
	bg := buttonColor
	fg := rl.RayWhite
	hot := enabled && hovered(r)
	switch {
	case !enabled:
		bg, fg = buttonDisabled, mutedTextColor
	case hot:
		bg = buttonHoverColor
	}
	rl.DrawRectangleRec(r, bg)
	rl.DrawRectangleLinesEx(r, 2, panelBorderColor)
	size := float32(24)
	drawLabel(label, r.X+(r.Width-measureLabel(label, size))/2, r.Y+(r.Height-size)/2, size, fg)
	return hot && rl.IsMouseButtonPressed(rl.MouseLeftButton)
}

// TextInput is a single line text field. Click it to focus.
type TextInput struct {
	Text    string
	MaxLen  int
	Focused bool
}

// Draw renders the field, handles typing while focused and reports whether
// Enter was pressed.
func (t *TextInput) Draw(r rl.Rectangle) bool {
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		t.Focused = hovered(r)
	}

	submitted := false
	if t.Focused {
		for c := rl.GetCharPressed(); c > 0; c = rl.GetCharPressed() {
			if c >= 32 && (t.MaxLen <= 0 || utf8.RuneCountInString(t.Text) < t.MaxLen) {
				t.Text += string(rune(c))
			}
		}
		if (rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace)) && t.Text != "" {
			_, size := utf8.DecodeLastRuneInString(t.Text)
			t.Text = t.Text[:len(t.Text)-size]
		}
		submitted = rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)
	}

	rl.DrawRectangleRec(r, inputColor)
	border := panelBorderColor
	if t.Focused {
		border = inputFocusedColor
	}
	rl.DrawRectangleLinesEx(r, 2, border)

	size := int32(22)
	text := t.Text
	if t.Focused && (int(rl.GetTime()*2))%2 == 0 {
		text += "_"
	}
	rl.DrawText(text, int32(r.X)+10, int32(r.Y+(r.Height-float32(size))/2), size, rl.RayWhite)
	return submitted
}
