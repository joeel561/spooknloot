package mobs

import (
	"container/heap"
	"fmt"
	"math"
	"spooknloot/pkg/player"
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
}

type Node struct {
	X, Y     int
	G, H, F  float64
	Parent   *Node
	Walkable bool
	index    int
}

var (
	path      []*Node
	pathIndex int
	grid      [][]*Node
	ghost     Mob
)

type PriorityQueue []*Node

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].F < pq[j].F }
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *PriorityQueue) Push(x interface{}) { *pq = append(*pq, x.(*Node)) }
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	node := old[n-1]
	*pq = old[0 : n-1]
	return node
}

// Manhattan-Heuristik
func heuristic(a, b *Node) float64 {
	return math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y))
}

func aStar(start, goal *Node, grid [][]*Node) []*Node {
	openSet := &PriorityQueue{}
	heap.Init(openSet)
	heap.Push(openSet, start)

	cameFrom := make(map[*Node]*Node)
	gScore := make(map[*Node]float64)
	gScore[start] = 0

	fScore := make(map[*Node]float64)
	fScore[start] = heuristic(start, goal)

	for openSet.Len() > 0 {
		current := heap.Pop(openSet).(*Node)
		if current == goal {
			// Pfad zurückverfolgen
			var path []*Node
			for c := current; c != nil; c = cameFrom[c] {
				path = append([]*Node{c}, path...)
			}
			return path
		}

		neighbors := getNeighbors(current, grid)
		for _, neighbor := range neighbors {
			if !neighbor.Walkable {
				continue
			}
			tentativeG := gScore[current] + 1
			if g, ok := gScore[neighbor]; !ok || tentativeG < g {
				cameFrom[neighbor] = current
				gScore[neighbor] = tentativeG
				fScore[neighbor] = tentativeG + heuristic(neighbor, goal)
				heap.Push(openSet, neighbor)
			}
		}
	}
	return nil
}

func getNeighbors(n *Node, grid [][]*Node) []*Node {
	var neighbors []*Node
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, d := range dirs {
		nx, ny := n.X+d[0], n.Y+d[1]
		if nx >= 0 && ny >= 0 && nx < world.WorldMap.MapWidth && ny < world.WorldMap.MapHeight {
			neighbors = append(neighbors, grid[ny][nx])
		}
	}
	return neighbors
}

func InitMobs() {
	pathIndex = 0

	// Build grid with map dimensions (width x height)
	width := world.WorldMap.MapWidth
	height := world.WorldMap.MapHeight
	grid = make([][]*Node, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]*Node, width)
		for x := 0; x < width; x++ {
			grid[y][x] = &Node{X: x, Y: y, Walkable: true}
		}
	}

	for _, layer := range world.WorldMap.Layers {
		if layer.Collider {
			for _, t := range layer.Tiles {
				if t.Y >= 0 && t.Y < height && t.X >= 0 && t.X < width {
					grid[t.Y][t.X].Walkable = false
				}
			}
		}
	}

	ghost = Mob{
		Sprite:       ghostSprite,
		Src:          rl.NewRectangle(0, 0, 16, 30),
		Dest:         rl.NewRectangle(495, 526, 16, 30),
		Dir:          5,
		Frame:        0,
		HitBox:       rl.NewRectangle(0, 0, 10, 10),
		FrameCount:   0,
		LastAttack:   0,
		IsAttacking:  false,
		AttackTimer:  0,
		MaxHealth:    5.0,
		Health:       5.0,
		HealthbarDir: 0,
		IsDead:       false,
		DeathTimer:   0,
	}
}

func MobMoving(playerPos rl.Vector2) {
	globalFrameCount++

	ghost.OldX, ghost.OldY = ghost.Dest.X, ghost.Dest.Y
	ghost.Src.X = ghost.Src.Width * float32(ghost.Frame)

	if ghost.FrameCount%10 == 1 {
		ghost.Frame++
	}

	if ghost.Frame >= 4 {
		ghost.Frame = 0
	}

	ghost.FrameCount++

	if !ghost.IsDead {
		dist := rl.Vector2Distance(rl.NewVector2(ghost.Dest.X, ghost.Dest.Y), playerPos)

		fmt.Println(dist)

		if dist > 150 {
			findPlayer(playerPos)
		}

		if !ghost.IsAttacking && dist < 150 && dist > 5 {
			directionX := playerPos.X - ghost.Dest.X
			directionY := playerPos.Y - ghost.Dest.Y

			length := rl.Vector2Length(rl.NewVector2(directionX, directionY))
			if length > 0 {
				directionX /= length
				directionY /= length
			}

			moveSpeed := float32(0.8)

			ghost.Dest.X += directionX * moveSpeed
			ghost.Dest.Y += directionY * moveSpeed
		}
	}

	ghost.HitBox.X = ghost.Dest.X + (ghost.Dest.Width / 2) - ghost.HitBox.Width/2
	ghost.HitBox.Y = ghost.Dest.Y + (ghost.Dest.Height / 2) + ghostHitBoxYOffset

	/* 	GhostCollision(i, world.Buildings)
	   	GhostCollision(i, world.Fence)
	   	GhostCollision(world.Markets)
	   	GhostCollision( world.Out)
	   	GhostCollision( world.Trees)
	   	GhostCollision(world.Bushes) */
}

func DrawMobs() {
	for _, node := range path {
		rl.DrawRectangle(
			int32(node.X*world.WorldMap.TileSize),
			int32(node.Y*world.WorldMap.TileSize),
			int32(world.WorldMap.TileSize-2),
			int32(world.WorldMap.TileSize-2),
			rl.Yellow,
		)
	}

	rl.DrawTexturePro(ghost.Sprite, ghost.Src, ghost.Dest, rl.NewVector2(0, 0), 0, rl.White)
}

func findPlayer(playerPos rl.Vector2) {
	if player.PlayerMove || path == nil || len(path) == 0 || (len(path) > 0 && ghost.Dest.X == float32(path[len(path)-1].X*world.WorldMap.TileSize) && ghost.Dest.Y == float32(path[len(path)-1].Y*world.WorldMap.TileSize)) {
		// Recalculate path
		tileSize := world.WorldMap.TileSize
		enemyX := int(ghost.Dest.X) / tileSize
		enemyY := int(ghost.Dest.Y) / tileSize
		playerX := int(playerPos.X) / tileSize
		playerY := int(playerPos.Y) / tileSize

		fmt.Println("Enemy Tile:", enemyX, enemyY, "Player Tile:", playerX, playerY)

		// Bounds check ("Bonus Check") – jetzt korrekt mit Tile-Indizes
		/* 		if enemyY >= 0 && enemyY < len(grid) && enemyX >= 0 && enemyX < len(grid[0]) && playerY >= 0 && playerY < len(grid) && playerX >= 0 && playerX < len(grid[0]) { */
		start := grid[enemyY][enemyX]
		goal := grid[playerY][playerX]
		path = aStar(start, goal, grid)
		pathIndex = 0
		/* 		} */
	}
	fmt.Println("Path Length:", len(path), pathIndex)
	if len(path) > 1 && pathIndex < len(path)-1 {

		pathIndex++
		next := path[pathIndex]
		ghost.Dest.X = float32(next.X * world.WorldMap.TileSize)
		ghost.Dest.Y = float32(next.Y * world.WorldMap.TileSize)
	}
}
