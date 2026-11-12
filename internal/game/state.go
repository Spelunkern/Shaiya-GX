// Package game holds the client's view of the world. It is pure: no
// sockets, no GL. The loop that owns a State feeds it every payload with
// Apply and sends the requests it builds through the network layer.
package game

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Phase follows the original screens.
type Phase int

const (
	PhaseConnecting Phase = iota
	PhaseFaction          // account has no country yet
	PhaseCharSelect       // 0x0101 list + 0x0109 received
	PhaseLoading          // 0x0104 accepted, initial state arriving
	PhaseInWorld          // 0x0201 for ourselves received
)

func (p Phase) String() string {
	return [...]string{"connecting", "faction", "char-select", "loading", "in-world"}[p]
}

type Kind uint8

const (
	KindUser Kind = iota + 1
	KindMob
	KindNpc
)

// Entity is anything placed in view.
type Entity struct {
	Sitting             bool
	Gesture             uint8
	GestureSerial       uint64
	PartyBadge          uint8
	MountShape          uint8
	Mounted             bool
	GuildID             uint32
	HP                  int32
	HealthKnown         bool
	PvPKills            uint32
	PvPKillsKnown       bool
	Dead                bool
	Kind                Kind
	ID                  uint32
	TypeID              uint16 // mob id or npc type id
	NpcType             uint8
	X, Y, Z             float32
	Motion              uint8
	MoveSerial          uint64
	PositionResetSerial uint64
	Dir                 uint16
	Shape               *proto.UserShape
	Speed               *proto.CharacterSpeed
}

// ChatLine is one received message.
type ChatLine struct {
	Op        proto.Opcode
	Direction uint8
	SenderID  uint32
	Sender    string
	Text      string
}

// Event tells the caller what a payload changed; Text is for logs.
type Event struct {
	Op   proto.Opcode
	Text string
}

type State struct {
	LastDetectInfo       proto.DetectInfo
	LastPeriodicEffect   proto.PeriodicEffect
	DeathLossFree        bool
	LastMotion           proto.CharacterMotionResult
	Blessing             BlessingState
	Mall                 MallState
	Guild                GuildState
	GuildDuel            GuildDuelState
	Exchange             ExchangeState
	Duel                 DuelState
	Quests               map[uint16]proto.ActiveQuest
	FinishedQuests       map[uint16]uint8
	QuestChoice          uint16
	QuestChoosing        bool
	LastQuestEnd         proto.QuestEndResult
	Warehouse            map[uint8]proto.InventoryItem
	WarehouseReady       bool
	WarehouseGold        uint32
	Party                PartyState
	LastItemUse          proto.ItemUseResult
	LastVehicle          proto.VehicleResult
	VehicleSerial        uint64
	vehicleShapeExpected bool
	LastGatekeeper       proto.GatekeeperResult
	GatekeeperSerial     uint64
	LastCompose          proto.ComposeResult
	ComposeSerial        uint64
	LastEnchant          proto.EnchantResult
	EnchantSerial        uint64
	LastGemAdd           proto.GemAddResult
	LastGemRemove        proto.GemRemoveResult
	GemRemoveSerial      uint64
	GemAddSerial         uint64
	LastShop             proto.ShopResult
	SkillBarReady        bool
	Skills               map[uint8]proto.LearnedSkill
	QuickSlots           []proto.QuickSlot
	LastLearn            proto.LearnResult
	LastSkill            proto.SkillResult
	LastSkillExtra       proto.SkillResult
	LastReflectedDamage  proto.ReflectedDamage
	LastCast             proto.SkillCast
	GroundItems          map[uint32]proto.GroundItem
	InventoryReady       bool
	Inventory            map[[2]uint8]proto.InventoryItem
	Target               uint32
	Pools                proto.Pools
	LastAttack           proto.AttackResult
	PvPCountsKnown       [4]bool
	Phase                Phase
	Faction              proto.Faction
	Slots                map[uint8]proto.CharacterSlot // non-empty slots only

	SelfID          uint32
	MapID           uint16
	Details         proto.Details
	Stats           proto.AdditionalStats
	StatsKnown      bool
	LastStatPoints  proto.StatPoints
	Buffs           []proto.ActiveBuff
	LastBuff        proto.ActiveBuff
	LastBuffRemoved uint32
	Self            Entity

	Entities map[uint32]*Entity
	Chat     []ChatLine

	Unknown map[proto.Opcode]int // opcodes received but not modeled
}

func New() *State {
	return &State{Skills: map[uint8]proto.LearnedSkill{}, GroundItems: map[uint32]proto.GroundItem{}, Inventory: map[[2]uint8]proto.InventoryItem{}, Slots: map[uint8]proto.CharacterSlot{}, Entities: map[uint32]*Entity{}, Unknown: map[proto.Opcode]int{}}
}

const maxChat = 200

// Apply updates the state from one payload. Malformed packets are
// reported as an error and leave the state as it was.
func (s *State) Apply(p []byte) (Event, error) {
	op := proto.Op(p)
	ev := Event{Op: op}
	fail := func(err error) (Event, error) { return ev, fmt.Errorf("0x%04X: %w", uint16(op), err) }
	switch op {
	case proto.CharacterMotion:
		m, err := proto.DecodeCharacterMotion(p)
		if err != nil {
			return fail(err)
		}
		s.LastMotion = m
		e := &s.Self
		if m.ID != s.SelfID {
			e = s.Entities[m.ID]
			if e == nil {
				e = &Entity{Kind: KindUser, ID: m.ID}
				s.Entities[m.ID] = e
			}
		}
		if m.Motion == proto.MotionSitDown {
			e.Sitting = true
		} else if m.Motion == proto.MotionStandUp {
			e.Sitting = false
		} else if _, ok := proto.GestureAction(m.Motion); ok || m.Motion == proto.MotionJump || m.Motion == proto.MotionBackstep || m.Motion == proto.MotionEvadeLeft || m.Motion == proto.MotionEvadeRight {
			e.Gesture, e.GestureSerial = m.Motion, e.GestureSerial+1
		}
	case proto.BlessInit, proto.CountryKillCount:
		value, err := proto.DecodeBlessing(p)
		if err != nil {
			return fail(err)
		}
		s.Blessing.apply(op, value, s.Phase == PhaseLoading)
	case proto.UserKillCountUpdate:
		value, err := proto.DecodePvPCountUpdate(p)
		if err != nil {
			return fail(err)
		}
		if value.Index < uint8(len(s.PvPCountsKnown)) {
			s.PvPCountsKnown[value.Index] = true
		}
		switch value.Index {
		case 0:
			s.Details.Kills = value.Value
			s.Self.PvPKills, s.Self.PvPKillsKnown = value.Value, true
			if s.Self.Shape != nil {
				s.Self.Shape.Kills = value.Value
			}
		case 1:
			s.Details.Deaths = value.Value
		case 2:
			s.Details.Wins = value.Value
		case 3:
			s.Details.Losses = value.Value
		}
	case proto.CharacterPvPKills:
		value, err := proto.DecodeVisiblePvPKills(p)
		if err != nil {
			return fail(err)
		}
		entity := &s.Self
		if value.ID != s.SelfID {
			entity = s.Entities[value.ID]
			if entity == nil {
				entity = &Entity{Kind: KindUser, ID: value.ID}
				s.Entities[value.ID] = entity
			}
		}
		entity.PvPKills, entity.PvPKillsKnown = value.Kills, true
		if entity.Shape != nil {
			entity.Shape.Kills = value.Kills
		}
		if value.ID == s.SelfID {
			s.Details.Kills = value.Kills
		}
	case proto.UseVehicle:
		result, err := proto.DecodeVehicleResult(p)
		if err != nil {
			return fail(err)
		}
		s.LastVehicle = result
		s.VehicleSerial++
		if result.Result == 1 {
			s.Self.Mounted = !s.Self.Mounted
			s.vehicleShapeExpected = true
		}
	case proto.CharacterMountShape:
		shape, err := proto.DecodeMountShape(p)
		if err != nil {
			return fail(err)
		}
		if shape.ID == s.SelfID {
			s.Self.MountShape = shape.Shape
			if s.vehicleShapeExpected {
				s.vehicleShapeExpected = false
			} else {
				s.Self.Mounted = shape.Shape != 0
			}
		} else {
			e := s.Entities[shape.ID]
			if e == nil {
				e = &Entity{Kind: KindUser, ID: shape.ID}
				s.Entities[shape.ID] = e
			}
			e.MountShape, e.Mounted = shape.Shape, shape.Shape != 0
		}
	case proto.Gatekeeper:
		result, err := proto.DecodeGatekeeperResult(p)
		if err != nil {
			return fail(err)
		}
		s.Details.Money = result.Money
		s.LastGatekeeper = result
		s.GatekeeperSerial++
	case proto.ItemCompose:
		result, err := proto.DecodeComposeResult(p)
		if err != nil {
			return fail(err)
		}
		key := [2]uint8{result.Bag, result.Slot}
		item, ok := s.Inventory[key]
		if result.Result == 0 && !ok {
			return fail(fmt.Errorf("recreation references missing equipment"))
		}
		if result.Result == 0 {
			item.Craft = result.Craft
			s.Inventory[key] = item
		}
		if result.Bag == 0 && result.Result == 0 {
			s.StatsKnown = false
		}
		s.LastCompose = result
		s.ComposeSerial++
	case proto.EnchantAdd:
		result, err := proto.DecodeEnchantResult(p)
		if err != nil {
			return fail(err)
		}
		lapKey := [2]uint8{result.LapisiaBag, result.LapisiaSlot}
		matKey := [2]uint8{result.MaterialBag, result.MaterialSlot}
		itemKey := [2]uint8{result.ItemBag, result.ItemSlot}
		lap, lapOK := s.Inventory[lapKey]
		mat, matOK := s.Inventory[matKey]
		item, itemOK := s.Inventory[itemKey]
		if !lapOK || !matOK || !itemOK {
			return fail(fmt.Errorf("enchant response references missing item"))
		}
		lap.Count, mat.Count, item.Craft = result.LapisiaCount, result.MaterialCount, result.Craft
		s.applyItem(lap)
		s.applyItem(mat)
		s.Inventory[itemKey] = item
		s.Details.Money = result.Money
		if result.ItemBag == 0 {
			s.StatsKnown = false
		}
		s.LastEnchant = result
		s.EnchantSerial++
	case proto.ItemRepair:
		result, err := proto.DecodeItemRepair(p)
		if err != nil {
			return fail(err)
		}
		s.Details.Money = result.Money
	case proto.GemRemove:
		result, err := proto.DecodeGemRemoveResult(p)
		if err != nil {
			return fail(err)
		}
		key := [2]uint8{result.ItemBag, result.ItemSlot}
		item, ok := s.Inventory[key]
		if result.Result == 1 && !ok {
			return fail(fmt.Errorf("gem removal references missing item"))
		}
		if ok && result.Result == 1 {
			for i, removed := range result.Removed {
				if removed != 0 {
					item.Gems[i] = 0
				}
			}
			s.Inventory[key] = item
		}
		s.Details.Money = result.Money
		if result.ItemBag == 0 {
			s.StatsKnown = false
		}
		s.LastGemRemove = result
		s.GemRemoveSerial++
	case proto.GemAdd:
		result, err := proto.DecodeGemAddResult(p)
		if err != nil {
			return fail(err)
		}
		gemKey, itemKey := [2]uint8{result.GemBag, result.GemSlot}, [2]uint8{result.ItemBag, result.ItemSlot}
		gem, gemOK := s.Inventory[gemKey]
		item, itemOK := s.Inventory[itemKey]
		if !gemOK || (result.Result == 1 && !itemOK) {
			return fail(fmt.Errorf("gem result references missing item"))
		}
		if result.Result == 1 {
			item.Gems[result.Position] = result.GemID
			s.Inventory[itemKey] = item
		}
		gem.Count = result.GemCount
		s.applyItem(gem)
		s.Details.Money = result.Money
		if result.ItemBag == 0 {
			s.StatsKnown = false
		}
		s.LastGemAdd = result
		s.GemAddSerial++
	case proto.GuildDuelRequest, proto.GuildDuelAnswer, proto.GuildDuelCountdown, proto.GuildDuelStart, proto.GuildDuelResult, proto.GuildDuelLeave:
		if err := s.applyGuildDuel(p); err != nil {
			return fail(err)
		}
	case proto.GuildListBegin, proto.GuildListEnd, proto.GuildList, proto.GuildNew, proto.GuildDisbanded, proto.GuildOnlineList, proto.GuildOfflineList, proto.GuildMemberAdded, proto.GuildApplicantList, proto.GuildApplicant, proto.GuildApplicantRemoved, proto.GuildApply, proto.GuildStatus, proto.GuildInvite, proto.GuildInviteAnswer, proto.GuildJoinResult, proto.GuildCreate, proto.GuildCreateAgree, proto.GuildCreateFailed, proto.GuildLeave, proto.GuildDisband, proto.GuildDisbandAgree, proto.GuildExpel, proto.GuildBankList, proto.GuildBankPlaced, proto.GuildBankTaken, proto.GuildBankResult, proto.GuildRankingNotice, proto.GuildRankingPoints, proto.GuildRankingList, proto.GuildEtin, proto.GuildRemark:
		if err := s.applyGuild(p); err != nil {
			return fail(err)
		}
	case proto.DuelRequest, proto.DuelAnswer, proto.DuelCountdown, proto.DuelStart, proto.DuelResult, proto.DuelEnd, proto.DuelWindow, proto.DuelWindowEnd, proto.DuelReady, proto.DuelAdd, proto.DuelRemove, proto.DuelMoney, proto.DuelAddOther:
		if err := s.applyDuel(p); err != nil {
			return fail(err)
		}
	case proto.ExchangeRequest, proto.ExchangeAnswer, proto.ExchangeStart, proto.ExchangeEnd, proto.ExchangeReady, proto.ExchangeAdd, proto.ExchangeAddDest, proto.ExchangeAddMoney, proto.ExchangeConfirm:
		if err := s.applyExchange(p); err != nil {
			return fail(err)
		}
	case proto.QuestList:
		quests, err := proto.DecodeQuestList(p)
		if err != nil {
			return fail(err)
		}
		if s.Quests == nil {
			s.Quests = make(map[uint16]proto.ActiveQuest)
		}
		for _, q := range quests {
			s.Quests[q.ID] = q
		}
	case proto.QuestFinishedList:
		quests, err := proto.DecodeFinishedQuests(p)
		if err != nil {
			return fail(err)
		}
		if s.FinishedQuests == nil {
			s.FinishedQuests = make(map[uint16]uint8)
		}
		for _, q := range quests {
			s.FinishedQuests[q.ID] = q.Success
		}
	case proto.QuestStart:
		_, id, err := proto.DecodeQuestStart(p)
		if err != nil {
			return fail(err)
		}
		if s.Quests == nil {
			s.Quests = make(map[uint16]proto.ActiveQuest)
		}
		s.Quests[id] = proto.ActiveQuest{ID: id}
	case proto.QuestCounter:
		id, slot, value, err := proto.DecodeQuestCounter(p)
		if err != nil {
			return fail(err)
		}
		if slot >= 3 {
			return fail(fmt.Errorf("invalid quest counter slot %d", slot))
		}
		if q, ok := s.Quests[id]; ok {
			q.Counters[slot] = value
			s.Quests[id] = q
		}
	case proto.QuestChoose:
		id, err := proto.DecodeQuestChoice(p)
		if err != nil {
			return fail(err)
		}
		s.QuestChoice, s.QuestChoosing = id, true
	case proto.QuestEnd:
		result, err := proto.DecodeQuestEnd(p)
		if err != nil {
			return fail(err)
		}
		s.LastQuestEnd = result
		delete(s.Quests, result.ID)
		if s.QuestChoosing && s.QuestChoice == result.ID {
			s.QuestChoosing = false
		}
	case proto.PartyRoster, proto.PartyJoined, proto.PartyInvite, proto.PartyInviteName, proto.PartyAnswer, proto.PartyLeft, proto.PartyExpelled, proto.PartyLeader, proto.PartyBadge, proto.PartyPools, proto.PartyLevel, proto.RaidRoster, proto.RaidJoined, proto.PartyLeaveAlt, proto.RaidConvert, proto.RaidDivision, proto.RaidAutoJoin, proto.RaidSwap, proto.RaidSubLeader, proto.RaidInvite, proto.RaidAnswer, proto.RaidDisband, proto.RaidLeader, proto.RaidInviteName, proto.RaidError, proto.RaidExpel, proto.RaidPools:
		if err := s.applyParty(p); err != nil {
			return fail(err)
		}
	case proto.CashPoint, proto.PointBuy, proto.PointGiftSend, proto.PointStoredItemList, proto.PointStoredItemGet:
		if err := s.applyMall(p); err != nil {
			return fail(err)
		}
	case proto.CharacterActiveBuffs:
		buffs, err := proto.DecodeActiveBuffs(p)
		if err != nil {
			return fail(err)
		}
		s.Buffs = nil
		for _, b := range buffs {
			if err := s.addBuff(b); err != nil {
				return fail(err)
			}
		}
	case proto.BuffAdd:
		b, err := proto.DecodeBuffAdd(p)
		if err != nil {
			return fail(err)
		}
		if err := s.addBuff(b); err != nil {
			return fail(err)
		}
		s.LastBuff = b
	case proto.BuffRemove:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return fail(err)
		}
		s.removeBuff(id)
		s.LastBuffRemoved = id
	case proto.SetStatPoint:
		d, err := proto.DecodeStatPoints(p)
		if err != nil {
			return fail(err)
		}
		s.applyStatPoints(d)
	case proto.StatsReset:
		d, err := proto.DecodeStatReset(p)
		if err != nil {
			return fail(err)
		}
		if d.Result == 1 {
			s.Details.StatPoint, s.StatsKnown = d.Points, false
			for i, c := range s.Slots {
				if c.ID == s.SelfID {
					c.Str, c.Rec, c.Int, c.Wis, c.Dex, c.Luc = d.Str, d.Rec, d.Int, d.Wis, d.Dex, d.Luc
					s.Slots[i] = c
				}
			}
		}
	case proto.CharacterAdditionalStats:
		stats, err := proto.DecodeAdditionalStats(p)
		if err != nil {
			return fail(err)
		}
		s.Stats, s.StatsKnown = stats, true
	case proto.CharacterMaxEnergy:
		m, err := proto.DecodeMaxEnergy(p)
		if err != nil {
			return fail(err)
		}
		if m.ID == s.SelfID {
			switch m.Kind {
			case 0:
				s.Details.MaxHP = m.Value
			case 1:
				s.Details.MaxSP = m.Value
			case 2:
				s.Details.MaxMP = m.Value
			}
		}
	case proto.CharacterDeathState:
		m, err := proto.DecodeCharacterSpeed(p)
		if err != nil {
			return fail(err)
		}
		e := &s.Self
		if m.ID != s.SelfID {
			e = s.Entities[m.ID]
			if e == nil {
				e = &Entity{Kind: KindUser, ID: m.ID}
				s.Entities[m.ID] = e
			}
		}
		e.Speed = &m
	case proto.UseItem:
		result, err := proto.DecodeItemUse(p)
		if err != nil {
			return fail(err)
		}
		if result.ID == s.SelfID {
			s.LastItemUse = result
			if result.Type != 0 {
				it := s.Inventory[[2]uint8{result.Bag, result.Slot}]
				it.Bag, it.Slot, it.Type, it.TypeID, it.Count = result.Bag, result.Slot, result.Type, result.TypeID, result.Count
				s.applyItem(it)
			}
		}
	case proto.ShopBuy, proto.ShopSell:
		result, err := proto.DecodeShopResult(p)
		if err != nil {
			return fail(err)
		}
		s.LastShop = result
		if result.Result == 0 {
			s.Details.Money = result.Money
			it := s.Inventory[[2]uint8{result.Bag, result.Slot}]
			it.Bag, it.Slot, it.Type, it.TypeID, it.Count = result.Bag, result.Slot, result.Type, result.TypeID, result.Count
			s.applyItem(it)
		}
	case proto.CharacterSkills:
		skills, err := proto.DecodeSkills(p)
		if err != nil {
			return fail(err)
		}
		s.Skills = map[uint8]proto.LearnedSkill{}
		for _, k := range skills {
			s.Skills[k.Number] = k
		}
	case proto.CharacterSkillBar:
		slots, err := proto.DecodeQuickSlots(p)
		if err != nil {
			return fail(err)
		}
		s.QuickSlots = slots
		s.SkillBarReady = true
	case proto.LearnNewSkill:
		result, err := proto.DecodeLearnResult(p)
		if err != nil {
			return fail(err)
		}
		s.LastLearn = result
		if result.Result == 0 {
			s.Skills[result.Number] = proto.LearnedSkill{ID: result.ID, Level: result.Level, Number: result.Number}
		}
	case proto.SkillCastStarted:
		cast, err := proto.DecodeSkillCast(p)
		if err != nil {
			return fail(err)
		}
		s.LastCast = cast
	case proto.SkillKeep, proto.MobPeriodicDamage:
		result, err := proto.DecodePeriodicEffect(p)
		if err != nil {
			return fail(err)
		}
		s.LastPeriodicEffect = result
	case proto.SkillDetectInfo:
		result, err := proto.DecodeDetectInfo(p)
		if err != nil {
			return fail(err)
		}
		s.LastDetectInfo = result
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain, proto.SkillPartyMember, proto.SkillMobOther:
		result, err := proto.DecodeSkillResult(p)
		if err != nil {
			return fail(err)
		}
		s.LastSkill = result
	case proto.SkillUserExtraDamage, proto.SkillExtraDamage:
		result, err := proto.DecodeSkillExtra(p)
		if err != nil {
			return fail(err)
		}
		s.LastSkillExtra = result
	case proto.SkillUserReflectedDamage, proto.SkillMobReflectedDamage:
		result, err := proto.DecodeReflectedDamage(p)
		if err != nil {
			return fail(err)
		}
		s.LastReflectedDamage = result
	case proto.SkillUsedSpMp:
		r := proto.NewReader(p)
		sp, mp := r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		s.Pools.SP -= min(s.Pools.SP, sp)
		s.Pools.MP -= min(s.Pools.MP, mp)
	case proto.ExpGain:
		r := proto.NewReader(p)
		gain := r.U32()
		r.U32()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		s.Details.Exp = uint32(min(uint64(s.Details.Exp)+uint64(gain), uint64(^uint32(0))))
	case proto.LevelUpSelf, proto.LevelUpOther:
		r := proto.NewReader(p)
		id, level, stats, skills := r.U32(), r.U16(), r.U16(), r.U16()
		prev, next := r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		for i, c := range s.Slots {
			if c.ID == id {
				// Native/Rapture levelUp grants +1 to the job's primary
				// attribute per level; it is not an allocated stat point.
				if level > c.Level && c.Class < 6 {
					var d proto.StatPoints
					d[[...]int{0, 2, 1, 5, 3, 4}[c.Class]] = level - c.Level
					addAttributes(&c, d)
				}
				c.Level = level
				s.Slots[i] = c
			}
		}
		if id == s.SelfID {
			s.StatsKnown = false
			s.Details.StatPoint, s.Details.SkillPoint = stats, skills
			s.Details.PrevExp, s.Details.NextExp = prev, next
		}
	case proto.CharacterMaxHpMpSp:
		r := proto.NewReader(p)
		id, hp, mp, sp := r.U32(), r.U32(), r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		if id == s.SelfID {
			s.Details.MaxHP, s.Details.MaxMP, s.Details.MaxSP = hp, mp, sp
		}
	case proto.MapAddItem:
		it, err := proto.DecodeGroundItem(p)
		if err != nil {
			return fail(err)
		}
		s.GroundItems[it.ID] = it
	case proto.MapRemoveItem:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return fail(err)
		}
		delete(s.GroundItems, id)
	case proto.CharacterItems:
		items, err := proto.DecodeInventory(p)
		if err != nil {
			return fail(err)
		}
		for _, it := range items {
			s.applyItem(it)
		}
		s.InventoryReady = true
	case proto.ItemBagToBag:
		items, money, hasMoney, err := proto.DecodeWarehouseMove(p)
		if err != nil {
			return fail(err)
		}
		for _, it := range items {
			s.applyItem(it)
		}
		if hasMoney {
			s.Details.Money = money
		}
	case proto.WarehouseItemList:
		list, err := proto.DecodeWarehouse(p)
		if err != nil {
			return fail(err)
		}
		stored := make(map[uint8]proto.InventoryItem, len(list.Items))
		for _, it := range list.Items {
			stored[it.Slot] = it
		}
		s.Warehouse, s.WarehouseGold, s.WarehouseReady = stored, list.Gold, true
	case proto.AddItem:
		if len(p) == 4 {
			ev.Text = fmt.Sprintf("item rejected: %d", p[3])
			break
		}
		it, err := proto.DecodeItemAdd(p)
		if err != nil {
			return fail(err)
		}
		// Rapture's gold acknowledgement uses a synthetic slot (26/0).
		// Only 0x0213 changes the balance; this packet must not replace a bag item.
		if it.Type != 26 || it.TypeID != 0 {
			s.applyItem(it)
		}
	case proto.RemoveItem:
		r := proto.NewReader(p)
		bag, slot, typ, id, count := r.U8(), r.U8(), r.U8(), r.U8(), r.U8()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		key := [2]uint8{bag, slot}
		it := s.Inventory[key]
		it.Bag, it.Slot, it.Type, it.TypeID, it.Count = bag, slot, typ, id, count
		s.applyItem(it)
	case proto.SetMoney:
		money, err := proto.DecodeU32(p)
		if err != nil {
			return fail(err)
		}
		s.Details.Money = money
	case proto.TargetMobGetState:
		r := proto.NewReader(p)
		id, hp := r.U32(), r.I32()
		r.U8()
		r.U8()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		if e := s.Entities[id]; e != nil && e.Kind == KindMob {
			e.HP = hp
			e.HealthKnown = true
			e.Dead = hp == 0
		}
	case proto.CharacterResurrect:
		m, err := proto.DecodeResurrection(p)
		if err != nil {
			return fail(err)
		}
		e := s.Entities[m.ID]
		if m.ID == s.SelfID {
			e = &s.Self
			s.Details.Exp = m.Exp
			s.Target = 0
			s.Duel.Outcome = 0
			s.DeathLossFree = false
			s.GuildDuel.deathLeavePending = false
		}
		if e != nil {
			e.Dead = false
			e.PositionResetSerial++
			e.X, e.Y, e.Z = m.X, m.Y, m.Z
			e.Motion = 0
		}
	case proto.CharacterReviveHitpoints:
		r := proto.NewReader(p)
		id := r.U32()
		hp, mp, sp := r.U32(), r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		if id == s.SelfID {
			s.Pools = proto.Pools{HP: hp, MP: mp, SP: sp}
		}
	case proto.CharacterCurrentHitpoints:
		pools, err := proto.DecodePools(p)
		if err != nil {
			return fail(err)
		}
		s.Pools = pools
	case proto.TargetMobHpUpdate, proto.MobCurrentHitpoints:
		hp, err := proto.DecodeMobHealth(p)
		if err != nil {
			return fail(err)
		}
		if e := s.Entities[hp.ID]; e != nil && e.Kind == KindMob {
			e.HP = hp.HP
			e.HealthKnown = true
		}
	case proto.CharacterMobAutoAttack, proto.CharacterUserAutoAttack, proto.MobAttackCharacter:
		attack, err := proto.DecodeAttackResult(p)
		if err != nil {
			return fail(err)
		}
		s.LastAttack = attack
	case proto.MobDeath, proto.CharacterDeath:
		r := proto.NewReader(p)
		id := r.U32()
		deathType := r.U8()
		r.U32()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		if id == s.SelfID && op == proto.CharacterDeath {
			if !s.Self.Dead {
				s.DeathLossFree = deathType == 1 && (s.Duel.Outcome == 2 || s.GuildDuel.deathLeavePending)
				s.GuildDuel.deathLeavePending = false
			}
			s.Self.Dead = true
		}
		if e := s.Entities[id]; e != nil {
			e.Dead = true
			e.HP = 0
			e.HealthKnown = true
		}
		if s.Target == id {
			s.Target = 0
		}
	case proto.AccountFaction:
		f, err := proto.DecodeFaction(p)
		if err != nil {
			return fail(err)
		}
		s.Faction = f
		if f.Faction > 1 {
			s.Phase = PhaseFaction
		} else if s.Phase < PhaseLoading {
			s.Phase = PhaseCharSelect
		}
		ev.Text = fmt.Sprintf("faction=%d maxMode=%d", f.Faction, f.MaxMode)
	case proto.CharacterList:
		c, err := proto.DecodeCharacterSlot(p)
		if err != nil {
			return fail(err)
		}
		if c.ID == 0 {
			delete(s.Slots, c.Slot)
		} else {
			s.Slots[c.Slot] = c
			ev.Text = fmt.Sprintf("slot %d: %s lv%d id=%d", c.Slot, c.Name, c.Level, c.ID)
		}
	case proto.CreateCharacter:
		v, err := proto.DecodeU8(p)
		if err != nil {
			return fail(err)
		}
		ev.Text = fmt.Sprintf("create result=%d", v)
	case proto.DeleteCharacter, proto.RestoreCharacter:
		r, err := proto.DecodeIDResult(p)
		if err != nil {
			return fail(err)
		}
		if r.Result == 0 {
			for slot, ch := range s.Slots {
				if ch.ID == r.ID {
					ch.PendingDelete = op == proto.DeleteCharacter
					s.Slots[slot] = ch
				}
			}
		}
		ev.Text = fmt.Sprintf("result=%d id=%d", r.Result, r.ID)
	case proto.SelectCharacter:
		r, err := proto.DecodeIDResult(p)
		if err != nil {
			return fail(err)
		}
		if r.Result == 0 {
			s.SelfID, s.Phase = r.ID, PhaseLoading
			s.DeathLossFree = false
			s.Blessing = BlessingState{}
			s.PvPCountsKnown = [4]bool{}
			s.Mall = MallState{}
			s.Exchange = ExchangeState{}
			s.Duel = DuelState{}
			s.Guild = GuildState{}
			s.GuildDuel = GuildDuelState{}
			s.Quests, s.FinishedQuests = nil, nil
			s.QuestChoosing, s.QuestChoice = false, 0
			s.Buffs = nil
			s.Inventory = map[[2]uint8]proto.InventoryItem{}
			s.LastVehicle, s.VehicleSerial, s.vehicleShapeExpected = proto.VehicleResult{}, 0, false
			s.LastGatekeeper, s.GatekeeperSerial = proto.GatekeeperResult{}, 0
			s.LastCompose, s.ComposeSerial = proto.ComposeResult{}, 0
			s.LastEnchant, s.EnchantSerial = proto.EnchantResult{}, 0
			s.LastGemAdd, s.GemAddSerial = proto.GemAddResult{}, 0
			s.LastGemRemove, s.GemRemoveSerial = proto.GemRemoveResult{}, 0
			s.InventoryReady = false
			s.Warehouse, s.WarehouseReady = nil, false
			s.Skills = map[uint8]proto.LearnedSkill{}
			s.QuickSlots = nil
			s.SkillBarReady = false
			s.GroundItems = map[uint32]proto.GroundItem{}
			s.Entities = map[uint32]*Entity{}
		}
		ev.Text = fmt.Sprintf("select result=%d id=%d", r.Result, r.ID)
	case proto.CharacterDetails:
		d, err := proto.DecodeDetails(p)
		if err != nil {
			return fail(err)
		}
		s.Details = d
		s.Guild.Name = d.Guild
		s.Self = Entity{Kind: KindUser, ID: s.SelfID, X: d.X, Y: d.Y, Z: d.Z, Dir: d.Dir}
		ev.Text = fmt.Sprintf("details hp=%d pos=(%.1f,%.1f,%.1f) money=%d", d.MaxHP, d.X, d.Y, d.Z, d.Money)
	case proto.CharacterMapTeleport:
		m, err := proto.DecodeMapPosition(p)
		if err != nil {
			return fail(err)
		}
		if m.ID == s.SelfID {
			if s.MapID != m.MapID {
				s.Guild.RankingPointsKnown = false
				s.GroundItems = map[uint32]proto.GroundItem{}
				s.Entities = map[uint32]*Entity{}
				s.Target = 0
			}
			s.MapID, s.Self.X, s.Self.Y, s.Self.Z = m.MapID, m.X, m.Y, m.Z
		}
		ev.Text = fmt.Sprintf("map %d pos=(%.1f,%.1f,%.1f)", m.MapID, m.X, m.Y, m.Z)
	case proto.CharacterEnteredMap:
		u, err := proto.DecodeUserEnter(p)
		if err != nil {
			return fail(err)
		}
		if u.ID == s.SelfID {
			s.Self.X, s.Self.Y, s.Self.Z = u.X, u.Y, u.Z
			s.Guild.ID = u.GuildID
			if info, ok := s.Guild.Directory[u.GuildID]; ok {
				s.Guild.Remark, s.Guild.RemarkKnown = info.Remark, true
			}
			s.Phase = PhaseInWorld
		} else {
			e := s.Entities[u.ID]
			if e == nil {
				e = &Entity{Kind: KindUser, ID: u.ID}
				s.Entities[u.ID] = e
			}
			e.X, e.Y, e.Z, e.GuildID = u.X, u.Y, u.Z, u.GuildID
			e.PositionResetSerial++
		}
		ev.Text = fmt.Sprintf("user %d enters", u.ID)
	case proto.UserLeave, proto.MobLeave, proto.MapNpcLeave:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return fail(err)
		}
		delete(s.Entities, id)
		if s.Target == id {
			s.Target = 0
		}
	case proto.ChangeAppearance:
		m, err := proto.DecodeAppearanceChange(p)
		if err != nil {
			return fail(err)
		}
		if m.ID == s.SelfID {
			for slot, c := range s.Slots {
				if c.ID == m.ID {
					c.Hair, c.Face, c.Height, c.Gender = m.Hair, m.Face, m.Height, m.Gender
					s.Slots[slot] = c
				}
			}
		}
		e := s.Entities[m.ID]
		if m.ID == s.SelfID {
			e = &s.Self
		}
		if e != nil && e.Shape != nil {
			shape := *e.Shape
			shape.Hair, shape.Face, shape.Height, shape.Gender = m.Hair, m.Face, m.Height, m.Gender
			e.Shape = &shape
		}
	case proto.CharacterEquipment:
		m, err := proto.DecodeEquipment(p)
		if err != nil {
			return fail(err)
		}
		e := s.Entities[m.ID]
		if m.ID == s.SelfID {
			e = &s.Self
		}
		if e != nil && e.Shape != nil && m.Slot < 8 {
			copyShape := *e.Shape
			copyShape.Equipment[m.Slot] = [3]uint8{m.Type, m.TypeID, m.Enchant}
			if m.Slot == 7 {
				copyShape.Cloak = m.Cloak
			}
			e.Shape = &copyShape
		}
	case proto.CharacterShape:
		shape, err := proto.DecodeUserShape(p)
		if err != nil {
			return fail(err)
		}
		e := &s.Self
		if shape.ID != s.SelfID {
			e = s.Entities[shape.ID]
			if e == nil {
				e = &Entity{Kind: KindUser, ID: shape.ID}
				s.Entities[shape.ID] = e
			}
		}
		e.Shape = &shape
		e.PvPKills, e.PvPKillsKnown = shape.Kills, true
		e.Dead = shape.Dead != 0
	case proto.CharacterMove:
		m, err := proto.DecodeUserMove(p)
		if err != nil {
			return fail(err)
		}
		e := &s.Self
		if m.ID != s.SelfID {
			if e = s.Entities[m.ID]; e == nil {
				e = &Entity{Kind: KindUser, ID: m.ID}
				s.Entities[m.ID] = e
			}
		}
		e.Motion, e.Dir, e.X, e.Y, e.Z = m.Motion, m.Dir, m.X, m.Y, m.Z
		e.MoveSerial++
	case proto.MobEnter:
		m, err := proto.DecodeMobEnter(p)
		if err != nil {
			return fail(err)
		}
		s.Entities[m.ID] = &Entity{Kind: KindMob, ID: m.ID, TypeID: m.MobID, X: m.X, Z: m.Z}
	case proto.MobMove:
		m, err := proto.DecodeMobMove(p)
		if err != nil {
			return fail(err)
		}
		if e := s.Entities[m.ID]; e != nil {
			e.Motion, e.X, e.Z = m.Motion, m.X, m.Z
			e.MoveSerial++
		}
	case proto.MapNpcEnter:
		n, err := proto.DecodeNpcEnter(p)
		if err != nil {
			return fail(err)
		}
		s.Entities[n.ID] = &Entity{Kind: KindNpc, ID: n.ID, NpcType: n.Type, TypeID: n.TypeID, X: n.X, Y: n.Y, Z: n.Z, Dir: n.Angle}
	case proto.MapNpcMove:
		n, err := proto.DecodeNpcMove(p)
		if err != nil {
			return fail(err)
		}
		if e := s.Entities[n.ID]; e != nil && e.Kind == KindNpc {
			e.Motion, e.X, e.Y, e.Z = n.Motion, n.X, n.Y, n.Z
			e.MoveSerial++
		}
	case proto.ChatNormal, proto.ChatParty, proto.ChatShout, proto.ChatNormalAdmin:
		c, err := proto.DecodeChatByID(p)
		if err != nil {
			return fail(err)
		}
		s.addChat(ChatLine{Op: op, SenderID: c.SenderID, Text: c.Text})
		ev.Text = fmt.Sprintf("chat %d: %s", c.SenderID, c.Text)
	case proto.ChatWorld, proto.ChatGuild, proto.ChatMap, proto.ChatWorldShout, proto.ChatWorldAdmin:
		c, err := proto.DecodeChatByName(p)
		if err != nil {
			return fail(err)
		}
		s.addChat(ChatLine{Op: op, Sender: c.Sender, Text: c.Text})
		ev.Text = fmt.Sprintf("chat %s: %s", c.Sender, c.Text)
	case proto.ChatWhisper:
		c, err := proto.DecodeWhisper(p)
		if err != nil {
			return fail(err)
		}
		s.addChat(ChatLine{Op: op, Sender: c.Sender, Text: c.Text, Direction: c.Direction})
	case proto.ChatError:
		r := proto.NewReader(p)
		code := r.U8()
		if err := r.Err(); err != nil {
			return fail(err)
		}
		message := fmt.Sprintf("Chat error %d.", code)
		if code == 1 {
			message = "Whisper recipient not found or unavailable."
		}
		s.addChat(ChatLine{Op: op, Text: message})
	case proto.Logout:
		s.Party = PartyState{}
		s.Guild = GuildState{}
		s.GuildDuel = GuildDuelState{}
		s.Phase, s.SelfID = PhaseCharSelect, 0
		s.Target = 0
		s.Entities = map[uint32]*Entity{}
		ev.Text = "back to character select"
	default:
		s.Unknown[op]++
	}
	return ev, nil
}

func (s *State) addChat(l ChatLine) {
	if l.Sender == "" && l.SenderID != 0 {
		if e := s.Entities[l.SenderID]; e != nil && e.Shape != nil {
			l.Sender = e.Shape.Name
		}
		if l.Sender == "" {
			for _, slot := range s.Slots {
				if slot.ID == l.SenderID {
					l.Sender = slot.Name
					break
				}
			}
		}
	}
	s.Chat = append(s.Chat, l)
	if len(s.Chat) > maxChat {
		s.Chat = s.Chat[len(s.Chat)-maxChat:]
	}
}

func (s *State) applyItem(it proto.InventoryItem) {
	if it.Bag == proto.GuildBankBag {
		if s.Guild.Bank == nil {
			s.Guild.Bank = make(map[uint8]proto.InventoryItem)
		}
		if it.Type == 0 || it.Count == 0 {
			delete(s.Guild.Bank, it.Slot)
		} else {
			s.Guild.Bank[it.Slot] = it
		}
		return
	}
	if it.Bag == proto.WarehouseBag {
		if s.Warehouse == nil {
			s.Warehouse = map[uint8]proto.InventoryItem{}
		}
		if it.Type == 0 || it.Count == 0 {
			delete(s.Warehouse, it.Slot)
		} else {
			s.Warehouse[it.Slot] = it
		}
		return
	}
	key := [2]uint8{it.Bag, it.Slot}
	if it.Type == 0 || it.Count == 0 {
		delete(s.Inventory, key)
	} else {
		s.Inventory[key] = it
	}
}
