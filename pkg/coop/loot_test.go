package coop

import (
	"testing"

	"spooknloot/pkg/sim"
)

// dungeonWithoutMobs puts player 1 alone into a dungeon with no mobs, so
// tests control all loot.
func dungeonWithoutMobs(t *testing.T, players int) (*game, *area) {
	g := newGame(t, players)
	g.sessions[1].EnterDungeon()
	g.step(6)
	a := g.hostState().run.area
	a.mobs.Mobs = nil
	a.potions = nil
	return g, a
}

func TestKilledMobDropsGoldThatCanBePickedUp(t *testing.T) {
	g := newGame(t, 1)
	s := g.sessions[1]
	s.EnterDungeon()
	g.step(6)
	a := g.hostState().run.area
	a.potions = nil
	m := &a.mobs.Mobs[0]
	// Kill it up close, then step back so the loot stays on the floor.
	for hits := 0; m.Alive(); hits++ {
		if hits > 20 {
			t.Fatal("mob does not die")
		}
		g.place(1, m.Center())
		s.Attack(m.ID)
		g.step(playerAttackCooldown + 1)
	}
	g.place(1, sim.Vec2{X: m.Center().X + 200, Y: m.Center().Y})
	// Picking up while standing on it may already have happened; count both.
	g.step(6)
	picked := s.Gold
	var gold *Drop
	for i := range s.Drops {
		if s.Drops[i].Kind == ItemGold {
			gold = &s.Drops[i]
		}
	}
	if gold == nil && picked == 0 {
		t.Fatal("killed mob dropped no gold")
	}
	if gold != nil {
		amount := int(gold.Amount)
		g.place(1, gold.Pos)
		g.step(6)
		if s.Gold != picked+amount {
			t.Fatalf("gold = %d, want %d", s.Gold, picked+amount)
		}
		for _, d := range s.Drops {
			if d.ID == gold.ID {
				t.Error("picked up gold is still on the floor")
			}
		}
	}
	if len(g.takeEvents(1, EventLoot)) == 0 {
		t.Error("no loot event for the pickup")
	}
}

func TestPotionPickupAndDrink(t *testing.T) {
	g, a := dungeonWithoutMobs(t, 1)
	s := g.sessions[1]
	spot := sim.Vec2{X: 100, Y: 100}
	g.hostState().addDrop(a, ItemPotion, 1, spot)
	g.place(1, a.drops[0].Pos)
	g.step(6)
	if s.PotionCount != 1 || len(s.Drops) != 0 {
		t.Fatalf("potions %d, drops %d after pickup", s.PotionCount, len(s.Drops))
	}

	// Full health: drinking would waste it, so nothing happens.
	s.UsePotion()
	g.step(6)
	if s.PotionCount != 1 || len(g.takeEvents(1, EventHeal)) != 0 {
		t.Fatal("potion used at full health")
	}

	g.setHealth(1, 40)
	s.UsePotion()
	g.step(6)
	heals := g.takeEvents(1, EventHeal)
	if s.PotionCount != 0 || len(heals) != 1 || heals[0].Amount != PotionDrinkHeal || heals[0].Quiet {
		t.Fatalf("after drinking: potions %d, heals %+v", s.PotionCount, heals)
	}

	s.UsePotion() // none left
	g.step(6)
	if len(g.takeEvents(1, EventHeal)) != 0 {
		t.Error("drank a potion that does not exist")
	}
}

func TestFullPotionBagLeavesPotionOnTheFloor(t *testing.T) {
	g, a := dungeonWithoutMobs(t, 1)
	g.hostState().inventory(1).potions = MaxPotions
	g.hostState().addDrop(a, ItemPotion, 1, sim.Vec2{X: 100, Y: 100})
	g.place(1, a.drops[0].Pos)
	g.step(6)
	if len(a.drops) != 1 {
		t.Fatal("potion picked up into a full bag")
	}
	if g.hostState().inventory(1).potions != MaxPotions {
		t.Error("potion count went over the maximum")
	}
}

func TestOnlyOnePlayerGetsADrop(t *testing.T) {
	g, a := dungeonWithoutMobs(t, 2)
	g.sessions[2].AcceptInvite()
	g.step(statusGrace)
	g.hostState().addDrop(a, ItemGold, 3, sim.Vec2{X: 100, Y: 100})
	pos := a.drops[0].Pos
	g.place(1, pos)
	g.place(2, pos)
	g.step(6)
	total := g.hostState().inventory(1).gold + g.hostState().inventory(2).gold
	if total != 3 {
		t.Errorf("gold handed out %d times the drop, want exactly 3 total", total)
	}
}

func TestDropsExpire(t *testing.T) {
	g, a := dungeonWithoutMobs(t, 1)
	g.hostState().addDrop(a, ItemGold, 1, sim.Vec2{X: 100, Y: 100})
	g.step(dropLifetime + 6)
	if len(a.drops) != 0 || len(g.sessions[1].Drops) != 0 {
		t.Error("drop did not expire")
	}
}

func TestBossRewardGoesToEveryone(t *testing.T) {
	g := newGame(t, 2)
	g.sessions[2].EnterDungeon()
	g.step(6)
	g.sessions[1].AcceptInvite()
	g.step(6)
	g.sessions[1].DebugBoss()
	g.step(6)
	g.killAll(2)
	for _, id := range []PeerID{1, 2} {
		s := g.sessions[id]
		// Player 2 also picks up loot of the other mobs on the way.
		if s.Gold < bossGold || s.PotionCount < bossLootPotions {
			t.Errorf("player %d: gold %d potions %d, want >= %d and %d", id, s.Gold, s.PotionCount, bossGold, bossLootPotions)
		}
	}
}
