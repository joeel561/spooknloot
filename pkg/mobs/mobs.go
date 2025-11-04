package mobs

import (
	"math"
	"math/rand"

	"spooknloot/pkg/boss"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Mob struct {
	Sprite       rl.Texture2D
	OldX, OldY   float32
	Src          rl.Rectangle
	Dest         rl.Rectangle
	Dir          int
	Frame        int
	HitBox       rl.Rectangle
	FrameCount   int
	LastAttack   int
	IsAttacking  bool
	AttackTimer  int
	MaxHealth    float32
	Health       float32
	HealthbarDir int
	IsDead       bool
	DeathTimer   int
	Damage       bool
	DamageTimer  int
}

var (
	deathDuration     int     = 120
	attackRange       float32 = 25
	attackDuration    int     = 20
	attackCooldown    int     = 60
	mobs              []Mob
	mobTexture        rl.Texture2D
	batSprite         rl.Texture2D
	globalFrameCount  int
	externalColliders []rl.Rectangle
	skeletonSprite1   rl.Texture2D
	skeletonSprite2   rl.Texture2D
	skeletonSprite3   rl.Texture2D
	zombieSprite      rl.Texture2D
	defaultMobTypes   = []string{"bat", "skeleton1", "skeleton2", "skeleton3", "zombie"}

	flowField          [][]rl.Vector2
	flowBlocked        [][]bool
	flowW, flowH       int
	flowTileSize       int
	flowDirty          bool
	flowRecalcInterval int = 6
	lastFlowCalcFrame  int
)

type Direction int

const (
	DirIdleDown = Direction(iota)
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

func InitMobs() {
	batSprite = rl.LoadTexture("assets/mobs/bat-spritesheet.png")
	skeletonSprite1 = rl.LoadTexture("assets/mobs/skeleton_1.png")
	skeletonSprite2 = rl.LoadTexture("assets/mobs/skeleton_2.png")
	zombieSprite = rl.LoadTexture("assets/mobs/zombie.png")
	skeletonSprite3 = rl.LoadTexture("assets/mobs/skeleton_3.png")
	InitBoss()
}

func SpawnMobs(amount int, mobType string) int {
	if len(world.Spawn) == 0 || amount <= 0 {
		return len(mobs)
	}

	selectMobTexture := func(t string) rl.Texture2D {
		switch t {
		case "bat":
			return batSprite
		case "skeleton1":
			return skeletonSprite1
		case "skeleton2":
			return skeletonSprite2
		case "skeleton3":
			return skeletonSprite3
		case "zombie":
			return zombieSprite
		default:
			return skeletonSprite1
		}
	}

	for len(mobs) < amount {
		randomIndex := rand.Intn(len(world.Spawn))
		selectedTile := world.Spawn[randomIndex]
		x := float32(selectedTile.X * world.WorldMap.TileSize)
		y := float32(selectedTile.Y * world.WorldMap.TileSize)

		chosenType := mobType
		if mobType == "random" {
			chosenType = defaultMobTypes[rand.Intn(len(defaultMobTypes))]
		}
		sprite := selectMobTexture(chosenType)

		newMob := Mob{
			Sprite:       sprite,
			Src:          rl.NewRectangle(0, 0, 16, 16),
			Dest:         rl.NewRectangle(x, y, 16, 16),
			Dir:          int(DirIdleDown),
			Frame:        0,
			HitBox:       rl.NewRectangle(0, 0, 8, 8),
			FrameCount:   0,
			LastAttack:   0,
			IsAttacking:  false,
			AttackTimer:  0,
			MaxHealth:    5.0,
			Health:       5.0,
			HealthbarDir: 0,
			IsDead:       false,
			DeathTimer:   0,
			Damage:       false,
		}

		mobs = append(mobs, newMob)
	}

	return len(mobs)
}

func SpawnMobsAtPositions(positions []rl.Vector2, mobType string) int {
	if len(positions) == 0 {
		return len(mobs)
	}

	selectMobTexture := func(t string) rl.Texture2D {
		switch t {
		case "bat":
			return batSprite
		case "skeleton1":
			return skeletonSprite1
		case "skeleton2":
			return skeletonSprite2
		case "skeleton3":
			return skeletonSprite3
		case "zombie":
			return zombieSprite
		default:
			return skeletonSprite1
		}
	}

	for _, p := range positions {
		chosenType := mobType
		if mobType == "random" {
			chosenType = defaultMobTypes[rand.Intn(len(defaultMobTypes))]
		}
		sprite := selectMobTexture(chosenType)

		newMob := Mob{
			Sprite:       sprite,
			Src:          rl.NewRectangle(0, 0, 16, 16),
			Dest:         rl.NewRectangle(p.X, p.Y, 16, 16),
			Dir:          int(DirIdleDown),
			Frame:        0,
			HitBox:       rl.NewRectangle(0, 0, 8, 8),
			FrameCount:   0,
			LastAttack:   0,
			IsAttacking:  false,
			AttackTimer:  0,
			MaxHealth:    5.0,
			Health:       5.0,
			HealthbarDir: 0,
			IsDead:       false,
			DeathTimer:   0,
			Damage:       false,
		}
		mobs = append(mobs, newMob)
	}

	return len(mobs)
}

func MobMoving(playerPos rl.Vector2, attackPlayerFunc func()) {
	globalFrameCount++

	if len(externalColliders) > 0 {
		ensureFlowGrid()
		if flowDirty || globalFrameCount-lastFlowCalcFrame >= flowRecalcInterval {
			buildFlowFieldTowards(playerPos)
			lastFlowCalcFrame = globalFrameCount
			flowDirty = false
		}
	}

	for i := range mobs {

		mobs[i].OldX, mobs[i].OldY = mobs[i].Dest.X, mobs[i].Dest.Y
		mobs[i].Src.X = mobs[i].Src.Width * float32(mobs[i].Frame)

		if mobs[i].Health <= 0 && !mobs[i].IsDead {
			continue
		}

		if mobs[i].FrameCount%10 == 1 {
			mobs[i].Frame++
		}

		if mobs[i].IsDead {
			if mobs[i].Frame >= 1 {
				mobs[i].Frame = 0
			}
		} else {
			if mobs[i].Frame >= 4 {
				mobs[i].Frame = 0
			}
		}

		if mobs[i].DamageTimer > 0 {
			if mobs[i].Frame >= 2 {
				mobs[i].Frame = 0
			}
		} else {
			if mobs[i].Frame >= 4 {
				mobs[i].Frame = 0
			}
		}

		if mobs[i].Frame >= 4 {
			mobs[i].Frame = 0
		}

		mobs[i].FrameCount++

		if mobs[i].IsDead {
			mobs[i].Dir = int(DirDeadDown)
			mobs[i].DeathTimer++
			if mobs[i].DeathTimer >= deathDuration {
				mobs[i].IsDead = false
				mobs[i].DeathTimer = 0
			}
		}

		if mobs[i].DamageTimer > 0 {
			mobs[i].DamageTimer--
			if mobs[i].DamageTimer == 0 {
				mobs[i].Damage = false
			}
		}

		if !mobs[i].IsDead {
			if mobs[i].DamageTimer > 0 {
				mobs[i].IsAttacking = false
			} else {
				// Distance based on hitbox centers for accurate melee range
				mobCenterX := mobs[i].HitBox.X + (mobs[i].HitBox.Width / 2)
				mobCenterY := mobs[i].HitBox.Y + (mobs[i].HitBox.Height / 2)
				dist := rl.Vector2Distance(rl.NewVector2(mobCenterX, mobCenterY), playerPos)

				currentAttackRange := attackRange
				if i == bossIndex {
					currentAttackRange = 48
				}
				if dist <= currentAttackRange && globalFrameCount-mobs[i].LastAttack >= attackCooldown && !mobs[i].IsAttacking {
					mobs[i].LastAttack = globalFrameCount
					mobs[i].IsAttacking = true
					mobs[i].AttackTimer = attackDuration
				}

				if mobs[i].IsAttacking {
					mobs[i].Dir = int(DirAttackDown)
					mobs[i].AttackTimer--

					if mobs[i].AttackTimer <= attackDuration-3 && mobs[i].AttackTimer > attackDuration-6 {
						attackPlayerFunc()
					}
					if mobs[i].AttackTimer <= 0 {
						mobs[i].IsAttacking = false
					}
				}

				if !mobs[i].IsAttacking && dist < 180 && dist > 8 {

					directionX := playerPos.X - mobCenterX
					directionY := playerPos.Y - mobCenterY

					if len(externalColliders) > 0 {

						fx, fy, ok := sampleFlowAt(mobCenterX, mobCenterY)
						if ok {
							directionX, directionY = fx, fy
						}
					}

					length := rl.Vector2Length(rl.NewVector2(directionX, directionY))
					if length > 0 {
						directionX /= length
						directionY /= length
					}

					if float32(math.Abs(float64(directionX))) > float32(math.Abs(float64(directionY))) {
						if directionX > 0 {
							mobs[i].Dir = int(DirMoveRight)

							if mobs[i].IsAttacking {
								mobs[i].Dir = int(DirAttackRight)
							}
						} else {
							mobs[i].Dir = int(DirMoveLeft)
							if mobs[i].IsAttacking {
								mobs[i].Dir = int(DirAttackLeft)
							}
						}
					} else {
						if directionY > 0 {
							mobs[i].Dir = int(DirMoveDown)
							if mobs[i].IsAttacking {
								mobs[i].Dir = int(DirAttackDown)
							}
						} else {
							mobs[i].Dir = int(DirMoveUp)
						}
					}

					moveSpeed := float32(0.6)
					if i == bossIndex {
						moveSpeed = 0.9
					}

					stepX := directionX * (moveSpeed / 2)
					stepY := directionY * (moveSpeed / 2)
					mobs[i].Dest.X += stepX
					mobs[i].Dest.Y += stepY
					mobs[i].Dest.X += stepX
					mobs[i].Dest.Y += stepY
				}
			}
		}

		mobs[i].HitBox.X = mobs[i].Dest.X + (mobs[i].Dest.Width / 2) - mobs[i].HitBox.Width/2
		mobs[i].HitBox.Y = mobs[i].Dest.Y + (mobs[i].Dest.Height / 2) - mobs[i].HitBox.Height/2

		mobs[i].Src.Y = mobs[i].Src.Height * float32(mobs[i].Dir)

		resolveMobCollisions(i)
	}
}

func mobCollisionRects(mobIndex int, rects []rl.Rectangle) {
	if len(rects) == 0 {
		return
	}
	for i := 0; i < len(rects); i++ {
		if mobs[mobIndex].HitBox.X < rects[i].X+rects[i].Width &&
			mobs[mobIndex].HitBox.X+mobs[mobIndex].HitBox.Width > rects[i].X &&
			mobs[mobIndex].HitBox.Y < rects[i].Y+rects[i].Height &&
			mobs[mobIndex].HitBox.Y+mobs[mobIndex].HitBox.Height > rects[i].Y {
			mobs[mobIndex].Dest.X = mobs[mobIndex].OldX
			mobs[mobIndex].Dest.Y = mobs[mobIndex].OldY
		}
	}
}

func updateMobHitBox(mobIndex int) {
	mobs[mobIndex].HitBox.X = mobs[mobIndex].Dest.X + (mobs[mobIndex].Dest.Width / 2) - mobs[mobIndex].HitBox.Width/2
	mobs[mobIndex].HitBox.Y = mobs[mobIndex].Dest.Y + (mobs[mobIndex].Dest.Height / 2) - mobs[mobIndex].HitBox.Height/2
}

func mobHitboxCollidesWithTiles(hit rl.Rectangle, tiles []world.Tile) bool {
	if len(tiles) == 0 {
		return false
	}
	jsonMap := world.WorldMap
	for i := 0; i < len(tiles); i++ {
		tx := float32(tiles[i].X * jsonMap.TileSize)
		ty := float32(tiles[i].Y * jsonMap.TileSize)
		if hit.X < tx+float32(jsonMap.TileSize) &&
			hit.X+hit.Width > tx &&
			hit.Y < ty+float32(jsonMap.TileSize) &&
			hit.Y+hit.Height > ty {
			return true
		}
	}
	return false
}

func mobHitboxCollidesWithRects(hit rl.Rectangle, rects []rl.Rectangle) bool {
	for i := 0; i < len(rects); i++ {
		if hit.X < rects[i].X+rects[i].Width &&
			hit.X+hit.Width > rects[i].X &&
			hit.Y < rects[i].Y+rects[i].Height &&
			hit.Y+hit.Height > rects[i].Y {
			return true
		}
	}
	return false
}

func mobCollidesAny(mobIndex int) bool {
	if len(externalColliders) > 0 {
		return false
	}
	if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, world.Bushes) {
		return true
	}
	if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, world.Out) {
		return true
	}
	if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, world.Markets) {
		return true
	}
	if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, world.Buildings) {
		return true
	}

	if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, world.Trees) {
		return true
	}
	if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, world.Fence) {
		return true
	}
	if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, world.Lamps) {
		return true
	}

	if len(boss.Out) > 0 {
		bt := make([]world.Tile, 0, len(boss.Out))
		for _, t := range boss.Out {
			bt = append(bt, world.Tile{X: t.X, Y: t.Y})
		}
		if mobHitboxCollidesWithTiles(mobs[mobIndex].HitBox, bt) {
			return true
		}
	}

	return false
}

func resolveMobCollisions(mobIndex int) {
	if !mobCollidesAny(mobIndex) {
		return
	}

	newX := mobs[mobIndex].Dest.X

	mobs[mobIndex].Dest.X = mobs[mobIndex].OldX
	updateMobHitBox(mobIndex)
	if !mobCollidesAny(mobIndex) {
		return
	}

	mobs[mobIndex].Dest.X = newX
	mobs[mobIndex].Dest.Y = mobs[mobIndex].OldY
	updateMobHitBox(mobIndex)
	if !mobCollidesAny(mobIndex) {
		return
	}

	mobs[mobIndex].Dest.X = mobs[mobIndex].OldX
	mobs[mobIndex].Dest.Y = mobs[mobIndex].OldY
	updateMobHitBox(mobIndex)
}

func SetExternalColliders(colliders []rl.Rectangle) {
	externalColliders = colliders
	flowDirty = true
}

func ClearExternalColliders() {
	externalColliders = nil
	flowField = nil
	flowBlocked = nil
}

func DrawMobsHealthBar(mobIndex int) {
	if mobs[mobIndex].Health <= 0 {
		return
	}

	maxBarWidth := mobs[mobIndex].Dest.Width
	barHeight := float32(1)
	padding := float32(0)

	barX := mobs[mobIndex].Dest.X + padding
	barY := mobs[mobIndex].Dest.Y - barHeight - 2

	healthPercent := mobs[mobIndex].Health / mobs[mobIndex].MaxHealth
	if healthPercent < 0 {
		healthPercent = 0
	} else if healthPercent > 1 {
		healthPercent = 1
	}

	currentWidth := maxBarWidth * healthPercent

	bgRect := rl.NewRectangle(barX, barY, maxBarWidth, barHeight)
	rl.DrawRectangleRec(bgRect, rl.NewColor(0, 0, 0, 160))

	fgColor := rl.Color{R: 190, G: 75, B: 75, A: 255}
	if healthPercent <= 0.2 {
		fgColor = rl.Color{R: 57, G: 108, B: 60, A: 255}
	} else if healthPercent <= 0.5 {
		fgColor = rl.Color{R: 231, G: 152, B: 50, A: 255}
	}

	fgRect := rl.NewRectangle(barX, barY, currentWidth, barHeight)
	rl.DrawRectangleRec(fgRect, fgColor)
}

func GetMobPosition() rl.Vector2 {
	for i := range mobs {
		if mobs[i].Health > 0 && !mobs[i].IsDead {
			return rl.NewVector2(mobs[i].Dest.X, mobs[i].Dest.Y)
		}
	}

	return rl.NewVector2(0, 0)
}

func IsMobAlive() bool {
	for i := range mobs {
		if mobs[i].Health > 0 && !mobs[i].IsDead {
			return true
		}
	}
	return false
}

func GetMobPositionByIndex(index int) rl.Vector2 {
	if index < 0 || index >= len(mobs) || mobs[index].Health <= 0 || mobs[index].IsDead {
		return rl.NewVector2(0, 0)
	}

	return rl.NewVector2(mobs[index].Dest.X, mobs[index].Dest.Y)
}

func GetMobHitboxCenterByIndex(index int) rl.Vector2 {
	if index < 0 || index >= len(mobs) || mobs[index].Health <= 0 || mobs[index].IsDead {
		return rl.NewVector2(0, 0)
	}

	cx := mobs[index].HitBox.X + (mobs[index].HitBox.Width / 2)
	cy := mobs[index].HitBox.Y + (mobs[index].HitBox.Height / 2)
	return rl.NewVector2(cx, cy)
}

func GetClosestMobIndex(playerPos rl.Vector2) int {
	closestIndex := -1
	closestIndexDistance := float32(999999)

	for i := range mobs {
		if mobs[i].Health > 0 && !mobs[i].IsDead {
			// Use mob hitbox center for accurate closest selection
			cx := mobs[i].HitBox.X + (mobs[i].HitBox.Width / 2)
			cy := mobs[i].HitBox.Y + (mobs[i].HitBox.Height / 2)
			distance := rl.Vector2Distance(playerPos, rl.NewVector2(cx, cy))
			if distance < closestIndexDistance {
				closestIndexDistance = distance
				closestIndex = i
			}
		}
	}

	return closestIndex
}

func DrawMobs() {
	for i := range mobs {
		if mobs[i].Health > 0 || mobs[i].IsDead {
			rl.DrawTexturePro(mobs[i].Sprite, mobs[i].Src, mobs[i].Dest, rl.NewVector2(0, 0), 0, rl.White)
			if mobs[i].Health > 0 && !mobs[i].IsDead && i != bossIndex {
				DrawMobsHealthBar(i)
			}
		}
	}
}

func DamageMob(mobIndex int, damage float32) {
	if mobIndex < 0 || mobIndex >= len(mobs) {
		return
	}

	mobs[mobIndex].Damage = true
	mobs[mobIndex].DamageTimer = 6 // ~0.2s at 60 FPS
	switch Direction(mobs[mobIndex].Dir) {
	case DirMoveUp, DirAttackUp:
		mobs[mobIndex].Dir = int(DirDamageUp)
	case DirMoveLeft, DirAttackLeft:
		mobs[mobIndex].Dir = int(DirDamageLeft)
	case DirMoveRight, DirAttackRight:
		mobs[mobIndex].Dir = int(DirDamageRight)
	default:
		mobs[mobIndex].Dir = int(DirDamageDown)
	}
	mobs[mobIndex].Frame = 0

	wasAlive := mobs[mobIndex].Health > 0

	if mobIndex == bossIndex {
		damage *= 0.3
	}
	mobs[mobIndex].Health -= damage
	if mobs[mobIndex].Health < 0 {
		mobs[mobIndex].Health = 0
	}

	if wasAlive && mobs[mobIndex].Health <= 0 {
		mobs[mobIndex].IsDead = true
		mobs[mobIndex].DeathTimer = 0
	}

	healthPercentage := mobs[mobIndex].Health / mobs[mobIndex].MaxHealth
	if healthPercentage > 0.8 {
		mobs[mobIndex].HealthbarDir = 0
	} else if healthPercentage > 0.6 {
		mobs[mobIndex].HealthbarDir = 1
	} else if healthPercentage > 0.4 {
		mobs[mobIndex].HealthbarDir = 2
	} else if healthPercentage > 0.2 {
		mobs[mobIndex].HealthbarDir = 3
	} else {
		mobs[mobIndex].HealthbarDir = 4
	}
}

func ResetMobs() {
	mobs = []Mob{}
	globalFrameCount = 0
	bossIndex = -1
}

func UnloadMobsTexture() {
	rl.UnloadTexture(mobTexture)
}

// -----------------------
// Flow field (dungeon)
// -----------------------

// ensureFlowGrid initializes the flow grid dimensions to match the dungeon map.
func ensureFlowGrid() {
	if len(externalColliders) == 0 {
		return
	}
	// Dungeon uses fixed sizes from dungeon package
	// Infer grid bounds from colliders’ max extents and tile size (16)
	flowTileSize = 16
	maxX, maxY := 0, 0
	for i := 0; i < len(externalColliders); i++ {
		r := externalColliders[i]
		ex := int(r.X) + int(r.Width)
		ey := int(r.Y) + int(r.Height)
		if ex > maxX {
			maxX = ex
		}
		if ey > maxY {
			maxY = ey
		}
	}
	w := (maxX + flowTileSize - 1) / flowTileSize
	h := (maxY + flowTileSize - 1) / flowTileSize
	if w <= 0 || h <= 0 {
		return
	}
	if w == flowW && h == flowH && flowField != nil && flowBlocked != nil {
		return
	}
	flowW, flowH = w, h
	flowField = make([][]rl.Vector2, flowH)
	flowBlocked = make([][]bool, flowH)
	for y := 0; y < flowH; y++ {
		flowField[y] = make([]rl.Vector2, flowW)
		flowBlocked[y] = make([]bool, flowW)
	}
	flowDirty = true
}

// buildFlowFieldTowards computes a Dijkstra-from-target cost field and derives direction vectors.
func buildFlowFieldTowards(target rl.Vector2) {
	if flowField == nil || flowBlocked == nil || flowW == 0 || flowH == 0 {
		return
	}

	// Mark blocked cells from external colliders
	for y := 0; y < flowH; y++ {
		for x := 0; x < flowW; x++ {
			flowBlocked[y][x] = false
		}
	}
	for i := 0; i < len(externalColliders); i++ {
		r := externalColliders[i]
		minGX := int(r.X) / flowTileSize
		minGY := int(r.Y) / flowTileSize
		maxGX := int(r.X+r.Width-1) / flowTileSize
		maxGY := int(r.Y+r.Height-1) / flowTileSize
		if minGX < 0 {
			minGX = 0
		}
		if minGY < 0 {
			minGY = 0
		}
		if maxGX >= flowW {
			maxGX = flowW - 1
		}
		if maxGY >= flowH {
			maxGY = flowH - 1
		}
		for gy := minGY; gy <= maxGY; gy++ {
			for gx := minGX; gx <= maxGX; gx++ {
				flowBlocked[gy][gx] = true
			}
		}
	}

	// Target grid cell
	tx := int(target.X) / flowTileSize
	ty := int(target.Y) / flowTileSize
	if tx < 0 {
		tx = 0
	}
	if ty < 0 {
		ty = 0
	}
	if tx >= flowW {
		tx = flowW - 1
	}
	if ty >= flowH {
		ty = flowH - 1
	}
	// Allow standing on a floor cell; avoid marking target as blocked
	// If target happens to be inside a collider cell, we still seed there to pull vectors inward.

	// Dijkstra cost grid
	const inf = 1e9
	cost := make([][]int, flowH)
	for y := 0; y < flowH; y++ {
		cost[y] = make([]int, flowW)
		for x := 0; x < flowW; x++ {
			cost[y][x] = int(inf)
		}
	}
	// BFS/Dijkstra with 4-neighborhood (cardinal moves)
	type cell struct{ x, y int }
	q := make([]cell, 0, flowW*flowH)
	push := func(cx, cy int) {
		q = append(q, cell{cx, cy})
	}
	pop := func() (cell, bool) {
		if len(q) == 0 {
			return cell{}, false
		}
		c := q[0]
		q = q[1:]
		return c, true
	}

	cost[ty][tx] = 0
	push(tx, ty)
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for {
		c, ok := pop()
		if !ok {
			break
		}
		for _, d := range dirs {
			nx, ny := c.x+d[0], c.y+d[1]
			if nx < 0 || ny < 0 || nx >= flowW || ny >= flowH {
				continue
			}
			if flowBlocked[ny][nx] {
				continue
			}
			if cost[ny][nx] > cost[c.y][c.x]+1 {
				cost[ny][nx] = cost[c.y][c.x] + 1
				push(nx, ny)
			}
		}
	}

	// Derive per-cell direction vectors toward lowest-cost neighbor
	for y := 0; y < flowH; y++ {
		for x := 0; x < flowW; x++ {
			if flowBlocked[y][x] {
				flowField[y][x] = rl.NewVector2(0, 0)
				continue
			}
			best := cost[y][x]
			vx, vy := 0, 0
			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= flowW || ny >= flowH {
					continue
				}
				if cost[ny][nx] < best {
					best = cost[ny][nx]
					vx, vy = d[0], d[1]
				}
			}
			// Normalize to unit vector in world space
			flowField[y][x] = rl.NewVector2(float32(vx), float32(vy))
		}
	}
}

// sampleFlowAt returns a direction vector from the flow field at world position (x,y).
func sampleFlowAt(x, y float32) (fx, fy float32, ok bool) {
	if flowField == nil || flowW == 0 || flowH == 0 || flowTileSize == 0 {
		return 0, 0, false
	}
	gx := int(x) / flowTileSize
	gy := int(y) / flowTileSize
	if gx < 0 || gy < 0 || gx >= flowW || gy >= flowH {
		return 0, 0, false
	}
	v := flowField[gy][gx]
	if v.X == 0 && v.Y == 0 {
		return 0, 0, false
	}
	return v.X, v.Y, true
}
