package coop

import (
	"time"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/sim"
)

const (
	inviteDuration   = 15 * time.Second
	mobTimeout       = 500 * time.Millisecond
	enterRetryDelay  = time.Second
	exitRetryDelay   = 250 * time.Millisecond
	maxMobSamples    = 8
	mobInterpolation = 100 * time.Millisecond
)

// reviveRetryDelay is a variable so tests can shorten it.
var reviveRetryDelay = 500 * time.Millisecond

type EventKind int

const (
	// EventEnterArea: move the player to Spawn in the new View.Area.
	EventEnterArea EventKind = iota
	// EventDamage: a mob hit the player for Amount.
	EventDamage
	// EventHeal: heal Amount (fraction of max health).
	EventHeal
	// EventExitOpened: all mobs dead, the dungeon exit appeared.
	EventExitOpened
	// EventRunComplete: the boss is dead, the player is back in the world.
	EventRunComplete
	// EventRevived: a teammate revived you; heal to Amount of max health.
	EventRevived
	// EventWipe: everybody in the run is down, the player is back in the world.
	EventWipe
)

// Status is what clients know about a run member's life state.
type Status struct {
	State   LifeState
	Since   time.Time // when State last changed (local time)
	Reviver PeerID
	// ReviveStart is when the current reviver started (local time).
	ReviveStart time.Time
}

type Event struct {
	Kind   EventKind
	Amount float32
	Spawn  sim.Vec2
}

// Invite is a pending "X entered the dungeon, press J to join".
type Invite struct {
	RunID uint32
	From  string
	Until time.Time
}

// Session is one player's view of the shared game, plus the simulation if
// this player is the host.
type Session struct {
	net  Net
	host *host

	// What this player currently sees.
	Area     lobby.Area
	Epoch    uint32
	Level    int
	Layout   *sim.DungeonLayout
	ExitOpen bool
	Potions  []sim.Vec2
	Invite   *Invite
	// Statuses holds run members who are not simply alive, by player.
	Statuses map[PeerID]Status

	reviveTarget PeerID
	mobs         map[uint16]*mobTrack
	kills        map[PeerID]int
	events       []Event
	hostInbox    []Message
	clientInbox  [][]byte
	lastEnter    time.Time
	lastExit     time.Time
	lastRevive   time.Time
}

type mobSample struct {
	at  time.Time
	mob MobView
}

type mobTrack struct{ samples []mobSample }

// NewSession starts a session. On the host it creates the simulation.
func NewSession(n Net, cfg Config, seed int64) *Session {
	s := &Session{net: n, Area: lobby.AreaWorld, Epoch: worldEpoch, mobs: map[uint16]*mobTrack{}, Statuses: map[PeerID]Status{}}
	if n.IsHost() {
		s.host = newHost(s, cfg, seed)
	}
	return s
}

// SetLocalState gives the host simulation the local player's state.
func (s *Session) SetLocalState(st lobby.PlayerState) {
	if s.host != nil {
		st.ID = s.net.LocalID()
		s.host.local = st
		s.host.hasLocal = true
	}
}

// Update processes network messages and, on the host, advances the
// simulation by one frame. Call it once per frame.
func (s *Session) Update() {
	for _, m := range s.net.Receive() {
		if len(m.Data) == 0 {
			continue
		}
		if s.host != nil {
			s.hostInbox = append(s.hostInbox, m)
		} else if m.From == netcode.HostPeerID {
			s.clientHandle(m.Data)
		}
	}
	if s.host != nil {
		inbox := s.hostInbox
		s.hostInbox = nil
		s.host.update(inbox)
	}
	for len(s.clientInbox) > 0 {
		msg := s.clientInbox[0]
		s.clientInbox = s.clientInbox[1:]
		s.clientHandle(msg)
	}
	if s.Invite != nil && time.Now().After(s.Invite.Until) {
		s.Invite = nil
	}
}

// TakeEvents returns what happened since the last call.
func (s *Session) TakeEvents() []Event {
	ev := s.events
	s.events = nil
	return ev
}

func (s *Session) sendToHost(data []byte, reliable bool) {
	if s.host != nil {
		s.hostInbox = append(s.hostInbox, Message{From: s.net.LocalID(), Data: data})
		return
	}
	s.net.Send(netcode.HostPeerID, data, reliable)
}

func (s *Session) hostSend(to PeerID, data []byte, reliable bool) {
	if to == s.net.LocalID() {
		s.clientInbox = append(s.clientInbox, data)
		return
	}
	s.net.Send(to, data, reliable)
}

// EnterDungeon is called while the player touches the dungeon door.
func (s *Session) EnterDungeon() {
	if s.Area != lobby.AreaWorld || time.Since(s.lastEnter) < enterRetryDelay {
		return
	}
	s.lastEnter = time.Now()
	s.sendToHost(newMsg(msgEnterDungeon).bytes(), true)
}

// AcceptInvite joins the dungeon run from the current invite.
func (s *Session) AcceptInvite() {
	if s.Invite == nil || s.Area != lobby.AreaWorld {
		return
	}
	w := newMsg(msgJoinRun)
	w.u32(s.Invite.RunID)
	s.sendToHost(w.bytes(), true)
	s.Invite = nil
}

// LeaveRun returns to the world after dying in a dungeon run.
func (s *Session) LeaveRun() {
	if s.Area == lobby.AreaWorld {
		return
	}
	s.sendToHost(newMsg(msgLeaveRun).bytes(), true)
	s.toWorld()
}

// ReachedExit is called while the player stands on the open exit.
func (s *Session) ReachedExit() {
	if !s.ExitOpen || time.Since(s.lastExit) < exitRetryDelay {
		return
	}
	s.lastExit = time.Now()
	w := newMsg(msgReachedExit)
	w.u32(s.Epoch)
	s.sendToHost(w.bytes(), true)
}

// Attack reports a hit on a mob; the host checks range and applies damage.
func (s *Session) Attack(mobID uint16) {
	w := newMsg(msgAttack)
	w.u32(s.Epoch)
	w.u16(mobID)
	s.sendToHost(w.bytes(), true)
}

// DebugBoss jumps straight to the boss (host only, B key).
func (s *Session) DebugBoss() { s.sendToHost(newMsg(msgDebugBoss).bytes(), true) }

// LocalStatus returns the local player's life state in the run.
func (s *Session) LocalStatus() Status { return s.Statuses[s.net.LocalID()] }

// SetReviving is called every frame with the downed player the local
// player is reviving (holding E next to), or 0 for none. Only changes are
// sent to the host.
func (s *Session) SetReviving(target PeerID) {
	if target == s.reviveTarget {
		// The host may have cancelled it (e.g. we were briefly out of
		// reach); ask again while still holding E.
		st := s.Statuses[target]
		if target == 0 || st.Reviver == s.net.LocalID() || time.Since(s.lastRevive) < reviveRetryDelay {
			return
		}
	}
	s.lastRevive = time.Now()
	if target == 0 {
		s.sendToHost(newMsg(msgReviveStop).bytes(), true)
	} else {
		w := newMsg(msgReviveStart)
		w.u32(uint32(target))
		s.sendToHost(w.bytes(), true)
	}
	s.reviveTarget = target
}

func (s *Session) toWorld() {
	s.Area, s.Epoch, s.Level, s.Layout = lobby.AreaWorld, worldEpoch, 0, nil
	s.ExitOpen, s.Potions = false, nil
	s.mobs = map[uint16]*mobTrack{}
	s.Statuses = map[PeerID]Status{}
	s.reviveTarget = 0
}

func (s *Session) clientHandle(data []byte) {
	r := &reader{b: data[1:]}
	switch int(data[0]) {
	case msgEnterArea:
		m := decodeEnterArea(r)
		if r.err != nil {
			return
		}
		s.Area, s.Epoch, s.Level = m.Area, m.Epoch, int(m.Level)
		s.Layout = nil
		if m.Area == lobby.AreaDungeon {
			s.Layout = sim.GenerateDungeon(m.Seed)
		}
		s.ExitOpen, s.Potions, s.Invite = false, nil, nil
		s.mobs = map[uint16]*mobTrack{}
		// Everyone starts a new area standing; the host sends who is down
		// to players joining mid-level.
		s.Statuses = map[PeerID]Status{}
		s.reviveTarget = 0
		s.events = append(s.events, Event{Kind: EventEnterArea, Spawn: m.Spawn})
	case msgAreaState:
		m := decodeAreaState(r)
		if r.err != nil || m.Epoch != s.Epoch {
			return
		}
		if m.ExitOpen && !s.ExitOpen {
			s.events = append(s.events, Event{Kind: EventExitOpened})
		}
		s.ExitOpen, s.Potions = m.ExitOpen, m.Potions
	case msgMobs:
		epoch, mobs := decodeMobChunk(r)
		if r.err != nil || epoch != s.Epoch {
			return
		}
		now := time.Now()
		for _, m := range mobs {
			t := s.mobs[m.ID]
			if t == nil {
				t = &mobTrack{}
				s.mobs[m.ID] = t
			}
			t.samples = append(t.samples, mobSample{now, m})
			if len(t.samples) > maxMobSamples {
				t.samples = t.samples[1:]
			}
		}
	case msgInvite:
		id, from := r.u32(), r.str()
		if r.err == nil && s.Area == lobby.AreaWorld {
			s.Invite = &Invite{RunID: id, From: from, Until: time.Now().Add(inviteDuration)}
		}
	case msgInviteEnded:
		if id := r.u32(); r.err == nil && s.Invite != nil && s.Invite.RunID == id {
			s.Invite = nil
		}
	case msgDamage:
		if a := r.f32(); r.err == nil && a > 0 {
			s.events = append(s.events, Event{Kind: EventDamage, Amount: a})
		}
	case msgHeal:
		if a := r.f32(); r.err == nil && a > 0 {
			s.events = append(s.events, Event{Kind: EventHeal, Amount: a})
		}
	case msgRunComplete:
		s.toWorld()
		s.events = append(s.events, Event{Kind: EventRunComplete})
	case msgPlayerStatus:
		id, state, reviver := PeerID(r.u32()), LifeState(r.u8()), PeerID(r.u32())
		if r.err != nil || s.Area == lobby.AreaWorld {
			return
		}
		old := s.Statuses[id]
		st := Status{State: state, Since: old.Since, Reviver: reviver, ReviveStart: old.ReviveStart}
		if state != old.State {
			st.Since = time.Now()
		}
		if reviver != old.Reviver {
			st.ReviveStart = time.Now()
		}
		if state == Alive {
			delete(s.Statuses, id)
		} else {
			s.Statuses[id] = st
		}
	case msgRevived:
		if a := r.f32(); r.err == nil {
			s.events = append(s.events, Event{Kind: EventRevived, Amount: a})
		}
	case msgScores:
		s.receiveScores(r)
	case msgWipe:
		s.toWorld()
		s.events = append(s.events, Event{Kind: EventWipe})
	}
}

// Mobs returns the mobs to draw, interpolated between snapshots.
func (s *Session) Mobs() []MobView {
	now := time.Now()
	renderTime := now.Add(-mobInterpolation)
	out := make([]MobView, 0, len(s.mobs))
	for id, t := range s.mobs {
		last := t.samples[len(t.samples)-1]
		if now.Sub(last.at) > mobTimeout {
			delete(s.mobs, id) // gone from the snapshots: removed on the host
			continue
		}
		out = append(out, t.at(renderTime))
	}
	return out
}

func (t *mobTrack) at(renderTime time.Time) MobView {
	s := t.samples
	for i := 1; i < len(s); i++ {
		if renderTime.Before(s[i].at) {
			a, b := s[i-1], s[i]
			if !renderTime.After(a.at) {
				return a.mob
			}
			f := float32(renderTime.Sub(a.at)) / float32(b.at.Sub(a.at))
			out := a.mob
			out.Pos.X += (b.mob.Pos.X - a.mob.Pos.X) * f
			out.Pos.Y += (b.mob.Pos.Y - a.mob.Pos.Y) * f
			return out
		}
	}
	return s[len(s)-1].mob
}

// Boss returns the boss health in percent, if a boss is in the area.
func (s *Session) Boss() (uint8, bool) {
	for _, t := range s.mobs {
		if m := t.samples[len(t.samples)-1].mob; m.Kind == sim.KindBoss {
			return m.Health, true
		}
	}
	return 0, false
}
