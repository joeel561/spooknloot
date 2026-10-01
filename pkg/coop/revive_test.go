package coop

import (
	"testing"
	"time"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/sim"
)

func (g *game) setHealth(id PeerID, percent uint8) {
	st := g.hub.states[id]
	st.Health = percent
	g.hub.states[id] = st
	g.sessions[id].SetLocalState(st)
}

// twoInRun puts players 1 (host) and 2 into the same dungeon, next to
// each other and away from the mobs' attacks.
func twoInRun(t *testing.T) *game {
	g := newGame(t, 2)
	g.sessions[2].EnterDungeon()
	g.step(6)
	g.sessions[1].AcceptInvite()
	g.step(6)
	// Remove the mobs so nobody takes damage during the test.
	g.hostState().run.area.mobs.Mobs = nil
	g.step(statusGrace + 1)
	return g
}

func TestDownedPlayerCanBeRevived(t *testing.T) {
	g := twoInRun(t)
	spot := sim.Vec2{X: 100, Y: 100}
	g.place(1, spot)
	g.place(2, sim.Vec2{X: spot.X + 10, Y: spot.Y})
	g.setHealth(2, 0)
	g.step(3)

	for _, id := range []PeerID{1, 2} {
		if st := g.sessions[id].Statuses[2]; st.State != Downed {
			t.Fatalf("player %d sees player 2 as %v, want downed", id, st.State)
		}
	}
	if g.hostState().run == nil {
		t.Fatal("run ended although player 1 is still standing")
	}

	g.sessions[1].SetReviving(2)
	g.step(sim.ClassWarrior.Stats().ReviveTime / 2)
	if st := g.sessions[2].Statuses[2]; st.Reviver != 1 {
		t.Fatalf("downed player does not see the reviver: %+v", st)
	}
	if len(g.takeEvents(2, EventRevived)) != 0 {
		t.Fatal("revived too early")
	}
	g.step(sim.ClassWarrior.Stats().ReviveTime/2 + 3)
	ev := g.takeEvents(2, EventRevived)
	if len(ev) != 1 || ev[0].Amount != ReviveHealth {
		t.Fatalf("revive events = %+v", ev)
	}
	g.setHealth(2, 30)
	g.step(3)
	if _, down := g.sessions[1].Statuses[2]; down {
		t.Error("player 2 still listed as down after the revive")
	}
}

func TestReviveStopsWhenWalkingAway(t *testing.T) {
	g := twoInRun(t)
	g.place(1, sim.Vec2{X: 100, Y: 100})
	g.place(2, sim.Vec2{X: 110, Y: 100})
	g.setHealth(2, 0)
	g.step(3)
	g.sessions[1].SetReviving(2)
	g.step(30)
	g.place(1, sim.Vec2{X: 300, Y: 100})
	g.step(sim.ClassWarrior.Stats().ReviveTime)
	if len(g.takeEvents(2, EventRevived)) != 0 {
		t.Fatal("revived from far away")
	}
	if st := g.sessions[1].Statuses[2]; st.State != Downed || st.Reviver != 0 {
		t.Errorf("status = %+v, want downed without reviver", st)
	}
}

func TestReviveFromTooFarIsRejected(t *testing.T) {
	g := twoInRun(t)
	g.place(1, sim.Vec2{X: 100, Y: 100})
	g.place(2, sim.Vec2{X: 200, Y: 100})
	g.setHealth(2, 0)
	g.step(3)
	g.sessions[1].SetReviving(2)
	g.step(3)
	if st := g.sessions[1].Statuses[2]; st.Reviver != 0 {
		t.Error("host accepted a revive from 100px away")
	}
}

func TestBledOutPlayerRespawnsOnNextLevel(t *testing.T) {
	g := twoInRun(t)
	g.place(1, sim.Vec2{X: 100, Y: 100})
	g.setHealth(2, 0)
	g.step(BleedOutTime + 3)
	if st := g.sessions[2].Statuses[2]; st.State != BledOut {
		t.Fatalf("status after bleeding out = %+v", st)
	}

	// Player 1 finishes the level alone.
	s := g.sessions[1]
	g.step(exitOpenDelay + 6)
	if !s.ExitOpen {
		t.Fatal("exit not open")
	}
	g.place(1, sim.Vec2{X: s.Layout.Exit.X + 8, Y: s.Layout.Exit.Y + 8})
	s.ReachedExit()
	g.step(6)
	if g.sessions[2].Level != 2 || len(g.sessions[2].Statuses) != 0 {
		t.Fatalf("bled out player: level %d statuses %v", g.sessions[2].Level, g.sessions[2].Statuses)
	}
	if len(g.takeEvents(2, EventEnterArea)) == 0 {
		t.Error("bled out player was not moved to the next level")
	}
}

func TestWipeWhenEveryoneIsDown(t *testing.T) {
	g := twoInRun(t)
	g.setHealth(1, 0)
	g.setHealth(2, 0)
	g.step(wipeDelay + 3)
	for _, id := range []PeerID{1, 2} {
		if len(g.takeEvents(id, EventWipe)) != 1 || g.sessions[id].Area != lobby.AreaWorld {
			t.Errorf("player %d was not sent back after the wipe", id)
		}
	}
	if g.hostState().run != nil {
		t.Error("run still active after the wipe")
	}
}

func TestSoloDeathEndsTheRun(t *testing.T) {
	g := newGame(t, 1)
	g.sessions[1].EnterDungeon()
	g.step(statusGrace + 6)
	g.setHealth(1, 0)
	g.step(wipeDelay + 3)
	if len(g.takeEvents(1, EventWipe)) != 1 {
		t.Fatal("dying alone did not end the run")
	}
}

func TestReviveRetriesAfterHostCancelled(t *testing.T) {
	defer func(d time.Duration) { reviveRetryDelay = d }(reviveRetryDelay)
	reviveRetryDelay = 0
	g := twoInRun(t)
	g.place(1, sim.Vec2{X: 100, Y: 100})
	g.place(2, sim.Vec2{X: 110, Y: 100})
	g.setHealth(2, 0)
	g.step(3)
	g.sessions[1].SetReviving(2)
	g.step(10)
	// Briefly out of reach (e.g. lag): the host cancels.
	g.place(1, sim.Vec2{X: 300, Y: 100})
	g.step(3)
	g.place(1, sim.Vec2{X: 100, Y: 100})
	// The client keeps holding E next to player 2 and asks again.
	for i := 0; i < 60; i++ {
		g.sessions[1].SetReviving(2)
		g.step(1)
	}
	if st := g.sessions[1].Statuses[2]; st.Reviver != 1 {
		t.Fatalf("revive did not restart: %+v", st)
	}
}
