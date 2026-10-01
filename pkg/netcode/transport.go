// Package netcode contains the networking layer: a transport abstraction,
// a QUIC based implementation and LAN discovery. The game code only talks to
// the Transport interface so other backends (e.g. Steam Networking Sockets)
// can be plugged in later without touching gameplay or lobby code.
package netcode

// PeerID identifies a connected peer. The host itself is always HostPeerID.
type PeerID uint32

const HostPeerID PeerID = 1

type EventKind int

const (
	EventConnected EventKind = iota
	EventDisconnected
	EventMessage
)

type Event struct {
	Kind EventKind
	Peer PeerID
	Data []byte
	// Err is set on EventDisconnected when the connection was lost unexpectedly.
	Err error
}

// Transport is a message based connection. On the host it manages one
// connection per client, on a client it holds a single connection to the
// host (addressed as HostPeerID).
type Transport interface {
	// Send delivers data to one peer. Reliable messages arrive in order;
	// unreliable ones may be dropped (used for frequent state snapshots).
	Send(to PeerID, data []byte, reliable bool) error
	// Broadcast sends to every connected peer (host only; on a client it sends to the host).
	Broadcast(data []byte, reliable bool)
	// Disconnect drops a single peer (host only).
	Disconnect(peer PeerID, reason string)
	// Poll returns all events received since the last call. Never blocks,
	// so it can be called once per frame from the game loop.
	Poll() []Event
	Close()
}
