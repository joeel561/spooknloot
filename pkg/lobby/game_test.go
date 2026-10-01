package lobby

import (
	"fmt"
	"math"
	"testing"
	"time"

	"spooknloot/pkg/sim"
)

func TestStatesRoundTrip(t *testing.T) {
	in := []PlayerState{
		{ID: 1, X: 495.5, Y: -12.25, Dir: 7, Frame: 2, Area: AreaWorld, Health: 100},
		{ID: 42, X: 0, Y: 1e6, Dir: 0, Frame: 0, Area: AreaBoss, Health: 3},
	}
	data := encodeStates(msgPlayerSnapshot, in)
	if msgType(data[0]) != msgPlayerSnapshot {
		t.Fatalf("type = %d", data[0])
	}
	out, err := decodeStates(data[1:])
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(out) != fmt.Sprint(in) {
		t.Errorf("got %v, want %v", out, in)
	}

	// 40 players must fit into a single QUIC datagram (~1200 bytes).
	if n := len(encodeStates(msgPlayerSnapshot, make([]PlayerState, MaxPlayers))); n > 1100 {
		t.Errorf("snapshot for %d players is %d bytes", MaxPlayers, n)
	}
}

func TestDecodeStatesRejectsGarbage(t *testing.T) {
	if _, err := decodeStates(nil); err == nil {
		t.Error("empty body accepted")
	}
	if _, err := decodeStates([]byte{2, 1, 2, 3}); err == nil {
		t.Error("truncated body accepted")
	}
	bad := encodeStates(msgPlayerState, []PlayerState{{X: float32(math.NaN())}})
	if _, err := decodeStates(bad[1:]); err == nil {
		t.Error("NaN position accepted")
	}
}

func TestInterpolation(t *testing.T) {
	t0 := time.Now()
	var tr remoteTrack
	tr.add(t0, PlayerState{X: 0, Y: 0, Frame: 1})
	tr.add(t0.Add(100*time.Millisecond), PlayerState{X: 10, Y: -20, Frame: 2})

	if s := tr.at(t0.Add(-time.Second)); s.X != 0 {
		t.Errorf("before first sample: X = %v", s.X)
	}
	if s := tr.at(t0.Add(50 * time.Millisecond)); s.X != 5 || s.Y != -10 || s.Frame != 1 {
		t.Errorf("halfway: %+v", s)
	}
	if s := tr.at(t0.Add(time.Second)); s.X != 10 || s.Frame != 2 {
		t.Errorf("after last sample: %+v", s)
	}
}

func TestPlayerStateRelay(t *testing.T) {
	host, err := newHost("Host", sim.ClassWarrior, "127.0.0.1", testPort+2, false)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Leave()
	addr := fmt.Sprintf("127.0.0.1:%d", testPort+2)
	a := Join(addr, "A", sim.ClassWarrior)
	b := Join(addr, "B", sim.ClassWarrior)
	all := []*Lobby{host, a, b}
	waitFor(t, "lobby", all, func() bool { return len(a.Players) == 3 && len(b.Players) == 3 })
	a.SetReady(true)
	b.SetReady(true)
	waitFor(t, "ready", all, host.CanStart)
	host.Start()
	waitFor(t, "start", all, func() bool { return a.State == StateInGame && b.State == StateInGame })

	host.SetLocalState(PlayerState{X: 1, Area: AreaWorld})
	a.SetLocalState(PlayerState{X: 2, Area: AreaWorld})
	b.SetLocalState(PlayerState{X: 3, Area: AreaDungeon})

	byID := func(l *Lobby) map[uint32]PlayerState {
		m := map[uint32]PlayerState{}
		for _, s := range l.RemoteStates(0) {
			m[uint32(s.ID)] = s
		}
		return m
	}
	waitFor(t, "everyone sees the two others", all, func() bool {
		return len(byID(host)) == 2 && len(byID(a)) == 2 && len(byID(b)) == 2
	})
	if s := byID(b)[uint32(a.LocalID)]; s.X != 2 {
		t.Errorf("b sees a at X=%v, want 2", s.X)
	}
	if s := byID(a)[uint32(b.LocalID)]; s.Area != AreaDungeon {
		t.Errorf("a sees b in area %v, want dungeon", s.Area)
	}
	if _, ok := byID(a)[uint32(a.LocalID)]; ok {
		t.Error("a sees itself as a remote player")
	}

	// A player who leaves disappears for the others.
	b.Leave()
	waitFor(t, "b gone", all, func() bool { return len(byID(a)) == 1 && len(byID(host)) == 1 })
}

func TestPingIsMeasuredAndShared(t *testing.T) {
	host, err := newHost("Host", sim.ClassWarrior, "127.0.0.1", testPort+3, false)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Leave()
	c := Join(fmt.Sprintf("127.0.0.1:%d", testPort+3), "C", sim.ClassWarrior)
	defer c.Leave()
	all := []*Lobby{host, c}
	waitFor(t, "lobby", all, func() bool { return c.State == StateInLobby })
	c.SetReady(true)
	waitFor(t, "ready", all, host.CanStart)
	host.Start()
	waitFor(t, "start", all, func() bool { return c.State == StateInGame })

	if _, ok := host.Ping(); !ok {
		t.Error("host should report a ping of 0")
	}
	waitFor(t, "client ping", all, func() bool { _, ok := c.Ping(); return ok })

	// Pretend a slow connection and check the value reaches the host.
	c.ping.rtt = 123 * time.Millisecond
	waitFor(t, "ping shared", all, func() bool {
		c.SetLocalState(PlayerState{X: 1})
		host.SetLocalState(PlayerState{})
		for _, s := range host.LatestStates() {
			// A real pong may smooth the value down a bit meanwhile.
			if s.ID == c.LocalID && s.Ping >= 50 {
				return true
			}
		}
		return false
	})
}
