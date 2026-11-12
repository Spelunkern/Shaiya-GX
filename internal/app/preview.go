package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Preview puts a screen in a fixed state for offscreen screenshots.
func (a *App) Preview(name string) error {
	menuPages := map[string]ui.MenuPage{"world-options": ui.MenuMain, "world-graphics": ui.MenuGraphics, "world-sound": ui.MenuSound, "world-user-options": ui.MenuUser, "world-hotkeys": ui.MenuKeys, "world-extra-hotkeys": ui.MenuExtraKeys, "world-help": ui.MenuHelp}
	if page, ok := menuPages[name]; ok {
		if err := a.Preview("world-inventory"); err != nil {
			return err
		}
		a.inventory.open = false
		a.openGameMenu()
		a.gameMenu.Open(page)
		return nil
	}
	if name == "world-guild-manage" {
		if err := a.Preview("world-guild-directory"); err != nil {
			return err
		}
		a.state.Guild.ID, a.state.Guild.Name, a.state.Guild.Rank = 1, "Guardians", 1
		return nil
	}
	if name == "world-guild-bank" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 6, TypeID: 1}
		a.shop.npcID, a.bank.npcID = 42, 42
		a.state.Guild.ID, a.state.Guild.Rank = 1, 1
		a.state.Guild.Bank = map[uint8]proto.InventoryItem{0: {Bag: proto.GuildBankBag, Slot: 0, Type: 25, TypeID: 4, Count: 5}}
		a.bank.guild = true
		return nil
	}
	if name == "world-guild-create" {
		if err := a.Preview("world-guild-directory"); err != nil {
			return err
		}
		a.guildCreateUI.Open()
		a.guildCreateUI.Name.Text = "Guardians"
		a.guildCreateUI.Remark.Text = "Explorers and friends."
		return nil
	}
	if name == "world-guild-directory" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 11}
		a.shop.npcID = 42
		a.state.Guild.DirectoryReady = true
		a.state.Guild.Directory = map[uint32]proto.GuildInfo{
			1: {ID: 1, Name: "Guardians", Master: "Master", Points: 1250, Remark: "Explorers and friends."},
			2: {ID: 2, Name: "Rangers", Master: "Archer", Points: 700, Remark: "Welcome to the guild."},
		}
		a.guild.directoryDirty, a.guild.directorySelected = true, 1
		return nil
	}
	if name == "world-guild-waiting" {
		if err := a.Preview("world-guild"); err != nil {
			return err
		}
		a.state.Guild.Applicants = map[uint32]proto.GuildCandidate{
			11: {ID: 11, Name: "Candidate", Level: 24, Class: 2},
			12: {ID: 12, Name: "Newcomer", Level: 15, Class: 4},
		}
		a.guild.waiting, a.guild.rosterDirty, a.guild.selected = true, true, 11
		return nil
	}
	if name == "world-guild-notice" {
		if err := a.Preview("world-guild"); err != nil {
			return err
		}
		a.state.Guild.Etin, a.state.Guild.EtinKnown = 1234, true
		a.guildCreateUI.OpenNotice(a.state.Guild.Name, "Explorers and friends.")
		return nil
	}
	if name == "world-guild" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		a.state.Guild.ID, a.state.Guild.Name, a.state.Guild.Rank = 42, "Guardians", 1
		a.state.Guild.Members = map[uint32]proto.GuildMember{
			1: {ID: 1, Name: "Master", Rank: 1, Level: 60, Class: 0, Online: true},
			2: {ID: 2, Name: "Ally", Rank: 2, Level: 45, Class: 4, Online: true},
			3: {ID: 3, Name: "Friend", Rank: 9, Level: 20, Class: 5},
		}
		a.guild.open, a.guild.rosterDirty = true, true
		a.guild.selected = 3
		return nil
	}
	if name == "world-duel" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		a.state.Duel.Phase, a.state.Duel.Opponent = game.DuelWager, 42
		a.state.Duel.OwnMoney, a.state.Duel.OtherMoney = 1234, 5678
		a.state.Duel.Own[0].Item = proto.ExchangeItem{Type: 25, TypeID: 4, Count: 3}
		a.state.Duel.Other[1] = proto.ExchangeItem{Type: 25, TypeID: 4, Count: 5}
		return nil
	}
	if name == "world-player-context" {
		if err := a.Preview("world-party"); err != nil {
			return err
		}
		a.playerContext = playerContext{id: 2, x: 260, y: 100, held: -1, open: true}
		return nil
	}
	if name == "world-exchange" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		a.state.Exchange.Active, a.state.Exchange.PartnerID = true, 42
		a.state.Exchange.OwnMoney, a.state.Exchange.OtherMoney = 1234, 5678
		a.state.Exchange.Own[0].Item = proto.ExchangeItem{Type: 25, TypeID: 4, Count: 3}
		a.state.Exchange.Other[1] = proto.ExchangeItem{Type: 25, TypeID: 4, Count: 5}
		return nil
	}
	if name == "world-npcquest" || name == "world-npcquest-detail" || name == "world-npcquest-reward" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		if err := a.loadQuests(); err != nil {
			return err
		}
		a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 8, TypeID: 83}
		a.shop.npcID = 42
		a.npcQuest = npcQuestView{npc: 42}
		if name == "world-npcquest-detail" {
			a.npcQuest.selected = 1
		}
		if name == "world-npcquest-reward" {
			a.state.Entities[42].TypeID = 53
			a.state.Quests = map[uint16]proto.ActiveQuest{226: {ID: 226, Counters: [3]uint8{5}}}
			a.state.QuestChoosing, a.state.QuestChoice = true, 226
			a.npcQuest.selected, a.npcQuest.ending = 226, true
		}
		return nil
	}
	if name == "world-quest" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		if err := a.loadQuests(); err != nil {
			return err
		}
		a.state.Quests = map[uint16]proto.ActiveQuest{1: {ID: 1, Counters: [3]uint8{2, 0, 0}}}
		a.questLog = questLogView{open: true, selected: 1}
		return nil
	}
	if name == "world-recording" || name == "world-recording-next" {
		a.state = game.New()
		a.state.Phase = game.PhaseInWorld
		a.state.MapID, a.state.SelfID = 19, 1
		a.state.Self.X, a.state.Self.Z = 1703, 1632
		a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Recording reference", Race: 0, Class: 5, Gender: 0, Height: 2, Face: 1, Hair: 1}
		a.screen = screenWorld
		a.enterWorld()
		if a.wv.world == nil {
			return fmt.Errorf("recording map 19 missing")
		}
		actor, err := scene.BuildCharacter(a.assets, scene.Appearance{Race: 0, Job: 5, Gender: 0, Height: 2, Face: 1, Hair: 1, Helmet: -1, Upper: 20, Lower: 20, Hand: 20, Foot: 20}, "000_normal")
		if err != nil {
			return err
		}
		a.wv.self = actor
		a.wv.facing = render.V3{0, 0, -1}
		a.wv.eye = a.wv.pos.Add(render.V3{0, 1.5, -4})
		if name == "world-recording-next" {
			actor.Time = 1.0 / 60
		}
		return nil
	}
	if name == "world-fog" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		v := a.wv
		v.pos = render.V3{1024, v.world.Height(1024, 1024), 1024}
		v.eye = v.pos.Add(render.V3{0, 90, -180})
		return nil
	}
	if name == "world-party" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		a.state.Pools = proto.Pools{HP: 250, MP: 50, SP: 100}
		a.state.Details.MaxHP, a.state.Details.MaxMP, a.state.Details.MaxSP = 300, 100, 150
		slot := a.state.Slots[0]
		slot.Name, slot.Level = "Fighter", 4
		a.state.Slots[0] = slot
		a.state.Party.Leader = 1
		for i, name := range []string{"Fighter", "Defender", "Ranger", "Archer", "Mage", "Priest"} {
			a.state.Party.Members = append(a.state.Party.Members, proto.PartyMember{ID: uint32(i + 1), Name: name, Level: uint16(4 + i), Class: uint8(i), HP: 150, MaxHP: 300, MP: 50, MaxMP: 100, SP: 100, MaxSP: 150})
		}
		return nil
	}
	if name == "world-shadow" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		a.wv.eye = a.wv.pos.Add(render.V3{3, 7, -7})
		return nil
	}
	if name == "world-water" || name == "world-water-next" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		v := a.wv
		v.pos = render.V3{1100, v.world.Height(1100, 420), 420}
		v.eye = v.pos.Add(render.V3{0, 25, -20})
		if name == "world-water-next" {
			v.world.Update(.12)
		}
		return nil
	}
	if name == "world-map" || name == "world-map-zoom" {
		if err := a.Preview("world-stream"); err != nil {
			return err
		}
		v := a.mapState()
		v.full = true
		if name == "world-map-zoom" {
			v.zoom, v.u, v.v = 2, .35, .8
		}
		return nil
	}
	if name == "chars-delete" || name == "chars-restore" || name == "create-error" {
		base := "chars"
		if name == "create-error" {
			base = "create"
		}
		if err := a.Preview(base); err != nil {
			return err
		}
		if name == "create-error" {
			a.create.Name.Text = "ExistingHero"
			a.accountMessage.Open(a.accountText(5046, "Name already in use."), false)
		} else {
			ch := a.state.Slots[0]
			ch.PendingDelete = name == "chars-restore"
			a.state.Slots[0] = ch
			a.chars.SetChars(a.state.Slots)
			a.confirmCharacterRemoval(ch)
		}
		return nil
	}
	if name == "create-face" || name == "create-fury-face" || name == "chars-face" || name == "chars-turned" || name == "create-turned" {
		base := name[:strings.LastIndex(name, "-")]
		if err := a.Preview(base); err != nil {
			return err
		}
		nav := &a.chars.Navigation
		if a.creating {
			nav = &a.create.Navigation
		}
		nav.Face = strings.HasSuffix(name, "-face")
		if !nav.Face {
			nav.Angle = .8
		}
		if !a.creating {
			if actor := a.selectActor(a.selectWorld(false)); actor != nil {
				actor.ReleaseAction()
				actor.SetClip("000_normal")
			}
		}
		return nil
	}
	if name == "create-fury" || name == "create-elf" {
		if err := a.Preview("create"); err != nil {
			return err
		}
		if name == "create-fury" {
			a.create.Open(0, 1, 3)
		} else {
			a.create.Draft.Race, a.create.Draft.Class, a.create.Draft.Gender = 1, 4, 1
		}
		return nil
	}

	if name == "world-shop-quantity" {
		if err := a.Preview("world-shop-inventory"); err != nil {
			return err
		}
		a.shop.quantity = &tradeQuantity{npcID: 11, buy: true}
		a.quantityUI.Open(700, 400, 1024, 768, 20)
		return nil
	}

	if name == "world-shop-drag" {
		if err := a.Preview("world-shop-inventory"); err != nil {
			return err
		}
		a.inventory.selected = false
		a.shop.dragging = true
		a.lastMouseX, a.lastMouseY = 900, 420
		return nil
	}

	switch name {
	case "world-stream", "world-eager", "world-stream-seam", "world-eager-seam":
		a.state = game.New()
		a.state.Phase = game.PhaseInWorld
		a.state.MapID, a.state.SelfID = 1, 1
		a.state.Self.X, a.state.Self.Y, a.state.Self.Z = 583.05, 78.56, 1767.63
		a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0, Class: 0, Height: 2}
		if strings.HasSuffix(name, "-seam") {
			a.state.Self.X, a.state.Self.Z = 1024, 1024
		}
		a.screen = screenWorld
		a.enterWorld()
		if strings.HasPrefix(name, "world-eager") {
			a.wv.world.Close()
			var err error
			a.wv.world, err = scene.LoadField(a.assets, "1")
			if err != nil {
				return err
			}
		}
	case "faction":
		a.state = game.New()
		a.state.Phase = game.PhaseFaction
		a.screen = screenCharacters
		a.factionUI.Reset()
	case "login":
		a.login.ID.Text, a.login.Password.Text = "admin1", "secret"
	case "login-settled":
		a.login.ID.Text, a.login.Password.Text = "admin1", "secret"
		if ls := a.loginBackdrop(); ls != nil {
			ls.Camera.Skip()
		}
	case "servers":
		a.servers.Servers = []proto.Server{
			{ID: 1, Name: "Rapture", Users: 10, Capacity: 100},
			{ID: 2, Name: "Busy", Users: 50, Capacity: 100},
			{ID: 3, Name: "Normal", Users: 40, Capacity: 100},
			{ID: 4, Name: "Full", Users: 90, Capacity: 100},
			{ID: 5, Name: "Closed", Status: 2},
		}
		a.servers.Selected = 0
		a.screen = screenServers
	case "world-players", "world-mobs", "world-inventory", "world-skills", "world-shop", "world-death", "world-equipment", "world-hud", "world-chat", "world-status", "world-status-expanded", "world-cast", "world-effect", "world-mesh", "world-projectile", "world-impact", "world-direct-impact", "world-self-effect", "world-shop-inventory":
		a.state = game.New()
		a.state.Phase = game.PhaseInWorld
		a.state.SelfID = 1
		a.screen = screenWorld
		a.wv = &worldView{world: a.selectWorld(false), pos: selectActorPos, facing: render.V3{0, 0, 1}, eye: selectActorPos.Add(render.V3{0, 2, -7})}
		if name == "world-status" || name == "world-status-expanded" {
			a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Status preview", Level: 4, Class: 0, Str: 24, Dex: 17, Rec: 19, Int: 8, Wis: 11, Luc: 13}
			a.state.Details = proto.Details{StatPoint: 12, SkillPoint: 5, MaxHP: 1234, MaxMP: 567, MaxSP: 890, Exp: 250, NextExp: 1000}
			a.state.Pools = proto.Pools{HP: 1234, MP: 567, SP: 890}
			a.state.Stats = proto.AdditionalStats{MinAttack: 123, MaxAttack: 145, MinMagic: 20, MaxMagic: 25, Defense: 60, Resistance: 30}
			a.state.StatsKnown = true
			a.status.open = true
			a.statusUI.Expanded = name == "world-status-expanded"
			if a.statusUI.Expanded {
				a.statusUI.Draft = proto.StatPoints{2, 1, 0, 0, 0, 0}
			}
		}
		if name == "world-chat" {
			a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Chat preview", Level: 4}
			a.state.Chat = append(a.state.Chat, game.ChatLine{Text: "System message preview."})
			for i := 1; i <= 12; i++ {
				a.state.Chat = append(a.state.Chat, game.ChatLine{Sender: "Player2", Text: fmt.Sprintf("Message %d: the chat keeps a readable history.", i)})
			}
			a.state.Chat = append(a.state.Chat, game.ChatLine{Sender: "Player3", Text: "A longer message wraps onto another line instead of disappearing outside the window."})
			a.chatOpen, a.chatInput.Text = true, "/party Ready to go?"
		}
		for i := uint32(0); i < 2; i++ {
			a.state.Entities[i+2] = &game.Entity{Kind: game.KindUser, ID: i + 2, X: selectActorPos[0] + float32(i)*1.5, Y: selectActorPos[1], Z: selectActorPos[2], Shape: &proto.UserShape{ID: i + 2, Name: fmt.Sprintf("Player%d", i+2), Race: uint8(i), Gender: uint8(i), Class: 3 * uint8(i), Height: 2}}
		}
		if name == "world-mobs" || name == "world-inventory" || name == "world-skills" {
			a.state.Entities = map[uint32]*game.Entity{}
			a.state.Target = 12
			for i := uint32(1); i <= 3; i++ {
				a.state.Entities[i+10] = &game.Entity{Kind: game.KindMob, ID: i + 10, TypeID: uint16(i), X: selectActorPos[0] + float32(i-2)*2, Y: selectActorPos[1], Z: selectActorPos[2]}
			}
		}
		if name == "world-hud" {
			a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "HUD preview", Level: 4}
			a.state.Pools = proto.Pools{HP: 167, MP: 44, SP: 55}
			a.state.Details.MaxHP, a.state.Details.MaxMP, a.state.Details.MaxSP = 334, 88, 220
			a.state.Details.Exp, a.state.Details.PrevExp, a.state.Details.NextExp = 450, 300, 900
			a.state.Entities[11] = &game.Entity{Kind: game.KindMob, ID: 11, TypeID: 2009, HP: 75, HealthKnown: true, X: selectActorPos[0] + 2, Y: selectActorPos[1], Z: selectActorPos[2]}
			a.state.Target = 11
			a.state.Skills[1] = proto.LearnedSkill{ID: 30, Level: 1, Number: 1}
			a.skills.cooldowns = map[uint16]float64{30: a.clock + 5}
			a.state.Skills[2] = proto.LearnedSkill{ID: 3, Level: 1, Number: 2}
			a.state.Skills[3] = proto.LearnedSkill{ID: 42, Level: 1, Number: 3}
			a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 25, TypeID: 4, Count: 3}
			a.state.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: 100, Number: 1}, {Bar: 0, Slot: 1, Bag: 100, Number: 2}, {Bar: 0, Slot: 2, Bag: 1, Number: 0}, {Bar: 0, Slot: 9, Bag: 100, Number: 3}}
		}
		if name == "world-cast" {
			sh := a.state.Entities[2].Shape
			sh.Class = 5
			sh.Name = "Casting Ice Arrow"
			delete(a.state.Entities, 3)
			actor, err := scene.BuildCharacter(a.assets, a.appearance(&proto.CharacterSlot{Race: sh.Race, Class: sh.Class, Height: sh.Height}), "000_normal")
			if err != nil {
				return err
			}
			a.wv.actors = map[uint32]*remoteActor{2: {shape: *sh, character: actor}}
			a.state.LastCast = proto.SkillCast{Source: 2, Target: 2, ID: 37, Level: 1}
			a.confirmedAnimation(proto.NewWriter(proto.SkillCastStarted).Bytes())
			a.confirmedEffects(proto.NewWriter(proto.SkillCastStarted).Bytes())
			if !actor.ActionActive() {
				return fmt.Errorf("cast preview: native ready action missing")
			}
		}
		if name == "world-projectile" || name == "world-impact" {
			sh := a.state.Entities[2].Shape
			sh.Class = 5
			sh.Name = "Ice Arrow"
			delete(a.state.Entities, 3)
			actor, err := scene.BuildCharacter(a.assets, a.appearance(&proto.CharacterSlot{Race: sh.Race, Class: sh.Class, Height: sh.Height}), "000_normal")
			if err != nil {
				return err
			}
			a.wv.actors = map[uint32]*remoteActor{2: {shape: *sh, character: actor}}
			a.state.Entities[11] = &game.Entity{Kind: game.KindMob, ID: 11, TypeID: 2009, X: selectActorPos[0] + 3, Y: selectActorPos[1], Z: selectActorPos[2]}
			a.state.LastSkill = proto.SkillResult{Source: 2, Target: 11, ID: 37, Level: 1, HP: 57}
			packet := proto.NewWriter(proto.SkillUse).Bytes()
			a.confirmedAnimation(packet)
			a.confirmedEffects(packet)
			if len(a.wv.projectiles) != 1 {
				return fmt.Errorf("projectile preview: Ice Arrow did not launch")
			}
			if name == "world-projectile" {
				a.wv.projectiles[0].age = -.25
			}
		}
		if name == "world-direct-impact" || name == "world-self-effect" {
			sh := a.state.Entities[2].Shape
			sh.Name = "Double Damage"
			if name == "world-self-effect" {
				sh.Name = "Bash"
			}
			delete(a.state.Entities, 3)
			actor, err := scene.BuildCharacter(a.assets, a.appearance(&proto.CharacterSlot{Race: sh.Race, Class: sh.Class, Height: sh.Height}), "000_normal")
			if err != nil {
				return err
			}
			a.wv.actors = map[uint32]*remoteActor{2: {shape: *sh, character: actor}}
			a.state.Entities[11] = &game.Entity{Kind: game.KindMob, ID: 11, TypeID: 2009, X: selectActorPos[0] + 3, Y: selectActorPos[1], Z: selectActorPos[2]}
			r := proto.SkillResult{Source: 2, Target: 11, ID: 57, Level: 1, HP: 57}
			if name == "world-self-effect" {
				r.ID, r.Target, r.HP = 30, 2, 0
			}
			a.state.LastSkill = r
			packet := proto.NewWriter(proto.SkillUse).Bytes()
			a.confirmedAnimation(packet)
			a.confirmedEffects(packet)
			if len(a.wv.impacts)+len(a.wv.impactCues) != 1 || len(a.wv.projectiles) != 0 {
				return fmt.Errorf("direct effect preview: skill %d did not create one target effect", r.ID)
			}
			if name == "world-direct-impact" && len(a.wv.impactCues) != 1 {
				return fmt.Errorf("direct effect preview: Double Damage did not wait for ALT contact")
			}
		}
		if name == "world-effect" || name == "world-mesh" {
			file := "Effect/CA_ice.EFT"
			if name == "world-mesh" {
				file = "Effect/EnergyBlackhole.EFT"
			}
			var err error
			a.wv.effect, err = scene.LoadEffect(a.assets, file, 0)
			if err != nil {
				return err
			}
			a.wv.effect.Origin = selectActorPos.Add(render.V3{0, 1, 0})
			if name == "world-mesh" {
				// EnergyBlackhole's animated mesh starts at t=1.5 seconds.
				for i := 0; i < 6; i++ {
					a.wv.effect.Update(.25)
				}
			}
			a.state.Entities = map[uint32]*game.Entity{}
		}
		if name == "world-inventory" || name == "world-shop-inventory" {
			a.inventory = inventoryView{open: true, bag: 1, selected: true, source: [2]uint8{1, 0}}
			a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0, Class: 0, Height: 2}
			a.state.InventoryReady = true
			a.state.Inventory[[2]uint8{0, 5}] = proto.InventoryItem{Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1}
			a.state.Inventory[[2]uint8{0, 6}] = proto.InventoryItem{Bag: 0, Slot: 6, Type: 19, TypeID: 1, Count: 1}
			a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}
			a.state.Inventory[[2]uint8{1, 1}] = proto.InventoryItem{Bag: 1, Slot: 1, Type: 25, TypeID: 4, Count: 3}
			a.state.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 19, TypeID: 1, Count: 1}
		}
		if name == "world-skills" {
			a.skills.open = true
			a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0, Class: 0, Level: 1, Mode: 1}
			a.state.Details.SkillPoint = 5
		}
		if name == "world-shop" || name == "world-shop-inventory" {
			a.state.Entities = map[uint32]*game.Entity{11: {Kind: game.KindNpc, ID: 11, NpcType: 1, TypeID: 1, X: selectActorPos[0], Y: selectActorPos[1], Z: selectActorPos[2]}}
			a.shop.npcID = 11
		}
		if name == "world-death" {
			a.state.Entities[2].Dead = true
		}
		if name == "world-equipment" {
			a.wv.eye = selectActorPos.Add(render.V3{0, 2, 7})
			a.state.Entities[2].Shape.Class = 1
			a.state.Entities[2].Shape.Equipment[5] = [3]uint8{1, 1, 0}
			a.state.Entities[2].Shape.Equipment[6] = [3]uint8{19, 1, 0}
			a.state.Entities[3].Shape.Equipment[5] = [3]uint8{11, 1, 0}
		}
	case "create":
		a.state = game.New()
		a.state.Phase = game.PhaseCharSelect
		a.screen = screenCharacters
		a.create.Open(0, 0, 3)
		a.creating = true
	case "chars":
		a.state = game.New()
		a.state.Phase = game.PhaseCharSelect
		a.state.Slots[0] = proto.CharacterSlot{Slot: 0, ID: 1, Name: "Interloper", Level: 60, Race: 0, Class: 0, Mode: 2, MapID: 42, Hair: 1, Face: 1, Height: 2}
		a.state.Slots[1] = proto.CharacterSlot{Slot: 1, ID: 2, Name: "Valiant", Level: 60, Class: 1, Mode: 2, MapID: 1}
		a.state.Slots[3] = proto.CharacterSlot{Slot: 3, ID: 4, Name: "Sylvan", Level: 12, Class: 4, Mode: 1, MapID: 3}
		a.screen = screenCharacters
		a.chars.SetChars(a.state.Slots)
		a.world = nil
		return nil
	default:
		return fmt.Errorf("unknown preview %q", name)
	}
	return nil
}
