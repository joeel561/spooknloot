package ui

import (
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ChatEntry is one chat message to show.
type ChatEntry struct {
	Name      string
	Text      string
	NameColor rl.Color
	System    bool
	Age       time.Duration
}

const (
	chatWidth      = 520
	chatFontSize   = 18
	chatLineHeight = 22
	chatShowFor    = 10 * time.Second
	chatFadeFor    = 2 * time.Second
	chatMaxClosed  = 5
	chatMaxOpen    = 10
)

// chatLine is one wrapped screen line; only the first line of a message
// carries the colored "Name: " prefix.
type chatLine struct {
	prefix      string
	prefixColor rl.Color
	text        string
	textColor   rl.Color
	alpha       float32
}

// DrawChat draws recent chat messages bottom left and, while open, the
// input box. It returns true when the player pressed Enter in the box.
func DrawChat(entries []ChatEntry, input *TextInput, open bool) bool {
	x := int32(20)
	bottom := int32(rl.GetScreenHeight()) - 150

	submitted := false
	if open {
		input.Focused = true // clicks in the game must not unfocus it
		submitted = input.Draw(rl.NewRectangle(float32(x)-8, float32(bottom)+4, chatWidth+16, 32))
	}

	lines := chatLines(entries, open)
	if len(lines) == 0 {
		return submitted
	}
	if open {
		top := bottom - int32(len(lines))*chatLineHeight
		rl.DrawRectangle(x-8, top-6, chatWidth+16, bottom-top+8, rl.NewColor(0, 0, 0, 150))
	}
	// lines are newest first; draw them upwards from the bottom.
	y := bottom - chatLineHeight
	for _, l := range lines {
		a := uint8(255 * l.alpha)
		if !open {
			w := rl.MeasureText(l.prefix+l.text, chatFontSize)
			rl.DrawRectangle(x-4, y-2, w+8, chatLineHeight, rl.NewColor(0, 0, 0, uint8(120*l.alpha)))
		}
		px := x
		if l.prefix != "" {
			c := l.prefixColor
			c.A = a
			rl.DrawText(l.prefix, px, y, chatFontSize, c)
			px += rl.MeasureText(l.prefix, chatFontSize)
		}
		c := l.textColor
		c.A = a
		rl.DrawText(l.text, px, y, chatFontSize, c)
		y -= chatLineHeight
	}
	return submitted
}

// chatLines wraps the newest messages into screen lines, newest first.
// Closed, only messages of the last few seconds are shown, fading out.
func chatLines(entries []ChatEntry, open bool) []chatLine {
	maxLines := chatMaxClosed
	if open {
		maxLines = chatMaxOpen
	}
	var lines []chatLine
	for i := len(entries) - 1; i >= 0 && len(lines) < maxLines; i-- {
		e := entries[i]
		alpha := float32(1)
		if !open {
			if e.Age > chatShowFor {
				break
			}
			if left := chatShowFor - e.Age; left < chatFadeFor {
				alpha = float32(left) / float32(chatFadeFor)
			}
		}
		prefix, textColor := e.Name+": ", rl.RayWhite
		if e.System {
			prefix, textColor = "", mutedTextColor
		}
		// The first line leaves room for the name prefix.
		first := chatWidth - rl.MeasureText(prefix, chatFontSize)
		wrapped := wrapText(e.Text, first, chatWidth, chatFontSize)
		for j := len(wrapped) - 1; j >= 0 && len(lines) < maxLines; j-- {
			l := chatLine{text: wrapped[j], textColor: textColor, alpha: alpha}
			if j == 0 {
				l.prefix, l.prefixColor = prefix, e.NameColor
			}
			lines = append(lines, l)
		}
	}
	return lines
}

// wrapText splits text into lines no wider than width pixels; the first
// line may be narrower (firstWidth). A single word longer than a line
// stays on its own line.
func wrapText(text string, firstWidth, width, size int32) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(text) {
		next := w
		if cur != "" {
			next = cur + " " + w
		}
		limit := width
		if len(lines) == 0 {
			limit = firstWidth
		}
		if cur == "" || rl.MeasureText(next, size) <= limit {
			cur = next
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
