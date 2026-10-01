package ui

import (
	"fmt"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type TitleAction int

const (
	TitleNone TitleAction = iota
	TitleSingleplayer
	TitleHost
	TitleJoin
	TitleQuit
)

func screenCenter() (float32, float32) {
	return float32(rl.GetScreenWidth()) / 2, float32(rl.GetScreenHeight()) / 2
}

func dimBackground() {
	rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()), rl.NewColor(0, 0, 0, 160))
}

// DrawTitle draws the start screen. status is shown below the buttons
// (e.g. why the last lobby was closed).
func DrawTitle(name *TextInput, status string) TitleAction {
	dimBackground()
	cx, cy := screenCenter()
	panel := rl.NewRectangle(cx-260, cy-250, 520, 500)
	drawPanel(panel)

	drawLabelCentered("SPOOK 'N LOOT", cx, panel.Y+30, 56, rl.RayWhite)

	drawLabel("Your name", panel.X+60, panel.Y+115, 22, mutedTextColor)
	name.Draw(rl.NewRectangle(panel.X+60, panel.Y+145, 400, 44))

	action := TitleNone
	bw, bh := float32(400), float32(48)
	bx := cx - bw/2
	y := panel.Y + 215
	if button(rl.NewRectangle(bx, y, bw, bh), "Singleplayer", true) {
		action = TitleSingleplayer
	}
	y += bh + 12
	if button(rl.NewRectangle(bx, y, bw, bh), "Host LAN game", true) {
		action = TitleHost
	}
	y += bh + 12
	if button(rl.NewRectangle(bx, y, bw, bh), "Join game", true) {
		action = TitleJoin
	}
	y += bh + 12
	if button(rl.NewRectangle(bx, y, bw, bh), "Quit", true) {
		action = TitleQuit
	}

	if status != "" {
		drawStatus(status, cx, panel.Y+panel.Height+16)
	}
	return action
}

func drawStatus(status string, cx, y float32) {
	size := int32(20)
	w := rl.MeasureText(status, size)
	rl.DrawRectangle(int32(cx)-w/2-12, int32(y)-6, w+24, size+12, rl.NewColor(0, 0, 0, 200))
	rl.DrawText(status, int32(cx)-w/2, int32(y), size, errorTextColor)
}

type JoinAction int

const (
	JoinNone JoinAction = iota
	JoinConnect
	JoinBack
)

// DrawJoin draws the server browser. It returns JoinConnect together with the
// address to connect to.
func DrawJoin(games []netcode.FoundGame, address *TextInput, status string) (JoinAction, string) {
	dimBackground()
	cx, cy := screenCenter()
	panel := rl.NewRectangle(cx-360, cy-320, 720, 640)
	drawPanel(panel)
	drawLabelCentered("JOIN GAME", cx, panel.Y+24, 44, rl.RayWhite)

	listX, listW := panel.X+40, panel.Width-80
	y := panel.Y + 90
	drawLabel("Games on your network", listX, y, 22, mutedTextColor)
	y += 32

	action, addr := JoinNone, ""
	rowH := float32(44)
	listH := rowH * 7
	rl.DrawRectangleLinesEx(rl.NewRectangle(listX, y, listW, listH), 1, panelBorderColor)
	if len(games) == 0 {
		rl.DrawText("Searching...", int32(listX)+14, int32(y)+12, 20, mutedTextColor)
	}
	for i, g := range games {
		if float32(i+1)*rowH > listH {
			break
		}
		row := rl.NewRectangle(listX, y+float32(i)*rowH, listW, rowH)
		compatible := g.Version == lobby.ProtocolVersion
		full := g.Players >= g.MaxPlayers
		joinable := compatible && !full
		if joinable && hovered(row) {
			rl.DrawRectangleRec(row, buttonHoverColor)
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				action, addr = JoinConnect, g.Addr
			}
		}
		info := fmt.Sprintf("%d/%d", g.Players, g.MaxPlayers)
		switch {
		case !compatible:
			info += "  other version"
		case full:
			info += "  full"
		case g.InGame:
			info += "  in game"
		}
		color := rl.RayWhite
		if !joinable {
			color = mutedTextColor
		}
		rl.DrawText(g.Name, int32(row.X)+14, int32(row.Y)+12, 22, color)
		iw := rl.MeasureText(info, 20)
		rl.DrawText(info, int32(row.X+row.Width)-iw-14, int32(row.Y)+13, 20, mutedTextColor)
	}
	y += listH + 30

	drawLabel("Or connect by IP address", listX, y, 22, mutedTextColor)
	y += 32
	if address.Draw(rl.NewRectangle(listX, y, listW-170, 44)) && address.Text != "" {
		action, addr = JoinConnect, address.Text
	}
	if button(rl.NewRectangle(listX+listW-155, y, 155, 44), "Connect", address.Text != "") {
		action, addr = JoinConnect, address.Text
	}

	if button(rl.NewRectangle(cx-100, panel.Y+panel.Height-68, 200, 44), "Back", true) || rl.IsKeyPressed(rl.KeyEscape) {
		action = JoinBack
	}
	if status != "" {
		drawStatus(status, cx, panel.Y+panel.Height+16)
	}
	return action, addr
}

type LobbyAction int

const (
	LobbyNone LobbyAction = iota
	LobbyToggleReady
	LobbyStart
	LobbyLeave
)

// DrawLobby draws the lobby: player list plus ready/start/leave buttons.
func DrawLobby(l *lobby.Lobby) LobbyAction {
	dimBackground()
	cx, cy := screenCenter()
	panel := rl.NewRectangle(cx-440, cy-380, 880, 760)
	drawPanel(panel)

	if l.State == lobby.StateConnecting {
		drawLabelCentered("Connecting...", cx, cy-40, 40, rl.RayWhite)
		if button(rl.NewRectangle(cx-100, cy+20, 200, 44), "Cancel", true) || rl.IsKeyPressed(rl.KeyEscape) {
			return LobbyLeave
		}
		return LobbyNone
	}

	drawLabelCentered(l.LobbyName, cx, panel.Y+22, 44, rl.RayWhite)
	sub := fmt.Sprintf("%d / %d players", len(l.Players), lobby.MaxPlayers)
	if l.IsHost {
		sub += "   -   LAN, port " + fmt.Sprint(netcode.DefaultGamePort)
	}
	drawLabelCentered(sub, cx, panel.Y+74, 20, mutedTextColor)

	// Two columns of 20 rows fit all 40 players.
	const rowsPerCol = 20
	colW := (panel.Width - 100) / 2
	rowH := float32(26)
	top := panel.Y + 115
	for i, p := range l.Players {
		col, row := i/rowsPerCol, i%rowsPerCol
		x := panel.X + 40 + float32(col)*(colW+20)
		y := top + float32(row)*rowH
		if p.ID == l.LocalID {
			rl.DrawRectangle(int32(x)-6, int32(y)-2, int32(colW)+12, int32(rowH), rl.NewColor(74, 48, 66, 180))
		}
		rl.DrawText(p.Name, int32(x), int32(y)+2, 20, rl.RayWhite)
		tag, color := "not ready", mutedTextColor
		switch {
		case p.Host:
			tag, color = "HOST", accentColor
		case p.Ready:
			tag, color = "ready", readyColor
		}
		tw := rl.MeasureText(tag, 18)
		rl.DrawText(tag, int32(x+colW)-tw, int32(y)+3, 18, color)
	}

	action := LobbyNone
	by := panel.Y + panel.Height - 72
	if l.IsHost {
		label := "Start game"
		if !l.CanStart() {
			label = "Waiting for players..."
		}
		if button(rl.NewRectangle(cx-290, by, 340, 50), label, l.CanStart()) {
			action = LobbyStart
		}
	} else {
		me, _ := l.LocalPlayer()
		label := "Ready"
		if me.Ready {
			label = "Not ready"
		}
		if button(rl.NewRectangle(cx-290, by, 340, 50), label, true) {
			action = LobbyToggleReady
		}
	}
	if button(rl.NewRectangle(cx+70, by, 220, 50), "Leave", true) || rl.IsKeyPressed(rl.KeyEscape) {
		action = LobbyLeave
	}
	return action
}

// DrawMultiplayerHUD shows a small lobby info box while playing online.
func DrawMultiplayerHUD(l *lobby.Lobby) {
	text := fmt.Sprintf("%s  -  %d players", l.LobbyName, len(l.Players))
	w := rl.MeasureText(text, 18)
	x := int32(rl.GetScreenWidth()) - w - 24
	rl.DrawRectangle(x-10, 12, w+20, 30, rl.NewColor(0, 0, 0, 150))
	rl.DrawText(text, x, 18, 18, rl.RayWhite)
}
