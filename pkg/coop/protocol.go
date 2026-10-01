package coop

import (
	"encoding/binary"
	"errors"
	"math"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/sim"
)

// Message types; values start at lobby.MsgGameFirst so the lobby passes
// them through to us.
const (
	// client -> host
	msgEnterDungeon = lobby.MsgGameFirst + iota // touched the dungeon door
	msgJoinRun                                  // accepted an invite
	msgLeaveRun                                 // died inside the run
	msgReachedExit                              // standing on the open exit
	msgAttack                                   // hit a mob
	msgDebugBoss                                // B key, host only

	// host -> client
	msgEnterArea   // teleport into an area
	msgAreaState   // exit and potions of the current area
	msgMobs        // mob snapshot chunk, unreliable
	msgInvite      // someone started a dungeon run
	msgInviteEnded // the run is over, drop the invite
	msgDamage      // a mob hit you
	msgHeal        // you picked up a potion
	msgRunComplete // boss defeated
)

// Epoch identifies one instance of an area (the world, or one dungeon
// level). Messages for an old epoch are ignored after a level change.
const worldEpoch uint32 = 1

type writer struct{ b []byte }

func newMsg(t int) *writer       { return &writer{b: []byte{byte(t)}} }
func (w *writer) u8(v uint8)     { w.b = append(w.b, v) }
func (w *writer) u16(v uint16)   { w.b = binary.BigEndian.AppendUint16(w.b, v) }
func (w *writer) u32(v uint32)   { w.b = binary.BigEndian.AppendUint32(w.b, v) }
func (w *writer) i64(v int64)    { w.b = binary.BigEndian.AppendUint64(w.b, uint64(v)) }
func (w *writer) f32(v float32)  { w.u32(math.Float32bits(v)) }
func (w *writer) vec(v sim.Vec2) { w.f32(v.X); w.f32(v.Y) }
func (w *writer) bool(v bool) {
	if v {
		w.u8(1)
	} else {
		w.u8(0)
	}
}
func (w *writer) str(s string) {
	w.u8(uint8(min(len(s), 255)))
	w.b = append(w.b, s[:min(len(s), 255)]...)
}
func (w *writer) bytes() []byte { return w.b }

var errShort = errors.New("message too short")

// reader parses a message body. After the first error every read returns
// zero values and err stays set, so callers check err once at the end.
type reader struct {
	b   []byte
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil || len(r.b) < n {
		r.err = errShort
		return make([]byte, n)
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v
}

func (r *reader) u8() uint8   { return r.take(1)[0] }
func (r *reader) u16() uint16 { return binary.BigEndian.Uint16(r.take(2)) }
func (r *reader) u32() uint32 { return binary.BigEndian.Uint32(r.take(4)) }
func (r *reader) i64() int64  { return int64(binary.BigEndian.Uint64(r.take(8))) }
func (r *reader) bool() bool  { return r.u8() != 0 }
func (r *reader) str() string { return string(r.take(int(r.u8()))) }

func (r *reader) f32() float32 {
	v := math.Float32frombits(r.u32())
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		r.err = errors.New("invalid number")
		return 0
	}
	return v
}

func (r *reader) vec() sim.Vec2 { return sim.Vec2{X: r.f32(), Y: r.f32()} }

// enterArea tells a player to move into an area.
type enterArea struct {
	Area  lobby.Area
	Epoch uint32
	RunID uint32
	Level uint16
	Seed  int64 // dungeon layout seed
	Spawn sim.Vec2
}

func (m enterArea) encode() []byte {
	w := newMsg(msgEnterArea)
	w.u8(uint8(m.Area))
	w.u32(m.Epoch)
	w.u32(m.RunID)
	w.u16(m.Level)
	w.i64(m.Seed)
	w.vec(m.Spawn)
	return w.bytes()
}

func decodeEnterArea(r *reader) enterArea {
	return enterArea{Area: lobby.Area(r.u8()), Epoch: r.u32(), RunID: r.u32(), Level: r.u16(), Seed: r.i64(), Spawn: r.vec()}
}

type areaState struct {
	Epoch    uint32
	ExitOpen bool
	Potions  []sim.Vec2
}

func (m areaState) encode() []byte {
	w := newMsg(msgAreaState)
	w.u32(m.Epoch)
	w.bool(m.ExitOpen)
	w.u8(uint8(len(m.Potions)))
	for _, p := range m.Potions {
		w.vec(p)
	}
	return w.bytes()
}

func decodeAreaState(r *reader) areaState {
	m := areaState{Epoch: r.u32(), ExitOpen: r.bool()}
	n := int(r.u8())
	for i := 0; i < n && r.err == nil; i++ {
		m.Potions = append(m.Potions, r.vec())
	}
	return m
}

// MobView is what clients know about a mob.
type MobView struct {
	ID     uint16
	Kind   sim.MobKind
	Pos    sim.Vec2
	Dir    uint8
	Frame  uint8
	Health uint8 // percent
	Dying  bool
}

func (m MobView) Center() sim.Vec2 {
	size := float32(16)
	if m.Kind == sim.KindBoss {
		size = 64
	}
	return sim.Vec2{X: m.Pos.X + size/2, Y: m.Pos.Y + size/2}
}

func mobView(m *sim.Mob) MobView {
	return MobView{
		ID:     m.ID,
		Kind:   m.Kind,
		Pos:    m.Pos,
		Dir:    uint8(m.Dir),
		Frame:  uint8(m.Frame),
		Health: uint8(math.Ceil(float64(m.Health / m.MaxHealth * 100))),
		Dying:  m.Dying,
	}
}

const (
	mobRecordSize = 2 + 1 + 8 + 1 + 1 + 1 + 1
	// Keeps every chunk inside one QUIC datagram (~1200 bytes).
	mobsPerChunk = 70
)

// encodeMobChunks splits a snapshot into datagram sized chunks.
func encodeMobChunks(epoch uint32, mobs []MobView) [][]byte {
	var out [][]byte
	for start := 0; start < len(mobs) || start == 0; start += mobsPerChunk {
		chunk := mobs[start:min(start+mobsPerChunk, len(mobs))]
		w := newMsg(msgMobs)
		w.u32(epoch)
		w.u8(uint8(len(chunk)))
		for _, m := range chunk {
			w.u16(m.ID)
			w.u8(uint8(m.Kind))
			w.vec(m.Pos)
			w.u8(m.Dir)
			w.u8(m.Frame)
			w.u8(m.Health)
			w.bool(m.Dying)
		}
		out = append(out, w.bytes())
		if len(mobs) == 0 {
			break
		}
	}
	return out
}

func decodeMobChunk(r *reader) (uint32, []MobView) {
	epoch := r.u32()
	n := int(r.u8())
	mobs := make([]MobView, 0, n)
	for i := 0; i < n && r.err == nil; i++ {
		mobs = append(mobs, MobView{
			ID:     r.u16(),
			Kind:   sim.MobKind(r.u8()),
			Pos:    r.vec(),
			Dir:    r.u8(),
			Frame:  r.u8(),
			Health: r.u8(),
			Dying:  r.bool(),
		})
	}
	return epoch, mobs
}
