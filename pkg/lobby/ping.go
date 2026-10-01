package lobby

import (
	"encoding/binary"
	"time"

	"spooknloot/pkg/netcode"
)

const pingInterval = time.Second

// pingState measures the round trip time from a client to the host.
type pingState struct {
	lastSent time.Time
	rtt      time.Duration
	measured bool
}

// Ping returns the smoothed round trip time to the host. The host itself
// has no ping; ok is false until the first answer arrived.
func (l *Lobby) Ping() (rtt time.Duration, ok bool) {
	if l.IsHost {
		return 0, true
	}
	return l.ping.rtt, l.ping.measured
}

func (l *Lobby) pingMillis() uint16 {
	rtt, _ := l.Ping()
	return uint16(min(rtt.Milliseconds(), 65535))
}

func (l *Lobby) updatePing() {
	if l.IsHost || l.transport == nil || l.State == StateConnecting || time.Since(l.ping.lastSent) < pingInterval {
		return
	}
	l.ping.lastSent = time.Now()
	msg := binary.BigEndian.AppendUint64([]byte{byte(msgPing)}, uint64(time.Now().UnixNano()))
	_ = l.transport.Send(netcode.HostPeerID, msg, false)
}

// hostAnswerPing echoes a ping; the client compares with its own clock.
func (l *Lobby) hostAnswerPing(from netcode.PeerID, body []byte) {
	if len(body) != 8 {
		return
	}
	_ = l.transport.Send(from, append([]byte{byte(msgPong)}, body...), false)
}

func (l *Lobby) clientReceivePong(body []byte) {
	if len(body) != 8 {
		return
	}
	sent := time.Unix(0, int64(binary.BigEndian.Uint64(body)))
	rtt := time.Since(sent)
	if rtt < 0 || rtt > time.Minute {
		return
	}
	if !l.ping.measured {
		l.ping.rtt, l.ping.measured = rtt, true
		return
	}
	// Smooth out jitter.
	l.ping.rtt = (l.ping.rtt*7 + rtt*3) / 10
}
