// Package mobs draws the mobs. Their behaviour lives in pkg/sim and runs on
// the host; clients get positions and animation frames via pkg/coop.
package mobs

import (
	"spooknloot/pkg/coop"
	"spooknloot/pkg/sim"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var sprites = map[sim.MobKind]rl.Texture2D{}

func InitMobs() {
	sprites[sim.KindBat] = rl.LoadTexture("assets/mobs/bat-spritesheet.png")
	sprites[sim.KindSkeleton1] = rl.LoadTexture("assets/mobs/skeleton_1.png")
	sprites[sim.KindSkeleton2] = rl.LoadTexture("assets/mobs/skeleton_2.png")
	sprites[sim.KindSkeleton3] = rl.LoadTexture("assets/mobs/skeleton_3.png")
	sprites[sim.KindZombie] = rl.LoadTexture("assets/mobs/zombie.png")
	sprites[sim.KindBoss] = rl.LoadTexture("assets/mobs/boss.png")
}

func UnloadMobsTexture() {
	for kind, tex := range sprites {
		rl.UnloadTexture(tex)
		delete(sprites, kind)
	}
}

// DrawMobs draws mobs in world space, with health bars on regular mobs.
func DrawMobs(views []coop.MobView) {
	for _, m := range views {
		tex, ok := sprites[m.Kind]
		if !ok {
			continue
		}
		size := float32(16)
		if m.Kind == sim.KindBoss {
			size = 64
		}
		src := rl.NewRectangle(size*float32(m.Frame), size*float32(m.Dir), size, size)
		dest := rl.NewRectangle(m.Pos.X, m.Pos.Y, size, size)
		rl.DrawTexturePro(tex, src, dest, rl.NewVector2(0, 0), 0, rl.White)
		if !m.Dying && m.Kind != sim.KindBoss {
			drawHealthBar(dest, float32(m.Health)/100)
		}
	}
}

func drawHealthBar(dest rl.Rectangle, percent float32) {
	if percent <= 0 {
		return
	}
	percent = min(percent, 1)
	barY := dest.Y - 3
	rl.DrawRectangleRec(rl.NewRectangle(dest.X, barY, dest.Width, 1), rl.NewColor(0, 0, 0, 160))

	color := rl.Color{R: 190, G: 75, B: 75, A: 255}
	if percent <= 0.2 {
		color = rl.Color{R: 57, G: 108, B: 60, A: 255}
	} else if percent <= 0.5 {
		color = rl.Color{R: 231, G: 152, B: 50, A: 255}
	}
	rl.DrawRectangleRec(rl.NewRectangle(dest.X, barY, dest.Width*percent, 1), color)
}

// Closest returns the living mob nearest to pos.
func Closest(views []coop.MobView, pos rl.Vector2) (coop.MobView, bool) {
	var best coop.MobView
	bestDist, found := float32(0), false
	for _, m := range views {
		if m.Dying {
			continue
		}
		c := m.Center()
		d := rl.Vector2Distance(pos, rl.NewVector2(c.X, c.Y))
		if !found || d < bestDist {
			best, bestDist, found = m, d, true
		}
	}
	return best, found
}
