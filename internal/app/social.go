package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"sort"
)

type socialView struct {
	searchName   string
	offset       int
	selected     uint32
	inviteSerial uint64
	pending      bool
	deadline     float64
}

func (a *App) socialRows() []proto.SocialContact {
	list := a.state.Social.Friends
	if a.guildUI.Page == 2 {
		list = a.state.Social.Blocked
	}
	rows := make([]proto.SocialContact, 0, len(list))
	for _, r := range list {
		rows = append(rows, r)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		if a.socialUI != nil && a.socialUI.Descending {
			left, right = right, left
		}
		if a.socialUI != nil {
			switch a.socialUI.SortColumn {
			case 0:
				if left.ID != right.ID {
					return left.ID < right.ID
				}
			case 2:
				if left.Class != right.Class {
					return left.Class < right.Class
				}
			case 3:
				if left.Memo != right.Memo {
					return left.Memo < right.Memo
				}
			case 4:
				if left.Online != right.Online {
					return left.Online
				}
			}
		}
		return left.Name < right.Name
	})
	return rows
}
func (a *App) sendSocialPacket(p []byte) {
	if a.world == nil {
		return
	}
	if err := a.world.Send(p); err != nil {
		a.backToLogin("Unable to send social request.")
		return
	}
	a.social.pending, a.social.deadline = true, a.clock+10
}
func (a *App) confirmedSocial(op proto.Opcode) {
	if a.state == nil {
		return
	}
	if op == proto.FriendInvite && a.socialInviteUI != nil {
		a.social.inviteSerial = a.state.Social.InviteSerial
		a.socialInviteUI.Open(fmt.Sprintf("Add %s to your Buddy list?", a.state.Social.Invitation), true)
	}
	if op == proto.FriendAnswer && a.state.Social.Result <= 2 {
		a.systemMessage(7, 399+int(a.state.Social.Result), nil)
	}
	if op == proto.SocialError {
		id := 403
		if a.state.Social.Result != 0 {
			id = 402
		}
		a.systemMessage(7, id, nil)
	}
	if op == proto.PartySearchRequest {
		id := 405
		if a.state.Social.Result != 0 {
			id = 404
		}
		a.systemMessage(7, id, nil)
	}
	switch op {
	case proto.FriendAnswer, proto.BlockAdd, proto.FriendDelete, proto.BlockDelete, proto.PartySearchRequest, proto.SocialError, proto.FriendMemo, proto.BlockMemo:
		a.social.pending = false
	}
}
func (a *App) updateSocialInvitation(in *platform.Input, w, h int) {
	result := a.socialInviteUI.Update(in, w, h)
	if result == ui.MessageNone {
		return
	}
	a.sendSocialPacket(proto.FriendAnswerPacket(result == ui.MessageOK))
	a.socialInviteUI.Visible = false
}
func (a *App) updateSocial(in *platform.Input, w, h int) bool {
	if a.socialUI == nil {
		return false
	}
	x, y := a.guildUI.Origin(w, h)
	page := a.guildUI.Page
	rows := a.socialRows()
	count := len(rows)
	if page == 3 {
		count = len(a.socialSearchRows())
	}
	if a.social.pending && a.clock >= a.social.deadline {
		a.social.pending = false
		a.questMessage("Social request timed out.")
	}
	a.social.offset = max(0, min(a.social.offset, max(0, count-15)))
	row, offset, action, closed := a.socialUI.Update(in, x, y, page, count, a.social.offset, a.social.selected != 0)
	a.social.offset = offset
	if closed {
		a.guild.open = false
		a.socialUI.Focus = false
		return true
	}
	if row >= 0 {
		if page == 3 {
			a.social.selected = uint32(row + 1)
			a.social.searchName = a.socialSearchRows()[row].Name
		} else {
			a.social.selected = rows[row].ID
		}
	}
	if in.RightPressed && page != 3 && in.MouseX >= x+36 && in.MouseX < x+303 && in.MouseY >= y+99 && in.MouseY < y+453 {
		index := offset + (in.MouseY-y-99)/23
		if index < len(rows) {
			r := rows[index]
			a.social.selected = r.ID
			a.socialUI.EditingMemo = r.ID
			a.socialUI.Name = ui.TextField{Text: r.Memo, Max: 50}
			a.socialUI.Focus = true
		}
	}
	var selected proto.SocialContact
	for _, r := range rows {
		if r.ID == a.social.selected {
			selected = r
			break
		}
	}
	if action >= 0 && !a.social.pending {
		switch {
		case action == 3 && a.socialUI.EditingMemo != 0:
			op := proto.FriendMemo
			if page == 2 {
				op = proto.BlockMemo
			}
			a.sendSocialPacket(proto.SocialMemoPacket(op, a.socialUI.EditingMemo, a.socialUI.Name.Text))
			a.socialUI.EditingMemo = 0
			a.socialUI.Name = ui.TextField{Max: 20}
			a.socialUI.Focus = false
		case page == 3 && action == 0:
			a.sendSocialPacket(proto.NewWriter(proto.PartySearchRequest).Bytes())
		case page == 3 && action == 1:
			if a.social.searchName != "" {
				a.whisperContact(a.social.searchName)
			}
		case page == 3 && action == 2:
			if a.social.searchName != "" {
				a.sendSocialPacket(proto.PartyInviteNamePacket(a.social.searchName))
				a.social.pending = false
			}
		case action == 0:
			op := proto.FriendInvite
			if page == 2 {
				op = proto.BlockAdd
			}
			if a.socialUI.Name.Text != "" {
				a.sendSocialPacket(proto.SocialNamePacket(op, a.socialUI.Name.Text))
				a.socialUI.Name.Text = ""
				a.socialUI.Focus = false
			}
		case page == 1 && action == 1 && selected.ID != 0:
			a.whisperContact(selected.Name)
		case (page == 1 && action == 2 || page == 2 && action == 1) && selected.ID != 0:
			op := proto.FriendDelete
			if page == 2 {
				op = proto.BlockDelete
			}
			a.sendSocialPacket(proto.SocialIDPacket(op, selected.ID))
		}
	}
	return a.socialUI.Focus || panelPointer(in, x, y, ui.GuildWidth, ui.GuildHeight)
}
func (a *App) whisperContact(name string) {
	if a.chatUI == nil || name == "" {
		return
	}
	a.chatUI.Channel, a.chatUI.Recipient = 5, name
	a.chatUI.ModeOpen, a.chatUI.Typing = false, true
	a.chatOpen = true
	a.chatInput = ui.TextField{Max: 120}
	a.chatHistoryIndex, a.chatDraft = len(a.chatHistory), ""
}
func (a *App) drawSocial(w, h int) {
	if a.socialUI == nil {
		return
	}
	x, y := a.guildUI.Origin(w, h)
	classes := [6]string{}
	for i := range classes {
		classes[i] = ui.ClassName(a.msg, a.state.Faction.Faction == 0, uint8(i))
	}
	a.socialUI.Draw(a.sprites, a.font, a.white, x, y, a.guildUI.Page, a.socialRows(), a.socialSearchRows(), a.social.offset, a.social.selected, classes)
	a.guildUI.DrawNavigation(a.sprites, x, y)
}

func (a *App) socialSearchRows() []proto.PartySearchEntry {
	rows := append([]proto.PartySearchEntry(nil), a.state.Social.Search...)
	if a.socialUI == nil {
		return rows
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		if a.socialUI.Descending {
			left, right = right, left
		}
		switch a.socialUI.SortColumn {
		case 2:
			if left.Class != right.Class {
				return left.Class < right.Class
			}
		case 3:
			if left.Level != right.Level {
				return left.Level < right.Level
			}
		}
		return left.Name < right.Name
	})
	return rows
}
