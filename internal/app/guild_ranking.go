package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (a *App) confirmedGuildRanking(p []byte) {
	if a.state == nil {
		return
	}
	switch proto.Op(p) {
	case proto.GuildRankingNotice:
		n := a.state.Guild.RankingNotice
		var message string
		switch n.Kind {
		case 100:
			message = fmt.Sprintf("Guild ranking battle opened on map %d.", n.Map)
		case 101:
			message = "Guild ranking battle ends in 10 minutes."
		case 102:
			message = "Guild ranking battle ends in 1 minute."
		default:
			if n.Kind < 24 {
				message = fmt.Sprintf("Guild ranking battle on map %d opens at %02d:00 server time.", n.Map, n.Kind)
			}
		}
		if message != "" {
			a.questMessage(message)
		}
	case proto.GuildRankingList:
		a.guild.directoryDirty = true
		a.questMessage("Guild ranking results updated.")
	}
}

// The score is shown only after Rapture sends the battle's live 0x0D27.
// Exact native placement and artwork remain to be recovered.
func (a *App) drawGuildRanking(w, h int) {
	if a.state == nil || !a.state.Guild.RankingPointsKnown {
		return
	}
	v := a.state.Guild.RankingPoints
	line := fmt.Sprintf("Guild battle: %d", v.Points)
	if v.Leader != 0 {
		leader := fmt.Sprintf("Guild %d", v.Leader)
		if info, ok := a.state.Guild.Directory[v.Leader]; ok && info.Name != "" {
			leader = info.Name
		}
		line += fmt.Sprintf("   Leader: %s (%d)", leader, v.Maximum)
	}
	a.font.Draw(a.sprites, line, max(0, (w-a.font.Width(line))/2), 80, white, 0)
}
