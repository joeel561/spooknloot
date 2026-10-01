package lobby

import (
	"fmt"
	"testing"
	"time"

	"spooknloot/pkg/netcode"
	"spooknloot/pkg/sim"
)

const testPort = 47878

func waitFor(t *testing.T, what string, lobbies []*Lobby, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, l := range lobbies {
			l.Update()
		}
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestLobbyFlow(t *testing.T) {
	host, err := newHost("Nicole", sim.ClassWarrior, "127.0.0.1", testPort, false)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Leave()

	addr := fmt.Sprintf("127.0.0.1:%d", testPort)
	a := Join(addr, "Alice", sim.ClassWarrior)
	b := Join("127.0.0.1:"+fmt.Sprint(testPort), "Alice", sim.ClassWarrior) // duplicate name
	all := []*Lobby{host, a, b}

	waitFor(t, "both clients in lobby", all, func() bool {
		return len(host.Players) == 3 && len(a.Players) == 3 && len(b.Players) == 3 &&
			a.State == StateInLobby && b.State == StateInLobby
	})
	// Both connect concurrently, so either one may get the suffix.
	pa, _ := a.LocalPlayer()
	pb, _ := b.LocalPlayer()
	if names := pa.Name + "|" + pb.Name; names != "Alice|Alice (2)" && names != "Alice (2)|Alice" {
		t.Errorf("names = %q, want Alice and Alice (2)", names)
	}
	if host.CanStart() {
		t.Fatal("host can start before clients are ready")
	}

	a.SetReady(true)
	b.SetReady(true)
	waitFor(t, "everyone ready", all, host.CanStart)

	host.Start()
	waitFor(t, "clients started", all, func() bool {
		return a.State == StateInGame && b.State == StateInGame
	})
	if a.Seed != host.Seed || b.Seed != host.Seed {
		t.Errorf("seeds differ: host %d a %d b %d", host.Seed, a.Seed, b.Seed)
	}
	if !a.TakeStarted() || a.TakeStarted() {
		t.Error("TakeStarted should report exactly once")
	}

	// A late joiner goes straight into the running game.
	c := Join(addr, "Late", sim.ClassWarrior)
	all = append(all, c)
	waitFor(t, "late joiner in game", all, func() bool { return c.State == StateInGame })

	// A client leaving is removed from everyone's list.
	b.Leave()
	waitFor(t, "b removed", all, func() bool { return len(host.Players) == 3 && len(a.Players) == 3 })

	// The host leaving closes the lobby for the clients with a reason.
	host.Leave()
	waitFor(t, "clients closed", all, func() bool {
		return a.State == StateClosed && c.State == StateClosed
	})
	if a.Err == nil {
		t.Error("expected a close reason for client a")
	}
}

func TestVersionMismatchRejected(t *testing.T) {
	host, err := newHost("Host", sim.ClassWarrior, "127.0.0.1", testPort+1, false)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Leave()

	hello := encode(msgHello, helloMsg{Name: "Old", Version: "0"})
	client, err := netcode.Dial(fmt.Sprintf("127.0.0.1:%d", testPort+1), hello, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		host.Update()
		for _, ev := range client.Poll() {
			if ev.Kind == netcode.EventDisconnected {
				if ev.Err == nil {
					t.Fatal("expected a rejection reason")
				}
				if len(host.Players) != 1 {
					t.Errorf("host has %d players, want 1", len(host.Players))
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("client was not rejected")
}

func TestJoinUnreachable(t *testing.T) {
	l := Join("127.0.0.1:1", "Nobody", sim.ClassWarrior)
	deadline := time.Now().Add(10 * time.Second)
	for l.State != StateClosed && time.Now().Before(deadline) {
		l.Update()
		time.Sleep(20 * time.Millisecond)
	}
	if l.State != StateClosed || l.Err == nil {
		t.Fatalf("state = %v err = %v, want closed with error", l.State, l.Err)
	}
}

func TestClassIsShared(t *testing.T) {
	host, err := newHost("Host", sim.ClassHealer, "127.0.0.1", testPort+4, false)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Leave()
	addr := fmt.Sprintf("127.0.0.1:%d", testPort+4)
	m := Join(addr, "M", sim.ClassMage)
	bad := Join(addr, "X", sim.Class(200))
	defer m.Leave()
	defer bad.Leave()
	all := []*Lobby{host, m, bad}
	waitFor(t, "lobby", all, func() bool { return len(m.Players) == 3 && len(bad.Players) == 3 })
	classes := map[string]sim.Class{}
	for _, p := range m.Players {
		classes[p.Name] = p.Class
	}
	if classes["Host"] != sim.ClassHealer || classes["M"] != sim.ClassMage || classes["X"] != sim.ClassWarrior {
		t.Errorf("classes = %v, want Host healer, M mage, X warrior (invalid class)", classes)
	}
}
