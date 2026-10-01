package main

import (
	"time"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	chatOpen  bool
	chatInput = ui.TextInput{MaxLen: lobby.MaxChatLength}
	// chatKeyUsed is set when Enter or Esc was used by the chat this frame,
	// so the same key press doesn't also send, leave the lobby or open
	// the menu.
	chatKeyUsed bool
)

// chatAvailable reports whether there is anyone to chat with.
func chatAvailable() bool {
	return gameLobby != nil && (currentScene == sceneLobby || currentScene == scenePlaying) && !menuOpen
}

// handleChatKeys opens and closes the chat box. It returns true while the
// chat box has the keyboard, so other game input must be ignored.
func handleChatKeys() bool {
	chatKeyUsed = false
	if !chatAvailable() {
		chatOpen = false
		return false
	}
	if !chatOpen {
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
			chatOpen, chatKeyUsed = true, true
			chatInput.Text = ""
			return true
		}
		return false
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		chatOpen, chatKeyUsed = false, true
	}
	return true
}

// drawChatHUD shows the chat and sends what the player typed.
func drawChatHUD() {
	if !chatAvailable() {
		return
	}
	now := time.Now()
	lines := gameLobby.Chat()
	entries := make([]ui.ChatEntry, len(lines))
	for i, l := range lines {
		entries[i] = ui.ChatEntry{
			Name:      l.Name,
			Text:      l.Text,
			NameColor: ui.ClassColor(classOf(l.From)),
			System:    l.System,
			Age:       now.Sub(l.At),
		}
	}
	if ui.DrawChat(entries, &chatInput, chatOpen) && !chatKeyUsed {
		gameLobby.SendChat(chatInput.Text)
		chatInput.Text = ""
		chatOpen = false
	}
}
