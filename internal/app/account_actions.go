package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type accountAction struct {
	op       proto.Opcode
	id       uint32
	deadline float64
	confirm  proto.CharacterSlot
}

func (a *App) accountText(id int, fallback string) string {
	if text := a.msg.Get(id); text != "" {
		return text
	}
	return fallback
}

func (a *App) sendAccountAction(op proto.Opcode, id uint32) {
	if a.world == nil || a.account.op != 0 {
		return
	}
	packet := proto.SelectCharacterPacket(id)
	switch op {
	case proto.DeleteCharacter:
		packet = proto.DeleteCharacterPacket(id)
	case proto.RestoreCharacter:
		packet = proto.RestoreCharacterPacket(id)
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to send character request.")
		return
	}
	a.account.op, a.account.id, a.account.deadline = op, id, a.clock+10
	a.chars.Busy = true
}

// The modal owns the entire frame, including its closing release/Enter.
func (a *App) updateAccountMessage(in *platform.Input, w, h int) bool {
	if a.accountMessage == nil || !a.accountMessage.Visible {
		return false
	}
	result := a.accountMessage.Update(in, w, h)
	if result == ui.MessageNone {
		return true
	}
	target := a.account.confirm
	a.account.confirm = proto.CharacterSlot{}
	if result == ui.MessageOK && target.ID != 0 {
		for _, ch := range a.state.Slots {
			if ch.ID == target.ID && ch.PendingDelete == target.PendingDelete {
				op := proto.DeleteCharacter
				if ch.PendingDelete {
					op = proto.RestoreCharacter
				}
				a.sendAccountAction(op, ch.ID)
				break
			}
		}
	}
	return true
}

func (a *App) confirmCharacterRemoval(ch proto.CharacterSlot) {
	a.account.confirm = ch
	id, fallback := 5064, "Do you really want to delete this character?"
	if ch.PendingDelete {
		id, fallback = 10093, "Do you want to restore this character?"
	}
	a.accountMessage.Open(a.accountText(id, fallback), true)
}

func (a *App) confirmedAccount(p []byte) {
	op := proto.Op(p)
	if op == proto.CreateCharacter && a.creating {
		result, err := proto.DecodeU8(p)
		if err != nil {
			return // State.Apply already reports malformed packets.
		}
		a.create.Busy = false
		if result == 0 {
			a.creating, a.actor = false, nil
		} else {
			a.accountMessage.Open(a.accountText(5046, "Unable to create character. Check the name."), false)
		}
		return
	}
	if op != a.account.op || a.account.op == 0 {
		return
	}
	r, err := proto.DecodeIDResult(p)
	if err != nil {
		return
	}
	if r.ID != a.account.id {
		log.Printf("character response %04x: id %d, expected %d", op, r.ID, a.account.id)
		return
	}
	a.account, a.chars.Busy = accountAction{}, false
	if r.Result != 0 {
		a.accountMessage.Open("The server rejected the character request.", false)
		return
	}
	if op == proto.DeleteCharacter {
		a.accountMessage.Open(a.accountText(10092, "The character can be restored within two hours."), false)
	} else if op == proto.RestoreCharacter {
		a.accountMessage.Open(a.accountText(10094, "Character restored."), false)
	}
}
