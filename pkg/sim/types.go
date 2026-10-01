// Package sim contains the game simulation that the host runs for everyone:
// mobs and dungeon generation. It is plain Go without raylib, so it can be
// tested and later run on a dedicated server without a window.
package sim

import "math"

const TileSize = 16

type Vec2 struct{ X, Y float32 }

type Rect struct{ X, Y, W, H float32 }

func (r Rect) Overlaps(o Rect) bool {
	return r.X < o.X+o.W && r.X+r.W > o.X && r.Y < o.Y+o.H && r.Y+r.H > o.Y
}

func Dist(a, b Vec2) float32 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// tileGrid marks blocked 16px tiles for fast collision and path finding.
type tileGrid struct {
	w, h    int
	blocked []bool
}

func newTileGrid(colliders []Rect) *tileGrid {
	g := &tileGrid{}
	for _, r := range colliders {
		if ex := int(math.Ceil(float64(r.X+r.W) / TileSize)); ex > g.w {
			g.w = ex
		}
		if ey := int(math.Ceil(float64(r.Y+r.H) / TileSize)); ey > g.h {
			g.h = ey
		}
	}
	g.blocked = make([]bool, g.w*g.h)
	for _, r := range colliders {
		x0, y0 := int(r.X)/TileSize, int(r.Y)/TileSize
		x1, y1 := int(r.X+r.W-1)/TileSize, int(r.Y+r.H-1)/TileSize
		for y := max(y0, 0); y <= y1 && y < g.h; y++ {
			for x := max(x0, 0); x <= x1 && x < g.w; x++ {
				g.blocked[y*g.w+x] = true
			}
		}
	}
	return g
}

func (g *tileGrid) isBlocked(x, y int) bool {
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return false
	}
	return g.blocked[y*g.w+x]
}

// overlaps reports whether the rectangle touches any blocked tile.
func (g *tileGrid) overlaps(r Rect) bool {
	x0, y0 := int(math.Floor(float64(r.X)/TileSize)), int(math.Floor(float64(r.Y)/TileSize))
	x1, y1 := int(math.Floor(float64(r.X+r.W)/TileSize)), int(math.Floor(float64(r.Y+r.H)/TileSize))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if !g.isBlocked(x, y) {
				continue
			}
			tile := Rect{float32(x * TileSize), float32(y * TileSize), TileSize, TileSize}
			if r.Overlaps(tile) {
				return true
			}
		}
	}
	return false
}
