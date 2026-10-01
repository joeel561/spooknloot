package sim

// Difficulty scales with the number of players in a dungeon run.
// n is the player count; values for one player match singleplayer.

const (
	BaseMobHealth  = 5
	BaseBossHealth = 100
	BaseMobHit     = 0.9 // one mob attack outside the boss room
	BaseBossHit    = 1.8 // attacks in the boss room hit harder
	MaxMobsPerArea = 150
)

func scale(base float32, perExtraPlayer float32, n int) float32 {
	if n < 1 {
		n = 1
	}
	return base * (1 + perExtraPlayer*float32(n-1))
}

func ScaledMobCount(base, players int) int {
	return min(int(scale(float32(base), 0.5, players)+0.5), MaxMobsPerArea)
}

func ScaledMobHealth(players int) float32  { return scale(BaseMobHealth, 0.3, players) }
func ScaledMobHit(players int) float32     { return scale(BaseMobHit, 0.1, players) }
func ScaledBossHealth(players int) float32 { return scale(BaseBossHealth, 0.8, players) }
func ScaledBossHit(players int) float32    { return scale(BaseBossHit, 0.15, players) }
