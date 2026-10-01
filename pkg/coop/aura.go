package coop

import "spooknloot/pkg/sim"

// updateAura lets healers heal standing teammates near them a little every
// few seconds.
func (h *host) updateAura() {
	if h.frame%sim.AuraInterval != 0 {
		return
	}
	for healer, hp := range h.players {
		heal := h.s.net.Class(healer).Stats().AuraHeal
		area := h.areaOf(healer)
		if heal == 0 || hp.Health == 0 || hp.Area != area.kind {
			continue
		}
		for id, p := range h.players {
			if id == healer || h.areaOf(id) != area || p.Area != area.kind || p.Health == 0 || p.Health >= 100 {
				continue
			}
			if sim.Dist(playerCenter(hp), playerCenter(p)) > sim.AuraRadius {
				continue
			}
			h.s.hostSend(id, encodeHeal(heal, true), true)
		}
	}
}

// encodeHeal builds a heal message; quiet heals (aura) play no sound.
func encodeHeal(fraction float32, quiet bool) []byte {
	w := newMsg(msgHeal)
	w.f32(fraction)
	w.bool(quiet)
	return w.bytes()
}
