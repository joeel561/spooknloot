package lobby

import (
	"encoding/binary"
	"errors"
	"math"
	"time"

	"spooknloot/pkg/netcode"
)

// Area tells which map a player is on. Players only see each other in the
// same area.
type Area uint8

const (
	AreaWorld Area = iota
	AreaDungeon
	AreaBoss
)

// PlayerState is what every player shares about themselves while playing.
// Each player is authoritative over their own movement; the host relays.
type PlayerState struct {
	ID     netcode.PeerID
	X, Y   float32
	Dir    uint8
	Frame  uint8
	Area   Area
	Health uint8 // percent of max health
}

const (
	stateSendInterval = 50 * time.Millisecond // 20 Hz
	maxSamples        = 16
	playerStateSize   = 4 + 4 + 4 + 1 + 1 + 1 + 1
)

// sample is a received state with its arrival time, used for interpolation.
type sample struct {
	at    time.Time
	state PlayerState
}

type remoteTrack struct {
	samples []sample
}

func (t *remoteTrack) add(at time.Time, s PlayerState) {
	t.samples = append(t.samples, sample{at, s})
	if len(t.samples) > maxSamples {
		t.samples = t.samples[len(t.samples)-maxSamples:]
	}
}

// at returns the state interpolated for the given render time.
func (t *remoteTrack) at(renderTime time.Time) PlayerState {
	s := t.samples
	if !renderTime.After(s[0].at) {
		return s[0].state
	}
	for i := 1; i < len(s); i++ {
		if renderTime.Before(s[i].at) {
			a, b := s[i-1], s[i]
			f := float32(renderTime.Sub(a.at)) / float32(b.at.Sub(a.at))
			out := a.state
			out.X += (b.state.X - a.state.X) * f
			out.Y += (b.state.Y - a.state.Y) * f
			return out
		}
	}
	// No newer data: hold the last known state instead of guessing.
	return s[len(s)-1].state
}

// gameState holds the in-game player sync data of a Lobby.
type gameState struct {
	local    PlayerState
	hasLocal bool
	lastSend time.Time
	remotes  map[netcode.PeerID]*remoteTrack
}

// SetLocalState records the local player's state. It is sent to the others
// at a fixed rate from Update, so it is fine to call this every frame.
func (l *Lobby) SetLocalState(s PlayerState) {
	s.ID = l.LocalID
	l.game.local = s
	l.game.hasLocal = true
}

// RemoteStates returns the other players' states, interpolated with the
// given delay so movement looks smooth despite 20 Hz updates.
func (l *Lobby) RemoteStates(delay time.Duration) []PlayerState {
	renderTime := time.Now().Add(-delay)
	out := make([]PlayerState, 0, len(l.game.remotes))
	for id, t := range l.game.remotes {
		if id == l.LocalID || len(t.samples) == 0 {
			continue
		}
		out = append(out, t.at(renderTime))
	}
	return out
}

func (l *Lobby) remoteTrack(id netcode.PeerID) *remoteTrack {
	if l.game.remotes == nil {
		l.game.remotes = make(map[netcode.PeerID]*remoteTrack)
	}
	t := l.game.remotes[id]
	if t == nil {
		t = &remoteTrack{}
		l.game.remotes[id] = t
	}
	return t
}

// updateGame sends the local state (clients) or the snapshot of all
// players (host) at the fixed send rate.
func (l *Lobby) updateGame() {
	if l.State != StateInGame || time.Since(l.game.lastSend) < stateSendInterval {
		return
	}
	l.game.lastSend = time.Now()

	if !l.IsHost {
		if l.game.hasLocal {
			_ = l.transport.Send(netcode.HostPeerID, encodeStates(msgPlayerState, []PlayerState{l.game.local}), false)
		}
		return
	}

	states := make([]PlayerState, 0, len(l.Players))
	if l.game.hasLocal {
		states = append(states, l.game.local)
	}
	for _, p := range l.Players {
		if t := l.game.remotes[p.ID]; t != nil && len(t.samples) > 0 {
			states = append(states, t.samples[len(t.samples)-1].state)
		}
	}
	l.transport.Broadcast(encodeStates(msgPlayerSnapshot, states), false)
}

// hostReceiveState stores a client's own state. The sender ID comes from
// the connection, never from the packet, so nobody can move other players.
func (l *Lobby) hostReceiveState(from netcode.PeerID, body []byte) {
	states, err := decodeStates(body)
	if err != nil || len(states) != 1 || !l.hasPlayer(from) {
		return
	}
	s := states[0]
	s.ID = from
	l.remoteTrack(from).add(time.Now(), s)
}

// clientReceiveSnapshot updates all remote players. Players missing from the
// snapshot have left (or not started yet) and are dropped.
func (l *Lobby) clientReceiveSnapshot(body []byte) {
	states, err := decodeStates(body)
	if err != nil {
		return
	}
	now := time.Now()
	seen := make(map[netcode.PeerID]bool, len(states))
	for _, s := range states {
		if s.ID == l.LocalID {
			continue
		}
		seen[s.ID] = true
		l.remoteTrack(s.ID).add(now, s)
	}
	for id := range l.game.remotes {
		if !seen[id] {
			delete(l.game.remotes, id)
		}
	}
}

// GameMessage is a gameplay message passed through the lobby connection.
// Data starts with its message type (>= MsgGameFirst).
type GameMessage struct {
	From netcode.PeerID
	Data []byte
}

// SendGame sends a gameplay message. On a client it always goes to the host.
func (l *Lobby) SendGame(to netcode.PeerID, data []byte, reliable bool) {
	if l.transport == nil || l.State != StateInGame {
		return
	}
	_ = l.transport.Send(to, data, reliable)
}

// TakeGameMessages returns the gameplay messages received since the last call.
func (l *Lobby) TakeGameMessages() []GameMessage {
	m := l.gameInbox
	l.gameInbox = nil
	return m
}

// LatestStates returns the most recent state of every other player
// without interpolation (used by the host for the simulation).
func (l *Lobby) LatestStates() []PlayerState {
	out := make([]PlayerState, 0, len(l.game.remotes))
	for id, t := range l.game.remotes {
		if id != l.LocalID && len(t.samples) > 0 {
			out = append(out, t.samples[len(t.samples)-1].state)
		}
	}
	return out
}

// PlayerName returns the display name of a player.
func (l *Lobby) PlayerName(id netcode.PeerID) string {
	for _, p := range l.Players {
		if p.ID == id {
			return p.Name
		}
	}
	return "?"
}

func (l *Lobby) hasPlayer(id netcode.PeerID) bool {
	for _, p := range l.Players {
		if p.ID == id {
			return true
		}
	}
	return false
}

// Player states are sent many times per second, so they use a compact
// binary layout: type byte, count byte, then fixed size records.
func encodeStates(t msgType, states []PlayerState) []byte {
	buf := make([]byte, 2, 2+len(states)*playerStateSize)
	buf[0] = byte(t)
	buf[1] = byte(len(states))
	for _, s := range states {
		buf = binary.BigEndian.AppendUint32(buf, uint32(s.ID))
		buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(s.X))
		buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(s.Y))
		buf = append(buf, s.Dir, s.Frame, byte(s.Area), s.Health)
	}
	return buf
}

// decodeStates parses the body after the type byte.
func decodeStates(body []byte) ([]PlayerState, error) {
	if len(body) < 1 {
		return nil, errors.New("short state message")
	}
	n := int(body[0])
	body = body[1:]
	if len(body) != n*playerStateSize {
		return nil, errors.New("bad state message length")
	}
	out := make([]PlayerState, n)
	for i := range out {
		r := body[i*playerStateSize:]
		x := math.Float32frombits(binary.BigEndian.Uint32(r[4:]))
		y := math.Float32frombits(binary.BigEndian.Uint32(r[8:]))
		if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) || math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) {
			return nil, errors.New("invalid position")
		}
		out[i] = PlayerState{
			ID:     netcode.PeerID(binary.BigEndian.Uint32(r)),
			X:      x,
			Y:      y,
			Dir:    r[12],
			Frame:  r[13],
			Area:   Area(r[14]),
			Health: r[15],
		}
	}
	return out, nil
}
