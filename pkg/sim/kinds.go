package sim

// Each mob kind plays differently: bats are quick and weak, zombies slow
// and tough, ghosts keep their distance and shoot slow projectiles that can
// be dodged.

type kindStats struct {
	Health float32 // multiplier on the area's base mob health
	Speed  float32 // pixels per frame
	Damage float32 // multiplier on the area's hit damage
	Ranged bool
}

var kindTable = map[MobKind]kindStats{
	KindBat:       {Health: 0.6, Speed: 0.9, Damage: 0.6},
	KindSkeleton1: {Health: 1, Speed: 0.6, Damage: 1},
	KindSkeleton2: {Health: 1, Speed: 0.6, Damage: 1},
	KindSkeleton3: {Health: 1, Speed: 0.6, Damage: 1},
	KindZombie:    {Health: 1.6, Speed: 0.4, Damage: 1.4},
	KindGhost:     {Health: 0.8, Speed: 0.5, Damage: 0.8, Ranged: true},
	KindBoss:      {Health: 1, Speed: 0.9, Damage: 1},
}

func (k MobKind) stats() kindStats {
	if s, ok := kindTable[k]; ok {
		return s
	}
	return kindTable[KindSkeleton1]
}

// IsRanged reports whether mobs of this kind attack from a distance.
func (k MobKind) IsRanged() bool { return k.stats().Ranged }

const (
	rangedKeepAway  = 60  // ghosts back off when players come closer
	rangedFireRange = 130 // and shoot when players are this close
	rangedCooldown  = 120 // frames between shots
	ProjectileSpeed = 1.2 // pixels per frame, slower than a walking player
	projectileTTL   = 120
	projectileHitR  = 6
	projectileSize  = 4
)

// Projectile is a ghost shot flying in a straight line.
type Projectile struct {
	ID     uint16
	Pos    Vec2
	Vel    Vec2
	Damage float32
	ttl    int
}

func (w *MobWorld) fire(from, at Vec2, damage float32) {
	dx, dy := at.X-from.X, at.Y-from.Y
	l := Dist(from, at)
	if l == 0 {
		return
	}
	w.nextProjectileID++
	w.Projectiles = append(w.Projectiles, Projectile{
		ID:     w.nextProjectileID,
		Pos:    from,
		Vel:    Vec2{dx / l * ProjectileSpeed, dy / l * ProjectileSpeed},
		Damage: damage,
		ttl:    projectileTTL,
	})
}

// updateProjectiles moves shots; they vanish on walls, after their range,
// or when they hit a player.
func (w *MobWorld) updateProjectiles(targets []Target) []Hit {
	var hits []Hit
	kept := w.Projectiles[:0]
	for _, p := range w.Projectiles {
		p.Pos.X += p.Vel.X
		p.Pos.Y += p.Vel.Y
		p.ttl--
		box := Rect{p.Pos.X - projectileSize/2, p.Pos.Y - projectileSize/2, projectileSize, projectileSize}
		if p.ttl <= 0 || w.grid.overlaps(box) {
			continue
		}
		hit := false
		for _, t := range targets {
			if Dist(p.Pos, t.Pos) <= projectileHitR {
				hits = append(hits, Hit{Target: t.ID, Damage: p.Damage})
				hit = true
				break
			}
		}
		if !hit {
			kept = append(kept, p)
		}
	}
	w.Projectiles = kept
	return hits
}

// updateRanged keeps a ghost at shooting distance and fires at the nearest
// player.
func (w *MobWorld) updateRanged(m *Mob, targets []Target) {
	center := m.Center()
	target, dist, found := nearest(center, targets)
	if !found || dist >= chaseRange {
		return
	}
	st := m.Kind.stats()

	if dist <= rangedFireRange && w.frame-m.lastAttack >= rangedCooldown {
		m.lastAttack = w.frame
		m.attacking = true
		m.attackTimer = attackDuration
		w.fire(center, target.Pos, w.HitDamage*st.Damage)
	}
	if m.attacking {
		m.Dir = DirAttackDown
		if m.attackTimer--; m.attackTimer <= 0 {
			m.attacking = false
		}
		return
	}

	dx, dy := target.Pos.X-center.X, target.Pos.Y-center.Y
	switch {
	case dist < rangedKeepAway:
		// Back off; also in dungeons this needs wall collision since the
		// flow field only leads towards players.
		w.move(m, -dx, -dy, st.Speed)
		w.resolveCollision(m)
	case dist > rangedFireRange:
		if w.useFlow {
			if f := w.flows[target.ID]; f != nil {
				if fx, fy, ok := f.sample(center); ok {
					dx, dy = fx, fy
				}
			}
		}
		w.move(m, dx, dy, st.Speed)
	default:
		m.Dir = DirIdleDown
	}
}
