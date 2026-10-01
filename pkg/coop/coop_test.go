package coop

import (
	"fmt"
	"sort"
	"testing"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/sim"
)

// hub is an in-memory network between a host and clients.
type hub struct {
	nets    map[PeerID]*fakeNet
	states  map[PeerID]lobby.PlayerState
	names   map[PeerID]string
	classes map[PeerID]sim.Class
}

type fakeNet struct {
	h     *hub
	id    PeerID
	inbox []Message
}

func (n *fakeNet) LocalID() PeerID { return n.id }
func (n *fakeNet) IsHost() bool    { return n.id == netcode.HostPeerID }
func (n *fakeNet) Send(to PeerID, data []byte, _ bool) {
	if dst := n.h.nets[to]; dst != nil {
		dst.inbox = append(dst.inbox, Message{From: n.id, Data: append([]byte(nil), data...)})
	}
}

func (n *fakeNet) Receive() []Message {
	in := n.inbox
	n.inbox = nil
	return in
}

func (n *fakeNet) Players() []PeerID {
	var out []PeerID
	for id := range n.h.nets {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (n *fakeNet) States() []lobby.PlayerState {
	var out []lobby.PlayerState
	for id, st := range n.h.states {
		if id != n.id && n.h.nets[id] != nil {
			out = append(out, st)
		}
	}
	return out
}

func (n *fakeNet) Name(id PeerID) string     { return n.h.names[id] }
func (n *fakeNet) Class(id PeerID) sim.Class { return n.h.classes[id] }

var testConfig = Config{
	WorldMobSpawns:  []sim.Vec2{{X: 300, Y: 300}},
	BossFloor:       []sim.Vec2{{X: 500, Y: 250}, {X: 520, Y: 250}, {X: 540, Y: 250}, {X: 560, Y: 250}, {X: 580, Y: 250}, {X: 600, Y: 250}},
	BossPlayerSpawn: sim.Vec2{X: 548, Y: 285},
	BossSpawn:       sim.Vec2{X: 548, Y: 200},
}

type game struct {
	t        *testing.T
	hub      *hub
	sessions map[PeerID]*Session
	events   map[PeerID][]Event
}

// newGame creates a host (ID 1) and clients 2..players.
func newGame(t *testing.T, players int) *game {
	g := &game{t: t, hub: &hub{nets: map[PeerID]*fakeNet{}, states: map[PeerID]lobby.PlayerState{}, names: map[PeerID]string{}, classes: map[PeerID]sim.Class{}},
		sessions: map[PeerID]*Session{}, events: map[PeerID][]Event{}}
	for i := 1; i <= players; i++ {
		id := PeerID(i)
		g.hub.nets[id] = &fakeNet{h: g.hub, id: id}
		g.hub.names[id] = fmt.Sprintf("P%d", i)
		// Everyone starts far away from the world mobs.
		g.place(id, sim.Vec2{X: 2000 + float32(i)*100, Y: 2000})
	}
	for i := 1; i <= players; i++ {
		id := PeerID(i)
		g.sessions[id] = NewSession(g.hub.nets[id], testConfig, 99)
		g.sessions[id].SetLocalState(g.hub.states[id])
	}
	return g
}

// place moves a player so its hitbox center is at p, in its current area.
func (g *game) place(id PeerID, p sim.Vec2) {
	area := lobby.AreaWorld
	if s := g.sessions[id]; s != nil {
		area = s.Area
	}
	st := lobby.PlayerState{ID: id, X: p.X - 24, Y: p.Y - 30, Area: area, Health: 100}
	g.hub.states[id] = st
	if s := g.sessions[id]; s != nil {
		s.SetLocalState(st)
	}
}

// step runs a few frames for everybody, host last so replies arrive.
func (g *game) step(frames int) {
	for i := 0; i < frames; i++ {
		for id := PeerID(len(g.sessions)); id >= 1; id-- {
			s := g.sessions[id]
			if s == nil {
				continue
			}
			s.Update()
			g.events[id] = append(g.events[id], s.TakeEvents()...)
			// Like the game: report the own state every frame, with the
			// area the player currently sees.
			st := g.hub.states[id]
			st.Area = s.Area
			g.hub.states[id] = st
			s.SetLocalState(st)
		}
	}
}

func (g *game) takeEvents(id PeerID, kind EventKind) []Event {
	var match, rest []Event
	for _, e := range g.events[id] {
		if e.Kind == kind {
			match = append(match, e)
		} else {
			rest = append(rest, e)
		}
	}
	g.events[id] = rest
	return match
}

// killAll lets player id kill every mob in its area.
func (g *game) killAll(id PeerID) {
	s := g.sessions[id]
	for round := 0; round < 400 && len(s.Mobs()) > 0; round++ {
		alive := false
		for _, m := range s.Mobs() {
			if m.Dying {
				continue
			}
			alive = true
			g.place(id, m.Center())
			s.Attack(m.ID)
			g.step(playerAttackCooldown + 1)
			break
		}
		if !alive {
			break
		}
	}
	g.step(10)
}

func TestSingleplayerDungeon(t *testing.T) {
	g := newGame(t, 1)
	s := g.sessions[1]
	g.step(5)
	if len(s.Mobs()) != worldMobCount {
		t.Fatalf("world has %d mobs, want %d", len(s.Mobs()), worldMobCount)
	}

	s.EnterDungeon()
	g.step(5)
	if len(g.takeEvents(1, EventEnterArea)) != 1 || s.Area != lobby.AreaDungeon || s.Layout == nil {
		t.Fatalf("did not enter the dungeon: area %v", s.Area)
	}
	if s.Invite != nil {
		t.Error("the initiator got an invite")
	}
	if n := len(s.Mobs()); n < 5 || n > 10 {
		t.Errorf("first level has %d mobs, want 5-10", n)
	}
	if len(s.Potions) != 1 {
		t.Errorf("dungeon has %d potions, want 1", len(s.Potions))
	}

	g.killAll(1)
	if !s.ExitOpen || len(g.takeEvents(1, EventExitOpened)) != 1 {
		t.Fatal("exit did not open after all mobs died")
	}
	epoch := s.Epoch
	g.place(1, sim.Vec2{X: s.Layout.Exit.X + 8, Y: s.Layout.Exit.Y + 8})
	s.ReachedExit()
	g.step(5)
	if s.Epoch == epoch || s.Level != 2 || s.Area != lobby.AreaDungeon {
		t.Fatalf("did not advance: epoch %d->%d level %d", epoch, s.Epoch, s.Level)
	}
}

func TestRunInviteJoinAndLateJoin(t *testing.T) {
	g := newGame(t, 4) // host 1, clients 2, 3, 4
	g.step(2)

	g.sessions[2].EnterDungeon()
	g.step(6)
	for _, id := range []PeerID{1, 3, 4} {
		inv := g.sessions[id].Invite
		if inv == nil || inv.From != "P2" {
			t.Fatalf("player %d invite = %+v, want one from P2", id, inv)
		}
	}
	if g.sessions[2].Area != lobby.AreaDungeon {
		t.Fatal("initiator not in the dungeon")
	}

	// Player 3 accepts and sees the very same dungeon.
	g.sessions[3].AcceptInvite()
	g.step(6)
	a, b := g.sessions[2], g.sessions[3]
	if b.Area != lobby.AreaDungeon || b.Epoch != a.Epoch || fmt.Sprint(b.Layout.Tiles) != fmt.Sprint(a.Layout.Tiles) {
		t.Fatal("joiner is not in the initiator's dungeon")
	}
	if len(a.Mobs()) == 0 || len(b.Mobs()) != len(a.Mobs()) {
		t.Errorf("joiner sees %d mobs, initiator %d", len(b.Mobs()), len(a.Mobs()))
	}

	// Player 4 ignores the invite; the host walks in later. That is a late
	// join and must not invite player 4 again.
	g.sessions[4].Invite = nil
	g.sessions[1].EnterDungeon()
	g.step(6)
	if g.sessions[1].Epoch != a.Epoch {
		t.Fatal("late joiner did not join the running dungeon")
	}
	if g.sessions[4].Invite != nil {
		t.Error("late join sent a new invite")
	}
}

func TestMobsTargetPlayersInTheirArea(t *testing.T) {
	g := newGame(t, 2)
	g.sessions[2].EnterDungeon()
	g.step(6)
	// Stand right on a melee mob (ghosts back off and shoot later) and
	// wait for attacks.
	var melee *sim.Mob
	for i, m := range g.hostState().run.area.mobs.Mobs {
		if !m.Kind.IsRanged() {
			melee = &g.hostState().run.area.mobs.Mobs[i]
			break
		}
	}
	if melee == nil {
		t.Fatal("no melee mob in the dungeon")
	}
	g.place(2, melee.Center())
	g.step(120)
	if len(g.takeEvents(2, EventDamage)) == 0 {
		t.Error("dungeon mob never hit the player standing on it")
	}
	if len(g.takeEvents(1, EventDamage)) != 0 {
		t.Error("host far away in the world took damage")
	}
}

func TestAttackOutOfRangeIsIgnored(t *testing.T) {
	g := newGame(t, 2)
	g.sessions[2].EnterDungeon()
	g.step(6)
	s := g.sessions[2]
	mob := g.hostState().run.area.mobs.Mobs[0]
	id := mob.ID
	c := mob.Center()
	g.place(2, sim.Vec2{X: c.X + 300, Y: c.Y})
	s.Attack(id)
	g.step(6)
	if g.hostState().run.area.mobs.Mob(id).Health < mob.MaxHealth {
		t.Fatal("mob took damage from a player 300px away")
	}
	// Control: the same attack from close by does hit.
	g.place(2, g.hostState().run.area.mobs.Mob(id).Center())
	s.Attack(id)
	g.step(6)
	if g.hostState().run.area.mobs.Mob(id).Health == mob.MaxHealth {
		t.Fatal("attack from close by did not hit either")
	}
}

func TestDeathAndDisconnectEndTheRun(t *testing.T) {
	g := newGame(t, 3)
	g.sessions[2].EnterDungeon()
	g.step(6)
	g.sessions[3].AcceptInvite()
	g.step(6)

	g.sessions[2].LeaveRun()
	g.step(6)
	if g.sessions[2].Area != lobby.AreaWorld {
		t.Fatal("player who died is not back in the world")
	}
	if g.hostState().run == nil {
		t.Fatal("run ended while player 3 is still inside")
	}

	// Player 3 disconnects: nobody left, the run is over.
	delete(g.hub.nets, 3)
	delete(g.sessions, 3)
	g.step(6)
	if g.hostState().run != nil {
		t.Fatal("run still exists without players")
	}
	if g.sessions[1].Invite != nil {
		t.Error("stale invite not removed after the run ended")
	}
}

func TestBossFightScalesAndCompletes(t *testing.T) {
	g := newGame(t, 2)
	g.sessions[2].EnterDungeon()
	g.step(6)
	g.sessions[1].AcceptInvite()
	g.step(6)
	g.sessions[1].DebugBoss()
	g.step(6)
	for _, id := range []PeerID{1, 2} {
		if g.sessions[id].Area != lobby.AreaBoss {
			t.Fatalf("player %d not in the boss room", id)
		}
	}
	boss := g.hostState().run.area.mobs.Boss()
	if boss.MaxHealth != sim.ScaledBossHealth(2) {
		t.Errorf("boss health %v, want %v for two players", boss.MaxHealth, sim.ScaledBossHealth(2))
	}

	g.killAll(2)
	for _, id := range []PeerID{1, 2} {
		if len(g.takeEvents(id, EventRunComplete)) != 1 || g.sessions[id].Area != lobby.AreaWorld {
			t.Errorf("player %d did not complete the run", id)
		}
	}
	if g.hostState().run != nil {
		t.Error("run still active after the boss died")
	}
}

func TestPotionHeals(t *testing.T) {
	g := newGame(t, 1)
	s := g.sessions[1]
	s.EnterDungeon()
	g.step(6)
	p := s.Potions[0]
	g.place(1, sim.Vec2{X: p.X + 8, Y: p.Y + 8})
	g.step(6)
	heals := g.takeEvents(1, EventHeal)
	if len(heals) != 1 || heals[0].Amount != potionHeal {
		t.Fatalf("heals = %+v", heals)
	}
	if len(s.Potions) != 0 {
		t.Error("potion still there after pickup")
	}
}

func TestMobChunksRoundTrip(t *testing.T) {
	var mobs []MobView
	for i := 0; i < 150; i++ {
		mobs = append(mobs, MobView{ID: uint16(i), Kind: sim.KindZombie, Pos: sim.Vec2{X: float32(i), Y: 2}, Health: 50})
	}
	chunks := encodeMobChunks(7, mobs)
	if len(chunks) != 3 {
		t.Fatalf("%d chunks, want 3", len(chunks))
	}
	var got []MobView
	for _, c := range chunks {
		if len(c) > 1100 {
			t.Errorf("chunk of %d bytes does not fit a datagram", len(c))
		}
		r := &reader{b: c[1:]}
		epoch, part := decodeMobChunk(r)
		if r.err != nil || epoch != 7 {
			t.Fatalf("decode: epoch %d err %v", epoch, r.err)
		}
		got = append(got, part...)
	}
	if fmt.Sprint(got) != fmt.Sprint(mobs) {
		t.Error("mobs changed in transit")
	}
	if r := (&reader{b: chunks[0][1:20]}); func() bool { decodeMobChunk(r); return r.err == nil }() {
		t.Error("truncated chunk decoded without error")
	}
}

func (g *game) hostState() *host { return g.sessions[1].host }

func TestKillsAreCountedForTheKiller(t *testing.T) {
	g := newGame(t, 2)
	g.sessions[2].EnterDungeon()
	g.step(6)
	total := len(g.sessions[2].Mobs())
	g.killAll(2)
	g.step(scoreSendEvery + 6)
	for _, id := range []PeerID{1, 2} {
		s := g.sessions[id]
		if s.Kills(2) != total || s.Kills(1) != 0 {
			t.Errorf("player %d sees kills %d/%d, want 0/%d", id, s.Kills(1), s.Kills(2), total)
		}
	}
}

func TestDyingMobGivesNoSecondKill(t *testing.T) {
	g := newGame(t, 1)
	s := g.sessions[1]
	s.EnterDungeon()
	g.step(6)
	m := s.Mobs()[0]
	g.place(1, m.Center())
	for i := 0; i < 5; i++ { // more hits than needed
		s.Attack(m.ID)
		g.step(playerAttackCooldown + 1)
	}
	g.step(scoreSendEvery)
	if s.Kills(1) != 1 {
		t.Errorf("kills = %d, want 1", s.Kills(1))
	}
}

func TestGhostShotsReachClients(t *testing.T) {
	g := newGame(t, 2)
	world := g.hostState().world
	world.mobs.Mobs = nil
	target := sim.Vec2{X: 2100, Y: 2000}
	g.place(1, sim.Vec2{X: 5000, Y: 5000}) // out of the way
	g.place(2, target)
	world.mobs.SpawnKind(sim.KindGhost, sim.Vec2{X: target.X - 108, Y: target.Y - 8}, 5)

	sawShot := false
	for i := 0; i < 400 && len(g.events[2]) == 0; i++ {
		g.step(1)
		if len(g.sessions[2].Projectiles()) > 0 {
			sawShot = true
		}
	}
	if !sawShot {
		t.Error("client never saw the ghost's shot")
	}
	if len(g.takeEvents(2, EventDamage)) == 0 {
		t.Error("ghost shot never hit the player")
	}
}
