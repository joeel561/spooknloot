package lobby

import (
	"encoding/json"
	"strings"
	"time"

	"spooknloot/pkg/netcode"
)

// Chat goes through the host: clients send a line, the host checks it and
// shares it with everyone (including the sender), so all players see the
// same messages in the same order.

const (
	MaxChatLength  = 120
	chatLogSize    = 50
	chatRateLimit  = 5 // messages per chatRateWindow and player
	chatRateWindow = 5 * time.Second
)

// ChatLine is one message in the chat log. System lines (joins, leaves)
// have no sender.
type ChatLine struct {
	From   netcode.PeerID
	Name   string
	Text   string
	System bool
	At     time.Time
}

type chatSendMsg struct {
	Text string `json:"text"`
}

type chatLineMsg struct {
	From   netcode.PeerID `json:"from"`
	Name   string         `json:"name"`
	Text   string         `json:"text"`
	System bool           `json:"system"`
}

type chatState struct {
	log    []ChatLine
	recent map[netcode.PeerID][]time.Time
}

// SanitizeChat trims a message, replaces characters the game font can't
// show and limits the length. It returns "" for messages not worth sending.
func SanitizeChat(text string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(text) {
		if b.Len() >= MaxChatLength {
			break
		}
		if r < 32 || r > 126 {
			r = '?'
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// Chat returns the chat log, oldest first.
func (l *Lobby) Chat() []ChatLine { return l.chat.log }

// SendChat sends a chat message to everyone.
func (l *Lobby) SendChat(text string) {
	text = SanitizeChat(text)
	if text == "" || l.transport == nil || l.State == StateConnecting || l.State == StateClosed {
		return
	}
	if l.IsHost {
		l.hostChat(l.LocalID, text)
		return
	}
	_ = l.transport.Send(netcode.HostPeerID, encode(msgChat, chatSendMsg{Text: text}), true)
}

// hostChat checks and shares a message from a player.
func (l *Lobby) hostChat(from netcode.PeerID, text string) {
	text = SanitizeChat(text)
	if text == "" || !l.hasPlayer(from) || !l.allowChat(from) {
		return
	}
	l.shareChat(chatLineMsg{From: from, Name: l.PlayerName(from), Text: text})
}

// hostSystemChat shares a line like "Alice joined".
func (l *Lobby) hostSystemChat(text string) {
	l.shareChat(chatLineMsg{Text: text, System: true})
}

func (l *Lobby) shareChat(m chatLineMsg) {
	l.addChatLine(m)
	l.transport.Broadcast(encode(msgChatLine, m), true)
}

// allowChat limits how fast a single player can send messages.
func (l *Lobby) allowChat(from netcode.PeerID) bool {
	if l.chat.recent == nil {
		l.chat.recent = map[netcode.PeerID][]time.Time{}
	}
	now := time.Now()
	var kept []time.Time
	for _, t := range l.chat.recent[from] {
		if now.Sub(t) < chatRateWindow {
			kept = append(kept, t)
		}
	}
	if len(kept) >= chatRateLimit {
		l.chat.recent[from] = kept
		return false
	}
	l.chat.recent[from] = append(kept, now)
	return true
}

func (l *Lobby) addChatLine(m chatLineMsg) {
	l.chat.log = append(l.chat.log, ChatLine{From: m.From, Name: m.Name, Text: m.Text, System: m.System, At: time.Now()})
	if len(l.chat.log) > chatLogSize {
		l.chat.log = l.chat.log[len(l.chat.log)-chatLogSize:]
	}
}

func (l *Lobby) hostReceiveChat(from netcode.PeerID, body []byte) {
	var m chatSendMsg
	if json.Unmarshal(body, &m) == nil {
		l.hostChat(from, m.Text)
	}
}

func (l *Lobby) clientReceiveChat(body []byte) {
	var m chatLineMsg
	if json.Unmarshal(body, &m) != nil {
		return
	}
	// The host already cleaned it, but never trust the network.
	m.Text = SanitizeChat(m.Text)
	m.Name = SanitizeName(m.Name)
	if m.Text != "" {
		l.addChatLine(m)
	}
}
