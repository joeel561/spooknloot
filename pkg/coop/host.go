package coop

import (
	"math/rand"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/sim"
)

// Config holds the static map data the host simulation needs.
type Config struct {
	WorldColliders  []sim.Rect
	WorldMobSpawns  []sim.Vec2
	BossWalls       []sim.Rect
	BossFloor       []sim.Vec2
	BossPlayerSpawn sim.Vec2
	BossSpawn       sim.Vec2
}

const (
	worldMobCount      = 8
	worldRespawnFrames = 30 * 60
	DungeonLevels      = 20 // levels before the boss
	bossExtraMobs      = 19
	bossPotions        = 5
	potionHeal         = 0.6
	exitOpenDelay      = 6
	snapshotEvery      = 3 // frames, 20 Hz at 60 FPS
	playerAttackDamage = 2.5
	// Player attack range is 40; the slack covers network delay.
	playerAttackReach    = 40 + 24
	playerAttackCooldown = 10
	exitReach            = 32
)

// area is one simulated map instance: the world, a dungeon level or the
// boss room.
type area struct {
	kind         lobby.Area
	epoch        uint32
	seed         int64
	spawn        sim.Vec2
	mobs         *sim.MobWorld
	layout       *sim.DungeonLayout
	potions      []sim.Vec2
	exitOpen     bool
	clearFrames  int
	respawnTimer int
	dirty        bool
}

// run is the dungeon run a group of players is in. There is at most one;
// players who walk into the door later join it.
type run struct {
	id         uint32
	level      int
	spawnCount int
	area       *area
	members    map[PeerID]bool
	status     map[PeerID]*memberStatus
	wipeTimer  int
}

type host struct {
	s          *Session
	cfg        Config
	rng        *rand.Rand
	world      *area
	run        *run
	nextEpoch  uint32
	nextRunID  uint32
	frame      int
	local      lobby.PlayerState
	hasLocal   bool
	lastAttack map[PeerID]int
	scores     scoreboard
	// players is rebuilt every frame: everyone connected and their state.
	players map[PeerID]*lobby.PlayerState
}

func newHost(s *Session, cfg Config, seed int64) *host {
	h := &host{
		s:          s,
		cfg:        cfg,
		rng:        rand.New(rand.NewSource(seed)),
		nextEpoch:  worldEpoch + 1,
		nextRunID:  1,
		lastAttack: map[PeerID]int{},
	}
	h.world = &area{kind: lobby.AreaWorld, epoch: worldEpoch, mobs: sim.NewOpenWorld(cfg.WorldColliders, sim.BaseMobHit)}
	h.respawnWorldMobs()
	return h
}

func (h *host) respawnWorldMobs() {
	missing := worldMobCount - len(h.world.mobs.Mobs)
	if missing <= 0 || len(h.cfg.WorldMobSpawns) == 0 {
		return
	}
	pos := make([]sim.Vec2, missing)
	for i := range pos {
		pos[i] = h.cfg.WorldMobSpawns[h.rng.Intn(len(h.cfg.WorldMobSpawns))]
	}
	h.world.mobs.SpawnRandom(pos, sim.BaseMobHealth, h.rng)
}

func (h *host) inRun(id PeerID) bool { return h.run != nil && h.run.members[id] }

func (h *host) areaOf(id PeerID) *area {
	if h.inRun(id) {
		return h.run.area
	}
	return h.world
}

func (h *host) update(inbox []Message) {
	h.frame++
	h.collectPlayers()
	for _, m := range inbox {
		if h.players[m.From] != nil {
			h.handle(m.From, m.Data)
		}
	}
	if h.run != nil {
		var gone []PeerID
		for id := range h.run.members {
			if h.players[id] == nil {
				gone = append(gone, id) // disconnected
			}
		}
		for _, id := range gone {
			if h.run != nil {
				h.removeMember(id)
			}
		}
	}

	h.updateArea(h.world)
	if h.world.mobs.AliveCount() < worldMobCount {
		h.world.respawnTimer++
		if h.world.respawnTimer >= worldRespawnFrames {
			h.world.respawnTimer = 0
			h.respawnWorldMobs()
		}
	}
	if h.run != nil {
		h.updateArea(h.run.area)
		h.updateRunProgress()
	}
	if h.run != nil {
		h.updateRevives()
	}
	h.sendAreas()
	h.sendScores()
}

func (h *host) collectPlayers() {
	h.players = map[PeerID]*lobby.PlayerState{}
	for _, id := range h.s.net.Players() {
		h.players[id] = &lobby.PlayerState{ID: id, Area: lobby.Area(255)}
	}
	for _, st := range h.s.net.States() {
		if p := h.players[st.ID]; p != nil {
			*p = st
		}
	}
	if h.hasLocal {
		if p := h.players[h.s.net.LocalID()]; p != nil {
			*p = h.local
		}
	}
}

// members returns who is in an area according to the host.
func (h *host) members(a *area) []PeerID {
	var out []PeerID
	for id := range h.players {
		if h.areaOf(id) == a {
			out = append(out, id)
		}
	}
	return out
}

// Player states report the sprite's top-left corner; mobs aim at the hitbox.
func playerCenter(st *lobby.PlayerState) sim.Vec2 { return sim.Vec2{X: st.X + 24, Y: st.Y + 30} }

func playerHitBox(st *lobby.PlayerState) sim.Rect {
	c := playerCenter(st)
	return sim.Rect{X: c.X - 3, Y: c.Y - 3, W: 6, H: 6}
}

func (h *host) updateArea(a *area) {
	var targets []sim.Target
	for _, id := range h.members(a) {
		st := h.players[id]
		// The player's own view must agree, otherwise they are still
		// teleporting and their position belongs to the old area.
		if st.Area == a.kind && st.Health > 0 {
			targets = append(targets, sim.Target{ID: uint32(id), Pos: playerCenter(st)})
		}
	}
	for _, hit := range a.mobs.Update(targets) {
		w := newMsg(msgDamage)
		w.f32(hit.Damage)
		h.s.hostSend(PeerID(hit.Target), w.bytes(), true)
	}
	a.mobs.RemoveGone()

	for _, t := range targets {
		hb := playerHitBox(h.players[PeerID(t.ID)])
		for i := 0; i < len(a.potions); i++ {
			if hb.Overlaps(sim.Rect{X: a.potions[i].X, Y: a.potions[i].Y, W: sim.TileSize, H: sim.TileSize}) {
				a.potions = append(a.potions[:i], a.potions[i+1:]...)
				a.dirty = true
				w := newMsg(msgHeal)
				w.f32(potionHeal)
				h.s.hostSend(PeerID(t.ID), w.bytes(), true)
				break
			}
		}
	}
}

func (h *host) updateRunProgress() {
	a := h.run.area
	switch a.kind {
	case lobby.AreaDungeon:
		if a.mobs.AliveCount() > 0 {
			a.clearFrames = 0
		} else if !a.exitOpen {
			a.clearFrames++
			if a.clearFrames >= exitOpenDelay {
				a.exitOpen = true
				a.dirty = true
			}
		}
	case lobby.AreaBoss:
		if b := a.mobs.Boss(); b == nil || !b.Alive() {
			for id := range h.run.members {
				h.s.hostSend(id, newMsg(msgRunComplete).bytes(), true)
			}
			h.endRun()
		}
	}
}

func (h *host) sendAreas() {
	areas := []*area{h.world}
	if h.run != nil {
		areas = append(areas, h.run.area)
	}
	for _, a := range areas {
		members := h.members(a)
		if a.dirty {
			a.dirty = false
			msg := areaState{Epoch: a.epoch, ExitOpen: a.exitOpen, Potions: a.potions}.encode()
			for _, id := range members {
				h.s.hostSend(id, msg, true)
			}
		}
		if h.frame%snapshotEvery != 0 {
			continue
		}
		views := make([]MobView, 0, len(a.mobs.Mobs))
		for i := range a.mobs.Mobs {
			views = append(views, mobView(&a.mobs.Mobs[i]))
		}
		for _, chunk := range encodeMobChunks(a.epoch, views) {
			for _, id := range members {
				h.s.hostSend(id, chunk, false)
			}
		}
	}
}

func (h *host) handle(from PeerID, data []byte) {
	r := &reader{b: data[1:]}
	switch int(data[0]) {
	case msgEnterDungeon:
		if h.inRun(from) || h.players[from].Area != lobby.AreaWorld {
			return
		}
		if h.run == nil {
			h.startRun(from)
		} else {
			h.addMember(from)
		}
	case msgJoinRun:
		id := r.u32()
		if r.err == nil && h.run != nil && h.run.id == id && !h.inRun(from) {
			h.addMember(from)
		}
	case msgLeaveRun:
		if h.inRun(from) {
			h.removeMember(from)
		}
	case msgReachedExit:
		epoch := r.u32()
		if r.err != nil || !h.inRun(from) {
			return
		}
		a := h.run.area
		if a.kind != lobby.AreaDungeon {
			return
		}
		exit := sim.Vec2{X: a.layout.Exit.X + sim.TileSize/2, Y: a.layout.Exit.Y + sim.TileSize/2}
		if a.epoch == epoch && a.exitOpen && sim.Dist(playerCenter(h.players[from]), exit) <= exitReach {
			h.nextLevel()
		}
	case msgAttack:
		epoch, mobID := r.u32(), r.u16()
		if r.err != nil || h.frame-h.lastAttack[from] < playerAttackCooldown {
			return
		}
		a := h.areaOf(from)
		m := a.mobs.Mob(mobID)
		if a.epoch != epoch || m == nil || sim.Dist(playerCenter(h.players[from]), m.Center()) > playerAttackReach+m.HitSize/2 {
			return
		}
		h.lastAttack[from] = h.frame
		if a.mobs.Damage(mobID, playerAttackDamage) && m.Dying {
			h.creditKill(from)
		}
	case msgReviveStart:
		target := PeerID(r.u32())
		if r.err == nil && h.players[target] != nil {
			h.handleReviveStart(from, target)
		}
	case msgReviveStop:
		h.handleReviveStop(from)
	case msgDebugBoss:
		if from != h.s.net.LocalID() {
			return
		}
		if h.run == nil {
			h.startRun(from)
		}
		h.run.level = DungeonLevels
		h.nextLevel()
	}
}

func (h *host) startRun(initiator PeerID) {
	h.run = &run{id: h.nextRunID, members: map[PeerID]bool{}, status: map[PeerID]*memberStatus{}}
	h.nextRunID++
	h.setupDungeon(1)
	h.addMember(initiator)

	w := newMsg(msgInvite)
	w.u32(h.run.id)
	w.str(h.s.net.Name(initiator))
	for id := range h.players {
		if !h.inRun(id) {
			h.s.hostSend(id, w.bytes(), true)
		}
	}
}

func (h *host) addMember(id PeerID) {
	h.run.members[id] = true
	h.run.status[id] = &memberStatus{graceUntil: h.frame + statusGrace}
	h.sendEnterArea(id)
	h.sendStatuses(id)
}

func (h *host) removeMember(id PeerID) {
	delete(h.run.members, id)
	delete(h.run.status, id)
	h.handleReviveStop(id)
	if len(h.run.members) == 0 {
		h.endRun()
	}
}

func (h *host) endRun() {
	w := newMsg(msgInviteEnded)
	w.u32(h.run.id)
	for id := range h.players {
		h.s.hostSend(id, w.bytes(), true)
	}
	h.run = nil
}

func (h *host) sendEnterArea(id PeerID) {
	a := h.run.area
	h.s.hostSend(id, enterArea{
		Area:  a.kind,
		Epoch: a.epoch,
		RunID: h.run.id,
		Level: uint16(h.run.level),
		Seed:  a.seed,
		Spawn: a.spawn,
	}.encode(), true)
	h.s.hostSend(id, areaState{Epoch: a.epoch, ExitOpen: a.exitOpen, Potions: a.potions}.encode(), true)
}

func (h *host) nextLevel() {
	if h.run.level >= DungeonLevels {
		h.setupBoss()
	} else {
		h.setupDungeon(h.run.level + 1)
	}
	// Everyone starts the new level standing; clients revive themselves on
	// the area change if they were down.
	h.resetStatuses()
	for id := range h.run.members {
		h.sendEnterArea(id)
	}
}

func (h *host) newArea(kind lobby.Area) *area {
	a := &area{kind: kind, epoch: h.nextEpoch}
	h.nextEpoch++
	return a
}

func (h *host) groupSize() int { return max(len(h.run.members), 1) }

func (h *host) setupDungeon(level int) {
	r := h.run
	r.level = level
	// Same progression as singleplayer: 5-10 mobs, then 2-5 more per level.
	if r.spawnCount == 0 {
		r.spawnCount = 5 + h.rng.Intn(6)
	} else {
		r.spawnCount = min(r.spawnCount+2+h.rng.Intn(4), 20)
	}

	n := h.groupSize()
	a := h.newArea(lobby.AreaDungeon)
	a.seed = h.rng.Int63()
	a.layout = sim.GenerateDungeon(a.seed)
	a.spawn = a.layout.Spawn
	a.mobs = sim.NewFlowWorld(a.layout.Walls, sim.ScaledMobHit(n))
	a.mobs.SpawnRandom(a.layout.RandomFloorPositions(sim.ScaledMobCount(r.spawnCount, n)), sim.ScaledMobHealth(n), h.rng)
	a.potions = a.layout.RandomFloorPositions(1)
	r.area = a
}

func (h *host) setupBoss() {
	n := h.groupSize()
	a := h.newArea(lobby.AreaBoss)
	a.spawn = h.cfg.BossPlayerSpawn
	a.mobs = sim.NewFlowWorld(h.cfg.BossWalls, sim.ScaledBossHit(n))
	a.mobs.SpawnBoss(h.cfg.BossSpawn, sim.ScaledBossHealth(n))

	floor := append([]sim.Vec2(nil), h.cfg.BossFloor...)
	h.rng.Shuffle(len(floor), func(i, j int) { floor[i], floor[j] = floor[j], floor[i] })
	mobCount := min(sim.ScaledMobCount(bossExtraMobs, n), len(floor))
	a.mobs.SpawnRandom(floor[:mobCount], sim.ScaledMobHealth(n), h.rng)
	h.rng.Shuffle(len(floor), func(i, j int) { floor[i], floor[j] = floor[j], floor[i] })
	a.potions = floor[:min(bossPotions, len(floor))]

	h.run.level = DungeonLevels + 1
	h.run.area = a
}
