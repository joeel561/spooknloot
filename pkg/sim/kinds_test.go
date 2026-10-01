package sim

import (
	"math/rand"
	"testing"
)

func TestKindStats(t *testing.T) {
	w := NewOpenWorld(nil, 1)
	for _, k := range regularKinds {
		w.SpawnKind(k, Vec2{}, 10)
	}
	health := map[MobKind]float32{}
	for _, m := range w.Mobs {
		health[m.Kind] = m.MaxHealth
	}
	if !(health[KindBat] < health[KindSkeleton1] && health[KindSkeleton1] < health[KindZombie]) {
		t.Errorf("health bat %v skeleton %v zombie %v, want bat < skeleton < zombie", health[KindBat], health[KindSkeleton1], health[KindZombie])
	}
	if !(KindZombie.stats().Speed < KindSkeleton1.stats().Speed && KindSkeleton1.stats().Speed < KindBat.stats().Speed) {
		t.Error("want zombie slower than skeleton slower than bat")
	}
	if !KindGhost.IsRanged() || KindZombie.IsRanged() || KindBoss.IsRanged() {
		t.Error("only ghosts are ranged")
	}
	if ProjectileSpeed >= 1.4 {
		t.Error("projectiles must be slower than a walking player (1.4) to be dodgeable")
	}

	// All regular kinds show up when spawning randomly.
	w = NewOpenWorld(nil, 1)
	pos := make([]Vec2, 300)
	w.SpawnRandom(pos, 5, rand.New(rand.NewSource(3)))
	seen := map[MobKind]bool{}
	for _, m := range w.Mobs {
		seen[m.Kind] = true
	}
	for _, k := range regularKinds {
		if !seen[k] {
			t.Errorf("kind %d never spawned", k)
		}
	}
}

func TestMeleeDamageDependsOnKind(t *testing.T) {
	hit := map[MobKind]float32{}
	for _, k := range []MobKind{KindBat, KindSkeleton1, KindZombie} {
		w := NewOpenWorld(nil, 1)
		w.SpawnKind(k, Vec2{100, 100}, 5)
		target := Target{ID: 1, Pos: w.Mobs[0].Center()}
		target.Pos.X += 10
		var hits []Hit
		for i := 0; i < 70; i++ {
			hits = append(hits, w.Update([]Target{target})...)
		}
		if len(hits) != 1 {
			t.Fatalf("kind %d landed %d hits, want 1", k, len(hits))
		}
		hit[k] = hits[0].Damage
	}
	// Skeletons deal the area's base damage; bats less, zombies more.
	if hit[KindSkeleton1] != 1 || !(hit[KindBat] < 1 && hit[KindZombie] > 1) {
		t.Errorf("hits bat %v skeleton %v zombie %v, want bat < 1 = skeleton < zombie", hit[KindBat], hit[KindSkeleton1], hit[KindZombie])
	}
}

func TestGhostShootsAndHits(t *testing.T) {
	w := NewOpenWorld(nil, 1)
	w.SpawnKind(KindGhost, Vec2{100, 100}, 5)
	g := &w.Mobs[0]
	target := Target{ID: 7, Pos: Vec2{g.Center().X + 100, g.Center().Y}}
	var hits []Hit
	for i := 0; i < rangedCooldown+120 && len(hits) == 0; i++ {
		hits = append(hits, w.Update([]Target{target})...)
	}
	// Ghost shots hit softer than a skeleton (base damage 1).
	if len(hits) != 1 || hits[0].Target != 7 || !(hits[0].Damage > 0 && hits[0].Damage < 1) {
		t.Fatalf("hits = %+v, want one projectile hit on 7", hits)
	}
	if d := Dist(g.Center(), target.Pos); d < rangedKeepAway {
		t.Errorf("ghost came too close: %v", d)
	}
	if len(w.Projectiles) != 0 {
		t.Error("projectile not removed after hitting")
	}
}

func TestGhostKeepsDistance(t *testing.T) {
	w := NewOpenWorld(nil, 1)
	w.SpawnKind(KindGhost, Vec2{100, 100}, 5)
	g := &w.Mobs[0]
	target := Target{ID: 1, Pos: Vec2{g.Center().X + 20, g.Center().Y}}
	start := Dist(g.Center(), target.Pos)
	for i := 0; i < 30; i++ {
		w.Update([]Target{target})
	}
	if d := Dist(g.Center(), target.Pos); d <= start {
		t.Errorf("ghost did not back off: %v -> %v", start, d)
	}
}

func TestProjectilesCanBeDodgedAndHitWalls(t *testing.T) {
	// Dodging: the player moves away from the shot's line after it is fired.
	w := NewOpenWorld(nil, 1)
	w.SpawnKind(KindGhost, Vec2{100, 100}, 5)
	g := &w.Mobs[0]
	target := Target{ID: 1, Pos: Vec2{g.Center().X + 100, g.Center().Y}}
	for i := 0; len(w.Projectiles) == 0; i++ {
		if i > rangedCooldown+10 {
			t.Fatal("ghost never fired")
		}
		if len(w.Update([]Target{target})) > 0 {
			t.Fatal("player 100 px away was hit the moment the ghost fired")
		}
	}
	target.Pos.Y += 30
	hits := 0
	for i := 0; i < 90; i++ {
		hits += len(w.Update([]Target{target}))
	}
	if hits != 0 {
		t.Error("projectile hit a player who stepped aside")
	}

	// Walls: a wall between ghost and player stops the shot.
	var walls []Rect
	for y := 0; y < 20; y++ {
		walls = append(walls, Rect{160, float32(y * TileSize), TileSize, TileSize})
	}
	w = NewOpenWorld(walls, 1)
	w.SpawnKind(KindGhost, Vec2{100, 100}, 5)
	target = Target{ID: 1, Pos: Vec2{220, 108}}
	hits = 0
	for i := 0; i < rangedCooldown+150; i++ {
		hits += len(w.Update([]Target{target}))
	}
	if hits != 0 {
		t.Error("projectile flew through a wall")
	}
}
