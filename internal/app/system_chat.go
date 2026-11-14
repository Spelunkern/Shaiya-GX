package app

import (
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// systemMessage is the only formatter for native gameplay notices. The data
// archive owns their language and wording; missing IDs do not invent messages.
func (a *App) systemMessage(style uint8, id int, values map[string]string) {
	if a.msg.Get(id) == "" {
		return
	}
	a.state.AddChat(game.ChatLine{Text: a.msg.Format(id, values), Style: style, MessageID: id})
}

// Keep diagnostic fallbacks usable without an archive in offline tools/tests.
func (a *App) localizedNotice(style uint8, id int, values map[string]string, fallback string) {
	if a.msg.Get(id) != "" {
		a.systemMessage(style, id, values)
	} else {
		a.questMessage(fallback)
	}
}

func (a *App) chatEntityName(id uint32) string {
	if id == a.state.SelfID {
		if own, ok := a.selfSlot(); ok {
			return own.Name
		}
	}
	if e := a.state.Entities[id]; e != nil {
		if e.Shape != nil {
			return e.Shape.Name
		}
		if e.Kind == game.KindMob && a.assets != nil {
			if d, err := a.assets.MonsterDefinition(e.TypeID); err == nil {
				return d.Name
			}
		}
	}
	if m := a.state.Party.Member(id); m != nil {
		return m.Name
	}
	return ""
}

// Called only after State.Apply accepts the packet. This reports acknowledgments,
// never predicted attacks, inventory changes, or rewards.
func (a *App) confirmedSystemChat(p []byte, previousMoney uint32) {
	s := a.state
	number := func(n uint32) string { return strconv.FormatUint(uint64(n), 10) }
	itemName := func(typ, id uint8) string {
		if a.arc != nil {
			if d := a.itemsItem(typ, id); d != nil {
				return d.Name
			}
		}
		return ""
	}
	skillName := func(id uint16, level uint8) string { d, _ := a.skillDefinition(id, level); return d.Name }
	switch proto.Op(p) {
	case proto.ExpGain:
		r := proto.NewReader(p)
		gain := r.U32()
		a.systemMessage(3, 50, map[string]string{"b": number(gain)})
	case proto.SetMoney:
		if s.Details.Money > previousMoney {
			a.systemMessage(3, 53, map[string]string{"b": number(s.Details.Money - previousMoney)})
		}
	case proto.LevelUpSelf, proto.LevelUpOther:
		r := proto.NewReader(p)
		id, level := r.U32(), r.U16()
		message := 218
		if id == s.SelfID {
			message = 217
		}
		a.systemMessage(3, message, map[string]string{"p": a.chatEntityName(id), "v": number(uint32(level))})
	case proto.AddItem:
		if len(p) == 4 {
			return
		}
		it, err := proto.DecodeItemAdd(p)
		if err == nil && it.Type != 26 {
			if name := itemName(it.Type, it.TypeID); name != "" {
				a.systemMessage(3, 52, map[string]string{"i": name})
			}
		}
	case proto.UseItem:
		it := s.LastItemUse
		if it.ID == s.SelfID {
			if a.arc != nil {
				if d := a.itemsItem(it.Type, it.TypeID); d != nil && (d.RecoveryHP > 0 || d.RecoveryMP > 0 || d.RecoverySP > 0) {
					return
				}
			}
			if name := itemName(it.Type, it.TypeID); name != "" {
				a.systemMessage(3, 111, map[string]string{"i": name})
			}
		}
	case proto.LearnNewSkill:
		if r := s.LastLearn; r.Result == 0 {
			if name := skillName(r.ID, r.Level); name != "" {
				a.systemMessage(3, 59, map[string]string{"s": name})
			}
		}
	case proto.CharacterMobAutoAttack, proto.CharacterUserAutoAttack, proto.MobAttackCharacter:
		r := s.LastAttack
		target := a.chatEntityName(r.Target)
		message, style := 1000, uint8(0)
		if r.Source == s.SelfID {
			if r.Result == 1 {
				message = 1002
			}
			if r.Result == 3 {
				message = 1004
			}
		} else if r.Target == s.SelfID {
			message, style, target = 1001, 1, a.chatEntityName(r.Source)
			if r.Result == 1 {
				message = 1003
			}
			if r.Result == 3 {
				message = 1005
			}
		} else {
			return
		}
		a.systemMessage(style, message, map[string]string{"t": target, "p": a.chatEntityName(r.Source), "v": number(uint32(r.HP))})
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain, proto.SkillPartyMember, proto.SkillMobOther, proto.MobSkillUse, proto.MobSkillDrain, proto.MobSkillOther:
		r := s.LastSkill
		if r.Source != s.SelfID && r.Target != s.SelfID {
			return
		}
		if r.Result > 3 {
			switch r.Result {
			case 5:
				a.systemMessage(1, 233, nil)
			case 11:
				a.systemMessage(1, 317, nil)
			}
			return
		}
		level := r.Level
		if proto.Op(p) == proto.MobSkillUse || proto.Op(p) == proto.MobSkillDrain || proto.Op(p) == proto.MobSkillOther {
			level = 100
		}
		name := skillName(r.ID, level)
		target := a.chatEntityName(r.Target)
		if r.Result == 3 {
			if r.Source == s.SelfID {
				a.systemMessage(0, 1108, map[string]string{"s": name, "t": target})
			} else {
				a.systemMessage(1, 1306, map[string]string{"s": name, "p": a.chatEntityName(r.Source)})
			}
			return
		}
		id := 1101
		if r.Result == 1 {
			id = 1102
		}
		if a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}].Effect == 2 {
			id = 1104
			if r.Target == s.SelfID {
				id = 1103
			}
		}
		style := uint8(0)
		if r.Source != s.SelfID {
			style = 1
			id = 1301
			if r.Result == 1 {
				id = 1302
			}
			if a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}].Effect == 2 {
				id = 1303
				style = 4
			}
		}
		if r.HP == 0 && r.MP == 0 && r.SP == 0 {
			notice := 1106
			if r.Source != s.SelfID {
				notice = 1305
			}
			a.systemMessage(style, notice, map[string]string{"s": name, "t": target, "p": a.chatEntityName(r.Source)})
			return
		}
		for _, pool := range []struct {
			amount uint16
			name   string
		}{{r.HP, "HP"}, {r.MP, "MP"}, {r.SP, "SP"}} {
			if pool.amount > 0 {
				a.systemMessage(style, id, map[string]string{"s": name, "t": target, "p": a.chatEntityName(r.Source), "v": number(uint32(pool.amount)), "hp": pool.name})
			}
		}
	case proto.PartyInvite, proto.PartyInviteName:
		name := s.Party.InviteName
		if name == "" {
			name = a.chatEntityName(s.Party.InviteID)
		}
		a.systemMessage(17, 204, map[string]string{"p": name})
	case proto.PartyJoined:
		if m, err := proto.DecodePartyJoined(p); err == nil {
			id := 205
			if m.ID == s.SelfID {
				id = 207
			}
			a.systemMessage(3, id, map[string]string{"p": m.Name})
		}
	case proto.PartyLeft, proto.PartyExpelled:
		if id, err := proto.DecodeU32(p); err == nil && id == s.SelfID {
			a.systemMessage(3, 358, nil)
		}
	case proto.CharacterDeath:
		r := proto.NewReader(p)
		id := r.U32()
		if id == s.SelfID {
			a.systemMessage(2, 1, nil)
		}
	}
}
