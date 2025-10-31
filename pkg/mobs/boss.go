package mobs

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	bossSprite rl.Texture2D
	bossIndex  int = -1
)

func InitBoss() {
	bossSprite = rl.LoadTexture("assets/mobs/boss.png")
}

func SpawnBossAtPosition(p rl.Vector2) int {
	newMob := Mob{
		Sprite:       bossSprite,
		Src:          rl.NewRectangle(0, 0, 64, 64),
		Dest:         rl.NewRectangle(p.X, p.Y, 64, 64),
		Dir:          int(DirIdleDown),
		Frame:        0,
		HitBox:       rl.NewRectangle(0, 0, 32, 32),
		FrameCount:   0,
		LastAttack:   0,
		IsAttacking:  false,
		AttackTimer:  0,
		MaxHealth:    100.0,
		Health:       100.0,
		HealthbarDir: 0,
		IsDead:       false,
		DeathTimer:   0,
		Damage:       false,
	}

	mobs = append(mobs, newMob)
	bossIndex = len(mobs) - 1
	return bossIndex
}

func IsBossAlive() bool {
	if bossIndex < 0 || bossIndex >= len(mobs) {
		return false
	}
	return mobs[bossIndex].Health > 0 && !mobs[bossIndex].IsDead
}

func GetBossHealth() (current float32, max float32, ok bool) {
	if bossIndex < 0 || bossIndex >= len(mobs) {
		return 0, 0, false
	}
	return mobs[bossIndex].Health, mobs[bossIndex].MaxHealth, true
}
