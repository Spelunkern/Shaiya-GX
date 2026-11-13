package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type guildView struct {
	inviteSerial, replySerial                     uint64
	applySerial, joinSerial                       uint64
	createSerial, voteSerial, failureSerial       uint64
	leaveSerial, disbandSerial, disbandVoteSerial uint64
	disbandedSerial                               uint64
	createPending                                 bool
	createDeadline                                float64
	lifecyclePending                              bool
	lifecycleDeadline                             float64
	voting                                        bool
	disbandVote                                   bool
	expires                                       float64
	open, rosterDirty                             bool
	waiting                                       bool
	offset                                        int
	selected                                      uint32
	kickTarget                                    uint32
	confirmKind                                   uint8
	confirmGuild                                  uint32
	lastAction                                    float64
	roster                                        []proto.GuildMember
	directory                                     []proto.GuildInfo
	directoryDirty                                bool
	directoryNPC, directorySelected               uint32
	directoryOffset                               int
	etinRequestedFor                              uint32
	remarkPending                                 bool
	remarkDeadline                                float64
}

func (a *App) refreshGuildDirectory() {
	if !a.guild.directoryDirty {
		return
	}
	a.guild.directory = a.guild.directory[:0]
	for _, guild := range a.state.Guild.Directory {
		a.guild.directory = append(a.guild.directory, guild)
	}
	sort.Slice(a.guild.directory, func(i, j int) bool {
		left, right := a.guild.directory[i], a.guild.directory[j]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.ID < right.ID
	})
	if _, ok := a.state.Guild.Directory[a.guild.directorySelected]; !ok {
		a.guild.directorySelected = 0
	}
	a.guild.directoryOffset = max(0, min(a.guild.directoryOffset, max(0, len(a.guild.directory)-ui.GuildDirectoryRows)))
	a.guild.directoryDirty = false
}

func (a *App) updateGuildDirectory(in *platform.Input, w, h int, e *game.Entity) bool {
	if a.guildDirectoryUI == nil {
		return false
	}
	if a.guild.directoryNPC != e.ID {
		a.guild.directoryNPC, a.guild.directorySelected, a.guild.directoryOffset = e.ID, 0, 0
		a.guild.directoryDirty = true
	}
	a.refreshGuildDirectory()
	_, exists := a.state.Guild.Directory[a.guild.directorySelected]
	canJoin := exists && a.state.Guild.ID == 0 && a.state.Guild.DirectoryReady
	x, y := a.guildDirectoryUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.GuildDirectoryWidth, ui.GuildDirectoryHeight)
	canCreate := a.state.Guild.ID == 0 && !a.guild.createPending && !a.guild.voting
	canLeave := a.state.Guild.ID != 0 && a.state.Guild.Rank > 1 && !a.guild.lifecyclePending && !a.guild.voting
	canDisband := a.state.Guild.ID != 0 && a.state.Guild.Rank == 1 && !a.guild.lifecyclePending && !a.guild.voting
	row, offset, join, create, leave, disband, close := a.guildDirectoryUI.Update(in, w, h, len(a.guild.directory), a.guild.directoryOffset, canJoin, canCreate, canLeave, canDisband)
	a.guild.directoryOffset = offset
	if row >= 0 {
		a.guild.directorySelected = a.guild.directory[row].ID
	}
	if join && canJoin {
		if err := a.world.Send(proto.GuildApplyPacket(a.guild.directorySelected)); err != nil {
			a.backToLogin("Unable to request guild membership.")
			return true
		}
	}
	if create && canCreate && a.guildCreateUI != nil {
		a.guildCreateUI.Open()
		return true
	}
	if (leave && canLeave || disband && canDisband) && a.guildConfirm != nil {
		a.guild.confirmGuild = a.state.Guild.ID
		if leave {
			a.guild.confirmKind = 2
			a.guildConfirm.Open("Leave "+a.state.Guild.Name+"?", true)
		} else {
			a.guild.confirmKind = 3
			a.guildConfirm.Open("Disband "+a.state.Guild.Name+"?", true)
		}
		return true
	}
	if close || in.HasKey(platform.KeyEscape) {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLogin("Unable to close guild manager.")
		} else {
			a.shop.npcID = 0
			a.guild.directoryNPC = 0
		}
		return true
	}
	return captured
}

func (a *App) drawGuildDirectory(w, h int) {
	if a.guildDirectoryUI == nil {
		return
	}
	a.refreshGuildDirectory()
	_, exists := a.state.Guild.Directory[a.guild.directorySelected]
	available := !a.guild.lifecyclePending && !a.guild.voting
	a.guildDirectoryUI.Draw(a.sprites, a.font, a.white, w, h, a.guild.directory, a.guild.directoryOffset, a.guild.directorySelected, exists && a.state.Guild.ID == 0 && a.state.Guild.DirectoryReady, a.state.Guild.ID == 0 && !a.guild.createPending && !a.guild.voting, a.state.Guild.ID != 0 && a.state.Guild.Rank > 1 && available, a.state.Guild.ID != 0 && a.state.Guild.Rank == 1 && available)
}

func (a *App) updateGuildCreate(in *platform.Input, w, h int) bool {
	if a.guildCreateUI == nil || !a.guildCreateUI.Visible || a.state == nil {
		return false
	}
	result := a.guildCreateUI.Update(in, w, h)
	if result == ui.MessageOK && a.guildCreateUI.NoticeMode {
		if a.state.Guild.ID == 0 || a.state.Guild.Rank != 1 || a.guild.remarkPending {
			a.questMessage("Only the guild master can change the notice.")
		} else if strings.ContainsRune(a.guildCreateUI.Remark.Text, '\'') {
			a.questMessage("Guild notices cannot contain apostrophes.")
		} else if err := a.world.Send(proto.GuildRemarkPacket(a.guildCreateUI.Remark.Text)); err != nil {
			a.backToLogin("Unable to update guild notice.")
		} else {
			a.guild.remarkPending, a.guild.remarkDeadline = true, a.clock+10
		}
		return true
	}
	if result == ui.MessageOK && a.state.Guild.ID == 0 && !a.guild.createPending {
		name := strings.TrimSpace(a.guildCreateUI.Name.Text)
		remark := strings.TrimSpace(a.guildCreateUI.Remark.Text)
		if err := a.world.Send(proto.GuildCreatePacket(name, remark)); err != nil {
			a.backToLogin("Unable to request guild creation.")
		} else {
			a.guild.createPending, a.guild.createDeadline = true, a.clock+10
		}
	}
	return true
}

func (a *App) updateGuildVote(in *platform.Input, w, h int) bool {
	if a.state == nil || a.guildVoteMessage == nil {
		return false
	}
	g, v := &a.state.Guild, &a.guild
	if g.VoteSerial != v.voteSerial {
		v.voteSerial = g.VoteSerial
		v.createPending = false
		v.voting = true
		v.disbandVote = false
		if a.guildCreateUI != nil {
			a.guildCreateUI.Visible = false
		}
		a.guildVoteMessage.Open("Create guild "+g.CreateVote.Name+" with your party?", true)
	}
	if g.DisbandVoteSerial != v.disbandVoteSerial {
		v.disbandVoteSerial = g.DisbandVoteSerial
		v.lifecyclePending = false
		v.voting = true
		v.disbandVote = true
		a.guildVoteMessage.Open("Disband "+g.Name+" with your party?", true)
	}
	if g.DisbandSerial != v.disbandSerial {
		v.disbandSerial = g.DisbandSerial
		v.lifecyclePending = false
		v.voting = false
		a.guildVoteMessage.Visible = false
		msg := "Guild disband request refused."
		if g.DisbandResult == 2 {
			msg = "Two other guild administrators must join your party."
		} else if g.DisbandResult == 3 {
			msg = "Guild disband vote was declined."
		}
		id := 380
		if g.DisbandResult == 2 {
			id = 382
		} else if g.DisbandResult == 3 {
			id = 383
		}
		a.localizedNotice(1, id, nil, msg)
	}
	if g.LeaveSerial != v.leaveSerial {
		v.leaveSerial = g.LeaveSerial
		v.lifecyclePending = false
		if g.LeaveResult == 1 {
			a.localizedNotice(17, 390, nil, "You left the guild.")
		} else {
			a.localizedNotice(1, 391, nil, "Guild leave request refused.")
		}
	}
	if g.DisbandedSerial != v.disbandedSerial {
		v.disbandedSerial = g.DisbandedSerial
		v.lifecyclePending = false
		v.voting = false
		a.guildVoteMessage.Visible = false
		a.localizedNotice(17, 395, nil, "Guild disbanded.")
	}
	if g.CreateSerial != v.createSerial {
		v.createSerial = g.CreateSerial
		v.createPending = false
		v.voting = false
		a.guildVoteMessage.Visible = false
		messages := map[uint8]string{1: "Guild created.", 3: "A party of seven is required.", 4: "Guild creator must reach level 11.", 5: "Normal mode or higher is required.", 6: "Guild name is unavailable.", 8: "A party member already belongs to a guild.", 9: "A party member must wait before joining a guild."}
		msg := messages[g.CreateResult]
		if msg == "" {
			msg = "Guild creation refused."
		}
		ids := map[uint8]int{1: 372, 3: 374, 4: 375, 5: 376, 6: 377, 8: 379}
		id := ids[g.CreateResult]
		if id == 0 {
			id = 371
		}
		name := g.Name
		if name == "" && a.guildCreateUI != nil {
			name = a.guildCreateUI.Name.Text
		}
		a.localizedNotice(17, id, map[string]string{"b": name}, msg)
	}
	if g.FailureSerial != v.failureSerial {
		v.failureSerial = g.FailureSerial
		v.createPending = false
		v.voting = false
		a.guildVoteMessage.Visible = false
		messages := map[uint8]string{2: "Not enough gold to create the guild.", 6: "Guild name is unavailable.", 7: "Guild creation was declined."}
		msg := messages[g.CreateFailure]
		if msg == "" {
			msg = "Guild creation failed."
		}
		id := 371
		switch g.CreateFailure {
		case 2:
			id = 373
		case 6:
			id = 377
		case 7:
			id = 378
		}
		name := g.Name
		if name == "" && a.guildCreateUI != nil {
			name = a.guildCreateUI.Name.Text
		}
		a.localizedNotice(1, id, map[string]string{"b": name}, msg)
	}
	if !a.guildVoteMessage.Visible {
		return false
	}
	if result := a.guildVoteMessage.Update(in, w, h); result != ui.MessageNone {
		packet := proto.GuildCreateAnswerPacket(result == ui.MessageOK)
		if v.disbandVote {
			packet = proto.GuildDisbandAnswerPacket(result == ui.MessageOK)
		}
		if err := a.world.Send(packet); err != nil {
			a.backToLogin("Unable to answer guild vote.")
		}
	}
	return true
}

func (a *App) toggleGuild() {
	a.guild.open = !a.guild.open
	if a.guild.open {
		a.questLog.open = false
		a.guild.etinRequestedFor = 0
	}
	a.guild.rosterDirty = true
}

func (a *App) updateGuild(in *platform.Input, w, h int) bool {
	if a.guild.remarkPending && a.clock >= a.guild.remarkDeadline {
		a.guild.remarkPending = false
		a.questMessage("Guild notice request timed out.")
	}
	for _, r := range in.Chars {
		if (r == 'g' || r == 'G') && !(a.socialUI != nil && a.socialUI.Focus && a.guild.open && a.guildUI != nil && a.guildUI.Page != 0) {
			a.toggleGuild()
			return true
		}
	}
	if !a.guild.open || a.guildUI == nil {
		return false
	}
	gx, gy := a.guildUI.Origin(w, h)
	if a.guildUI.UpdateNavigation(in, gx, gy) {
		a.social.offset = 0
		a.social.selected = 0
		return true
	}
	if a.guildUI.Page != 0 {
		return a.updateSocial(in, w, h)
	}
	if a.state.Guild.ID != 0 && a.guild.etinRequestedFor != a.state.Guild.ID && a.world != nil {
		if err := a.world.Send(proto.GuildEtinPacket()); err != nil {
			a.backToLogin("Unable to request guild Etin.")
			return true
		}
		a.guild.etinRequestedFor = a.state.Guild.ID
	}
	a.refreshGuildRoster()
	x, y := a.guildUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.GuildWidth, ui.GuildHeight)
	canToggle := a.state.Guild.ID != 0 && a.state.Guild.Rank > 0 && a.state.Guild.Rank <= 2
	if a.guild.waiting && !canToggle {
		a.guild.waiting, a.guild.rosterDirty, a.guild.selected = false, true, 0
		a.refreshGuildRoster()
	}
	allowed := a.guildActions(a.guild.selected)
	if a.guild.waiting {
		_, ok := a.state.Guild.Applicants[a.guild.selected]
		allowed = [3]bool{canToggle && ok, canToggle && ok, false}
	}
	row, scroll, action, toggle, close := a.guildUI.Update(in, w, h, len(a.guild.roster), a.guild.offset, a.guild.waiting, canToggle, allowed)
	if a.guildUI.SortChanged {
		a.guild.rosterDirty = true
		a.refreshGuildRoster()
	}
	canEdit := !a.guild.waiting && a.state.Guild.ID != 0 && a.state.Guild.Rank == 1 && !a.guild.remarkPending && a.guildCreateUI != nil
	if a.guildUI.UpdateNotice(in, w, h, canEdit) {
		a.guildCreateUI.OpenNotice(a.state.Guild.Name, a.state.Guild.Remark)
		return true
	}
	a.guild.offset = scroll
	if toggle {
		a.guild.waiting, a.guild.rosterDirty, a.guild.selected, a.guild.offset = !a.guild.waiting, true, 0, 0
	}
	if row >= 0 {
		a.guild.selected = a.guild.roster[row].ID
	}
	if action >= 0 {
		if a.guild.waiting {
			a.answerGuildApplicant(action == 0)
		} else {
			a.guildAction(action)
		}
	}
	if close || in.HasKey(platform.KeyEscape) {
		a.guild.open = false
		return true
	}
	return captured
}

func (a *App) refreshGuildRoster() {
	if !a.guild.rosterDirty {
		return
	}
	a.guild.roster = a.guild.roster[:0]
	if a.guild.waiting {
		for _, m := range a.state.Guild.Applicants {
			a.guild.roster = append(a.guild.roster, proto.GuildMember{ID: m.ID, Name: m.Name, Level: m.Level, Class: m.Class})
		}
	} else {
		for _, m := range a.state.Guild.Members {
			a.guild.roster = append(a.guild.roster, m)
		}
	}
	sort.Slice(a.guild.roster, func(i, j int) bool {
		left, right := a.guild.roster[i], a.guild.roster[j]
		column, descending := 3, false
		if a.guildUI != nil {
			column, descending = a.guildUI.SortColumn, a.guildUI.Descending
		}
		order := 0
		switch column {
		case 0:
			order = int(left.Level) - int(right.Level)
		case 1:
			order = strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
		case 2:
			order = int(left.Class) - int(right.Class)
		case 3:
			order = int(left.Rank) - int(right.Rank)
		case 4:
			if left.Online != right.Online {
				if left.Online {
					order = -1
				} else {
					order = 1
				}
			}
		}
		if order != 0 {
			if descending {
				return order > 0
			}
			return order < 0
		}
		if left.Rank != right.Rank {
			return left.Rank < right.Rank
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.ID < right.ID
	})
	if a.guild.waiting {
		if _, ok := a.state.Guild.Applicants[a.guild.selected]; !ok {
			a.guild.selected = 0
		}
	} else {
		if _, ok := a.state.Guild.Members[a.guild.selected]; !ok {
			a.guild.selected = 0
		}
	}
	a.guild.offset = max(0, min(a.guild.offset, max(0, len(a.guild.roster)-ui.GuildRows)))
	a.guild.rosterDirty = false
}

func (a *App) drawGuild(w, h int) {
	if !a.guild.open || a.guildUI == nil {
		return
	}
	x, y := a.guildUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.GuildWidth, ui.GuildHeight)
	if a.guildUI.Page != 0 {
		a.drawSocial(w, h)
		return
	}
	a.refreshGuildRoster()
	canToggle := a.state.Guild.ID != 0 && a.state.Guild.Rank > 0 && a.state.Guild.Rank <= 2
	allowed := a.guildActions(a.guild.selected)
	if a.guild.waiting {
		_, ok := a.state.Guild.Applicants[a.guild.selected]
		allowed = [3]bool{canToggle && ok, canToggle && ok, false}
	}
	for i := range a.guildUI.ClassNames {
		a.guildUI.ClassNames[i] = ui.ClassName(a.msg, a.state.Faction.Faction == 0, uint8(i))
	}
	name := a.state.Guild.Name
	if suffix := a.msg.Get(0x13e0); name != "" && suffix != "" {
		name += " " + suffix
	}
	a.guildUI.Draw(a.sprites, a.font, a.white, w, h, name, a.guild.roster, a.guild.offset, a.guild.selected, a.guild.waiting, canToggle, allowed)
	if !a.guild.waiting {
		a.guildUI.DrawNotice(a.sprites, a.font, a.white, w, h, a.state.Guild.Rank == 1 && !a.guild.remarkPending, a.state.Guild.Etin, a.state.Guild.EtinKnown)
	}
}

func (a *App) answerGuildApplicant(accept bool) {
	g := &a.state.Guild
	id := a.guild.selected
	if g.ID == 0 || g.Rank == 0 || g.Rank > 2 {
		return
	}
	if _, ok := g.Applicants[id]; !ok {
		return
	}
	if a.clock < a.guild.lastAction {
		return
	}
	if err := a.world.Send(proto.GuildApplicantAnswerPacket(id, accept)); err != nil {
		a.backToLogin("Unable to answer guild application.")
		return
	}
	a.guild.lastAction = a.clock + .5
}

func (a *App) guildActions(id uint32) [3]bool {
	var enabled [3]bool
	if a.state == nil || a.state.Guild.ID == 0 || id == 0 || id == a.state.SelfID {
		return enabled
	}
	g := &a.state.Guild
	target, ok := g.Members[id]
	if !ok || g.Rank == 0 || g.Rank > 2 || target.Rank <= g.Rank {
		return enabled
	}
	enabled[0] = target.Rank > 2 && (g.Rank == 1 || target.Rank > 3)
	enabled[1] = target.Rank < 9
	enabled[2] = true
	return enabled
}

func (a *App) guildAction(action int) {
	id := a.guild.selected
	if action < 0 || action >= 3 || !a.guildActions(id)[action] {
		return
	}
	if action == 2 {
		if a.guildConfirm != nil {
			a.guild.kickTarget = id
			a.guild.confirmKind = 1
			a.guild.confirmGuild = a.state.Guild.ID
			a.guildConfirm.Open("Remove "+a.state.Guild.Members[id].Name+" from the guild?", true)
		}
		return
	}
	if a.clock < a.guild.lastAction {
		return
	}
	if err := a.world.Send(proto.GuildRankPacket(id, action == 1)); err != nil {
		a.backToLogin("Unable to request guild rank change.")
		return
	}
	a.guild.lastAction = a.clock + .5
}

func (a *App) updateGuildConfirm(in *platform.Input, w, h int) bool {
	if a.guildConfirm == nil || !a.guildConfirm.Visible || a.state == nil {
		return false
	}
	if result := a.guildConfirm.Update(in, w, h); result != ui.MessageNone {
		id, kind, guildID := a.guild.kickTarget, a.guild.confirmKind, a.guild.confirmGuild
		a.guild.kickTarget = 0
		a.guild.confirmKind, a.guild.confirmGuild = 0, 0
		if result != ui.MessageOK || guildID == 0 || a.state.Guild.ID != guildID {
			return true
		}
		var packet []byte
		switch kind {
		case 1:
			if a.guildActions(id)[2] {
				packet = proto.GuildExpelPacket(id)
			}
		case 2:
			if a.state.Guild.Rank > 1 && !a.guild.lifecyclePending && !a.guild.voting {
				packet = proto.GuildLeavePacket()
			}
		case 3:
			if a.state.Guild.Rank == 1 && !a.guild.lifecyclePending && !a.guild.voting {
				packet = proto.GuildDisbandPacket()
			}
		}
		if packet != nil {
			if err := a.world.Send(packet); err != nil {
				a.backToLogin("Unable to send guild request.")
			} else if kind == 2 || kind == 3 {
				a.guild.lifecyclePending = true
				a.guild.lifecycleDeadline = a.clock + 10
			}
		}
	}
	return true
}

func (a *App) updateGuildInvite(in *platform.Input, w, h int) bool {
	if a.guildMessage == nil || a.state == nil {
		return false
	}
	g, v := &a.state.Guild, &a.guild
	if g.InviteSerial != v.inviteSerial {
		v.inviteSerial = g.InviteSerial
		if g.InviteID != 0 && g.ID == 0 {
			name := fmt.Sprintf("Player %d", g.InviteID)
			if e := a.state.Entities[g.InviteID]; e != nil && e.Shape != nil && e.Shape.Name != "" {
				name = e.Shape.Name
			}
			a.guildMessage.Open(name+" invites you to a guild. Join?", true)
			if v.expires == 0 {
				v.expires = a.clock + 30
			}
		}
	}
	if g.ReplySerial != v.replySerial {
		v.replySerial = g.ReplySerial
		if g.LastReply.Result == 1 {
			a.questMessage("Guild invitation accepted.")
		} else {
			a.questMessage("Guild invitation declined or unavailable.")
		}
	}
	if g.ApplySerial != v.applySerial {
		v.applySerial = g.ApplySerial
		messages := [...]string{"Guild application refused.", "Guild application sent.", "You must wait before joining another guild."}
		message := "Guild application failed."
		if int(g.ApplyResult) < len(messages) {
			message = messages[g.ApplyResult]
		}
		a.questMessage(message)
	}
	if g.JoinSerial != v.joinSerial {
		v.joinSerial = g.JoinSerial
		if g.JoinResult == 0 {
			a.localizedNotice(17, 385, nil, "Guild application declined.")
		} else if g.JoinResult == 1 {
			a.localizedNotice(17, 384, nil, "You joined a guild.")
		}
	}
	if !a.guildMessage.Visible {
		return false
	}
	if g.ID != 0 || g.InviteID == 0 || a.clock >= v.expires {
		a.guildMessage.Visible = false
		g.InviteID = 0
		return false
	}
	if answer := a.guildMessage.Update(in, w, h); answer != ui.MessageNone {
		if err := a.world.Send(proto.GuildAnswerPacket(answer == ui.MessageOK)); err != nil {
			a.backToLogin("Unable to answer guild invitation.")
			return true
		}
		// The server does not echo a declined invitation to its recipient.
		g.InviteID = 0
	}
	return true
}

func (a *App) guildCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	if strings.EqualFold(fields[0], "/guildapply") {
		if len(fields) < 2 || a.state.Guild.ID != 0 {
			a.questMessage("Use /guildapply guild name while not in a guild.")
			return true
		}
		name := strings.Join(fields[1:], " ")
		for id, guild := range a.state.Guild.Directory {
			if strings.EqualFold(guild.Name, name) {
				if err := a.world.Send(proto.GuildApplyPacket(id)); err != nil {
					a.backToLogin("Unable to request guild membership.")
				}
				return true
			}
		}
		a.questMessage("Guild not found in the current directory.")
		return true
	}
	if !strings.EqualFold(fields[0], "/guildinvite") {
		return false
	}
	if a.state.Guild.ID == 0 || a.state.Guild.Rank == 0 || a.state.Guild.Rank > 2 {
		a.questMessage("Only a guild master or administrator can invite.")
		return true
	}
	e := a.state.Entities[a.state.Target]
	if len(fields) > 1 {
		e = nil
		name := strings.Join(fields[1:], " ")
		for _, candidate := range a.state.Entities {
			if candidate.Kind == game.KindUser && candidate.Shape != nil && strings.EqualFold(candidate.Shape.Name, name) {
				e = candidate
				break
			}
		}
	}
	if e == nil || e.Kind != game.KindUser || e.ID == a.state.SelfID {
		a.questMessage("Select a nearby player to invite.")
		return true
	}
	if _, member := a.state.Guild.Members[e.ID]; member {
		a.questMessage("Player is already in your guild.")
		return true
	}
	if err := a.world.Send(proto.GuildInvitePacket(e.ID)); err != nil {
		a.backToLogin("Unable to invite player to guild.")
	}
	return true
}
