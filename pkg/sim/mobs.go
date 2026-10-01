package sim

import (
	"math"
	"math/rand"
)

// MobKind selects sprite and stats.
type MobKind uint8

const (
	KindBat MobKind = iota
	KindSkeleton1
	KindSkeleton2
	KindSkeleton3
	KindZombie
	KindBoss
)

var regularKinds = []MobKind{KindBat, KindSkeleton1, KindSkeleton2, KindSkeleton3, KindZombie}

// Sprite sheet rows, same order as the mob sprite sheets.
const (
	DirIdleDown = iota
	DirIdleLeft
	DirIdleRight
	DirIdleUp
	DirMoveDown
	DirMoveLeft
	DirMoveRight
	DirMoveUp
	DirAttackDown
	DirAttackLeft
	DirAttackRight
	DirAttackUp
	DirDeadUp
	DirDeadLeft
	DirDeadRight
	DirDeadDown
	DirDamageDown
	DirDamageLeft
	DirDamageRight
	DirDamageUp
)

const (
	deathDuration    = 120
	attackRange      = 25
	bossAttackRange  = 48
	attackDuration   = 20
	attackCooldown   = 60
	chaseRange       = 180
	moveSpeed        = 0.6
	bossMoveSpeed    = 0.9
	bossDamageTaken  = 0.6
	flowRecalcFrames = 6
	damageFlashTimer = 6
)

type Mob struct {
	ID        uint16
	Kind      MobKind
	Pos       Vec2 // top-left of the sprite
	Size      float32
	HitSize   float32
	Dir       int
	Frame     int
	MaxHealth float32
	Health    float32
	// Dying is true while the death animation plays after Health hit 0.
	Dying bool

	frameCount   int
	lastAttack   int
	attacking    bool
	attackTimer  int
	attackTarget uint32
	deathTimer   int
	damageTimer  int
	oldPos       Vec2
}

func (m *Mob) Center() Vec2 { return Vec2{m.Pos.X + m.Size/2, m.Pos.Y + m.Size/2} }

func (m *Mob) HitBox() Rect {
	c := m.Center()
	return Rect{c.X - m.HitSize/2, c.Y - m.HitSize/2, m.HitSize, m.HitSize}
}

// Alive means the mob can still fight.
func (m *Mob) Alive() bool { return m.Health > 0 && !m.Dying }

// Visible means the mob is alive or still playing its death animation.
func (m *Mob) Visible() bool { return m.Health > 0 || m.Dying }

// Target is a player the mobs can chase and attack.
type Target struct {
	ID  uint32
	Pos Vec2 // hitbox center
}

// Hit is damage a mob dealt to a player this frame.
type Hit struct {
	Target uint32
	Damage float32
}

// MobWorld simulates the mobs of one area.
type MobWorld struct {
	Mobs []Mob
	// HitDamage is the damage of a single mob attack.
	HitDamage float32

	frame   int
	nextID  uint16
	grid    *tileGrid
	useFlow bool
	flows   map[uint32]*flowField
}

// NewOpenWorld creates mobs that walk straight at players and slide along
// the given colliders (the world map).
func NewOpenWorld(colliders []Rect, hitDamage float32) *MobWorld {
	return &MobWorld{HitDamage: hitDamage, grid: newTileGrid(colliders), nextID: 1}
}

// NewFlowWorld creates mobs that path find around walls (dungeon, boss room).
func NewFlowWorld(walls []Rect, hitDamage float32) *MobWorld {
	return &MobWorld{HitDamage: hitDamage, grid: newTileGrid(walls), useFlow: true, flows: map[uint32]*flowField{}, nextID: 1}
}

func (w *MobWorld) add(m Mob) *Mob {
	m.ID = w.nextID
	w.nextID++
	m.Dir = DirIdleDown
	w.Mobs = append(w.Mobs, m)
	return &w.Mobs[len(w.Mobs)-1]
}

// SpawnRandom adds mobs of random regular kinds at the given positions.
func (w *MobWorld) SpawnRandom(positions []Vec2, health float32, rng *rand.Rand) {
	for _, p := range positions {
		w.add(Mob{
			Kind:      regularKinds[rng.Intn(len(regularKinds))],
			Pos:       p,
			Size:      16,
			HitSize:   8,
			MaxHealth: health,
			Health:    health,
		})
	}
}

func (w *MobWorld) SpawnBoss(p Vec2, health float32) {
	w.add(Mob{Kind: KindBoss, Pos: p, Size: 64, HitSize: 32, MaxHealth: health, Health: health})
}

// AliveCount returns how many mobs can still fight.
func (w *MobWorld) AliveCount() int {
	n := 0
	for i := range w.Mobs {
		if w.Mobs[i].Alive() {
			n++
		}
	}
	return n
}

func (w *MobWorld) Mob(id uint16) *Mob {
	for i := range w.Mobs {
		if w.Mobs[i].ID == id {
			return &w.Mobs[i]
		}
	}
	return nil
}

// Boss returns the boss mob, if there is one.
func (w *MobWorld) Boss() *Mob {
	for i := range w.Mobs {
		if w.Mobs[i].Kind == KindBoss {
			return &w.Mobs[i]
		}
	}
	return nil
}

// RemoveGone drops mobs that died and finished their death animation.
func (w *MobWorld) RemoveGone() {
	out := w.Mobs[:0]
	for _, m := range w.Mobs {
		if m.Visible() {
			out = append(out, m)
		}
	}
	w.Mobs = out
}

// Damage applies a player's hit. It returns false if the mob can't be hit.
func (w *MobWorld) Damage(id uint16, damage float32) bool {
	m := w.Mob(id)
	if m == nil || !m.Alive() {
		return false
	}
	m.damageTimer = damageFlashTimer
	switch m.Dir {
	case DirMoveUp, DirAttackUp:
		m.Dir = DirDamageUp
	case DirMoveLeft, DirAttackLeft:
		m.Dir = DirDamageLeft
	case DirMoveRight, DirAttackRight:
		m.Dir = DirDamageRight
	default:
		m.Dir = DirDamageDown
	}
	m.Frame = 0
	if m.Kind == KindBoss {
		damage *= bossDamageTaken
	}
	m.Health = max(m.Health-damage, 0)
	if m.Health <= 0 {
		m.Dying = true
		m.deathTimer = 0
	}
	return true
}

// Update advances all mobs by one frame (60 per second) and returns the
// hits they landed on players.
func (w *MobWorld) Update(targets []Target) []Hit {
	w.frame++
	if w.useFlow && w.frame%flowRecalcFrames == 1 {
		w.rebuildFlows(targets)
	}

	var hits []Hit
	for i := range w.Mobs {
		m := &w.Mobs[i]
		m.oldPos = m.Pos
		if !m.Visible() {
			continue
		}

		if m.frameCount%10 == 1 {
			m.Frame++
		}
		if m.Dying {
			if m.Frame >= 1 {
				m.Frame = 0
			}
		} else if m.Frame >= 4 {
			m.Frame = 0
		}
		if m.damageTimer > 0 && m.Frame >= 2 {
			m.Frame = 0
		}
		m.frameCount++

		if m.Dying {
			m.Dir = DirDeadDown
			m.deathTimer++
			if m.deathTimer >= deathDuration {
				m.Dying = false
			}
			continue
		}

		if m.damageTimer > 0 {
			m.damageTimer--
			m.attacking = false
			continue
		}

		if hit, ok := w.updateMob(m, targets); ok {
			hits = append(hits, hit)
		}
		if !w.useFlow {
			w.resolveCollision(m)
		}
	}
	return hits
}

func (w *MobWorld) updateMob(m *Mob, targets []Target) (Hit, bool) {
	center := m.Center()
	target, dist, found := nearest(center, targets)

	rangeLimit := float32(attackRange)
	speed := float32(moveSpeed)
	if m.Kind == KindBoss {
		rangeLimit = bossAttackRange
		speed = bossMoveSpeed
	}

	if found && dist <= rangeLimit && w.frame-m.lastAttack >= attackCooldown && !m.attacking {
		m.lastAttack = w.frame
		m.attacking = true
		m.attackTimer = attackDuration
		m.attackTarget = target.ID
	}

	var hit Hit
	landed := false
	if m.attacking {
		m.Dir = DirAttackDown
		m.attackTimer--
		// The attack connects a few frames into the animation.
		if m.attackTimer == attackDuration-3 {
			hit, landed = Hit{Target: m.attackTarget, Damage: w.HitDamage}, true
		}
		if m.attackTimer <= 0 {
			m.attacking = false
		}
	}

	if !found || m.attacking || dist >= chaseRange || dist <= 8 {
		return hit, landed
	}

	dx, dy := target.Pos.X-center.X, target.Pos.Y-center.Y
	if w.useFlow {
		if f := w.flows[target.ID]; f != nil {
			if fx, fy, ok := f.sample(center); ok {
				dx, dy = fx, fy
			}
		}
	}
	if l := float32(math.Hypot(float64(dx), float64(dy))); l > 0 {
		dx, dy = dx/l, dy/l
	}
	if abs(dx) > abs(dy) {
		if dx > 0 {
			m.Dir = DirMoveRight
		} else {
			m.Dir = DirMoveLeft
		}
	} else if dy > 0 {
		m.Dir = DirMoveDown
	} else {
		m.Dir = DirMoveUp
	}
	m.Pos.X += dx * speed
	m.Pos.Y += dy * speed
	return hit, landed
}

// resolveCollision slides along obstacles: try undoing X, then Y, then both.
func (w *MobWorld) resolveCollision(m *Mob) {
	if !w.grid.overlaps(m.HitBox()) {
		return
	}
	newX := m.Pos.X
	m.Pos.X = m.oldPos.X
	if !w.grid.overlaps(m.HitBox()) {
		return
	}
	m.Pos.X = newX
	m.Pos.Y = m.oldPos.Y
	if !w.grid.overlaps(m.HitBox()) {
		return
	}
	m.Pos = m.oldPos
}

func nearest(from Vec2, targets []Target) (Target, float32, bool) {
	best, bestDist, found := Target{}, float32(math.MaxFloat32), false
	for _, t := range targets {
		if d := Dist(from, t.Pos); d < bestDist {
			best, bestDist, found = t, d, true
		}
	}
	return best, bestDist, found
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (w *MobWorld) rebuildFlows(targets []Target) {
	seen := make(map[uint32]bool, len(targets))
	for _, t := range targets {
		seen[t.ID] = true
		f := w.flows[t.ID]
		if f == nil {
			f = newFlowField(w.grid)
			w.flows[t.ID] = f
		}
		f.build(t.Pos)
	}
	for id := range w.flows {
		if !seen[id] {
			delete(w.flows, id)
		}
	}
}

// flowField points every tile towards one target along the shortest path.
type flowField struct {
	grid *tileGrid
	dirs []Vec2
	cost []int
}

func newFlowField(g *tileGrid) *flowField {
	return &flowField{grid: g, dirs: make([]Vec2, g.w*g.h), cost: make([]int, g.w*g.h)}
}

func (f *flowField) build(target Vec2) {
	g := f.grid
	if g.w == 0 || g.h == 0 {
		return
	}
	const inf = math.MaxInt32
	for i := range f.cost {
		f.cost[i] = inf
	}
	tx := clamp(int(target.X)/TileSize, 0, g.w-1)
	ty := clamp(int(target.Y)/TileSize, 0, g.h-1)
	f.cost[ty*g.w+tx] = 0
	queue := []int{ty*g.w + tx}
	steps := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		cx, cy := c%g.w, c/g.w
		for _, s := range steps {
			nx, ny := cx+s[0], cy+s[1]
			if nx < 0 || ny < 0 || nx >= g.w || ny >= g.h || g.blocked[ny*g.w+nx] {
				continue
			}
			if n := ny*g.w + nx; f.cost[n] > f.cost[c]+1 {
				f.cost[n] = f.cost[c] + 1
				queue = append(queue, n)
			}
		}
	}
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			i := y*g.w + x
			f.dirs[i] = Vec2{}
			if g.blocked[i] {
				continue
			}
			best := f.cost[i]
			for _, s := range steps {
				nx, ny := x+s[0], y+s[1]
				if nx < 0 || ny < 0 || nx >= g.w || ny >= g.h {
					continue
				}
				if c := f.cost[ny*g.w+nx]; c < best {
					best = c
					f.dirs[i] = Vec2{float32(s[0]), float32(s[1])}
				}
			}
		}
	}
}

func (f *flowField) sample(p Vec2) (float32, float32, bool) {
	gx, gy := int(p.X)/TileSize, int(p.Y)/TileSize
	if gx < 0 || gy < 0 || gx >= f.grid.w || gy >= f.grid.h {
		return 0, 0, false
	}
	d := f.dirs[gy*f.grid.w+gx]
	if d.X == 0 && d.Y == 0 {
		return 0, 0, false
	}
	return d.X, d.Y, true
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
