// Package coop runs the shared game: the host simulates mobs in the world
// and in dungeon runs, clients show what the host sends. Singleplayer is a
// host without network peers, so there is only one code path.
package coop

import (
	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/sim"
)

type PeerID = netcode.PeerID

// Message is a gameplay message from another peer.
type Message struct {
	From PeerID
	Data []byte
}

// Net is the network as seen by a coop session.
type Net interface {
	LocalID() PeerID
	IsHost() bool
	// Send goes to the host on a client, or to one client on the host.
	Send(to PeerID, data []byte, reliable bool)
	Receive() []Message
	// Players lists every connected player including the host (host only).
	Players() []PeerID
	// States returns the latest state of every remote player (host only).
	States() []lobby.PlayerState
	Name(id PeerID) string
	Class(id PeerID) sim.Class
}

// LobbyNet adapts a lobby to Net.
type LobbyNet struct{ L *lobby.Lobby }

func (n LobbyNet) LocalID() PeerID { return n.L.LocalID }
func (n LobbyNet) IsHost() bool    { return n.L.IsHost }
func (n LobbyNet) Send(to PeerID, data []byte, reliable bool) {
	n.L.SendGame(to, data, reliable)
}

func (n LobbyNet) Receive() []Message {
	in := n.L.TakeGameMessages()
	out := make([]Message, len(in))
	for i, m := range in {
		out[i] = Message{From: m.From, Data: m.Data}
	}
	return out
}

func (n LobbyNet) Players() []PeerID {
	out := make([]PeerID, len(n.L.Players))
	for i, p := range n.L.Players {
		out[i] = p.ID
	}
	return out
}

func (n LobbyNet) States() []lobby.PlayerState { return n.L.LatestStates() }
func (n LobbyNet) Name(id PeerID) string       { return n.L.PlayerName(id) }

func (n LobbyNet) Class(id PeerID) sim.Class {
	for _, p := range n.L.Players {
		if p.ID == id {
			return p.Class
		}
	}
	return sim.ClassWarrior
}

// LocalNet is the network of a singleplayer game: just the local host
// playing the given class.
type LocalNet struct{ PlayerClass sim.Class }

func (LocalNet) LocalID() PeerID             { return netcode.HostPeerID }
func (LocalNet) IsHost() bool                { return true }
func (LocalNet) Send(PeerID, []byte, bool)   {}
func (LocalNet) Receive() []Message          { return nil }
func (LocalNet) Players() []PeerID           { return []PeerID{netcode.HostPeerID} }
func (LocalNet) States() []lobby.PlayerState { return nil }
func (LocalNet) Name(PeerID) string          { return "You" }
func (n LocalNet) Class(PeerID) sim.Class    { return n.PlayerClass }
