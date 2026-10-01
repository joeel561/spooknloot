package coop

import (
	"spooknloot/pkg/sim"
)

// Killed mobs drop loot (gold, sometimes a health potion). The first player
// to walk over it picks it up. Inventories live on the host so items can't
// be lost or duplicated; each player is told about their own inventory.

type ItemKind uint8

const (
	ItemGold ItemKind = iota
	ItemPotion
)

const (
	MaxPotions       = 9
	PotionDrinkHeal  = 0.6 // fraction of max health
	dropLifetime     = 60 * 60
	dropSize         = 10
	potionDropChance = 0.2
	bossGold         = 50
	bossLootPotions  = 3
	dropResendEvery  = 120 // frames; players coming back to an area get the drops
)

// Drop is loot lying on the floor.
type Drop struct {
	ID     uint16
	Kind   ItemKind
	Amount uint16
	Pos    sim.Vec2 // center
}

type drop struct {
	Drop
	expires int
}

type inventory struct {
	gold    int
	potions int
}

func (h *host) inventory(id PeerID) *inventory {
	if h.inventories == nil {
		h.inventories = map[PeerID]*inventory{}
	}
	inv := h.inventories[id]
	if inv == nil {
		inv = &inventory{}
		h.inventories[id] = inv
	}
	return inv
}

func (h *host) sendInventory(id PeerID) {
	inv := h.inventory(id)
	w := newMsg(msgInventory)
	w.u32(uint32(inv.gold))
	w.u8(uint8(inv.potions))
	h.s.hostSend(id, w.bytes(), true)
}

func (h *host) addDrop(a *area, kind ItemKind, amount int, at sim.Vec2) {
	a.nextDropID++
	// Spread drops a little so several items don't stack on one spot.
	at.X += float32(h.rng.Intn(9) - 4)
	at.Y += float32(h.rng.Intn(9) - 4)
	a.drops = append(a.drops, drop{
		Drop:    Drop{ID: a.nextDropID, Kind: kind, Amount: uint16(amount), Pos: at},
		expires: h.frame + dropLifetime,
	})
	a.dropsDirty = true
}

// dropLoot is called when a mob dies. The boss drops nothing: the run ends
// right away, so its reward goes straight into the inventories instead.
func (h *host) dropLoot(a *area, m *sim.Mob) {
	if m.Kind == sim.KindBoss {
		return
	}
	c := m.Center()
	h.addDrop(a, ItemGold, 1+h.rng.Intn(3), c)
	if h.rng.Float32() < potionDropChance {
		h.addDrop(a, ItemPotion, 1, c)
	}
}

// rewardBoss gives everyone in the run the boss loot.
func (h *host) rewardBoss() {
	for id := range h.run.members {
		inv := h.inventory(id)
		inv.gold += bossGold
		inv.potions = min(inv.potions+bossLootPotions, MaxPotions)
		h.sendInventory(id)
	}
}

// updateDrops expires old loot and lets standing players pick it up.
func (h *host) updateDrops(a *area, pickers []sim.Target) {
	kept := a.drops[:0]
	for _, d := range a.drops {
		if h.frame >= d.expires {
			a.dropsDirty = true
			continue
		}
		if picker, ok := h.pickUp(d, pickers); ok {
			h.sendInventory(picker)
			a.dropsDirty = true
			continue
		}
		kept = append(kept, d)
	}
	a.drops = kept
}

func (h *host) pickUp(d drop, pickers []sim.Target) (PeerID, bool) {
	box := sim.Rect{X: d.Pos.X - dropSize/2, Y: d.Pos.Y - dropSize/2, W: dropSize, H: dropSize}
	for _, t := range pickers {
		id := PeerID(t.ID)
		if !playerHitBox(h.players[id]).Overlaps(box) {
			continue
		}
		inv := h.inventory(id)
		switch d.Kind {
		case ItemGold:
			inv.gold += int(d.Amount)
		case ItemPotion:
			if inv.potions >= MaxPotions {
				continue // bag full, leave it for someone else
			}
			inv.potions = min(inv.potions+int(d.Amount), MaxPotions)
		}
		return id, true
	}
	return 0, false
}

func (h *host) sendDrops(a *area, to []PeerID) {
	w := newMsg(msgDrops)
	w.u32(a.epoch)
	w.u16(uint16(len(a.drops)))
	for _, d := range a.drops {
		w.u16(d.ID)
		w.u8(uint8(d.Kind))
		w.u16(d.Amount)
		w.vec(d.Pos)
	}
	for _, id := range to {
		h.s.hostSend(id, w.bytes(), true)
	}
}

func (h *host) handleUsePotion(from PeerID) {
	inv := h.inventory(from)
	p := h.players[from]
	if inv.potions == 0 || p.Health == 0 || p.Health >= 100 {
		return
	}
	inv.potions--
	h.s.hostSend(from, encodeHeal(PotionDrinkHeal, false), true)
	h.sendInventory(from)
}

// Client side.

// UsePotion drinks a health potion from the inventory.
func (s *Session) UsePotion() {
	if s.PotionCount <= 0 {
		return
	}
	s.sendToHost(newMsg(msgUsePotion).bytes(), true)
}

func (s *Session) receiveInventory(r *reader) {
	gold, potions := int(r.u32()), int(r.u8())
	if r.err != nil {
		return
	}
	if gold > s.Gold {
		s.events = append(s.events, Event{Kind: EventLoot, Item: ItemGold, Count: gold - s.Gold})
	}
	if potions > s.PotionCount {
		s.events = append(s.events, Event{Kind: EventLoot, Item: ItemPotion, Count: potions - s.PotionCount})
	}
	s.Gold, s.PotionCount = gold, potions
}

func (s *Session) receiveDrops(r *reader) {
	epoch := r.u32()
	n := int(r.u16())
	drops := make([]Drop, 0, n)
	for i := 0; i < n && r.err == nil; i++ {
		drops = append(drops, Drop{ID: r.u16(), Kind: ItemKind(r.u8()), Amount: r.u16(), Pos: r.vec()})
	}
	if r.err == nil && epoch == s.Epoch {
		s.Drops = drops
	}
}
