package ui

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ScoreRow is one player in the scoreboard; rows come sorted by kills.
type ScoreRow struct {
	Name      string
	Kills     int
	Ping      string
	PingColor rl.Color
	Local     bool
}

const liveRankingSize = 5

// DrawLiveRanking shows the top players by kills under the health bar,
// plus the local player if they are not in the top.
func DrawLiveRanking(rows []ScoreRow) {
	if len(rows) == 0 {
		return
	}
	const size, lineH = 18, 22
	shown := rows[:min(liveRankingSize, len(rows))]
	localRank := -1
	for i, r := range rows {
		if r.Local {
			localRank = i
		}
	}
	extra := localRank >= liveRankingSize

	lines := len(shown) + 1
	if extra {
		lines++
	}
	x, y := int32(20), int32(84)
	rl.DrawRectangle(x-8, y-8, 230, int32(lines*lineH)+12, rl.NewColor(0, 0, 0, 140))
	rl.DrawText("Kills  (Tab: scoreboard)", x, y, 16, mutedTextColor)
	y += lineH

	drawRow := func(rank int, r ScoreRow) {
		color := rl.RayWhite
		if r.Local {
			color = accentColor
		}
		rl.DrawText(fmt.Sprintf("%d. %s", rank, r.Name), x, y, size, color)
		k := fmt.Sprint(r.Kills)
		rl.DrawText(k, x+210-rl.MeasureText(k, size), y, size, color)
		y += lineH
	}
	for i, r := range shown {
		drawRow(i+1, r)
	}
	if extra {
		drawRow(localRank+1, rows[localRank])
	}
}

// DrawScoreboard shows every player with kills and ping (held Tab).
func DrawScoreboard(rows []ScoreRow) {
	const rowsPerCol, rowH, colW = 20, 26, float32(400)
	cols := 1
	if len(rows) > rowsPerCol {
		cols = 2
	}
	visible := min(len(rows), rowsPerCol)
	w := colW*float32(cols) + 40 + float32(cols-1)*20
	h := float32(visible*rowH) + 120
	cx, cy := screenCenter()
	panel := rl.NewRectangle(cx-w/2, cy-h/2, w, h)
	drawPanel(panel)
	drawLabelCentered("SCOREBOARD", cx, panel.Y+16, 36, rl.RayWhite)

	for c := 0; c < cols; c++ {
		x := panel.X + 20 + float32(c)*(colW+20)
		y := panel.Y + 66
		rl.DrawText("Player", int32(x)+8, int32(y), 16, mutedTextColor)
		rl.DrawText("Kills", int32(x+colW)-150, int32(y), 16, mutedTextColor)
		rl.DrawText("Ping", int32(x+colW)-70, int32(y), 16, mutedTextColor)
	}
	for i, r := range rows {
		col, row := i/rowsPerCol, i%rowsPerCol
		x := panel.X + 20 + float32(col)*(colW+20)
		y := panel.Y + 92 + float32(row*rowH)
		if r.Local {
			rl.DrawRectangle(int32(x), int32(y)-3, int32(colW), rowH, rl.NewColor(74, 48, 66, 200))
		}
		rl.DrawText(fmt.Sprintf("%d. %s", i+1, r.Name), int32(x)+8, int32(y), 20, rl.RayWhite)
		rl.DrawText(fmt.Sprint(r.Kills), int32(x+colW)-150, int32(y), 20, rl.RayWhite)
		rl.DrawText(r.Ping, int32(x+colW)-70, int32(y), 20, r.PingColor)
	}
}
