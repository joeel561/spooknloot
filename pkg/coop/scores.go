package coop

import "sort"

// The host counts kills per player (last hit wins) and shares the table
// with everyone whenever it changes.

const scoreSendEvery = 15 // frames, at most 4 updates per second

type scoreboard struct {
	kills map[PeerID]int
	dirty bool
	// sentPlayers is the player count at the last send, so new players
	// get the table too.
	sentPlayers int
}

func (h *host) creditKill(id PeerID) {
	if h.scores.kills == nil {
		h.scores.kills = map[PeerID]int{}
	}
	h.scores.kills[id]++
	h.scores.dirty = true
}

func (h *host) sendScores() {
	sb := &h.scores
	if h.frame%scoreSendEvery != 0 || (!sb.dirty && sb.sentPlayers == len(h.players)) {
		return
	}
	sb.dirty, sb.sentPlayers = false, len(h.players)

	ids := make([]PeerID, 0, len(sb.kills))
	for id := range sb.kills {
		if h.players[id] != nil {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	w := newMsg(msgScores)
	w.u8(uint8(len(ids)))
	for _, id := range ids {
		w.u32(uint32(id))
		w.u32(uint32(sb.kills[id]))
	}
	for id := range h.players {
		h.s.hostSend(id, w.bytes(), true)
	}
}

func (s *Session) receiveScores(r *reader) {
	n := int(r.u8())
	kills := make(map[PeerID]int, n)
	for i := 0; i < n && r.err == nil; i++ {
		id, k := PeerID(r.u32()), int(r.u32())
		kills[id] = k
	}
	if r.err == nil {
		s.kills = kills
	}
}

// Kills returns how many mobs a player has killed this game.
func (s *Session) Kills(id PeerID) int { return s.kills[id] }
