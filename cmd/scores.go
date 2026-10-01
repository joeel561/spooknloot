package main

import (
	"sort"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/ui"
)

// scoreRows builds the scoreboard: every player with kills and ping,
// sorted by kills.
func scoreRows() []ui.ScoreRow {
	if session == nil {
		return nil
	}
	if gameLobby == nil {
		return []ui.ScoreRow{{
			Name:      lobby.SanitizeName(nameInput.Text),
			Class:     selectedClass,
			Kills:     session.Kills(netcode.HostPeerID),
			Ping:      "--",
			PingColor: ui.MutedTextColor,
			Local:     true,
		}}
	}

	pings := map[netcode.PeerID]uint16{}
	for _, s := range gameLobby.RemoteStates(0) {
		pings[s.ID] = s.Ping
	}
	rows := make([]ui.ScoreRow, 0, len(gameLobby.Players))
	for _, p := range gameLobby.Players {
		row := ui.ScoreRow{Name: p.Name, Kills: session.Kills(p.ID), Local: p.ID == gameLobby.LocalID, Class: p.Class}
		switch ms, ok := pings[p.ID]; {
		case row.Local:
			row.Ping, row.PingColor = ui.PingLabel(gameLobby)
		case p.Host:
			row.Ping, row.PingColor = "host", ui.MutedTextColor
		case ok:
			row.Ping, row.PingColor = ui.PingText(ms)
		default:
			row.Ping, row.PingColor = "--", ui.MutedTextColor
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Kills != rows[j].Kills {
			return rows[i].Kills > rows[j].Kills
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}
