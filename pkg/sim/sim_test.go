package sim

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestDungeonIsDeterministic(t *testing.T) {
	a, b := GenerateDungeon(1234), GenerateDungeon(1234)
	if fmt.Sprint(a.Tiles, a.Spawn, a.Exit, a.Decals, a.Torches) != fmt.Sprint(b.Tiles, b.Spawn, b.Exit, b.Decals, b.Torches) {
		t.Fatal("same seed produced different dungeons")
	}
	if fmt.Sprint(GenerateDungeon(1).Tiles) == fmt.Sprint(GenerateDungeon(2).Tiles) {
		t.Error("different seeds produced the same dungeon")
	}
}

func TestDungeonIsConnected(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		l := GenerateDungeon(seed)
		// Flood fill from spawn over floor and exit tiles must reach the exit.
		walkable := func(x, y int) bool {
			return x >= 0 && y >= 0 && x < DungeonW && y < DungeonH &&
				(l.Tiles[y][x] == TileFloor || l.Tiles[y][x] == TileExit)
		}
		seen := map[[2]int]bool{}
		stack := [][2]int{{int(l.Spawn.X) / TileSize, int(l.Spawn.Y) / TileSize}}
		for len(stack) > 0 {
			c := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[c] || !walkable(c[0], c[1]) {
				continue
			}
			seen[c] = true
			stack = append(stack, [2]int{c[0] + 1, c[1]}, [2]int{c[0] - 1, c[1]}, [2]int{c[0], c[1] + 1}, [2]int{c[0], c[1] - 1})
		}
		if !seen[[2]int{int(l.Exit.X) / TileSize, int(l.Exit.Y) / TileSize}] {
			t.Fatalf("seed %d: exit not reachable from spawn", seed)
		}
	}
}

func TestRandomFloorPositionsAreFloor(t *testing.T) {
	l := GenerateDungeon(7)
	ps := l.RandomFloorPositions(30)
	if len(ps) != 30 {
		t.Fatalf("got %d positions", len(ps))
	}
	seen := map[Vec2]bool{}
	for _, p := range ps {
		if l.Tiles[int(p.Y)/TileSize][int(p.X)/TileSize] != TileFloor {
			t.Errorf("%v is not floor", p)
		}
		if seen[p] {
			t.Errorf("%v picked twice", p)
		}
		seen[p] = true
	}
}

func TestMobsChaseNearestPlayer(t *testing.T) {
	w := NewOpenWorld(nil, 1)
	w.SpawnRandom([]Vec2{{100, 100}}, 5, rand.New(rand.NewSource(1)))
	near := Target{ID: 1, Pos: Vec2{150, 108}}
	far := Target{ID: 2, Pos: Vec2{0, 108}}
	start := w.Mobs[0].Center()
	for i := 0; i < 30; i++ {
		w.Update([]Target{near, far})
	}
	if c := w.Mobs[0].Center(); c.X <= start.X {
		t.Errorf("mob moved from %v to %v, expected towards the nearer player on the right", start, c)
	}
}

func TestMobAttacksAndRespectsCooldown(t *testing.T) {
	w := NewOpenWorld(nil, 0.9)
	w.SpawnRandom([]Vec2{{100, 100}}, 5, rand.New(rand.NewSource(1)))
	target := Target{ID: 7, Pos: w.Mobs[0].Center()}
	target.Pos.X += 15 // in attack range, not moving

	var hits []Hit
	for i := 0; i < 190; i++ {
		hits = append(hits, w.Update([]Target{target})...)
	}
	// Like before, mobs attack every 60 frames starting at frame 60, so
	// 190 frames allow exactly 3 attacks.
	if len(hits) != 3 {
		t.Fatalf("got %d hits, want 3", len(hits))
	}
	if hits[0].Target != 7 || hits[0].Damage != 0.9 {
		t.Errorf("hit = %+v", hits[0])
	}
}

func TestMobDiesAndDisappears(t *testing.T) {
	w := NewOpenWorld(nil, 1)
	w.SpawnRandom([]Vec2{{0, 0}}, 5, rand.New(rand.NewSource(1)))
	id := w.Mobs[0].ID
	w.Damage(id, 2.5)
	if !w.Mobs[0].Alive() {
		t.Fatal("mob died from the first hit")
	}
	w.Damage(id, 2.5)
	if w.Mobs[0].Alive() || !w.Mobs[0].Dying {
		t.Fatal("mob should be dying")
	}
	if w.Damage(id, 2.5) {
		t.Error("dying mob took damage")
	}
	if w.AliveCount() != 0 {
		t.Error("dying mob counted as alive")
	}
	for i := 0; i < deathDuration+1; i++ {
		w.Update(nil)
	}
	w.RemoveGone()
	if len(w.Mobs) != 0 {
		t.Error("dead mob was not removed after its death animation")
	}
}

func TestBossTakesReducedDamage(t *testing.T) {
	w := NewFlowWorld(nil, 1)
	w.SpawnBoss(Vec2{0, 0}, 100)
	w.Damage(w.Boss().ID, 10)
	if h := w.Boss().Health; h != 94 {
		t.Errorf("boss health = %v, want 94", h)
	}
}

func TestMobsCollideWithWalls(t *testing.T) {
	// A wall column at x=128..144 between mob and player.
	var walls []Rect
	for y := 0; y < 20; y++ {
		walls = append(walls, Rect{128, float32(y * TileSize), TileSize, TileSize})
	}
	w := NewOpenWorld(walls, 1)
	w.SpawnRandom([]Vec2{{100, 100}}, 5, rand.New(rand.NewSource(1)))
	target := Target{ID: 1, Pos: Vec2{200, 108}}
	for i := 0; i < 200; i++ {
		w.Update([]Target{target})
	}
	if hb := w.Mobs[0].HitBox(); hb.X+hb.W > 128 {
		t.Errorf("mob walked through the wall: hitbox %v", hb)
	}
}

func TestFlowFieldLeadsAroundWalls(t *testing.T) {
	tested := 0
	for seed := int64(0); seed < 30; seed++ {
		l := GenerateDungeon(seed)
		target := Target{ID: 1, Pos: Vec2{l.Spawn.X + 8, l.Spawn.Y + 8}}
		// Mobs give up when the straight-line distance exceeds the chase
		// range (as before), so only test starts whose walking path stays
		// within range, but which are not trivially close.
		path := newFlowField(newTileGrid(l.Walls))
		path.build(target.Pos)
		var starts []Vec2
		for _, p := range l.floorList {
			steps := path.cost[(int(p.Y)/TileSize)*path.grid.w+int(p.X)/TileSize]
			if steps >= 4 && float32(steps*TileSize) < chaseRange*0.7 {
				starts = append(starts, p)
			}
		}
		if len(starts) == 0 {
			continue
		}
		w := NewFlowWorld(l.Walls, 1)
		w.SpawnRandom(starts, 5, rand.New(rand.NewSource(1)))
		for i := 0; i < 900; i++ {
			w.Update([]Target{target})
		}
		for _, m := range w.Mobs {
			if d := Dist(m.Center(), target.Pos); d > attackRange+8 {
				t.Errorf("seed %d: mob stuck %v away from the player", seed, d)
			}
		}
		tested += len(starts)
	}
	if tested == 0 {
		t.Fatal("no mob positions tested")
	}
}

func TestScaling(t *testing.T) {
	if ScaledMobHealth(1) != BaseMobHealth || ScaledBossHealth(1) != BaseBossHealth || ScaledMobCount(10, 1) != 10 {
		t.Error("one player must match singleplayer values")
	}
	if got := ScaledMobCount(10, 3); got != 20 {
		t.Errorf("mob count for 3 players = %d, want 20", got)
	}
	if got := ScaledMobCount(20, 40); got != MaxMobsPerArea {
		t.Errorf("mob count for 40 players = %d, want cap %d", got, MaxMobsPerArea)
	}
	if got := ScaledBossHealth(2); got != 180 {
		t.Errorf("boss health for 2 players = %v, want 180", got)
	}
}

func TestClassStats(t *testing.T) {
	for _, c := range AllClasses {
		s := c.Stats()
		if s.Name == "" || s.MaxHealth <= 0 || s.Damage <= 0 || s.Range <= 0 || s.ReviveTime <= 0 {
			t.Errorf("class %d has incomplete stats: %+v", c, s)
		}
	}
	if Class(99).Stats().Name != ClassWarrior.Stats().Name || Class(99).Valid() {
		t.Error("unknown class must fall back to warrior")
	}
}
