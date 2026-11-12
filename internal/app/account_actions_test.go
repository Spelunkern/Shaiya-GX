package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestAccountTimeoutDisconnectsWithoutChangingCharacter(t *testing.T) {
	a := inputApp()
	a.state.Phase = game.PhaseCharSelect
	a.state.Slots[0] = proto.CharacterSlot{ID: 10}
	st := a.state
	a.account = accountAction{op: proto.DeleteCharacter, id: 10, deadline: 1}
	a.accountMessage.Open("Pending", false)
	a.updateCharacters(&platform.Input{}, 1024, 768, 2)
	if a.state != nil || a.screen != screenLogin || a.accountMessage.Visible || a.account.op != 0 {
		t.Fatal("timed-out request left a usable ambiguous account session")
	}
	if st.Slots[0].PendingDelete {
		t.Fatal("timeout predicted a deletion")
	}
}

func TestAccountResponseMustMatchPendingCharacter(t *testing.T) {
	a := inputApp()
	a.account = accountAction{op: proto.DeleteCharacter, id: 10, deadline: 10}
	w := proto.NewWriter(proto.DeleteCharacter)
	w.U8(0)
	w.U32(20)
	a.confirmedAccount(w.Bytes())
	if a.account.op != proto.DeleteCharacter || a.account.id != 10 || a.accountMessage.Visible {
		t.Fatal("another character's response completed the pending request")
	}
}

func TestAccountModalOwnsDismissalAndRosterChangesInvalidateConfirmation(t *testing.T) {
	a := inputApp()
	a.state.Phase = game.PhaseCharSelect
	a.state.Slots[0] = proto.CharacterSlot{ID: 10}
	a.confirmCharacterRemoval(a.state.Slots[0])
	a.state.Slots[0] = proto.CharacterSlot{ID: 20}
	if !a.updateAccountMessage(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768) {
		t.Fatal("closing Enter was not consumed")
	}
	if a.account.confirm.ID != 0 || a.account.op != 0 || a.accountMessage.Visible {
		t.Fatal("stale confirmation was retained or changed its target")
	}
}
