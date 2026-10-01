package coop

import (
	"spooknloot/pkg/sim"
)

// Players who drop to 0 health inside a dungeon run are downed instead of
// dead. A teammate holding E next to them for a few seconds revives them.
// Downed players bleed out after a while and come back on the next level.
// When nobody in the run is standing anymore, the run is over (wipe).

// LifeState is a run member's state as decided by the host.
type LifeState uint8

const (
	Alive LifeState = iota
	Downed
	BledOut
)

const (
	BleedOutTime  = 30 * 60 // frames
	ReviveTime    = 3 * 60  // frames of holding E
	ReviveReach   = 28      // pixels between player centers
	reviveSlack   = 12      // network delay allowance on the host
	ReviveHealth  = 0.3     // fraction of max health after a revive
	RespawnHealth = 0.5     // fraction after bleeding out, on the next level
	wipeDelay     = 90      // frames, lets the last death animation play
	// After a revive or level change the player's reported health is still
	// 0 for a moment; ignore it for this many frames.
	statusGrace = 60
)

type memberStatus struct {
	state      LifeState
	downFrame  int
	reviver    PeerID
	progress   int
	graceUntil int
}

func encodeStatus(id PeerID, st *memberStatus) []byte {
	w := newMsg(msgPlayerStatus)
	w.u32(uint32(id))
	w.u8(uint8(st.state))
	w.u32(uint32(st.reviver))
	return w.bytes()
}

func (h *host) broadcastStatus(id PeerID) {
	msg := encodeStatus(id, h.run.status[id])
	for m := range h.run.members {
		h.s.hostSend(m, msg, true)
	}
}

// sendStatuses tells a new run member who is currently down.
func (h *host) sendStatuses(to PeerID) {
	for id, st := range h.run.status {
		if st.state != Alive {
			h.s.hostSend(to, encodeStatus(id, st), true)
		}
	}
}

// resetStatuses stands everyone up, e.g. on a new level.
func (h *host) resetStatuses() {
	for id := range h.run.members {
		h.run.status[id] = &memberStatus{graceUntil: h.frame + statusGrace}
	}
	h.run.wipeTimer = 0
}

func (h *host) updateRevives() {
	r := h.run
	standing := 0
	for id := range r.members {
		st := r.status[id]
		p := h.players[id]
		switch st.state {
		case Alive:
			if h.frame >= st.graceUntil && p.Area == r.area.kind && p.Health == 0 {
				*st = memberStatus{state: Downed, downFrame: h.frame}
				h.broadcastStatus(id)
			}
		case Downed:
			if h.frame-st.downFrame >= BleedOutTime {
				*st = memberStatus{state: BledOut}
				h.broadcastStatus(id)
			} else if st.reviver != 0 {
				h.advanceRevive(id, st)
			}
		}
		if st.state == Alive {
			standing++
		}
	}

	if standing > 0 {
		r.wipeTimer = 0
		return
	}
	r.wipeTimer++
	if r.wipeTimer >= wipeDelay {
		for id := range r.members {
			h.s.hostSend(id, newMsg(msgWipe).bytes(), true)
		}
		h.endRun()
	}
}

func (h *host) advanceRevive(id PeerID, st *memberStatus) {
	if !h.canRevive(st.reviver, id) {
		st.reviver, st.progress = 0, 0
		h.broadcastStatus(id)
		return
	}
	st.progress++
	if st.progress < ReviveTime {
		return
	}
	*st = memberStatus{graceUntil: h.frame + statusGrace}
	w := newMsg(msgRevived)
	w.f32(ReviveHealth)
	h.s.hostSend(id, w.bytes(), true)
	h.broadcastStatus(id)
}

// canRevive checks that the reviver is up, close and in the same area.
func (h *host) canRevive(reviver, target PeerID) bool {
	if !h.inRun(reviver) || h.run.status[reviver].state != Alive {
		return false
	}
	rp, tp := h.players[reviver], h.players[target]
	if rp.Health == 0 || rp.Area != h.run.area.kind {
		return false
	}
	return sim.Dist(playerCenter(rp), playerCenter(tp)) <= ReviveReach+reviveSlack
}

func (h *host) handleReviveStart(from PeerID, target PeerID) {
	if !h.inRun(target) || !h.canRevive(from, target) {
		return
	}
	st := h.run.status[target]
	if st.state != Downed || st.reviver != 0 {
		return
	}
	st.reviver, st.progress = from, 0
	h.broadcastStatus(target)
}

func (h *host) handleReviveStop(from PeerID) {
	if h.run == nil {
		return
	}
	for id, st := range h.run.status {
		if st.reviver == from {
			st.reviver, st.progress = 0, 0
			h.broadcastStatus(id)
		}
	}
}
