package coop

import (
	"testing"

	"spooknloot/pkg/sim"
)

// mobInDungeon puts player 1 with the given class into a fresh dungeon and
// returns the host's first mob there.
func mobInDungeon(t *testing.T, class sim.Class) (*game, *sim.Mob) {
	g := newGame(t, 1)
	g.hub.classes[1] = class
	g.sessions[1].EnterDungeon()
	g.step(6)
	return g, &g.hostState().run.area.mobs.Mobs[0]
}

func TestMageHitsFromRangeWarriorDoesNot(t *testing.T) {
	for _, tc := range []struct {
		class   sim.Class
		wantHit bool
	}{{sim.ClassMage, true}, {sim.ClassWarrior, false}} {
		g, m := mobInDungeon(t, tc.class)
		c := m.Center()
		g.place(1, sim.Vec2{X: c.X + 80, Y: c.Y})
		g.sessions[1].Attack(m.ID)
		g.step(2)
		if hit := m.Health < m.MaxHealth; hit != tc.wantHit {
			t.Errorf("%s hitting from 80px: hit=%v, want %v", tc.class.Stats().Name, hit, tc.wantHit)
		}
	}
}

func TestClassDamage(t *testing.T) {
	for _, class := range sim.AllClasses {
		g, m := mobInDungeon(t, class)
		// Mob kinds differ in health; make sure one hit can't kill it.
		m.MaxHealth, m.Health = 10, 10
		g.place(1, m.Center())
		g.sessions[1].Attack(m.ID)
		g.step(2)
		if want := m.MaxHealth - class.Stats().Damage; m.Health != want {
			t.Errorf("%s: mob health %v, want %v", class.Stats().Name, m.Health, want)
		}
	}
}
func TestHealerAuraHealsNearbyTeammates(t *testing.T) {
	g := newGame(t, 3)
	g.hub.classes[1] = sim.ClassHealer
	g.place(1, sim.Vec2{X: 2000, Y: 2000})
	g.place(2, sim.Vec2{X: 2030, Y: 2000}) // in range
	g.place(3, sim.Vec2{X: 2300, Y: 2000}) // too far
	g.setHealth(2, 50)
	g.setHealth(3, 50)
	g.step(sim.AuraInterval + 6)

	heals := g.takeEvents(2, EventHeal)
	if len(heals) != 1 || heals[0].Amount != sim.ClassHealer.Stats().AuraHeal || !heals[0].Quiet {
		t.Errorf("near teammate heals = %+v", heals)
	}
	if len(g.takeEvents(3, EventHeal)) != 0 {
		t.Error("far teammate was healed")
	}
	if len(g.takeEvents(1, EventHeal)) != 0 {
		t.Error("healer healed itself")
	}
}

func TestHealerRevivesFaster(t *testing.T) {
	g := twoInRun(t)
	g.hub.classes[1] = sim.ClassHealer
	g.place(1, sim.Vec2{X: 100, Y: 100})
	g.place(2, sim.Vec2{X: 110, Y: 100})
	g.setHealth(2, 0)
	g.step(3)
	g.sessions[1].SetReviving(2)
	g.step(sim.ClassHealer.Stats().ReviveTime + 6)
	if len(g.takeEvents(2, EventRevived)) != 1 {
		t.Fatal("healer did not revive within its revive time")
	}
	if sim.ClassHealer.Stats().ReviveTime >= sim.ClassWarrior.Stats().ReviveTime {
		t.Error("healer should revive faster than a warrior")
	}
}

func TestPotionHealIsNotQuiet(t *testing.T) {
	g := newGame(t, 1)
	s := g.sessions[1]
	s.EnterDungeon()
	g.step(6)
	p := s.Potions[0]
	g.place(1, sim.Vec2{X: p.X + 8, Y: p.Y + 8})
	g.step(6)
	if h := g.takeEvents(1, EventHeal); len(h) != 1 || h[0].Quiet {
		t.Errorf("potion heal = %+v, want one non-quiet heal", h)
	}
}
