package mobs

import (
	"spooknloot/pkg/coop"
	"spooknloot/pkg/sim"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// The ghost sheet has its own layout: 16x24 cells, measured from the
// image. Each entry is the top of a row; rows hold 4 frames (idle, move),
// 1 (dead) or 2 (damage). Dead and damage cells start 1 px further right.
var (
	ghostIdleRows   = [4]float32{1, 25, 49, 73}
	ghostMoveRows   = [4]float32{99, 123, 147, 171}
	ghostDeadRows   = [4]float32{423, 447, 471, 495}
	ghostDamageRows = [4]float32{521, 545, 569, 593}
)

const ghostW, ghostH = 16, 24

// ghostSource picks the sprite for a sim direction and frame.
func ghostSource(dir, frame int) rl.Rectangle {
	var top, left float32
	frames := 4
	switch {
	case dir >= sim.DirDamageDown && dir <= sim.DirDamageUp:
		top, left, frames = ghostDamageRows[dir-sim.DirDamageDown], 1, 2
	case dir >= sim.DirDeadUp && dir <= sim.DirDeadDown:
		top, left, frames = ghostDeadRows[dir-sim.DirDeadUp], 1, 1
	case dir >= sim.DirAttackDown && dir <= sim.DirAttackUp:
		// The sheet's attack frames are larger effects; casting uses the
		// move rows.
		top = ghostMoveRows[dir-sim.DirAttackDown]
	case dir >= sim.DirMoveDown && dir <= sim.DirMoveUp:
		top = ghostMoveRows[dir-sim.DirMoveDown]
	case dir >= sim.DirIdleDown && dir <= sim.DirIdleUp:
		top = ghostIdleRows[dir]
	default:
		top = ghostIdleRows[0]
	}
	return rl.NewRectangle(left+float32((frame%frames)*ghostW), top, ghostW, ghostH)
}

func drawGhost(tex rl.Texture2D, m coop.MobView) rl.Rectangle {
	// The cell is 8 px taller than the 16x16 mob; keep the feet in place.
	dest := rl.NewRectangle(m.Pos.X, m.Pos.Y-(ghostH-16), ghostW, ghostH)
	rl.DrawTexturePro(tex, ghostSource(int(m.Dir), int(m.Frame)), dest, rl.NewVector2(0, 0), 0, rl.White)
	return dest
}

// DrawProjectiles draws ghost shots as glowing orbs in world space.
func DrawProjectiles(shots []coop.ProjectileView) {
	for _, p := range shots {
		c := rl.NewVector2(p.Pos.X, p.Pos.Y)
		rl.DrawCircleV(c, 3.5, rl.NewColor(120, 255, 160, 90))
		rl.DrawCircleV(c, 2, rl.NewColor(190, 255, 210, 230))
	}
}
