package ui

import (
	"fmt"
	"math"
	"time"

	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/sim"

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
func DrawTitle(name *TextInput, class *sim.Class, status string) TitleAction {
	dimBackground()
	cx, cy := screenCenter()
	panel := rl.NewRectangle(cx-260, cy-305, 520, 610)
	drawPanel(panel)

	drawLabelCentered("SPOOK 'N LOOT", cx, panel.Y+30, 56, rl.RayWhite)

	drawLabel("Your name", panel.X+60, panel.Y+115, 22, mutedTextColor)
	name.Draw(rl.NewRectangle(panel.X+60, panel.Y+145, 400, 44))

	drawLabel("Class", panel.X+60, panel.Y+205, 22, mutedTextColor)
	cw, gap := float32(128), float32(8)
	for i, c := range sim.AllClasses {
		r := rl.NewRectangle(panel.X+60+float32(i)*(cw+gap), panel.Y+235, cw, 44)
		if classButton(r, c, *class == c) {
			*class = c
		}
	}
	drawLabelCentered(class.Stats().Description, cx, panel.Y+290, 18, ClassColor(*class))

	action := TitleNone
	bw, bh := float32(400), float32(48)
	bx := cx - bw/2
	y := panel.Y + 325
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

// classButton is a toggle button for one class; the selected one is
// highlighted in the class color.
func classButton(r rl.Rectangle, c sim.Class, selected bool) bool {
	clicked := button(r, c.Stats().Name, true)
	if selected {
		rl.DrawRectangleLinesEx(r, 3, ClassColor(c))
	}
	return clicked
}

// ClassColor is the tint that tells classes apart in the game and menus.
func ClassColor(c sim.Class) rl.Color {
	switch c {
	case sim.ClassMage:
		return rl.NewColor(150, 180, 255, 255)
	case sim.ClassHealer:
		return rl.NewColor(160, 255, 170, 255)
	default:
		return rl.NewColor(255, 170, 160, 255)
	}
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
// escape lets Esc leave (false while the chat box uses Esc).
func DrawLobby(l *lobby.Lobby, escape bool) LobbyAction {
	dimBackground()
	cx, cy := screenCenter()
	panel := rl.NewRectangle(cx-440, cy-380, 880, 760)
	drawPanel(panel)

	if l.State == lobby.StateConnecting {
		drawLabelCentered("Connecting...", cx, cy-40, 40, rl.RayWhite)
		if button(rl.NewRectangle(cx-100, cy+20, 200, 44), "Cancel", true) || (escape && rl.IsKeyPressed(rl.KeyEscape)) {
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
		nw := rl.MeasureText(p.Name, 20)
		rl.DrawText(p.Class.Stats().Name, int32(x)+nw+10, int32(y)+5, 16, ClassColor(p.Class))
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
	if button(rl.NewRectangle(cx+70, by, 220, 50), "Leave", true) || (escape && rl.IsKeyPressed(rl.KeyEscape)) {
		action = LobbyLeave
	}
	return action
}

// DrawInvite shows "<name> entered the dungeon" with the J hint and the
// seconds left to join.
func DrawInvite(from string, until time.Time) {
	secs := int(math.Ceil(time.Until(until).Seconds()))
	title := fmt.Sprintf("%s entered the dungeon!", from)
	hint := fmt.Sprintf("Press J to join (%d)", max(secs, 0))
	cx := float32(rl.GetScreenWidth()) / 2
	w := measureLabel(title, 30)
	if hw := measureLabel(hint, 22); hw > w {
		w = hw
	}
	w += 48
	box := rl.NewRectangle(cx-w/2, 90, w, 86)
	drawPanel(box)
	drawLabelCentered(title, cx, box.Y+12, 30, rl.RayWhite)
	drawLabelCentered(hint, cx, box.Y+50, 22, accentColor)
}

// DrawMultiplayerHUD shows a small lobby info box while playing online.
func DrawMultiplayerHUD(l *lobby.Lobby) {
	text := fmt.Sprintf("%s  -  %d players  -  ", l.LobbyName, len(l.Players))
	ping, color := PingLabel(l)
	w := rl.MeasureText(text, 18)
	pw := rl.MeasureText(ping, 18)
	x := int32(rl.GetScreenWidth()) - w - pw - 24
	rl.DrawRectangle(x-10, 12, w+pw+20, 30, rl.NewColor(0, 0, 0, 150))
	rl.DrawText(text, x, 18, 18, rl.RayWhite)
	rl.DrawText(ping, x+w, 18, 18, color)
}

// PingLabel formats the local ping: "host" on the host, otherwise
// milliseconds colored by quality.
func PingLabel(l *lobby.Lobby) (string, rl.Color) {
	if l.IsHost {
		return "host", accentColor
	}
	rtt, ok := l.Ping()
	if !ok {
		return "-- ms", mutedTextColor
	}
	return PingText(uint16(min(rtt.Milliseconds(), 65535)))
}

// PingText formats a ping in milliseconds, colored by quality.
func PingText(ms uint16) (string, rl.Color) {
	color := readyColor
	switch {
	case ms >= 150:
		color = errorTextColor
	case ms >= 80:
		color = accentColor
	}
	return fmt.Sprintf("%d ms", ms), color
}

// DrawRunProgress shows how far the group is on the way to the boss:
// one segment per dungeon level, the current one highlighted.
func DrawRunProgress(level, levels int) {
	const segW, segH, gap = float32(16), float32(10), float32(3)
	barW := float32(levels)*(segW+gap) - gap
	bossLabel := "BOSS"
	bossW := measureLabel(bossLabel, 20)
	w := barW + 16 + bossW + 32
	cx := float32(rl.GetScreenWidth()) / 2
	box := rl.NewRectangle(cx-w/2, 12, w, 62)
	drawPanel(box)

	title := fmt.Sprintf("Level %d / %d", level, levels)
	drawLabelCentered(title, cx, box.Y+8, 22, rl.RayWhite)

	x := box.X + 16
	y := box.Y + 40
	for i := 1; i <= levels; i++ {
		color := rl.NewColor(60, 45, 60, 255)
		switch {
		case i < level:
			color = readyColor
		case i == level:
			color = accentColor
		}
		rl.DrawRectangleRec(rl.NewRectangle(x, y, segW, segH), color)
		x += segW + gap
	}
	drawLabel(bossLabel, x+16-gap, y-6, 20, errorTextColor)
}
