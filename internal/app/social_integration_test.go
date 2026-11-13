//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func verifyNativeSocial(t *testing.T, a, b *App, pump func(func() bool)) {
	t.Helper()
	name := b.chatEntityName(b.state.SelfID)
	a.sendSocialPacket(proto.SocialNamePacket(proto.FriendInvite, name))
	pump(func() bool { return b.socialInviteUI.Visible })
	if len(a.state.Social.Friends) != 0 {
		t.Fatal("friend predicted before acceptance")
	}
	b.updateSocialInvitation(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 734)
	pump(func() bool {
		return a.state.Social.Friends[b.state.SelfID].Online && b.state.Social.Friends[a.state.SelfID].Online
	})
	a.sendSocialPacket(proto.SocialMemoPacket(proto.FriendMemo, b.state.SelfID, "Tank"))
	pump(func() bool { return a.state.Social.Friends[b.state.SelfID].Memo == "Tank" })
	a.sendSocialPacket(proto.SocialNamePacket(proto.BlockAdd, name))
	pump(func() bool { return a.state.Social.Blocked[b.state.SelfID].Name == name })
	before := len(a.state.Chat)
	beforeB := len(b.state.Chat)
	if err := b.world.Send(proto.WhisperPacket(a.chatEntityName(a.state.SelfID), "blocked whisper")); err != nil {
		t.Fatal(err)
	}
	pump(func() bool {
		for _, line := range b.state.Chat[beforeB:] {
			if line.MessageID == 347 {
				return true
			}
		}
		return false
	})
	for _, line := range a.state.Chat[before:] {
		if line.Text == "blocked whisper" {
			t.Fatal("blocked whisper delivered")
		}
	}
	a.sendSocialPacket(proto.SocialIDPacket(proto.BlockDelete, b.state.SelfID))
	pump(func() bool { return len(a.state.Social.Blocked) == 0 })
	b.sendSocialPacket(proto.NewWriter(proto.PartySearchRequest).Bytes())
	pump(func() bool { return !b.social.pending })
	a.state.Social.Search = nil
	a.sendSocialPacket(proto.NewWriter(proto.PartySearchRequest).Bytes())
	pump(func() bool { return len(a.state.Social.Search) > 0 })
	found := false
	for _, row := range a.state.Social.Search {
		if row.Name == name {
			found = true
		}
	}
	if !found {
		t.Fatal("party search omitted eligible partner")
	}
	a.sendSocialPacket(proto.SocialIDPacket(proto.FriendDelete, b.state.SelfID))
	pump(func() bool { return len(a.state.Social.Friends) == 0 })
	t.Log("native Buddy invitation/acceptance, memo, Block whisper enforcement, unblock, party search and removal confirmed")
}
