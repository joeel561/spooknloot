// Package lobby implements the pre-game lobby on top of a netcode.Transport:
// players join, set themselves ready and the host starts the game.
package lobby

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"spooknloot/pkg/netcode"
)

const (
	MaxPlayers    = 40
	MaxNameLength = 16
	dialTimeout   = 5 * time.Second
)

type State int

const (
	StateConnecting State = iota
	StateInLobby
	StateInGame
	StateClosed
)

type Player struct {
	ID    netcode.PeerID `json:"id"`
	Name  string         `json:"name"`
	Ready bool           `json:"ready"`
	Host  bool           `json:"host"`
}

type Lobby struct {
	IsHost    bool
	LobbyName string
	LocalID   netcode.PeerID
	Players   []Player
	State     State
	// Err explains why the lobby closed (nil when the local player left).
	Err error
	// Seed is shared by all players once the game starts.
	Seed int64

	transport netcode.Transport
	announcer *netcode.Announcer
	dialDone  chan dialResult
	started   bool
	game      gameState
	gameInbox []GameMessage
}

type dialResult struct {
	client *netcode.QUICClient
	err    error
}

// NewHost opens a lobby on the given port and announces it on the LAN.
func NewHost(playerName string, port int) (*Lobby, error) {
	return newHost(playerName, "", port, true)
}

// NewHostOn binds to listenIP only ("" for all interfaces). Local tests use
// 127.0.0.1 without LAN announcements so Windows Firewall does not prompt.
func NewHostOn(playerName, listenIP string, port int, announce bool) (*Lobby, error) {
	return newHost(playerName, listenIP, port, announce)
}

func newHost(playerName, listenIP string, port int, announce bool) (*Lobby, error) {
	h, err := netcode.Host(net.JoinHostPort(listenIP, fmt.Sprint(port)))
	if err != nil {
		return nil, fmt.Errorf("could not open port %d: %w", port, err)
	}
	name := SanitizeName(playerName)
	l := &Lobby{
		IsHost:    true,
		LobbyName: name + "'s game",
		LocalID:   netcode.HostPeerID,
		Players:   []Player{{ID: netcode.HostPeerID, Name: name, Ready: true, Host: true}},
		State:     StateInLobby,
		transport: h,
	}
	if !announce {
		return l, nil
	}
	ann, err := netcode.StartAnnouncer(netcode.Announcement{
		Version:    ProtocolVersion,
		Name:       l.LobbyName,
		Port:       port,
		Players:    1,
		MaxPlayers: MaxPlayers,
	})
	if err == nil {
		l.announcer = ann
	}
	// Without an announcer the lobby still works via direct IP.
	return l, nil
}

// Join connects to a host in the background. addr may omit the port.
func Join(addr, playerName string) *Lobby {
	l := &Lobby{State: StateConnecting, dialDone: make(chan dialResult, 1)}
	addr = withDefaultPort(strings.TrimSpace(addr))
	hello := encode(msgHello, helloMsg{Name: SanitizeName(playerName), Version: ProtocolVersion})
	go func() {
		c, err := netcode.Dial(addr, hello, dialTimeout)
		l.dialDone <- dialResult{c, err}
	}()
	return l
}

func withDefaultPort(addr string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(addr, fmt.Sprint(netcode.DefaultGamePort))
}

// SanitizeName trims a player name and limits it to MaxNameLength characters.
func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > MaxNameLength {
		name = string([]rune(name)[:MaxNameLength])
	}
	if name == "" {
		name = "Player"
	}
	return name
}

// Update processes network events. Call it once per frame.
func (l *Lobby) Update() {
	if l.State == StateClosed {
		return
	}
	if l.transport == nil && l.dialDone != nil {
		select {
		case r := <-l.dialDone:
			l.dialDone = nil
			if r.err != nil {
				l.close(fmt.Errorf("could not connect: %w", r.err))
				return
			}
			l.transport = r.client
		default:
			return
		}
	}
	if l.transport == nil {
		return
	}
	for _, ev := range l.transport.Poll() {
		if l.IsHost {
			l.hostHandle(ev)
		} else {
			l.clientHandle(ev)
		}
		if l.State == StateClosed {
			return
		}
	}
	l.updateGame()
}

func (l *Lobby) hostHandle(ev netcode.Event) {
	switch ev.Kind {
	case netcode.EventConnected:
		// Wait for the hello before the peer counts as a player.
	case netcode.EventDisconnected:
		if l.removePlayer(ev.Peer) {
			l.broadcastState()
		}
	case netcode.EventMessage:
		t, body, err := decode(ev.Data)
		if err != nil {
			return
		}
		if t >= MsgGameFirst {
			if l.State == StateInGame && l.hasPlayer(ev.Peer) {
				l.gameInbox = append(l.gameInbox, GameMessage{From: ev.Peer, Data: ev.Data})
			}
			return
		}
		switch t {
		case msgPlayerState:
			l.hostReceiveState(ev.Peer, body)
		case msgHello:
			var m helloMsg
			if json.Unmarshal(body, &m) != nil {
				l.transport.Disconnect(ev.Peer, "invalid hello")
				return
			}
			l.hostAddPlayer(ev.Peer, m)
		case msgSetReady:
			var m setReadyMsg
			if json.Unmarshal(body, &m) != nil {
				return
			}
			for i := range l.Players {
				if l.Players[i].ID == ev.Peer {
					l.Players[i].Ready = m.Ready
				}
			}
			l.broadcastState()
		}
	}
}

func (l *Lobby) hostAddPlayer(peer netcode.PeerID, m helloMsg) {
	if m.Version != ProtocolVersion {
		l.transport.Disconnect(peer, fmt.Sprintf("version mismatch (host %s, you %s)", ProtocolVersion, m.Version))
		return
	}
	if len(l.Players) >= MaxPlayers {
		l.transport.Disconnect(peer, "lobby is full")
		return
	}
	l.Players = append(l.Players, Player{ID: peer, Name: l.uniqueName(SanitizeName(m.Name))})
	_ = l.transport.Send(peer, encode(msgWelcome, welcomeMsg{YourID: peer}), true)
	l.broadcastState()
	// Late joiners go straight into the running game.
	if l.State == StateInGame {
		_ = l.transport.Send(peer, encode(msgStartGame, startGameMsg{Seed: l.Seed}), true)
	}
}

func (l *Lobby) uniqueName(name string) string {
	taken := func(n string) bool {
		for _, p := range l.Players {
			if p.Name == n {
				return true
			}
		}
		return false
	}
	if !taken(name) {
		return name
	}
	for i := 2; ; i++ {
		n := fmt.Sprintf("%s (%d)", name, i)
		if !taken(n) {
			return n
		}
	}
}

func (l *Lobby) removePlayer(id netcode.PeerID) bool {
	delete(l.game.remotes, id)
	for i, p := range l.Players {
		if p.ID == id {
			l.Players = append(l.Players[:i], l.Players[i+1:]...)
			return true
		}
	}
	return false
}

func (l *Lobby) broadcastState() {
	l.transport.Broadcast(encode(msgLobbyState, lobbyStateMsg{
		LobbyName: l.LobbyName,
		Players:   l.Players,
		InGame:    l.State == StateInGame,
	}), true)
	if l.announcer != nil {
		n, inGame := len(l.Players), l.State == StateInGame
		l.announcer.Update(func(a *netcode.Announcement) {
			a.Players = n
			a.InGame = inGame
		})
	}
}

func (l *Lobby) clientHandle(ev netcode.Event) {
	switch ev.Kind {
	case netcode.EventDisconnected:
		err := ev.Err
		if err == nil {
			err = errors.New("disconnected from host")
		}
		l.close(err)
	case netcode.EventMessage:
		t, body, err := decode(ev.Data)
		if err != nil {
			return
		}
		if t >= MsgGameFirst {
			l.gameInbox = append(l.gameInbox, GameMessage{From: ev.Peer, Data: ev.Data})
			return
		}
		switch t {
		case msgPlayerSnapshot:
			l.clientReceiveSnapshot(body)
		case msgWelcome:
			var m welcomeMsg
			if json.Unmarshal(body, &m) == nil {
				l.LocalID = m.YourID
				l.State = StateInLobby
			}
		case msgLobbyState:
			var m lobbyStateMsg
			if json.Unmarshal(body, &m) == nil {
				l.LobbyName = m.LobbyName
				l.Players = m.Players
			}
		case msgStartGame:
			var m startGameMsg
			if json.Unmarshal(body, &m) == nil {
				l.Seed = m.Seed
				l.State = StateInGame
				l.started = true
			}
		}
	}
}

// LocalPlayer returns the local player's entry, if already known.
func (l *Lobby) LocalPlayer() (Player, bool) {
	for _, p := range l.Players {
		if p.ID == l.LocalID {
			return p, true
		}
	}
	return Player{}, false
}

func (l *Lobby) SetReady(ready bool) {
	if l.IsHost || l.State != StateInLobby {
		return
	}
	_ = l.transport.Send(netcode.HostPeerID, encode(msgSetReady, setReadyMsg{Ready: ready}), true)
}

// CanStart reports whether the host may start: every other player is ready.
func (l *Lobby) CanStart() bool {
	if !l.IsHost || l.State != StateInLobby {
		return false
	}
	for _, p := range l.Players {
		if !p.Ready {
			return false
		}
	}
	return true
}

// Start begins the game for everyone in the lobby (host only).
func (l *Lobby) Start() {
	if !l.CanStart() {
		return
	}
	l.Seed = rand.Int63()
	l.State = StateInGame
	l.started = true
	l.transport.Broadcast(encode(msgStartGame, startGameMsg{Seed: l.Seed}), true)
	l.broadcastState()
}

// TakeStarted returns true once after the game was started.
func (l *Lobby) TakeStarted() bool {
	s := l.started
	l.started = false
	return s
}

// Leave closes the lobby. For the host this ends it for everyone.
func (l *Lobby) Leave() { l.close(nil) }

func (l *Lobby) close(err error) {
	if l.State == StateClosed {
		return
	}
	l.State = StateClosed
	l.Err = err
	if l.announcer != nil {
		l.announcer.Close()
		l.announcer = nil
	}
	if l.transport != nil {
		l.transport.Close()
	}
	// A dial still in flight is closed when it finishes.
	if l.dialDone != nil {
		go func(ch chan dialResult) {
			if r := <-ch; r.client != nil {
				r.client.Close()
			}
		}(l.dialDone)
	}
}
