package lobby

import (
	"encoding/json"
	"errors"

	"spooknloot/pkg/netcode"
	"spooknloot/pkg/sim"
)

// ProtocolVersion must match between host and clients. Bump it whenever a
// message format changes.
const ProtocolVersion = "5"

// MsgGameFirst is the first message type reserved for gameplay messages.
// The lobby passes those through untouched (see Lobby.SendGame).
const MsgGameFirst = 64

type msgType byte

const (
	msgHello msgType = iota + 1
	msgWelcome
	msgLobbyState
	msgSetReady
	msgStartGame
	msgPlayerState    // client -> host, own state, unreliable
	msgPlayerSnapshot // host -> clients, all states, unreliable
	msgPing           // client -> host, send time, unreliable
	msgPong           // host -> client, echoes the ping
)

// Lobby messages are rare and small, so they are a type byte followed by
// JSON. Frequent gameplay messages will get a compact binary format.

type helloMsg struct {
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Class   sim.Class `json:"class"`
}

type welcomeMsg struct {
	YourID netcode.PeerID `json:"yourId"`
}

type lobbyStateMsg struct {
	LobbyName string   `json:"lobbyName"`
	Players   []Player `json:"players"`
	InGame    bool     `json:"inGame"`
}

type setReadyMsg struct {
	Ready bool `json:"ready"`
}

type startGameMsg struct {
	Seed int64 `json:"seed"`
}

func encode(t msgType, v any) []byte {
	body, _ := json.Marshal(v)
	return append([]byte{byte(t)}, body...)
}

func decode(data []byte) (msgType, []byte, error) {
	if len(data) == 0 {
		return 0, nil, errors.New("empty message")
	}
	return msgType(data[0]), data[1:], nil
}
