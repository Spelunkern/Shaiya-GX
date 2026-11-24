# Shared codec appendix

> Generated from Go source by `tools/generate_server_reference.py`; do not edit by hand.

Complete shared codec source inventory, including helper readers, nested row definitions and codecs without a statically mapped opcode. Each file lists types and function definitions. This closes gaps left by dynamic dispatch; it is not a claim that every declared packet is accepted by the server.

[Opcode catalog](server-opcodes.md) · [Wire conventions](server-protocol.md)

## auction

[internal/proto/auction.go](../../internal/proto/auction.go#L1)

<details><summary>Types and codecs</summary>

```go
func AuctionHours(duration uint8) int64 {
	if int(duration) >= len(auctionHours) {
		return 0
	}
	return auctionHours[duration]
}
```

```go
func AuctionDeposit(start uint32, duration uint8) uint32 {
	return uint32(AuctionHours(duration)) * 5 * uint32((uint64(start)+499999)/500000)
}
```

```go
type AuctionRequest struct {
	Item                         *InventoryItem
	Sequence                     uint64
	Sort                         int
	Action                       string
	NPC                          uint32
	ID, Version                  int64
	View, Page                   int
	Query                        string
	Category, MinLevel, MaxLevel int
	Subcategory                  int    `json:",omitempty"`
	Faction                      uint8  `json:",omitempty"`
	NameGrade                    *uint8 `json:",omitempty"`
	ItemType, ItemID             uint8
	Duration                     uint8
	Count                        uint16
	Start, Buyout, Amount        uint32
	Source                       string
}
```

```go
type AuctionRow struct {
	ID, Version                 int64
	Name, Seller, Craft, Reason string
	Type, TypeID                uint8
	Level, Count, Quality       uint16
	Gems                        [6]uint8
	Price, Buyout, Gold         uint32
	Ends                        int64
	Active, Mine, HighBidder    bool
}
```

```go
type AuctionReply struct {
	Sequence       uint64
	NPC            uint32
	View, Page     int
	More           bool
	Error, Message string
	Rows           []AuctionRow
}
```

```go
func AuctionPacket(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil || len(body) > 12000 {
		return nil
	}
	w := NewWriter(GXAuction)
	w.Raw(body)
	return w.Bytes()
}
```

```go
func DecodeAuction(p []byte, limit int, value any) error {
	if len(p) < 3 || len(p) > limit || Op(p) != GXAuction {
		return errors.New("invalid auction packet")
	}
	return json.Unmarshal(p[2:], value)
}
```

```go
func AuctionCategory(realType, itemType uint8) int {
	switch {
	case realType >= 1 && realType <= 15:
		return 1
	case realType >= 16 && realType <= 21 || realType == 24:
		return 2
	case realType == 22 || realType == 23 || realType == 25:
		return 3
	case itemType == 30 || itemType == 95:
		return 4
	default:
		return 5
	}
}
```

```go
func (r AuctionRequest) ValidSearchFilters() bool {
	return r.Category >= 0 && r.Category < len(AuctionSubcategories) &&
		r.Subcategory >= 0 && r.Subcategory <= len(AuctionSubcategories[r.Category]) &&
		r.Faction <= AuctionFury && (r.NameGrade == nil || *r.NameGrade <= 5 || *r.NameGrade == 7)
}
```

</details>

## blessing

[internal/proto/blessing.go](../../internal/proto/blessing.go#L1)

<details><summary>Types and codecs</summary>

```go
type BlessingUpdate struct {
	Country            uint8
	Count, RemainingMS uint32
}
```

```go
func DecodeBlessing(p []byte) (BlessingUpdate, error) {
	r := NewReader(p)
	b := BlessingUpdate{Country: r.U8(), Count: r.U32()}
	if Op(p) == BlessInit {
		b.RemainingMS = r.U32()
	}
	if err := r.Err(); err != nil {
		return b, err
	}
	if b.Country > 1 || r.Remaining() != 0 {
		return b, fmt.Errorf("invalid blessing country or trailing bytes")
	}
	return b, nil
}
```

```go
type BlessingSnapshot struct {
	Country                     uint8
	Entry                       bool
	Count, Maximum, RemainingMS uint32
	Bonuses                     [27]uint16
}
```

```go
func DecodeBlessingSnapshot(p []byte) (BlessingSnapshot, error) {
	r := NewReader(p)
	country, flags := r.U8(), r.U8()
	b := BlessingSnapshot{Country: country, Entry: flags == 1, Count: r.U32(), Maximum: r.U32(), RemainingMS: r.U32()}
	for i := range b.Bonuses {
		b.Bonuses[i] = r.U16()
	}
	if err := r.Err(); err != nil {
		return b, err
	}
	if country > 1 || flags > 1 || b.Maximum == 0 || b.Count > b.Maximum || b.RemainingMS > 600000 ||
		b.RemainingMS != 0 && b.Count != b.Maximum || r.Remaining() != 0 {
		return b, fmt.Errorf("invalid blessing snapshot")
	}
	return b, nil
}
```

</details>

## buffs

[internal/proto/buffs.go](../../internal/proto/buffs.go#L1)

<details><summary>Types and codecs</summary>

```go
type ActiveBuff struct {
	Instance     uint32
	Skill        uint16
	Level        uint8
	Remaining    uint32
	HasRemaining bool
}
```

```go
func DecodeActiveBuffs(p []byte) (out []ActiveBuff, err error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/11 {
		return nil, ErrShort
	}
	out = make([]ActiveBuff, n)
	for i := range out {
		out[i] = ActiveBuff{Instance: r.U32(), Skill: r.U16(), Level: r.U8(), Remaining: r.U32(), HasRemaining: true}
	}
	return out, r.Err()
}
```

```go
func DecodeBuffAdd(p []byte) (b ActiveBuff, err error) {
	r := NewReader(p)
	b.Instance, b.Skill, b.Level = r.U32(), r.U16(), r.U8()
	return b, r.Err()
}
```

```go
func CancelBuffPacket(skill uint16) []byte { return UseSkillPacket(0xff, uint32(skill), true) }
```

</details>

## chat items

[internal/proto/chat_items.go](../../internal/proto/chat_items.go#L1)

<details><summary>Types and codecs</summary>

```go
type ChatItems [MaxChatItems]InventoryItem
```

```go
func (items ChatItems) Len() int {
	for i, item := range items {
		if item.Type == 0 {
			return i
		}
	}
	return len(items)
}
```

```go
func WithChatItemSlots(p []byte, slots [][2]uint8) []byte {
	if len(p) == 0 || len(slots) == 0 || len(slots) > MaxChatItems {
		return p
	}
	p = append(p, chatItemsTag, byte(len(slots)))
	for _, slot := range slots {
		p = append(p, slot[:]...)
	}
	return p
}
```

```go
func ChatItemSlots(p []byte, textOffset int) ([][2]uint8, bool) {
	if len(p) <= textOffset {
		return nil, false
	}
	end := textOffset + 1 + int(p[textOffset])
	if end > len(p) {
		return nil, false
	}
	tail := p[end:]
	if len(tail) == 0 {
		return nil, true
	}
	if len(tail) < 2 || tail[0] != chatItemsTag {
		return nil, false
	}
	n := int(tail[1])
	if n < 1 || n > MaxChatItems || len(tail) != 2+n*2 {
		return nil, false
	}
	slots := make([][2]uint8, n)
	for i := range slots {
		copy(slots[i][:], tail[2+i*2:4+i*2])
	}
	return slots, true
}
```

```go
func WithChatItems(p []byte, items ChatItems) []byte {
	count := items.Len()
	if count == 0 {
		return p
	}
	w := NewWriter(0)
	for _, it := range items[:count] {
		w.U8(it.Type)
		w.U8(it.TypeID)
		w.U16(it.Quality)
		w.Count(it.Count)
		for _, gem := range it.Gems {
			w.U8(gem)
		}
		w.Fixed(it.Craft, 21, false)
	}
	p = append(p, chatItemsTag, byte(count))
	return append(p, w.Bytes()[2:]...)
}
```

```go
func readChatItems(r *Reader) (items ChatItems) {
	if r.Remaining() < 2 || r.U8() != chatItemsTag {
		return items
	}
	n := int(r.U8())
	if n < 1 || n > MaxChatItems || r.Remaining() != n*32 {
		return items
	}
	for i := 0; i < n; i++ {
		it := &items[i]
		it.Type, it.TypeID, it.Quality, it.Count = r.U8(), r.U8(), r.U16(), r.Count()
		for j := range it.Gems {
			it.Gems[j] = r.U8()
		}
		it.Craft = r.Fixed(21)
		if it.Type == 0 {
			return ChatItems{}
		}
	}
	return items
}
```

</details>

## cloak fits

[internal/proto/cloak_fits.go](../../internal/proto/cloak_fits.go#L1)

<details><summary>Types and codecs</summary>

```go
type ChestCloakFit struct{ Type, TypeID, Fit uint8 }
```

```go
func ChestCloakFitsPacket(rows []ChestCloakFit) []byte {
	w := NewWriter(GXChestCloakFits)
	w.U16(uint16(len(rows)))
	for _, v := range rows {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Fit)
	}
	return w.Bytes()
}
```

```go
func DecodeChestCloakFits(p []byte) ([]ChestCloakFit, error) {
	if len(p) < 4 || Op(p) != GXChestCloakFits || len(p) != 4+3*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid chest cloak fits")
	}
	rows := make([]ChestCloakFit, 0, (len(p)-4)/3)
	for i := 4; i < len(p); i += 3 {
		v := ChestCloakFit{p[i], p[i+1], p[i+2]}
		if v.Type == 0 || v.TypeID == 0 || v.Fit > 7 {
			return nil, fmt.Errorf("invalid chest cloak fit")
		}
		rows = append(rows, v)
	}
	return rows, nil
}
```

</details>

## combat

[internal/proto/combat.go](../../internal/proto/combat.go#L1)

<details><summary>Types and codecs</summary>

```go
type Pools struct{ HP, MP, SP uint32 }
```

```go
func DecodePools(p []byte) (m Pools, err error) {
	r := NewReader(p)
	m.HP, m.MP, m.SP = r.U32(), r.U32(), r.U32()
	return m, r.Err()
}
```

```go
type MobHealth struct {
	ID uint32
	HP int32
}
```

```go
func DecodeMobHealth(p []byte) (m MobHealth, err error) {
	r := NewReader(p)
	m.ID, m.HP = r.U32(), r.I32()
	return m, r.Err()
}
```

```go
type AttackResult struct {
	Result         uint8
	Source, Target uint32
	HP, SP, MP     uint16
}
```

```go
func DecodeAttackResult(p []byte) (m AttackResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	return m, r.Err()
}
```

```go
type Resurrection struct {
	ID      uint32
	Kind    uint8
	Exp     uint32
	X, Y, Z float32
}
```

```go
func DecodeResurrection(p []byte) (m Resurrection, err error) {
	r := NewReader(p)
	m.ID, m.Kind, m.Exp = r.U32(), r.U8(), r.U32()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}
```

```go
func TargetMobPacket(id uint32) []byte { return idPacket(TargetMobGetState, id) }
```

```go
type ReviveMode uint8
```

```go
func RevivePacket() []byte { return ReviveChoicePacket(ReviveBind) }
```

```go
func ReviveChoicePacket(mode ReviveMode) []byte {
	w := NewWriter(ResurrectRequest)
	w.U8(uint8(mode))
	return w.Bytes()
}
```

```go
func StopAttackPacket() []byte { return NewWriter(AutoAttackStop).Bytes() }
```

</details>

## compose

[internal/proto/compose.go](../../internal/proto/compose.go#L1)

<details><summary>Types and codecs</summary>

```go
type ComposeResult struct {
	Result, Bag, Slot uint8
	Craft             string
}
```

```go
func DecodeComposeResult(p []byte) (ComposeResult, error) {
	r := NewReader(p)
	v := ComposeResult{Result: r.U8(), Bag: r.U8(), Slot: r.U8(), Craft: r.Fixed(21)}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.Bag > 5 || v.Slot >= 24 {
		return v, fmt.Errorf("invalid recreation result")
	}
	return v, nil
}
```

```go
func ComposePacket(runeBag, runeSlot, itemBag, itemSlot uint8) []byte {
	w := NewWriter(ItemCompose)
	w.Raw([]byte{runeBag, runeSlot, itemBag, itemSlot})
	return w.Bytes()
}
```

</details>

## detect

[internal/proto/detect.go](../../internal/proto/detect.go#L1)

<details><summary>Types and codecs</summary>

```go
type DetectInfo struct {
	DefenseElement uint32
	Level          uint16
}
```

```go
func DecodeDetectInfo(p []byte) (v DetectInfo, err error) {
	r := NewReader(p)
	v.DefenseElement, v.Level = r.U32(), r.U16()
	r.U8() // reserved byte
	return v, r.Err()
}
```

</details>

## duel

[internal/proto/duel.go](../../internal/proto/duel.go#L1)

<details><summary>Types and codecs</summary>

```go
type DuelChallenge struct{ Requester, Target uint32 }
```

```go
type DuelResponse struct {
	Result uint8
	Player uint32
}
```

```go
func DecodeDuelChallenge(p []byte) (d DuelChallenge, err error) {
	r := NewReader(p)
	d = DuelChallenge{r.U32(), r.U32()}
	return d, r.Err()
}
```

```go
func DecodeDuelResponse(p []byte) (d DuelResponse, err error) {
	r := NewReader(p)
	d = DuelResponse{r.U8(), r.U32()}
	return d, r.Err()
}
```

```go
func DecodeDuelCenter(p []byte) (x, z float32, err error) {
	r := NewReader(p)
	x, z = r.F32(), r.F32()
	return x, z, r.Err()
}
```

```go
func DecodeDuelEmpty(p []byte) error     { return NewReader(p).Err() }
```

```go
func DuelRequestPacket(id uint32) []byte { return idPacket(DuelRequest, id) }
```

```go
func DuelAnswerPacket(accept bool) []byte {
	w := NewWriter(DuelAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
func DuelGiveUpPacket() []byte       { return NewWriter(DuelEnd).Bytes() }
```

```go
func DuelReadyPacket(v uint8) []byte { w := NewWriter(DuelReady); w.U8(v); return w.Bytes() }
```

```go
func DuelAddPacket(bag, slot uint8, count uint16, dest uint8) []byte {
	w := NewWriter(DuelAdd)
	w.Raw([]byte{bag, slot})
	w.Count(count)
	w.U8(dest)
	return w.Bytes()
}
```

```go
func DuelRemovePacket(slot uint8) []byte  { w := NewWriter(DuelRemove); w.U8(slot); return w.Bytes() }
```

```go
func DuelMoneyPacket(money uint32) []byte { w := NewWriter(DuelMoney); w.U32(money); return w.Bytes() }
```

</details>

## emblem

[internal/proto/emblem.go](../../internal/proto/emblem.go#L1)

<details><summary>Types and codecs</summary>

```go
type EmblemResult struct {
	Result, Bag, Slot uint8
	Pattern           [6]uint8
	Money             uint32
}
```

```go
func DecodeEmblemResult(p []byte) (EmblemResult, error) {
	r := NewReader(p)
	v := EmblemResult{Result: r.U8(), Bag: r.U8(), Slot: r.U8()}
	for i := range v.Pattern {
		v.Pattern[i] = r.U8()
	}
	v.Money = r.U32()
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.Bag > 5 || v.Slot >= 24 {
		return v, fmt.Errorf("invalid emblem result")
	}
	return v, nil
}
```

```go
type EmblemRemoved struct{ Bag, Slot uint8 }
```

```go
func DecodeEmblemRemoved(p []byte) (EmblemRemoved, error) {
	r := NewReader(p)
	v := EmblemRemoved{Bag: r.U8(), Slot: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Bag > 5 || v.Slot >= 24 {
		return v, fmt.Errorf("invalid emblem removal")
	}
	return v, nil
}
```

```go
func EmblemApplyPacket(bag, slot uint8, pattern [6]uint8) []byte {
	w := NewWriter(CloakAdd)
	w.Raw([]byte{bag, slot})
	w.Raw(pattern[:])
	return w.Bytes()
}
```

```go
func EmblemRemovePacket(bag, slot uint8) []byte {
	w := NewWriter(CloakRemove)
	w.Raw([]byte{bag, slot})
	return w.Bytes()
}
```

</details>

## enchant

[internal/proto/enchant.go](../../internal/proto/enchant.go#L1)

<details><summary>Types and codecs</summary>

```go
type EnchantResult struct {
	Result, LapisiaBag, LapisiaSlot uint8
	LapisiaCount                    uint16
	MaterialBag, MaterialSlot       uint8
	MaterialCount                   uint16
	ItemBag, ItemSlot               uint8
	Money                           uint32
	Craft                           string
}
```

```go
func DecodeEnchantResult(p []byte) (EnchantResult, error) {
	r := NewReader(p)
	v := EnchantResult{Result: r.U8(), LapisiaBag: r.U8(), LapisiaSlot: r.U8(), LapisiaCount: r.Count(),
		MaterialBag: r.U8(), MaterialSlot: r.U8(), MaterialCount: r.Count(), ItemBag: r.U8(), ItemSlot: r.U8(), Money: r.U32(), Craft: r.Fixed(21)}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.LapisiaBag > 5 || v.MaterialBag > 5 || v.ItemBag > 5 || v.LapisiaSlot >= 24 || v.MaterialSlot >= 24 || v.ItemSlot >= 24 {
		return v, fmt.Errorf("invalid enchant result")
	}
	l, m, i := [2]uint8{v.LapisiaBag, v.LapisiaSlot}, [2]uint8{v.MaterialBag, v.MaterialSlot}, [2]uint8{v.ItemBag, v.ItemSlot}
	if l == m || l == i || m == i {
		return v, fmt.Errorf("overlapping enchant positions")
	}
	return v, nil
}
```

```go
func EnchantPacket(lapisiaBag, lapisiaSlot, materialBag, materialSlot uint8, materialCount uint16, itemBag, itemSlot uint8) []byte {
	w := NewWriter(EnchantAdd)
	w.Raw([]byte{lapisiaBag, lapisiaSlot, materialBag, materialSlot})
	w.Count(materialCount)
	w.Raw([]byte{itemBag, itemSlot})
	return w.Bytes()
}
```

</details>

## enchant rules

[internal/proto/enchant_rules.go](../../internal/proto/enchant_rules.go#L1)

<details><summary>Types and codecs</summary>

```go
type EnchantRules struct {
	MaxLevel                uint8
	WeaponBonus, ArmorBonus [50]uint32
}
```

```go
func EnchantRulesPacket(r EnchantRules) []byte {
	w := NewWriter(GXEnchantRules)
	w.U8(r.MaxLevel)
	for _, table := range [2][50]uint32{r.WeaponBonus, r.ArmorBonus} {
		for _, v := range table {
			w.U32(v)
		}
	}
	return w.Bytes()
}
```

```go
func DecodeEnchantRules(p []byte) (EnchantRules, error) {
	var r EnchantRules
	if len(p) != 403 || Op(p) != GXEnchantRules || p[2] > 49 {
		return r, fmt.Errorf("invalid enchant rules")
	}
	r.MaxLevel = p[2]
	off := 3
	for _, table := range []*[50]uint32{&r.WeaponBonus, &r.ArmorBonus} {
		for i := range table {
			v := binary.LittleEndian.Uint32(p[off:])
			off += 4
			if v > 2147483647 {
				return EnchantRules{}, fmt.Errorf("invalid enchant bonus")
			}
			table[i] = v
		}
	}
	return r, nil
}
```

</details>

## entities

[internal/proto/entities.go](../../internal/proto/entities.go#L1)

<details><summary>Types and codecs</summary>

```go
type AppearanceChange struct {
	ID                         uint32
	Hair, Face, Height, Gender uint8
}
```

```go
func ChangeAppearancePacket(bag, slot, hair, face, height, gender uint8) []byte {
	w := NewWriter(ChangeAppearance)
	w.U8(bag)
	w.U8(slot)
	w.U8(hair)
	w.U8(face)
	w.U8(height)
	w.U8(gender)
	return w.Bytes()
}
```

```go
func DecodeAppearanceChange(p []byte) (m AppearanceChange, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Hair, m.Face, m.Height, m.Gender = r.U8(), r.U8(), r.U8(), r.U8()
	return m, r.Err()
}
```

```go
type UserShape struct {
	ID                                                 uint32
	Dead, Motion, Faction                              uint8
	Race, Hair, Face, Height, Class, Gender, Unk, Mode uint8
	Kills                                              uint32
	Equipment                                          [8][3]uint8
	Name                                               string
	Cloak                                              [6]uint8
	Guild                                              string
}
```

```go
func DecodeUserShape(p []byte) (m UserShape, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Dead, m.Motion, m.Faction = r.U8(), r.U8(), r.U8()
	m.Race, m.Hair, m.Face, m.Height, m.Class, m.Gender, m.Unk, m.Mode = r.U8(), r.U8(), r.U8(), r.U8(), r.U8(), r.U8(), r.U8(), r.U8()
	m.Kills = r.U32()
	for i := range m.Equipment {
		for j := range m.Equipment[i] {
			m.Equipment[i][j] = r.U8()
		}
	}
	m.Name = r.Fixed(21)
	// The original reads cloak gems only when equipment slot 7 is set.
	// Rapture also sends six zero bytes without a cloak (discrepancy #2).
	if m.Equipment[7][0] != 0 || r.Remaining() == 31 {
		for i := range m.Cloak {
			m.Cloak[i] = r.U8()
		}
	}
	m.Guild = r.Fixed(25)
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("user shape: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type EquipmentChange struct {
	ID                          uint32
	Slot, Type, TypeID, Enchant uint8
	Cloak                       [6]uint8
}
```

```go
func DecodeEquipment(p []byte) (m EquipmentChange, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Slot, m.Type, m.TypeID, m.Enchant = r.U8(), r.U8(), r.U8(), r.U8()
	for i := range m.Cloak {
		m.Cloak[i] = r.U8()
	}
	return m, r.Err()
}
```

</details>

## exchange

[internal/proto/exchange.go](../../internal/proto/exchange.go#L1)

<details><summary>Types and codecs</summary>

```go
type ExchangeItem struct {
	Type, TypeID uint8
	Count        uint16
	Quality      uint16
	Gems         [6]uint8
	Craft        string
}
```

```go
type ExchangeOwnItem struct {
	Bag, Slot, Destination uint8
	Count                  uint16
}
```

```go
type ExchangeUpdate struct{ Kind, Value uint8 }
```

```go
type ExchangeMoneyUpdate struct {
	Kind  uint8
	Money uint32
}
```

```go
func DecodeExchangeID(p []byte) (uint32, error) { r := NewReader(p); id := r.U32(); return id, r.Err() }
```

```go
func DecodeExchangeStart(p []byte) (id uint32, result uint8, err error) {
	r := NewReader(p)
	id = r.U32()
	result = r.U8()
	return id, result, r.Err()
}
```

```go
func DecodeExchangeByte(p []byte) (uint8, error) { r := NewReader(p); v := r.U8(); return v, r.Err() }
```

```go
func DecodeExchangeUpdate(p []byte) (u ExchangeUpdate, err error) {
	r := NewReader(p)
	u = ExchangeUpdate{r.U8(), r.U8()}
	return u, r.Err()
}
```

```go
func DecodeExchangeMoney(p []byte) (m ExchangeMoneyUpdate, err error) {
	r := NewReader(p)
	m = ExchangeMoneyUpdate{r.U8(), r.U32()}
	return m, r.Err()
}
```

```go
func DecodeExchangeOwnItem(p []byte) (item ExchangeOwnItem, err error) {
	r := NewReader(p)
	item = ExchangeOwnItem{Bag: r.U8(), Slot: r.U8(), Count: r.Count(), Destination: r.U8()}
	if err = r.Err(); err == nil && item.Destination >= ExchangeSlots {
		err = fmt.Errorf("exchange: invalid slot %d", item.Destination)
	}
	return
}
```

```go
func DecodeExchangePartnerItem(p []byte) (slot uint8, item ExchangeItem, err error) {
	r := NewReader(p)
	slot = r.U8()
	item.Type, item.TypeID, item.Count = r.U8(), r.U8(), r.Count()
	item.Quality = r.U16()
	for i := range item.Gems {
		item.Gems[i] = r.U8()
	}
	item.Craft = r.Fixed(21)
	if err = r.Err(); err == nil && slot >= ExchangeSlots {
		err = fmt.Errorf("exchange: invalid slot %d", slot)
	}
	return
}
```

```go
func ExchangeRequestPacket(id uint32) []byte {
	w := NewWriter(ExchangeRequest)
	w.U32(id)
	return w.Bytes()
}
```

```go
func ExchangeAnswerPacket(accept bool) []byte {
	w := NewWriter(ExchangeAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	return w.Bytes()
}
```

```go
func ExchangeReadyPacket(value uint8) []byte {
	w := NewWriter(ExchangeReady)
	w.U8(value)
	return w.Bytes()
}
```

```go
func ExchangeAddPacket(bag, slot uint8, count uint16, dest uint8) []byte {
	w := NewWriter(ExchangeAdd)
	w.Raw([]byte{bag, slot})
	w.Count(count)
	w.U8(dest)
	return w.Bytes()
}
```

```go
func ExchangeRemovePacket(slot uint8) []byte {
	w := NewWriter(ExchangeRemove)
	w.U8(slot)
	return w.Bytes()
}
```

```go
func ExchangeMoneyPacket(money uint32) []byte {
	w := NewWriter(ExchangeAddMoney)
	w.U32(money)
	return w.Bytes()
}
```

```go
func ExchangeConfirmPacket(confirm bool) []byte {
	w := NewWriter(ExchangeConfirm)
	if confirm {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

</details>

## experience

[internal/proto/experience.go](../../internal/proto/experience.go#L1)

<details><summary>Types and codecs</summary>

```go
type ExperienceGain struct{ Base, Perfect uint32 }
```

```go
func DecodeExperienceGain(p []byte) (ExperienceGain, error) {
	r := NewReader(p)
	gain := ExperienceGain{Base: r.U32(), Perfect: r.U32()}
	return gain, r.Err()
}
```

</details>

## fall damage

[internal/proto/fall_damage.go](../../internal/proto/fall_damage.go#L1)

<details><summary>Types and codecs</summary>

```go
type EnvironmentalDamage struct{ Target, Amount uint32 }
```

```go
type FallDamage = EnvironmentalDamage
```

```go
func DecodeFallDamage(p []byte) (FallDamage, error) {
	return decodeEnvironmentalDamage(p, "fall")
}
```

```go
func decodeEnvironmentalDamage(p []byte, cause string) (v EnvironmentalDamage, err error) {
	if len(p) != 10 {
		return v, fmt.Errorf("%s damage: expected 10 bytes, got %d", cause, len(p))
	}
	r := NewReader(p)
	v.Target, v.Amount = r.U32(), r.U32()
	return v, r.Err()
}
```

</details>

## field

[internal/proto/field.go](../../internal/proto/field.go#L1)

<details><summary>Types and codecs</summary>

```go
type Field struct {
	Owner   uint32
	Skill   uint16
	Level   uint8
	X, Y, Z float32
	Seconds uint16
}
```

```go
func DecodeField(p []byte) (v Field, err error) {
	if len(p) != 23 {
		return v, fmt.Errorf("invalid GX field length")
	}
	r := NewReader(p)
	v.Owner, v.Skill, v.Level = r.U32(), r.U16(), r.U8()
	v.X, v.Y, v.Z = r.F32(), r.F32(), r.F32()
	v.Seconds = r.U16()
	return v, r.Err()
}
```

</details>

## firecracker

[internal/proto/firecracker.go](../../internal/proto/firecracker.go#L1)

<details><summary>Types and codecs</summary>

```go
type Firecracker struct {
	Variant uint8
	X, Y, Z float32
	Name    string
}
```

```go
func FirecrackerVariant(itemType, itemID uint8) uint8 {
	if itemType != 100 {
		return 0
	}
	switch itemID {
	case 115, 116, 117:
		return itemID - 114
	case 128:
		return 4
	default:
		return 0
	}
}
```

```go
func FirecrackerPacket(v Firecracker) []byte {
	w := NewWriter(GXFirecracker)
	w.U8(v.Variant)
	w.F32(v.X)
	w.F32(v.Y)
	w.F32(v.Z)
	w.Text(v.Name)
	return w.Bytes()
}
```

```go
func DecodeFirecracker(p []byte) (Firecracker, error) {
	r := NewReader(p)
	v := Firecracker{Variant: r.U8(), X: r.F32(), Y: r.F32(), Z: r.F32(), Name: r.Text()}
	if Op(p) != GXFirecracker || r.Err() != nil || r.Remaining() != 0 || p[len(p)-1] != 0 ||
		v.Variant < 1 || v.Variant > 4 || strings.TrimSpace(v.Name) == "" {
		return Firecracker{}, fmt.Errorf("invalid firecracker event")
	}
	for _, coordinate := range []float32{v.X, v.Y, v.Z} {
		if math.IsNaN(float64(coordinate)) || math.IsInf(float64(coordinate), 0) {
			return Firecracker{}, fmt.Errorf("invalid firecracker position")
		}
	}
	return v, nil
}
```

</details>

## forge quote

[internal/proto/forge_quote.go](../../internal/proto/forge_quote.go#L1)

<details><summary>Types and codecs</summary>

```go
type ForgeQuoteRequest struct {
	Serial                        uint32
	Kind                          uint8
	Item, Lapis, Material, Hammer [2]uint8
	Count                         uint8
}
```

```go
type ForgeQuote struct {
	Serial     uint32
	Kind       uint8
	Valid      bool
	Rate, Gold uint32 // rate in successes per million
	Lapis      uint8
	Material   uint32
}
```

```go
func ForgeQuotePacket(q ForgeQuoteRequest) []byte {
	w := NewWriter(GXForgeQuote)
	w.U32(q.Serial)
	w.U8(q.Kind)
	for _, k := range [][2]uint8{q.Item, q.Lapis, q.Material, q.Hammer} {
		w.Raw(k[:])
	}
	w.U8(q.Count)
	return w.Bytes()
}
```

```go
func DecodeForgeQuoteRequest(p []byte) (q ForgeQuoteRequest, err error) {
	r := NewReader(p)
	q.Serial = r.U32()
	q.Kind = r.U8()
	for _, k := range []*[2]uint8{&q.Item, &q.Lapis, &q.Material, &q.Hammer} {
		k[0], k[1] = r.U8(), r.U8()
	}
	q.Count = r.U8()
	if r.Err() != nil || r.Remaining() != 0 || q.Kind > ForgeQuoteExtract {
		return q, fmt.Errorf("invalid forge quote request")
	}
	return q, nil
}
```

```go
func DecodeForgeQuote(p []byte) (q ForgeQuote, err error) {
	r := NewReader(p)
	q.Serial = r.U32()
	q.Kind = r.U8()
	q.Valid = r.Bool()
	q.Rate = r.U32()
	q.Gold = r.U32()
	q.Lapis = r.U8()
	q.Material = r.U32()
	if r.Err() != nil || r.Remaining() != 0 || q.Kind > ForgeQuoteExtract || q.Rate > 1000000 {
		return q, fmt.Errorf("invalid forge quote")
	}
	return q, nil
}
```

</details>

## gatekeeper

[internal/proto/gatekeeper.go](../../internal/proto/gatekeeper.go#L1)

<details><summary>Types and codecs</summary>

```go
type GatekeeperResult struct {
	Result uint8
	Money  uint32
}
```

```go
func DecodeGatekeeperResult(p []byte) (GatekeeperResult, error) {
	r := NewReader(p)
	v := GatekeeperResult{Result: r.U8(), Money: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 {
		return v, fmt.Errorf("invalid gatekeeper result")
	}
	return v, nil
}
```

```go
func GatekeeperPacket(id uint32, destination uint8) []byte {
	w := NewWriter(Gatekeeper)
	w.U32(id)
	w.U8(destination)
	return w.Bytes()
}
```

</details>

## gem

[internal/proto/gem.go](../../internal/proto/gem.go#L1)

<details><summary>Types and codecs</summary>

```go
type GemAddResult struct {
	Result, GemBag, GemSlot            uint8
	GemCount                           uint16
	ItemBag, ItemSlot, Position, GemID uint8
	Money                              uint32
}
```

```go
func DecodeGemAddResult(p []byte) (GemAddResult, error) {
	r := NewReader(p)
	v := GemAddResult{Result: r.U8(), GemBag: r.U8(), GemSlot: r.U8(), GemCount: r.Count(), ItemBag: r.U8(), ItemSlot: r.U8(), Position: r.U8(), GemID: r.U8()}
	r.U32()
	v.Money = r.U32()
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.GemBag > 5 || v.GemSlot >= 24 || v.ItemBag > 5 || v.ItemSlot >= 24 || (v.Result == 1 && (v.Position >= 6 || v.GemID == 0)) {
		return v, fmt.Errorf("invalid gem add result")
	}
	return v, nil
}
```

```go
func GemAddPacket(gemBag, gemSlot, itemBag, itemSlot uint8) []byte {
	w := NewWriter(GemAdd)
	w.Raw([]byte{gemBag, gemSlot, itemBag, itemSlot})
	return w.Bytes()
}
```

```go
type GemRemoveResult struct {
	Result, ItemBag, ItemSlot uint8
	Removed, IDs, Bags, Slots [6]uint8
	Money                     uint32
}
```

```go
func DecodeGemRemoveResult(p []byte) (GemRemoveResult, error) {
	r := NewReader(p)
	v := GemRemoveResult{Result: r.U8(), ItemBag: r.U8(), ItemSlot: r.U8()}
	for i := range v.Removed {
		v.Removed[i] = r.U8()
	}
	for i := range v.IDs {
		v.IDs[i] = r.U8()
	}
	for i := range v.Bags {
		v.Bags[i] = r.U8()
	}
	for i := range v.Slots {
		v.Slots[i] = r.U8()
	}
	v.Money = r.U32()
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.ItemBag > 5 || v.ItemSlot >= 24 {
		return v, fmt.Errorf("invalid gem remove result")
	}
	for i, removed := range v.Removed {
		if removed > 1 || (removed != 0 && (v.Result == 0 || v.IDs[i] == 0 || v.Bags[i] < 1 || v.Bags[i] > 5 || v.Slots[i] >= 24)) {
			return v, fmt.Errorf("invalid extracted gem")
		}
	}
	return v, nil
}
```

```go
func GemRemovePacket(itemBag, itemSlot uint8) []byte {
	w := NewWriter(GemRemove)
	w.Raw([]byte{itemBag, itemSlot})
	return w.Bytes()
}
```

```go
func GemAddHammerPacket(gemBag, gemSlot, itemBag, itemSlot, hammerBag, hammerSlot uint8) []byte {
	w := NewWriter(GemAdd)
	w.Raw([]byte{gemBag, gemSlot, itemBag, itemSlot, hammerBag, hammerSlot})
	return w.Bytes()
}
```

```go
func GemRemoveHammerPacket(item, hammer [2]uint8, socket uint8) []byte {
	w := NewWriter(GemRemove)
	w.Raw([]byte{item[0], item[1], hammer[0], hammer[1], socket, 0})
	return w.Bytes()
}
```

</details>

## gm

[internal/proto/gm.go](../../internal/proto/gm.go#L1)

<details><summary>Types and codecs</summary>

```go
type GMState struct {
	ID        uint32
	GM, Turbo bool
	Message   string
}
```

```go
func DecodeGMState(p []byte) (GMState, error) {
	if len(p) < 10 {
		return GMState{}, fmt.Errorf("short GX GM state")
	}
	n := int(binary.LittleEndian.Uint16(p[8:10]))
	if len(p) != 10+n || p[6] > 1 || p[7] > 1 || (p[6] == 0 && p[7] != 0) || !utf8.Valid(p[10:]) {
		return GMState{}, fmt.Errorf("invalid GX GM state")
	}
	return GMState{ID: binary.LittleEndian.Uint32(p[2:6]), GM: p[6] == 1, Turbo: p[7] == 1, Message: string(p[10:])}, nil
}
```

```go
type GMModes struct {
	ID                       uint32
	Concealed, Immune, Turbo bool
}
```

```go
func DecodeGMModes(p []byte) (GMModes, error) {
	if len(p) != 9 || p[6] > 1 || p[7] > 1 || p[8] > 1 {
		return GMModes{}, fmt.Errorf("invalid GX GM modes")
	}
	return GMModes{ID: binary.LittleEndian.Uint32(p[2:6]), Concealed: p[6] == 1, Immune: p[7] == 1, Turbo: p[8] == 1}, nil
}
```

</details>

## gm map

[internal/proto/gm_map.go](../../internal/proto/gm_map.go#L1)

<details><summary>Types and codecs</summary>

```go
func GMMapTeleportPacket(mapID uint16, u, v float32) []byte {
	w := NewWriter(GXGMMapTeleport)
	w.U16(mapID)
	w.F32(u)
	w.F32(v)
	return w.Bytes()
}
```

```go
func DecodeGMMapTeleport(p []byte) (mapID uint16, u, v float32, err error) {
	if len(p) != 12 || Op(p) != GXGMMapTeleport {
		return 0, 0, 0, fmt.Errorf("invalid GM map request")
	}
	mapID = binary.LittleEndian.Uint16(p[2:4])
	u = math.Float32frombits(binary.LittleEndian.Uint32(p[4:8]))
	v = math.Float32frombits(binary.LittleEndian.Uint32(p[8:12]))
	if math.IsNaN(float64(u)) || math.IsNaN(float64(v)) || u < 0 || u > 1 || v < 0 || v > 1 {
		err = fmt.Errorf("invalid GM map coordinate")
	}
	return
}
```

</details>

## guild

[internal/proto/guild.go](../../internal/proto/guild.go#L1)

<details><summary>Types and codecs</summary>

```go
type GuildInfo struct {
	ID                   uint32
	Name, Master, Remark string
	Rank                 uint8
	Points               uint32
}
```

```go
type GuildMember struct {
	ID     uint32
	Rank   uint8
	Level  uint16
	Class  uint8
	Name   string
	Online bool
}
```

```go
type GuildCandidate struct {
	ID    uint32
	Level uint16
	Class uint8
	Name  string
}
```

```go
func readGuildApplicant(r *Reader) GuildCandidate {
	return GuildCandidate{ID: r.U32(), Level: r.U16(), Class: r.U8(), Name: r.Fixed(21)}
}
```

```go
func DecodeGuildApplicants(p []byte) ([]GuildCandidate, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/28 {
		return nil, ErrShort
	}
	list := make([]GuildCandidate, n)
	for i := range list {
		list[i] = readGuildApplicant(r)
	}
	return list, r.Err()
}
```

```go
func DecodeGuildApplicant(p []byte) (GuildCandidate, error) {
	r := NewReader(p)
	v := readGuildApplicant(r)
	return v, r.Err()
}
```

```go
type GuildJoin struct {
	Result uint8
	ID     uint32
	Rank   uint8
	Name   string
}
```

```go
type GuildCreateVote struct {
	Master       uint32
	Name, Remark string
}
```

```go
func DecodeGuildCreateVote(p []byte) (GuildCreateVote, error) {
	r := NewReader(p)
	v := GuildCreateVote{Master: r.U32(), Name: r.Fixed(25), Remark: r.Fixed(65)}
	return v, r.Err()
}
```

```go
func DecodeGuildCreateFailure(p []byte) (uint8, error) {
	r := NewReader(p)
	result := r.U8()
	r.Bytes(95)
	return result, r.Err()
}
```

```go
func readGuildInfo(r *Reader) GuildInfo {
	return GuildInfo{ID: r.U32(), Name: r.Fixed(25), Master: r.Fixed(21), Remark: r.Fixed(65), Rank: r.U8(), Points: r.U32()}
}
```

```go
func DecodeGuildList(p []byte) ([]GuildInfo, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/120 {
		return nil, ErrShort
	}
	list := make([]GuildInfo, n)
	for i := range list {
		list[i] = readGuildInfo(r)
	}
	return list, r.Err()
}
```

```go
func DecodeGuildInfo(p []byte) (GuildInfo, error) {
	r := NewReader(p)
	v := readGuildInfo(r)
	return v, r.Err()
}
```

```go
func readGuildMember(r *Reader, online bool) GuildMember {
	return GuildMember{ID: r.U32(), Rank: r.U8(), Level: r.U16(), Class: r.U8(), Name: r.Fixed(21), Online: online}
}
```

```go
func DecodeGuildMembers(p []byte, online bool) ([]GuildMember, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/29 {
		return nil, ErrShort
	}
	list := make([]GuildMember, n)
	for i := range list {
		list[i] = readGuildMember(r, online)
	}
	return list, r.Err()
}
```

```go
func DecodeGuildMemberAdded(p []byte) (GuildMember, error) {
	r := NewReader(p)
	online := r.U8() != 0
	m := readGuildMember(r, online)
	return m, r.Err()
}
```

```go
func DecodeGuildJoin(p []byte) (GuildJoin, error) {
	r := NewReader(p)
	v := GuildJoin{Result: r.U8(), ID: r.U32(), Rank: r.U8(), Name: r.Fixed(25)}
	return v, r.Err()
}
```

```go
func DecodeGuildCreate(p []byte) (GuildJoin, string, error) {
	r := NewReader(p)
	v := GuildJoin{Result: r.U8(), ID: r.U32(), Rank: r.U8(), Name: r.Fixed(25)}
	remark := r.Fixed(65)
	return v, remark, r.Err()
}
```

```go
func GuildInvitePacket(id uint32) []byte { return idPacket(GuildInvite, id) }
```

```go
func GuildAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildInviteAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
func GuildLeavePacket() []byte   { return NewWriter(GuildLeave).Bytes() }
```

```go
func GuildDisbandPacket() []byte { return NewWriter(GuildDisband).Bytes() }
```

```go
func GuildDisbandAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildDisbandAgree)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
func GuildExpelPacket(id uint32) []byte { return idPacket(GuildExpel, id) }
```

```go
func GuildApplyPacket(id uint32) []byte { return idPacket(GuildApply, id) }
```

```go
func GuildCreatePacket(name, remark string) []byte {
	w := NewWriter(GuildCreate)
	w.Fixed(name, 25, true)
	w.Fixed(remark, 65, true)
	return w.Bytes()
}
```

```go
func GuildCreateAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildCreateAgree)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
func GuildApplicantAnswerPacket(id uint32, accept bool) []byte {
	w := NewWriter(GuildJoinResult)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	w.U32(id)
	return w.Bytes()
}
```

```go
func GuildRankPacket(id uint32, demote bool) []byte {
	w := NewWriter(GuildStatus)
	if demote {
		w.U8(1)
	} else {
		w.U8(0)
	}
	w.U32(id)
	return w.Bytes()
}
```

</details>

## guild bank

[internal/proto/guild_bank.go](../../internal/proto/guild_bank.go#L1)

<details><summary>Types and codecs</summary>

```go
type GuildBankResultInfo struct{ Result, Arg uint8 }
```

```go
type GuildBankEvent struct {
	Item  InventoryItem
	Actor uint32
}
```

```go
func readGuildBankItem(r *Reader) InventoryItem {
	m := InventoryItem{Bag: GuildBankBag, Slot: r.U8(), Type: r.U8(), TypeID: r.U8(), Quality: r.U16()}
	for i := range m.Gems {
		m.Gems[i] = r.U8()
	}
	m.Count, m.Craft = r.Count(), r.Fixed(21)
	return m
}
```

```go
func DecodeGuildBankList(p []byte) ([]InventoryItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/33 {
		return nil, ErrShort
	}
	items := make([]InventoryItem, n)
	var seen [GuildBankSlots]bool
	for i := range items {
		items[i] = readGuildBankItem(r)
		if int(items[i].Slot) >= GuildBankSlots || seen[items[i].Slot] {
			return nil, fmt.Errorf("guild bank: invalid or duplicate slot %d", items[i].Slot)
		}
		seen[items[i].Slot] = true
	}
	return items, r.Err()
}
```

```go
func DecodeGuildBankEvent(p []byte) (GuildBankEvent, error) {
	r := NewReader(p)
	event := GuildBankEvent{Item: readGuildBankItem(r), Actor: r.U32()}
	if err := r.Err(); err != nil {
		return event, err
	}
	if int(event.Item.Slot) >= GuildBankSlots {
		return event, fmt.Errorf("guild bank: invalid slot %d", event.Item.Slot)
	}
	return event, nil
}
```

```go
func DecodeGuildBankResult(p []byte) (GuildBankResultInfo, error) {
	r := NewReader(p)
	result := GuildBankResultInfo{Result: r.U8(), Arg: r.U8()}
	return result, r.Err()
}
```

</details>

## guild duel

[internal/proto/guild_duel.go](../../internal/proto/guild_duel.go#L1)

<details><summary>Types and codecs</summary>

```go
type GuildDuelCenter struct {
	X, Z          float32
	OpponentGuild uint32
}
```

```go
func DecodeGuildDuelCenter(p []byte) (GuildDuelCenter, error) {
	r := NewReader(p)
	v := GuildDuelCenter{X: r.F32(), Z: r.F32(), OpponentGuild: r.U32()}
	return v, r.Err()
}
```

```go
func GuildDuelRequestPacket(id uint32) []byte { return idPacket(GuildDuelRequest, id) }
```

```go
func GuildDuelAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildDuelAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

</details>

## guild house

[internal/proto/guild_house.go](../../internal/proto/guild_house.go#L1)

<details><summary>Types and codecs</summary>

```go
type GuildNpc struct {
	Type  uint16
	Level uint8
	Slot  uint8
}
```

```go
type GuildNpcUpgradeResult struct {
	Result uint8
	Npc    GuildNpc
}
```

```go
type GuildHouseBuyResult struct {
	Result uint8
	Money  uint32
}
```

```go
func DecodeGuildNpcList(p []byte) ([]GuildNpc, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/4 {
		return nil, ErrShort
	}
	out := make([]GuildNpc, n)
	for i := range out {
		out[i] = GuildNpc{Type: r.U16(), Level: r.U8(), Slot: r.U8()}
	}
	return out, r.Err()
}
```

```go
func DecodeGuildNpcUpgrade(p []byte) (GuildNpcUpgradeResult, error) {
	r := NewReader(p)
	v := GuildNpcUpgradeResult{Result: r.U8()}
	v.Npc = GuildNpc{Type: r.U16(), Level: r.U8(), Slot: r.U8()}
	return v, r.Err()
}
```

```go
func DecodeGuildHouseBuy(p []byte) (GuildHouseBuyResult, error) {
	r := NewReader(p)
	v := GuildHouseBuyResult{Result: r.U8(), Money: r.U32()}
	return v, r.Err()
}
```

```go
func DecodeGuildDonation(p []byte) ([][2]uint8, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/2 {
		return nil, ErrShort
	}
	out := make([][2]uint8, n)
	for i := range out {
		out[i] = [2]uint8{r.U8(), r.U8()}
	}
	return out, r.Err()
}
```

```go
func GuildNpcUpgradePacket(npcType uint16, level uint8) []byte {
	w := NewWriter(GuildNpcUpgrade)
	w.U16(npcType)
	w.U8(level)
	return w.Bytes()
}
```

```go
func GuildHouseBuyPacket() []byte { w := NewWriter(GuildHouseBuy); w.U32(0); return w.Bytes() }
```

```go
func GuildDonatePacket() []byte   { w := NewWriter(GuildDonate); w.U32(0); return w.Bytes() }
```

</details>

## guild notice

[internal/proto/guild_notice.go](../../internal/proto/guild_notice.go#L1)

<details><summary>Types and codecs</summary>

```go
type GuildRemarkResult struct {
	Result uint8
	Remark string
}
```

```go
func GuildRemarkPacket(remark string) []byte {
	w := NewWriter(GuildRemark)
	w.Fixed(remark, 65, true)
	return w.Bytes()
}
```

```go
func DecodeGuildRemark(p []byte) (GuildRemarkResult, error) {
	r := NewReader(p)
	v := GuildRemarkResult{Result: r.U8(), Remark: r.Fixed(65)}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("guild remark: trailing bytes")
	}
	return v, nil
}
```

```go
func GuildEtinPacket() []byte { return NewWriter(GuildEtin).Bytes() }
```

</details>

## guild ranking

[internal/proto/guild_ranking.go](../../internal/proto/guild_ranking.go#L1)

<details><summary>Types and codecs</summary>

```go
type GuildRankingNoticeInfo struct {
	Map  uint16
	Kind uint8
}
```

```go
type GuildRankingPointsInfo struct{ Points, Maximum, Leader uint32 }
```

```go
type GuildRankingEntry struct {
	Guild, Points uint32
	Rank          uint8
}
```

```go
func DecodeGuildRankingNotice(p []byte) (GuildRankingNoticeInfo, error) {
	r := NewReader(p)
	v := GuildRankingNoticeInfo{Map: r.U16(), Kind: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("guild ranking notice: trailing bytes")
	}
	return v, nil
}
```

```go
func DecodeGuildRankingPoints(p []byte) (GuildRankingPointsInfo, error) {
	r := NewReader(p)
	v := GuildRankingPointsInfo{Points: r.U32(), Maximum: r.U32(), Leader: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("guild ranking points: trailing bytes")
	}
	return v, nil
}
```

```go
func DecodeGuildRankingList(p []byte) ([]GuildRankingEntry, error) {
	r := NewReader(p)
	count := int(r.U8())
	if r.Err() != nil {
		return nil, r.Err()
	}
	// Game.exe's handler has space for 40 rows; Rapture sends at most 30.
	if count > 40 {
		return nil, fmt.Errorf("guild ranking: too many rows %d", count)
	}
	if r.Remaining() != count*9 {
		return nil, fmt.Errorf("guild ranking: incorrect row length")
	}
	out := make([]GuildRankingEntry, 0, count)
	seen := make(map[uint32]bool, count)
	for i := 0; i < count; i++ {
		v := GuildRankingEntry{Guild: r.U32(), Points: r.U32(), Rank: r.U8()}
		if v.Guild == 0 || seen[v.Guild] {
			return nil, fmt.Errorf("guild ranking: duplicate or zero guild")
		}
		seen[v.Guild] = true
		out = append(out, v)
	}
	return out, r.Err()
}
```

</details>

## inventory

[internal/proto/inventory.go](../../internal/proto/inventory.go#L1)

<details><summary>Types and codecs</summary>

```go
type InventoryItem struct {
	Bag, Slot, Type, TypeID uint8
	Quality                 uint16
	Gems                    [6]uint8
	Count                   uint16
	Craft                   string
}
```

```go
func DecodeInventory(p []byte) ([]InventoryItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/34 {
		return nil, ErrShort
	}
	items := make([]InventoryItem, n)
	for i := range items {
		m := &items[i]
		m.Bag, m.Slot, m.Type, m.TypeID = r.U8(), r.U8(), r.U8(), r.U8()
		m.Quality = r.U16()
		for j := range m.Gems {
			m.Gems[j] = r.U8()
		}
		m.Count = r.Count()
		m.Craft = r.Fixed(21)
		if m.Bag > 5 || m.Slot >= 24 {
			return nil, fmt.Errorf("inventory: invalid position %d/%d", m.Bag, m.Slot)
		}
	}
	return items, r.Err()
}
```

```go
func readItemSlot(r *Reader) InventoryItem {
	m := InventoryItem{Bag: r.U8(), Slot: r.U8(), Type: r.U8(), TypeID: r.U8(), Count: r.Count()}
	m.Quality = r.U16()
	for j := range m.Gems {
		m.Gems[j] = r.U8()
	}
	m.Craft = r.Fixed(21)
	return m
}
```

```go
func DecodeItemMove(p []byte) ([2]InventoryItem, error) {
	r := NewReader(p)
	m := [2]InventoryItem{readItemSlot(r), readItemSlot(r)}
	return m, r.Err()
}
```

```go
func DecodeItemAdd(p []byte) (InventoryItem, error) {
	r := NewReader(p)
	m := readItemSlot(r)
	return m, r.Err()
}
```

```go
func ItemMovePacket(srcBag, srcSlot, dstBag, dstSlot uint8) []byte {
	w := NewWriter(ItemBagToBag)
	w.Raw([]byte{srcBag, srcSlot, dstBag, dstSlot})
	return w.Bytes()
}
```

```go
type GroundItem struct {
	ID                uint32
	Unk, Type, TypeID uint8
	Count             uint16
	X, Y, Z           float32
}
```

```go
func DecodeGroundItem(p []byte) (m GroundItem, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Unk, m.Type, m.TypeID, m.Count = r.U8(), r.U8(), r.U8(), r.Count()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}
```

```go
func PickupPacket(id uint32) []byte { return idPacket(AddItem, id) }
```

```go
func UseItemPacket(bag, slot uint8) []byte {
	w := NewWriter(UseItem)
	w.U8(bag)
	w.U8(slot)
	return w.Bytes()
}
```

```go
func TargetResurrectionItemPacket(bag, slot uint8, target uint32) []byte {
	w := NewWriter(UseItem)
	w.U8(bag)
	w.U8(slot)
	w.U32(target)
	return w.Bytes()
}
```

```go
type ItemUseResult struct {
	ID                      uint32
	Bag, Slot, Type, TypeID uint8
	Count                   uint16
}
```

```go
func DecodeItemUse(p []byte) (m ItemUseResult, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Bag, m.Slot, m.Type, m.TypeID, m.Count = r.U8(), r.U8(), r.U8(), r.U8(), r.Count()
	return m, r.Err()
}
```

</details>

## kill notice

[internal/proto/kill_notice.go](../../internal/proto/kill_notice.go#L1)

<details><summary>Types and codecs</summary>

```go
func DecodeKillNotice(p []byte) (string, error) {
	if len(p) != 23 {
		return "", fmt.Errorf("invalid GX kill notice length")
	}
	r := NewReader(p)
	name := r.Fixed(21)
	return name, r.Err()
}
```

</details>

## ladder

[internal/proto/ladder.go](../../internal/proto/ladder.go#L1)

<details><summary>Types and codecs</summary>

```go
type LadderState struct {
	ID    uint32
	Index uint16
	Mode  uint8
}
```

```go
func LadderPacket(index uint16, mode uint8) []byte {
	w := NewWriter(GXLadder)
	w.U16(index)
	w.U8(mode)
	return w.Bytes()
}
```

```go
func DecodeLadderState(p []byte) (v LadderState, err error) {
	if len(p) != 9 {
		return v, fmt.Errorf("ladder state: expected 9 bytes, got %d", len(p))
	}
	r := NewReader(p)
	v.ID = r.U32()
	v.Index = r.U16()
	v.Mode = r.U8()
	if v.Mode > 3 {
		return v, fmt.Errorf("invalid ladder mode")
	}
	return v, r.Err()
}
```

</details>

## login

[internal/proto/login.go](../../internal/proto/login.go#L1)

<details><summary>Types and codecs</summary>

```go
type LoginHello struct {
	Unk        uint8
	ExponentLE []byte
	ModulusLE  []byte
}
```

```go
func DecodeLoginHello(p []byte) (m LoginHello, err error) {
	r := NewReader(p)
	m.Unk = r.U8()
	el, ml := int(r.U8()), int(r.U8())
	m.ExponentLE = r.Bytes(el)
	m.ModulusLE = r.Bytes(ml)
	return m, r.Err()
}
```

```go
func KeyReply(block []byte) ([]byte, error) {
	if len(block) != 128 {
		return nil, fmt.Errorf("proto: RSA block is %d bytes, the client sends 128", len(block))
	}
	w := NewWriter(LoginHandshake)
	w.U8(128)
	w.Raw(block)
	return w.Bytes(), nil
}
```

```go
func LoginRequestPacket(user, password string) []byte {
	w := NewWriter(LoginRequest)
	w.Fixed(user, 32, true)
	w.Fixed(password, 16, false)
	return w.Bytes()
}
```

```go
type LoginResult struct {
	Result     uint8 // 0 = ok
	UserUID    uint32
	Permission uint8
	Token      [16]byte
}
```

```go
func DecodeLoginResult(p []byte) (m LoginResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	if m.Result == 0 {
		m.UserUID = r.U32()
		m.Permission = r.U8()
		copy(m.Token[:], r.Bytes(16))
	}
	return m, r.Err()
}
```

```go
type Server struct {
	ID       uint8
	Status   uint8
	Users    uint16
	Capacity uint16
	Name     string
}
```

```go
func DecodeServerList(p []byte) ([]Server, error) {
	r := NewReader(p)
	n := int(r.U8())
	out := make([]Server, 0, n)
	for i := 0; i < n && r.Err() == nil; i++ {
		out = append(out, Server{ID: r.U8(), Status: r.U8(), Users: r.U16(), Capacity: r.U16(), Name: r.Fixed(32)})
	}
	return out, r.Err()
}
```

```go
func SelectServerPacket(serverID uint8, version uint32) []byte {
	w := NewWriter(SelectServer)
	w.U8(serverID)
	w.U32(version)
	return w.Bytes()
}
```

```go
type SelectServerResult struct {
	Status uint8
	IPv4   [4]byte
}
```

```go
func DecodeSelectServer(p []byte) (m SelectServerResult, err error) {
	r := NewReader(p)
	m.Status = r.U8()
	copy(m.IPv4[:], r.Bytes(4))
	return m, r.Err()
}
```

```go
func WorldHelloPacket(userUID uint32, token [16]byte) []byte {
	w := NewWriter(GameHandshake)
	w.U32(userUID)
	w.Raw(token[:])
	return w.Bytes()
}
```

```go
type WorldHandshake struct {
	Status uint8
	Mode   uint8
	Seed   [16]byte
}
```

```go
func DecodeWorldHandshake(p []byte) (m WorldHandshake, err error) {
	r := NewReader(p)
	m.Status = r.U8()
	m.Mode = r.U8()
	copy(m.Seed[:], r.Bytes(16))
	return m, r.Err()
}
```

</details>

## loot notice

[internal/proto/loot_notice.go](../../internal/proto/loot_notice.go#L1)

<details><summary>Types and codecs</summary>

```go
type LootNotice struct {
	Receiver     uint32
	Type, TypeID uint8
}
```

```go
func DecodeLootNotice(p []byte) (LootNotice, error) {
	if len(p) != 8 {
		return LootNotice{}, fmt.Errorf("invalid GX loot notice length")
	}
	r := NewReader(p)
	n := LootNotice{Receiver: r.U32(), Type: r.U8(), TypeID: r.U8()}
	return n, r.Err()
}
```

</details>

## mall

[internal/proto/mall.go](../../internal/proto/mall.go#L1)

<details><summary>Types and codecs</summary>

```go
type MallItem struct {
	Bag, Slot, Type, TypeID uint8
	Count                   uint16
}
```

```go
type MallPurchase struct {
	Result          uint8
	HasDetails      bool
	Points          uint32
	Code            string
	Timestamp, Cost uint32
	Items           []MallItem
}
```

```go
type MallGiftResult struct {
	Result          uint8
	HasDetails      bool
	Target          string
	Points          uint32
	Code            string
	Timestamp, Cost uint32
}
```

```go
type MallStoredItem struct {
	Slot, Type, TypeID uint8
	Count              uint16
}
```

```go
type MallClaim struct{ Slot, Bag, InventorySlot uint8 }
```

```go
func DecodeMallPurchase(p []byte) (MallPurchase, error) {
	r := NewReader(p)
	result := MallPurchase{Result: r.U8()}
	if err := r.Err(); err != nil {
		return result, err
	}
	if result.Result != 0 && r.Remaining() == 0 {
		return result, nil
	}
	result.Points, result.Code, result.Timestamp, result.Cost = r.U32(), r.Fixed(21), r.U32(), r.U32()
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return result, err
	}
	if n > r.Remaining()/5 {
		return result, ErrShort
	}
	result.Items = make([]MallItem, n)
	for i := range result.Items {
		result.Items[i] = MallItem{r.U8(), r.U8(), r.U8(), r.U8(), r.Count()}
	}
	result.HasDetails = true
	return result, r.Err()
}
```

```go
func DecodeMallGiftResult(p []byte) (MallGiftResult, error) {
	r := NewReader(p)
	result := MallGiftResult{Result: r.U8()}
	if err := r.Err(); err != nil {
		return result, err
	}
	if result.Result != 0 && r.Remaining() == 0 {
		return result, nil
	}
	result.Target, result.Points, result.Code, result.Timestamp, result.Cost = r.Fixed(21), r.U32(), r.Fixed(21), r.U32(), r.U32()
	if err := r.Err(); err != nil {
		return result, err
	}
	result.HasDetails = true
	return result, nil
}
```

```go
func DecodeMallStored(p []byte) ([]MallStoredItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return nil, err
	}
	if n > 240 || n > r.Remaining()/4 {
		return nil, ErrShort
	}
	items := make([]MallStoredItem, n)
	var seen [240]bool
	for i := range items {
		it := MallStoredItem{r.U8(), r.U8(), r.U8(), r.Count()}
		if it.Slot >= 240 || it.Type == 0 || it.Count == 0 || seen[it.Slot] {
			return nil, fmt.Errorf("mall: invalid stored item %d", it.Slot)
		}
		seen[it.Slot] = true
		items[i] = it
	}
	return items, r.Err()
}
```

```go
func DecodeMallClaim(p []byte) (MallClaim, error) {
	r := NewReader(p)
	m := MallClaim{r.U8(), r.U8(), r.U8()}
	if err := r.Err(); err != nil {
		return m, err
	}
	if m.Slot >= 240 || m.Bag < 1 || m.Bag > 5 || m.InventorySlot >= 24 {
		return m, fmt.Errorf("mall: invalid claim location")
	}
	return m, nil
}
```

```go
func MallPointsPacket() []byte { return NewWriter(CashPoint).Bytes() }
```

```go
func MallBuyPacket(code string) []byte {
	w := NewWriter(PointBuy)
	w.Fixed(code, 21, true)
	return w.Bytes()
}
```

```go
func MallGiftPacket(code, target string) []byte {
	w := NewWriter(PointGiftSend)
	w.Fixed(code, 21, true)
	w.Fixed(target, 21, true)
	return w.Bytes()
}
```

```go
func MallClaimPacket(slot uint8) []byte {
	w := NewWriter(PointStoredItemGet)
	w.U8(slot)
	return w.Bytes()
}
```

</details>

## mob speech

[internal/proto/mob_speech.go](../../internal/proto/mob_speech.go#L1)

<details><summary>Types and codecs</summary>

```go
type MobSpeech struct {
	MobID     uint32
	MessageID uint16
	Text      string
}
```

```go
func DecodeMobSpeech(p []byte) (MobSpeech, error) {
	r := NewReader(p)
	var speech MobSpeech
	switch Op(p) {
	case MobSpeechText:
		if r.U8() != 1 {
			return speech, fmt.Errorf("invalid mob speech type")
		}
		speech.MobID = r.U32()
		speech.Text = r.Text()
	case MobSpeechIndex:
		speech.MobID = r.U32()
		speech.MessageID = r.U16()
	default:
		return speech, fmt.Errorf("invalid mob speech opcode")
	}
	if e := r.Err(); e != nil {
		return speech, e
	}
	if r.Remaining() != 0 {
		return speech, fmt.Errorf("trailing mob speech bytes")
	}
	return speech, nil
}
```

</details>

## motion

[internal/proto/motion.go](../../internal/proto/motion.go#L1)

<details><summary>Types and codecs</summary>

```go
func GestureAction(motion uint8) (int, bool) {
	if motion < 116 || motion > 125 {
		return 0, false
	}
	if motion >= 124 {
		return int(motion), true
	}
	return int(motion ^ 1), true
}
```

```go
type CharacterMotionResult struct {
	ID     uint32
	Motion uint8
}
```

```go
func MotionPacket(motion uint8) []byte {
	w := NewWriter(CharacterMotion)
	w.U8(motion)
	return w.Bytes()
}
```

```go
func DecodeCharacterMotion(p []byte) (CharacterMotionResult, error) {
	r := NewReader(p)
	m := CharacterMotionResult{ID: r.U32(), Motion: r.U8()}
	if r.Err() != nil {
		return CharacterMotionResult{}, r.Err()
	}
	if r.Remaining() != 0 {
		return CharacterMotionResult{}, fmt.Errorf("motion: trailing bytes")
	}
	return m, nil
}
```

```go
func LocomotionAction(motion uint8) (int, bool) {
	switch motion {
	case MotionJump:
		return 8, true
	case MotionBackstep:
		return 13, true
	case MotionEvadeLeft:
		return 14, true
	case MotionEvadeRight:
		return 15, true
	}
	return 0, false
}
```

</details>

## mount state

[internal/proto/mount_state.go](../../internal/proto/mount_state.go#L1)

<details><summary>Types and codecs</summary>

```go
type VehicleState struct {
	ID                               uint32
	State, Race, Model, Speed, Seats uint8
	Driver, Passenger                uint32
	CastMS, CooldownMS               uint32
}
```

```go
func VehicleStatePacket(v VehicleState) []byte {
	w := NewWriter(GXVehicleState)
	w.U32(v.ID)
	w.U8(v.State)
	w.U8(v.Race)
	w.U8(v.Model)
	w.U8(v.Speed)
	w.U8(v.Seats)
	w.U32(v.Driver)
	w.U32(v.Passenger)
	w.U32(v.CastMS)
	w.U32(v.CooldownMS)
	return w.Bytes()
}
```

```go
func DecodeVehicleState(p []byte) (VehicleState, error) {
	if len(p) != 27 || Op(p) != GXVehicleState {
		return VehicleState{}, fmt.Errorf("invalid vehicle state size")
	}
	r := NewReader(p)
	v := VehicleState{ID: r.U32(), State: r.U8(), Race: r.U8(), Model: r.U8(), Speed: r.U8(), Seats: r.U8(), Driver: r.U32(), Passenger: r.U32(), CastMS: r.U32(), CooldownMS: r.U32()}
	if v.State > VehiclePassenger || v.Race > 3 || (v.State >= VehicleDriver && (v.Speed < 4 || v.Speed > 5 || v.Seats < 1 || v.Seats > 2 || v.Driver == 0)) {
		return v, fmt.Errorf("invalid vehicle state")
	}
	return v, r.Err()
}
```

```go
func VehicleInvitePacket(id uint32) []byte {
	w := NewWriter(VehicleInvite)
	w.U32(id)
	return w.Bytes()
}
```

```go
func VehicleAnswerPacket(accept bool) []byte {
	w := NewWriter(VehicleAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	return w.Bytes()
}
```

```go
func VehicleLeavePacket() []byte { return NewWriter(VehicleLeave).Bytes() }
```

</details>

## myshop

[internal/proto/myshop.go](../../internal/proto/myshop.go#L1)

<details><summary>Types and codecs</summary>

```go
func MyShopOpenPacket(title string) []byte {
	w := NewWriter(MyShopOpen)
	text := []byte(textwire.Prefix(title, MyShopTitleBytes))
	w.U8(uint8(len(text)))
	w.Raw(text)
	return w.Bytes()
}
```

```go
func MyShopAddPacket(index, bag, slot uint8, price uint32) []byte {
	w := NewWriter(MyShopAdd)
	w.U8(index)
	w.U8(bag)
	w.U8(slot)
	w.U32(price)
	return w.Bytes()
}
```

```go
func MyShopRemovePacket(index uint8) []byte {
	w := NewWriter(MyShopRemove)
	w.U8(index)
	return w.Bytes()
}
```

```go
func MyShopPricePacket(index uint8, price uint32) []byte {
	w := NewWriter(MyShopPrice)
	w.U8(index)
	w.U32(price)
	return w.Bytes()
}
```

```go
func MyShopEnterPacket(seller uint32) []byte {
	w := NewWriter(MyShopEnter)
	w.U32(seller)
	return w.Bytes()
}
```

```go
func MyShopBuyPacket(index, count uint8) []byte {
	w := NewWriter(MyShopBuy)
	w.U8(index)
	w.U8(count)
	return w.Bytes()
}
```

```go
type MyShopItem struct {
	Index byte
	Price uint32
	Item  InventoryItem
}
```

```go
func DecodeMyShopList(p []byte) ([]MyShopItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return nil, err
	}
	const recordSize = 37
	if Op(p) != MyShopList || n > MyShopSlots || n*recordSize != r.Remaining() {
		return nil, fmt.Errorf("market list: invalid count %d", n)
	}
	out := make([]MyShopItem, n)
	var seen [MyShopSlots]bool
	for i := range out {
		e := &out[i]
		e.Index, e.Price = r.U8(), r.U32()
		e.Item.Type, e.Item.TypeID, e.Item.Count = r.U8(), r.U8(), uint16(r.U8())
		e.Item.Quality = r.U16()
		for j := range e.Item.Gems {
			e.Item.Gems[j] = r.U8()
		}
		e.Item.Craft = r.Fixed(21)
		if int(e.Index) >= MyShopSlots || seen[e.Index] || e.Price == 0 || e.Price > MyShopPriceMax || e.Item.Type == 0 || e.Item.Count == 0 {
			return nil, fmt.Errorf("market list: invalid item at index %d", e.Index)
		}
		seen[e.Index] = true
	}
	return out, r.Err()
}
```

```go
func DecodeShopBubble(p []byte) (id uint32, title string, err error) {
	r := NewReader(p)
	id = r.U32()
	on := r.U8()
	if on != 0 {
		n := int(r.U8())
		if on != 1 || n > MyShopTitleBytes || n != r.Remaining() {
			return id, "", fmt.Errorf("invalid market message length %d", n)
		}
		title = textwire.Decode(r.Bytes(n))
	} else if r.Remaining() != 0 {
		return id, "", fmt.Errorf("trailing bytes in closed market sign")
	}
	return id, title, r.Err()
}
```

</details>

## oj limits

[internal/proto/oj_limits.go](../../internal/proto/oj_limits.go#L1)

<details><summary>Types and codecs</summary>

```go
type OJLimit struct{ Type, TypeID, Stats, Value uint8 }
```

```go
func OJLimitsPacket(limits []OJLimit) []byte {
	w := NewWriter(GXOJLimits)
	w.U16(uint16(len(limits)))
	for _, v := range limits {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Stats)
		w.U8(v.Value)
	}
	return w.Bytes()
}
```

```go
func DecodeOJLimits(p []byte) ([]OJLimit, error) {
	if len(p) < 4 || Op(p) != GXOJLimits || len(p) != 4+4*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid OJ limits")
	}
	out := make([]OJLimit, 0, (len(p)-4)/4)
	for i := 4; i < len(p); i += 4 {
		v := OJLimit{p[i], p[i+1], p[i+2], p[i+3]}
		if v.Type == 0 || v.TypeID == 0 || v.Stats > 9 || v.Value > 99 || (v.Stats == 0) != (v.Value == 0) {
			return nil, fmt.Errorf("invalid OJ limit")
		}
		out = append(out, v)
	}
	return out, nil
}
```

</details>

## oxygen

[internal/proto/oxygen.go](../../internal/proto/oxygen.go#L1)

<details><summary>Types and codecs</summary>

```go
type OxygenPhase uint8
```

```go
type OxygenStatus struct {
	Phase       OxygenPhase
	RemainingMS uint16
}
```

```go
func OxygenPacket(v OxygenStatus) []byte {
	w := NewWriter(GXOxygen)
	w.U8(uint8(v.Phase))
	w.U16(v.RemainingMS)
	return w.Bytes()
}
```

```go
func DecodeOxygen(p []byte) (OxygenStatus, error) {
	r := NewReader(p)
	v := OxygenStatus{Phase: OxygenPhase(r.U8()), RemainingMS: r.U16()}
	if Op(p) != GXOxygen || r.Err() != nil || r.Remaining() != 0 ||
		v.Phase > OxygenRefilling || v.RemainingMS > uint16(OxygenCapacity.Milliseconds()) ||
		v.Phase == OxygenHidden && v.RemainingMS != 0 {
		return OxygenStatus{}, fmt.Errorf("invalid oxygen status")
	}
	return v, nil
}
```

```go
func DecodeDrowningDamage(p []byte) (EnvironmentalDamage, error) {
	return decodeEnvironmentalDamage(p, "drowning")
}
```

```go
func DecodeDrowningDeath(p []byte) (uint32, error) {
	r := NewReader(p)
	id := r.U32()
	if Op(p) != GXDrowningDeath || r.Err() != nil || r.Remaining() != 0 {
		return 0, fmt.Errorf("invalid drowning death")
	}
	return id, nil
}
```

</details>

## party

[internal/proto/party.go](../../internal/proto/party.go#L1)

<details><summary>Types and codecs</summary>

```go
type PartyEffect struct {
	Skill     uint16
	Level     uint8
	Remaining uint32
}
```

```go
type PartyMember struct {
	ID                              uint32
	Name                            string
	Level                           uint16
	Class                           uint8
	HP, MaxHP, MP, MaxMP, SP, MaxSP uint32
	Instance                        uint16
	X, Y, Z                         float32
	Effects                         []PartyEffect
}
```

```go
type PartyList struct {
	LeaderIndex uint8
	Members     []PartyMember
}
```

```go
type PartyReply struct {
	Result uint8
	ID     uint32
}
```

```go
func readPartyMember(r *Reader) (m PartyMember) {
	m.ID, m.Name, m.Level, m.Class = r.U32(), r.Fixed(21), r.U16(), r.U8()
	m.HP, m.MaxHP, m.MP, m.MaxMP, m.SP, m.MaxSP = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	m.Instance, m.X, m.Y, m.Z = r.U16(), r.F32(), r.F32(), r.F32()
	n := int(r.U8())
	if n > r.Remaining()/7 {
		r.err = ErrShort
		return m
	}
	m.Effects = make([]PartyEffect, n)
	for i := range m.Effects {
		m.Effects[i] = PartyEffect{r.U16(), r.U8(), r.U32()}
	}
	return m
}
```

```go
func DecodePartyList(p []byte) (list PartyList, err error) {
	r := NewReader(p)
	list.LeaderIndex = r.U8()
	n := int(r.U8())
	// Native scratch contains eight records; the normal party has at most seven.
	if n > 8 {
		return list, fmt.Errorf("party: invalid member count %d", n)
	}
	if n > r.Remaining()/67 {
		return list, ErrShort
	}
	list.Members = make([]PartyMember, n)
	for i := range list.Members {
		list.Members[i] = readPartyMember(r)
	}
	return list, r.Err()
}
```

```go
func DecodePartyJoined(p []byte) (PartyMember, error) {
	r := NewReader(p)
	m := readPartyMember(r)
	return m, r.Err()
}
```

```go
func DecodePartyReply(p []byte) (PartyReply, error) {
	r := NewReader(p)
	m := PartyReply{r.U8(), r.U32()}
	return m, r.Err()
}
```

```go
func DecodePartyInviteName(p []byte) (string, error) {
	r := NewReader(p)
	s := r.Fixed(21)
	return s, r.Err()
}
```

```go
func PartyInvitePacket(id uint32) []byte { return idPacket(PartyInvite, id) }
```

```go
func PartyAnswerPacket(accept bool, characterID uint32) []byte {
	w := NewWriter(PartyAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	w.U32(characterID)
	return w.Bytes()
}
```

```go
func PartyLeavePacket() []byte           { return NewWriter(PartyLeft).Bytes() }
```

```go
func PartyExpelPacket(id uint32) []byte  { return idPacket(PartyExpelled, id) }
```

```go
func PartyLeaderPacket(id uint32) []byte { return idPacket(PartyLeader, id) }
```

```go
func PartyInviteNamePacket(name string) []byte {
	w := NewWriter(PartyInviteName)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
```

```go
func PartyGlobalInvitePacket(name string) []byte {
	w := NewWriter(GXPartyInviteName)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
```

```go
func (m *PartyMember) AddEffect(skill uint16, level uint8) {
	for i := range m.Effects {
		e := &m.Effects[i]
		if e.Skill != skill {
			continue
		}
		if e.Level == level {
			return
		}
		if level != 100 {
			e.Level = level
			return
		}
	}
	m.Effects = append(m.Effects, PartyEffect{Skill: skill, Level: level})
}
```

```go
func (m *PartyMember) RemoveEffect(skill uint16, level uint8) {
	for i, e := range m.Effects {
		if e.Skill == skill && e.Level == level {
			m.Effects = append(m.Effects[:i], m.Effects[i+1:]...)
			return
		}
	}
}
```

</details>

## party assist

[internal/proto/party_assist.go](../../internal/proto/party_assist.go#L1)

<details><summary>Types and codecs</summary>

```go
func PartyAssistPacket(action uint8, id uint32) []byte {
	w := NewWriter(GXPartyAssist)
	w.U8(action)
	w.U32(id)
	return w.Bytes()
}
```

</details>

## periodic

[internal/proto/periodic.go](../../internal/proto/periodic.go#L1)

<details><summary>Types and codecs</summary>

```go
type PeriodicEffect struct {
	Target                   uint32
	Skill                    uint16
	Level                    uint8
	HP, Resource1, Resource2 uint16
}
```

```go
func DecodePeriodicEffect(p []byte) (v PeriodicEffect, err error) {
	r := NewReader(p)
	v.Target, v.Skill, v.Level = r.U32(), r.U16(), r.U8()
	v.HP, v.Resource1, v.Resource2 = r.U16(), r.U16(), r.U16()
	return v, r.Err()
}
```

</details>

## player feedback

[internal/proto/player_feedback.go](../../internal/proto/player_feedback.go#L1)

<details><summary>Types and codecs</summary>

```go
func PlayerFeedbackPacket(id uint32, kind uint8) []byte {
	w := NewWriter(GXPlayerFeedback)
	w.U32(id)
	w.U8(kind)
	return w.Bytes()
}
```

```go
func DecodePlayerFeedback(p []byte) (id uint32, kind uint8, err error) {
	r := NewReader(p)
	id, kind = r.U32(), r.U8()
	if Op(p) != GXPlayerFeedback || r.Err() != nil || r.Remaining() != 0 ||
		id == 0 || (kind != PlayerSpawnFeedback && kind != PlayerSkillFeedback && kind != PlayerLevelFeedback) {
		err = fmt.Errorf("invalid player feedback")
	}
	return
}
```

</details>

## player forms

[internal/proto/player_forms.go](../../internal/proto/player_forms.go#L1)

<details><summary>Types and codecs</summary>

```go
type PlayerFormKind uint8
```

```go
func SkillForm(effect uint8, detail, change uint16) PlayerFormKind {
	if detail != 90 {
		return FormNone
	}
	if effect == 3 && change == 101 {
		return FormDegeneration
	}
	if effect != 4 {
		return FormNone
	}
	switch change {
	case 100:
		return FormDisguise
	case 102:
		return FormMonsterCopy
	case 10, 11, 12:
		return FormEvolution
	default:
		return FormNone
	}
}
```

```go
type PlayerFormAppearance struct {
	User    uint32
	Kind    PlayerFormKind
	Skill   uint16
	Rank    uint8
	Shape   UserShape
	Attacks [2]FormAttack
}
```

```go
type FormAttack struct {
	Skill uint16
	Slot  uint8
}
```

```go
func DecodePlayerForm(p []byte) (v PlayerFormAppearance, err error) {
	r := NewReader(p)
	v.User, v.Kind = r.U32(), PlayerFormKind(r.U8())
	switch v.Kind {
	case FormNone:
		if r.Remaining() == 0 {
			return v, r.Err()
		}
	case FormEvolution:
		v.Skill, v.Rank = r.U16(), r.U8()
		for i := range v.Attacks {
			v.Attacks[i] = FormAttack{Skill: r.U16(), Slot: r.U8()}
		}
		if r.Remaining() == 0 {
			return v, r.Err()
		}
	case FormDisguise:
		if len(p) >= 9 && Op(p[7:]) == CharacterShape {
			v.Shape, err = DecodeUserShape(p[7:])
			if err == nil && v.Shape.ID != v.User {
				err = fmt.Errorf("invalid disguise character")
			}
			return v, err
		}
	}
	return v, fmt.Errorf("invalid player form")
}
```

```go
type EvolutionAction struct {
	Skill              uint16
	Range              uint8
	CooldownMS, LeftMS uint32
}
```

```go
type EvolutionBar struct {
	Generation uint32
	Actions    [2]EvolutionAction
}
```

```go
func DecodeEvolutionBar(p []byte) (v EvolutionBar, err error) {
	if len(p) != 28 {
		return v, fmt.Errorf("invalid evolution bar length")
	}
	r := NewReader(p)
	v.Generation = r.U32()
	for i := range v.Actions {
		a := &v.Actions[i]
		a.Skill, a.Range, a.CooldownMS, a.LeftMS = r.U16(), r.U8(), r.U32(), r.U32()
	}
	return v, r.Err()
}
```

```go
func EvolutionCastPacket(generation uint32, cell uint8, target uint32) []byte {
	w := NewWriter(GXEvolutionCast)
	w.U32(generation)
	w.U8(cell)
	w.U32(target)
	return w.Bytes()
}
```

</details>

## player physics

[internal/proto/player_physics.go](../../internal/proto/player_physics.go#L1)

<details><summary>Types and codecs</summary>

```go
type PlayerCorrection struct {
	ID             uint32
	X, Y, Z        float32
	Motion         uint8
	Direction      uint16
	Jumping        bool
	JumpFall, Fall float32
	SlopeFrames    float32
}
```

```go
func DecodePlayerCorrection(p []byte) (c PlayerCorrection, err error) {
	if len(p) != 34 || p[21] > 1 {
		return c, fmt.Errorf("invalid player physics correction")
	}
	r := NewReader(p)
	c.ID = r.U32()
	c.X, c.Y, c.Z = r.F32(), r.F32(), r.F32()
	c.Motion, c.Direction, c.Jumping = r.U8(), r.U16(), r.Bool()
	c.JumpFall, c.Fall = r.F32(), r.F32()
	c.SlopeFrames = r.F32()
	if c.SlopeFrames < 0 {
		return PlayerCorrection{}, fmt.Errorf("negative slope pressure")
	}
	for _, v := range []float32{c.X, c.Y, c.Z, c.JumpFall, c.Fall, c.SlopeFrames} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return PlayerCorrection{}, fmt.Errorf("non-finite physics correction")
		}
	}
	return c, r.Err()
}
```

</details>

## pvp count

[internal/proto/pvp_count.go](../../internal/proto/pvp_count.go#L1)

<details><summary>Types and codecs</summary>

```go
type PvPCountUpdate struct {
	Index uint8
	Value uint32
}
```

```go
func DecodePvPCountUpdate(p []byte) (PvPCountUpdate, error) {
	r := NewReader(p)
	v := PvPCountUpdate{Index: r.U8(), Value: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("pvp count: %d trailing bytes", r.Remaining())
	}
	return v, nil
}
```

```go
type VisiblePvPKills struct{ ID, Kills uint32 }
```

```go
func DecodeVisiblePvPKills(p []byte) (VisiblePvPKills, error) {
	r := NewReader(p)
	v := VisiblePvPKills{ID: r.U32(), Kills: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("visible pvp kills: %d trailing bytes", r.Remaining())
	}
	return v, nil
}
```

</details>

## pvp reward

[internal/proto/pvp_reward.go](../../internal/proto/pvp_reward.go#L1)

<details><summary>Types and codecs</summary>

```go
type PvPRewards struct {
	KillStage, DeathStage uint8
	Known                 bool
}
```

```go
type PvPRewardResult struct {
	OK    bool
	Value uint32
}
```

```go
func DecodePvPRewards(p []byte) (PvPRewards, error) {
	r := NewReader(p)
	v := PvPRewards{KillStage: r.U8(), DeathStage: r.U8(), Known: true}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || v.KillStage < 1 || v.KillStage > PvPRewardFinishedStage || v.DeathStage < 1 || v.DeathStage > PvPRewardFinishedStage {
		return v, fmt.Errorf("invalid PvP reward stages")
	}
	return v, nil
}
```

```go
func DecodePvPRewardResult(p []byte, death bool) (PvPRewardResult, error) {
	r := NewReader(p)
	result := r.U8()
	v := PvPRewardResult{OK: result == 1}
	if death {
		v.Value = r.U32()
	} else {
		v.Value = uint32(r.U8())
	}
	if err := r.Err(); err != nil {
		return v, err
	}
	if result > 1 || r.Remaining() != 0 {
		return v, fmt.Errorf("invalid PvP reward result")
	}
	return v, nil
}
```

</details>

## quest

[internal/proto/quest.go](../../internal/proto/quest.go#L1)

<details><summary>Types and codecs</summary>

```go
type ActiveQuest struct {
	ID, MinutesLeft uint16
	Counters        [3]uint8
}
```

```go
type FinishedQuest struct {
	ID      uint16
	Success uint8
}
```

```go
type QuestEndResult struct {
	NPC               uint32
	ID                uint16
	Success, Variant  uint8
	Experience, Money uint32
	Item              [4]uint8
}
```

```go
func DecodeQuestList(p []byte) ([]ActiveQuest, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/7 {
		return nil, ErrShort
	}
	out := make([]ActiveQuest, n)
	for i := range out {
		q := &out[i]
		q.ID = r.U16()
		q.MinutesLeft = r.U16()
		for j := range q.Counters {
			q.Counters[j] = r.U8()
		}
	}
	return out, r.Err()
}
```

```go
func DecodeFinishedQuests(p []byte) ([]FinishedQuest, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/3 {
		return nil, ErrShort
	}
	out := make([]FinishedQuest, n)
	for i := range out {
		out[i] = FinishedQuest{r.U16(), r.U8()}
	}
	return out, r.Err()
}
```

```go
func DecodeQuestStart(p []byte) (npc uint32, id uint16, err error) {
	r := NewReader(p)
	npc = r.U32()
	id = r.U16()
	return npc, id, r.Err()
}
```

```go
func DecodeQuestCounter(p []byte) (id uint16, slot, value uint8, err error) {
	r := NewReader(p)
	id = r.U16()
	slot = r.U8()
	value = r.U8()
	return id, slot, value, r.Err()
}
```

```go
func DecodeQuestEnd(p []byte) (q QuestEndResult, err error) {
	r := NewReader(p)
	q.NPC = r.U32()
	q.ID = r.U16()
	q.Success = r.U8()
	q.Variant = r.U8()
	q.Experience = r.U32()
	q.Money = r.U32()
	for i := range q.Item {
		q.Item[i] = r.U8()
	}
	return q, r.Err()
}
```

```go
func DecodeQuestChoice(p []byte) (uint16, error) {
	r := NewReader(p)
	id := r.U16()
	return id, r.Err()
}
```

```go
func QuestStartPacket(npc uint32, id uint16) []byte {
	w := NewWriter(QuestStart)
	w.U32(npc)
	w.U16(id)
	return w.Bytes()
}
```

```go
func QuestEndPacket(npc uint32, id uint16) []byte {
	w := NewWriter(QuestEnd)
	w.U32(npc)
	w.U16(id)
	return w.Bytes()
}
```

```go
func QuestChoosePacket(id uint16, choice uint8) []byte {
	w := NewWriter(QuestChoose)
	w.U16(id)
	w.U8(choice)
	return w.Bytes()
}
```

```go
func QuestAbandonPacket(id uint16) []byte { w := NewWriter(QuestAbandon); w.U16(id); return w.Bytes() }
```

</details>

## quest map

[internal/proto/quest_map.go](../../internal/proto/quest_map.go#L1)

<details><summary>Types and codecs</summary>

```go
type QuestNPC struct {
	Type   uint8
	TypeID uint16
	X, Z   float32
}
```

```go
type QuestNPCChunk struct {
	MapID uint16
	Flags uint8
	NPCs  []QuestNPC
}
```

```go
func DecodeQuestNPCs(p []byte) (QuestNPCChunk, error) {
	r := NewReader(p)
	c := QuestNPCChunk{MapID: r.U16(), Flags: r.U8()}
	n := int(r.U16())
	if r.Err() != nil || n > QuestNPCChunkLimit || c.Flags & ^uint8(3) != 0 || r.Remaining() != n*11 {
		return c, fmt.Errorf("invalid quest NPC catalog")
	}
	c.NPCs = make([]QuestNPC, n)
	for i := range c.NPCs {
		q := &c.NPCs[i]
		q.Type, q.TypeID, q.X, q.Z = r.U8(), r.U16(), r.F32(), r.F32()
		if q.Type < 1 || q.Type > 13 || q.TypeID == 0 || math.IsNaN(float64(q.X)) || math.IsNaN(float64(q.Z)) || math.IsInf(float64(q.X), 0) || math.IsInf(float64(q.Z), 0) {
			return c, fmt.Errorf("invalid quest NPC position")
		}
	}
	return c, r.Err()
}
```

</details>

## raid

[internal/proto/raid.go](../../internal/proto/raid.go#L1)

<details><summary>Types and codecs</summary>

```go
type RaidMember struct {
	Slot   uint16
	Member PartyMember
}
```

```go
type RaidRosterChunk struct {
	More                      bool
	LeaderSlot, SubLeaderSlot uint8
	Division                  uint16
	Kind                      uint8
	AutoJoin                  bool
	Members                   []RaidMember
}
```

```go
func DecodeRaidRoster(p []byte) (RaidRosterChunk, error) {
	r := NewReader(p)
	var chunk RaidRosterChunk
	chunk.More = r.U8() != 0
	chunk.LeaderSlot, chunk.SubLeaderSlot = r.U8(), r.U8()
	chunk.Division, chunk.Kind, chunk.AutoJoin = r.U16(), r.U8(), r.U8() != 0
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return chunk, err
	}
	if n > 6 {
		return chunk, fmt.Errorf("raid: invalid chunk size %d", n)
	}
	if n > r.Remaining()/69 {
		return chunk, ErrShort
	}
	chunk.Members = make([]RaidMember, n)
	for i := range chunk.Members {
		chunk.Members[i] = RaidMember{r.U16(), readPartyMember(r)}
	}
	return chunk, r.Err()
}
```

```go
func DecodeRaidJoined(p []byte) (RaidMember, error) {
	r := NewReader(p)
	m := RaidMember{r.U16(), readPartyMember(r)}
	return m, r.Err()
}
```

```go
func RaidConvertPacket(kind uint8, autoJoin bool, division uint32) []byte {
	w := NewWriter(RaidConvert)
	w.U8(kind)
	if autoJoin {
		w.U8(1)
	} else {
		w.U8(0)
	}
	w.U32(division)
	return w.Bytes()
}
```

```go
func RaidDivisionPacket(division uint32) []byte { return idPacket(RaidDivision, division) }
```

```go
func RaidAutoJoinPacket(enabled bool) []byte {
	w := NewWriter(RaidAutoJoin)
	if enabled {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
func RaidSwapPacket(first, second uint32) []byte {
	w := NewWriter(RaidSwap)
	w.U32(first)
	w.U32(second)
	return w.Bytes()
}
```

```go
func RaidInvitePacket(id uint32) []byte    { return idPacket(RaidInvite, id) }
```

```go
func RaidSubLeaderPacket(id uint32) []byte { return idPacket(RaidSubLeader, id) }
```

```go
func RaidLeaderPacket(id uint32) []byte    { return idPacket(RaidLeader, id) }
```

```go
func RaidExpelPacket(id uint32) []byte     { return idPacket(RaidExpel, id) }
```

```go
func RaidDisbandPacket() []byte            { return NewWriter(RaidDisband).Bytes() }
```

```go
func RaidAnswerPacket(accept bool, id uint32) []byte {
	w := NewWriter(RaidAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	w.U32(id)
	return w.Bytes()
}
```

```go
func RaidNamePacket(op Opcode, name string) []byte {
	w := NewWriter(op)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
```

</details>

## relic

[internal/proto/relic.go](../../internal/proto/relic.go#L1)

<details><summary>Types and codecs</summary>

```go
type Relic struct {
	ID    uint32
	Owner uint8
	X, Z  float32
}
```

```go
func DecodeRelicList(p []byte) ([]Relic, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > 100 || r.Remaining() != n*13 {
		return nil, fmt.Errorf("invalid relic list length")
	}
	out := make([]Relic, 0, n)
	seen := make(map[uint32]bool, n)
	for range n {
		v := Relic{ID: r.U32(), Owner: r.U8(), X: r.F32(), Z: r.F32()}
		if seen[v.ID] || !validRelicOwner(v.Owner) || math.IsNaN(float64(v.X)) || math.IsNaN(float64(v.Z)) || math.IsInf(float64(v.X), 0) || math.IsInf(float64(v.Z), 0) {
			return nil, fmt.Errorf("invalid relic list record")
		}
		seen[v.ID] = true
		out = append(out, v)
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("relic list: %d trailing bytes", r.Remaining())
	}
	return out, nil
}
```

```go
type RelicOwnerChange struct {
	ID    uint32
	Owner uint8
}
```

```go
func DecodeRelicOwner(p []byte) (RelicOwnerChange, error) {
	r := NewReader(p)
	v := RelicOwnerChange{ID: r.U32(), Owner: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || !validRelicOwner(v.Owner) {
		return v, fmt.Errorf("relic owner: %d trailing bytes", r.Remaining())
	}
	return v, nil
}
```

```go
type RelicDestroyed struct {
	MobID              uint16
	OldOwner, NewOwner uint8
}
```

```go
func DecodeRelicNotice(p []byte) (RelicDestroyed, error) {
	r := NewReader(p)
	v := RelicDestroyed{MobID: r.U16(), OldOwner: r.U8(), NewOwner: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || v.OldOwner > 2 || v.NewOwner > 2 {
		return v, fmt.Errorf("relic notice: %d trailing bytes", r.Remaining())
	}
	return v, nil
}
```

```go
func validRelicOwner(v uint8) bool { return v <= 2 || v >= 100 && v <= 103 }
```

```go
type PortalOwnerNotice struct {
	PortalID           uint16
	OldOwner, NewOwner uint8
}
```

```go
func DecodePortalNotice(p []byte) (PortalOwnerNotice, error) {
	r := NewReader(p)
	v := PortalOwnerNotice{r.U16(), r.U8(), r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || v.OldOwner < 100 || v.OldOwner > 103 || v.NewOwner < 100 || v.NewOwner > 103 {
		return v, fmt.Errorf("invalid portal notice")
	}
	return v, nil
}
```

</details>

## repair

[internal/proto/repair.go](../../internal/proto/repair.go#L1)

<details><summary>Types and codecs</summary>

```go
type ItemRepairResult struct {
	Bag, Slot uint8
	Money     uint32
}
```

```go
func DecodeItemRepair(p []byte) (ItemRepairResult, error) {
	r := NewReader(p)
	v := ItemRepairResult{Bag: r.U8(), Slot: r.U8(), Money: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if !(v.Bag == 255 && v.Slot == 255) && (v.Bag > 5 || v.Bag == 0 && v.Slot >= 13 || v.Bag != 0 && v.Slot >= 24) {
		return v, fmt.Errorf("invalid repair position")
	}
	return v, nil
}
```

```go
func ItemRepairPacket(bag, slot uint8) []byte {
	w := NewWriter(ItemRepair)
	w.Raw([]byte{bag, slot})
	return w.Bytes()
}
```

</details>

## rw

[internal/proto/rw.go](../../internal/proto/rw.go#L1)

<details><summary>Types and codecs</summary>

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
func (r *Reader) take(n int) []byte {
	if r.err != nil || n < 0 || r.off+n > len(r.b) {
		r.err = ErrShort
		return nil
	}
	p := r.b[r.off : r.off+n]
	r.off += n
	return p
}
```

```go
func (r *Reader) Err() error     { return r.err }
```

```go
func (r *Reader) Remaining() int { return len(r.b) - r.off }
```

```go
func (r *Reader) U8() uint8 {
	if p := r.take(1); p != nil {
		return p[0]
	}
	return 0
}
```

```go
func (r *Reader) Count() uint16 {
	if r.wide {
		v := r.U16()
		if v > itemstack.Max {
			r.err = fmt.Errorf("item quantity %d exceeds %d", v, itemstack.Max)
		}
		return v
	}
	return uint16(r.U8())
}
```

```go
func (r *Reader) Bool() bool { return r.U8() != 0 }
```

```go
func (r *Reader) U16() uint16 {
	if p := r.take(2); p != nil {
		return binary.LittleEndian.Uint16(p)
	}
	return 0
}
```

```go
func (r *Reader) U32() uint32 {
	if p := r.take(4); p != nil {
		return binary.LittleEndian.Uint32(p)
	}
	return 0
}
```

```go
func (r *Reader) I32() int32   { return int32(r.U32()) }
```

```go
func (r *Reader) F32() float32 { return math.Float32frombits(r.U32()) }
```

```go
func (r *Reader) U64() uint64 {
	if p := r.take(8); p != nil {
		return binary.LittleEndian.Uint64(p)
	}
	return 0
}
```

```go
func (r *Reader) Bytes(n int) []byte { return append([]byte(nil), r.take(n)...) }
```

```go
func (r *Reader) Fixed(n int) string { return DecodeText(r.take(n)) }
```

```go
func (r *Reader) Text() string { return DecodeText(r.take(int(r.U8()))) }
```

```go
func DecodeText(b []byte) string { return textwire.Decode(b) }
```

```go
func EncodeText(s string) []byte { return textwire.Encode(s) }
```

```go
type Writer struct{ b []byte }
```

```go
func NewWriter(op Opcode) *Writer {
	w := &Writer{b: make([]byte, 0, 64)}
	w.U16(uint16(op))
	return w
}
```

```go
func (w *Writer) Count(v uint16) {
	if v > itemstack.Max {
		panic(fmt.Sprintf("item quantity %d exceeds %d", v, itemstack.Max))
	}
	w.U8(uint8(v))
}
```

```go
func (w *Writer) U8(v uint8)   { w.b = append(w.b, v) }
```

```go
func (w *Writer) U16(v uint16) { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
```

```go
func (w *Writer) U32(v uint32) { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
```

```go
func (w *Writer) F32(v float32) {
	w.U32(math.Float32bits(v))
}
```

```go
func (w *Writer) Bool(v bool) {
	if v {
		w.U8(1)
	} else {
		w.U8(0)
	}
}
```

```go
func (w *Writer) Raw(p []byte) { w.b = append(w.b, p...) }
```

```go
func (w *Writer) Fixed(s string, n int, keepNul bool) {
	b := EncodeText(s)
	max := n
	if keepNul {
		max = n - 1
	}
	if len(b) > max {
		b = []byte(textwire.Prefix(string(b), max))
	}
	w.b = append(w.b, b...)
	w.b = append(w.b, make([]byte, n-len(b))...)
}
```

```go
func (w *Writer) Text(s string) {
	b := EncodeText(s)
	if len(b) > 127 {
		b = []byte(textwire.Prefix(string(b), 127))
	}
	w.U8(uint8(len(b) + 1))
	w.b = append(append(w.b, b...), 0)
}
```

```go
func (w *Writer) Bytes() []byte { return w.b }
```

```go
func Op(payload []byte) Opcode {
	if len(payload) < 2 {
		return 0
	}
	return Opcode(itemstack.Base(binary.LittleEndian.Uint16(payload)))
}
```

</details>

## server notice

[internal/proto/server_notice.go](../../internal/proto/server_notice.go#L1)

<details><summary>Types and codecs</summary>

```go
func DecodeServerNotice(payload []byte) (string, error) {
	if Op(payload) != GXServerNotice || len(payload) < 5 || len(payload) > 131 || payload[len(payload)-1] != 0 {
		return "", fmt.Errorf("invalid server notice")
	}
	r := NewReader(payload)
	text := r.Text()
	if err := r.Err(); err != nil {
		return "", err
	}
	if r.Remaining() != 0 || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("invalid server notice text")
	}
	return text, nil
}
```

</details>

## shop

[internal/proto/shop.go](../../internal/proto/shop.go#L1)

<details><summary>Types and codecs</summary>

```go
func ShopBuyPacket(npc uint32, index uint8, count uint16) []byte {
	w := NewWriter(ShopBuy)
	w.U32(npc)
	w.U8(index)
	w.Count(count)
	return w.Bytes()
}
```

```go
func ShopSellPacket(bag, slot uint8, count uint16) []byte {
	w := NewWriter(ShopSell)
	w.Raw([]byte{bag, slot})
	w.Count(count)
	return w.Bytes()
}
```

```go
type ShopResult struct {
	Result, Bag, Slot, Type, TypeID uint8
	Count                           uint16
	Money                           uint32
}
```

```go
func DecodeShopResult(p []byte) (m ShopResult, err error) {
	r := NewReader(p)
	m.Result, m.Bag, m.Slot, m.Type, m.TypeID, m.Count = r.U8(), r.U8(), r.U8(), r.U8(), r.U8(), r.Count()
	m.Money = r.U32()
	return m, r.Err()
}
```

```go
func ItemDropPacket(bag, slot uint8, count uint16) []byte {
	w := NewWriter(RemoveItem)
	w.Raw([]byte{bag, slot})
	w.Count(count)
	return w.Bytes()
}
```

</details>

## skill cooldown

[internal/proto/skill_cooldown.go](../../internal/proto/skill_cooldown.go#L1)

<details><summary>Types and codecs</summary>

```go
type SkillCooldown struct{ Skill, Seconds, Total uint16 }
```

```go
func DecodeSkillCooldown(p []byte) (v SkillCooldown, err error) {
	if len(p) != 8 {
		return v, fmt.Errorf("invalid GX skill cooldown length")
	}
	r := NewReader(p)
	v.Skill, v.Seconds, v.Total = r.U16(), r.U16(), r.U16()
	return v, r.Err()
}
```

</details>

## skill dash

[internal/proto/skill_dash.go](../../internal/proto/skill_dash.go#L1)

<details><summary>Types and codecs</summary>

```go
type SkillDash struct {
	Stopped  bool
	Source   uint32
	From, To [3]float32
	Started  float64 // synchronized map physics seconds
	Contact  float32 // native ALT contact frame
}
```

```go
func SkillDashPacket(d SkillDash) []byte {
	w := NewWriter(GXSkillDash)
	w.U32(d.Source)
	for _, p := range [][3]float32{d.From, d.To} {
		for _, v := range p {
			w.F32(v)
		}
	}
	bits := math.Float64bits(d.Started)
	w.U32(uint32(bits))
	w.U32(uint32(bits >> 32))
	w.F32(d.Contact)
	if d.Stopped {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
func DecodeSkillDash(p []byte) (d SkillDash, err error) {
	if len(p) != 43 || p[42] > 1 || Op(p) != GXSkillDash {
		return d, fmt.Errorf("invalid skill dash size")
	}
	r := NewReader(p)
	d.Source = r.U32()
	for i := range d.From {
		d.From[i] = r.F32()
	}
	for i := range d.To {
		d.To[i] = r.F32()
	}
	d.Started = math.Float64frombits(r.U64())
	d.Contact = r.F32()
	d.Stopped = r.Bool()
	if d.Source == 0 || math.IsNaN(d.Started) || math.IsInf(d.Started, 0) || d.Started < 0 || d.Contact < 10 || d.Contact > 30 {
		return SkillDash{}, fmt.Errorf("invalid skill dash metadata")
	}
	for _, v := range append(d.From[:], d.To[:]...) {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return SkillDash{}, fmt.Errorf("non-finite skill dash")
		}
	}
	if math.IsNaN(float64(d.Contact)) || d.Distance() > MaxSkillDashDistance {
		return SkillDash{}, fmt.Errorf("invalid skill dash range")
	}
	return d, r.Err()
}
```

```go
func (d SkillDash) Distance() float64 {
	var n float64
	for i := range d.From {
		x := float64(d.To[i] - d.From[i])
		n += x * x
	}
	return math.Sqrt(n)
}
```

```go
func (d SkillDash) TravelEnd() float64 {
	return skilltiming.StingerWindup + d.Distance()/skilltiming.StingerSpeed
}
```

```go
func (d SkillDash) Duration() float64 { return d.TravelEnd() + float64(d.Contact-10)/30 }
```

```go
func (d SkillDash) Position(elapsed float64) [3]float32 {
	f := float32(1)
	if n := d.Distance(); n > 0 {
		f = float32(min(1, max(0, (elapsed-skilltiming.StingerWindup)*skilltiming.StingerSpeed/n)))
	}
	p := d.From
	for i := range p {
		p[i] += (d.To[i] - d.From[i]) * f
	}
	return p
}
```

```go
func (d SkillDash) Frame(elapsed float64) float32 {
	if elapsed <= skilltiming.StingerWindup {
		return float32(max(elapsed, 0) * 30)
	}
	return 10 + float32(max(0, elapsed-d.TravelEnd())*30)
}
```

</details>

## skill failure

[internal/proto/skill_failure.go](../../internal/proto/skill_failure.go#L1)

<details><summary>Types and codecs</summary>

```go
type SkillFailure struct{ Skill, Message uint16 }
```

```go
func DecodeSkillFailure(p []byte) (v SkillFailure, err error) {
	if len(p) != 6 {
		return v, fmt.Errorf("invalid GX skill failure length")
	}
	r := NewReader(p)
	v.Skill, v.Message = r.U16(), r.U16()
	return v, r.Err()
}
```

</details>

## skills

[internal/proto/skills.go](../../internal/proto/skills.go#L1)

<details><summary>Types and codecs</summary>

```go
func SupportsRaidSkillTarget(effect uint8) bool { return effect == 2 || effect == 4 }
```

```go
type LearnedSkill struct {
	ID            uint16
	Level, Number uint8
	Delay         int32
}
```

```go
func DecodeSkills(p []byte) ([]LearnedSkill, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/8 {
		return nil, ErrShort
	}
	out := make([]LearnedSkill, n)
	for i := range out {
		out[i] = LearnedSkill{r.U16(), r.U8(), r.U8(), r.I32()}
	}
	return out, r.Err()
}
```

```go
type QuickSlot struct {
	Bar, Slot, Bag uint8
	Number         uint16
}
```

```go
func DecodeQuickSlots(p []byte) ([]QuickSlot, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/5 {
		return nil, ErrShort
	}
	out := make([]QuickSlot, n)
	for i := range out {
		out[i] = QuickSlot{r.U8(), r.U8(), r.U8(), r.U16()}
	}
	return out, r.Err()
}
```

```go
func SkillBarPacket(slots []QuickSlot) ([]byte, error) {
	if len(slots) > 128 {
		return nil, fmt.Errorf("skill bar: too many slots")
	}
	w := NewWriter(CharacterSkillBar)
	w.U8(uint8(len(slots)))
	for _, q := range slots {
		if q.Bar >= 5 || q.Slot >= 10 || !((q.Bag <= 5 && q.Number < 24) || (q.Bag == 100 && q.Number > 0)) {
			return nil, fmt.Errorf("skill bar: invalid slot %+v", q)
		}
		w.U8(q.Bar)
		w.U8(q.Slot)
		w.U8(q.Bag)
		w.U16(q.Number)
	}
	return w.Bytes(), nil
}
```

```go
type LearnResult struct {
	Result, Number uint8
	ID             uint16
	Level          uint8
}
```

```go
func DecodeLearnResult(p []byte) (m LearnResult, err error) {
	r := NewReader(p)
	m.Result, m.Number, m.ID, m.Level = r.U8(), r.U8(), r.U16(), r.U8()
	return m, r.Err()
}
```

```go
func LearnSkillPacket(id uint16, level uint8) []byte {
	w := NewWriter(LearnNewSkill)
	w.U16(id)
	w.U8(level)
	return w.Bytes()
}
```

```go
func UseSkillPacket(number uint8, target uint32, userTarget bool) []byte {
	op := SkillUse
	if userTarget {
		op = SkillUseNoTarget
	}
	w := NewWriter(op)
	w.U8(number)
	w.U32(target)
	return w.Bytes()
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

```go
type SkillCast struct {
	Source, Target uint32
	ID             uint16
	Level          uint8
}
```

```go
func DecodeSkillCast(p []byte) (m SkillCast, err error) {
	r := NewReader(p)
	m.Source, m.Target, m.ID, m.Level = r.U32(), r.U32(), r.U16(), r.U8()
	return m, r.Err()
}
```

```go
func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	if Op(p) == MobSkillUse || Op(p) == MobSkillDrain {
		m.AttackSlot = r.U8()
	}
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain || Op(p) == MobSkillDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
func DecodeSkillExtra(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Source, m.Target, m.ID, m.Level = r.U32(), r.U32(), r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill extra: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type ReflectedDamage struct {
	Source, Target uint32
	HP, SP, MP     uint16
}
```

```go
func DecodeReflectedDamage(p []byte) (m ReflectedDamage, err error) {
	r := NewReader(p)
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("reflected damage: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

</details>

## social

[internal/proto/social.go](../../internal/proto/social.go#L1)

<details><summary>Types and codecs</summary>

```go
type SocialContact struct {
	ID         uint32
	Class      uint8
	Online     bool
	Name, Memo string
}
```

```go
type PartySearchEntry struct {
	Level, Class uint8
	Name         string
}
```

```go
func DecodeSocialList(p []byte) ([]SocialContact, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > 50 {
		return nil, fmt.Errorf("social list count %d", n)
	}
	rows := make([]SocialContact, n)
	for i := range rows {
		v := &rows[i]
		v.ID = r.U32()
		if Op(p) == FriendList {
			v.Class = r.U8()
			v.Online = r.U8() != 0
		}
		v.Name = r.Fixed(21)
		v.Memo = r.Text()
		if len(v.Memo) > 50 {
			return nil, fmt.Errorf("social memo too long")
		}
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("social list trailing data")
	}
	return rows, nil
}
```

```go
func DecodePartySearch(p []byte) ([]PartySearchEntry, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > 100 {
		return nil, fmt.Errorf("party search count")
	}
	rows := make([]PartySearchEntry, n)
	for i := range rows {
		rows[i] = PartySearchEntry{r.U8(), r.U8(), r.Fixed(21)}
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("party search trailing data")
	}
	return rows, nil
}
```

```go
func SocialNamePacket(op Opcode, name string) []byte {
	w := NewWriter(op)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
```

```go
func SocialIDPacket(op Opcode, id uint32) []byte { w := NewWriter(op); w.U32(id); return w.Bytes() }
```

```go
func FriendAnswerPacket(accept bool) []byte {
	w := NewWriter(FriendAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
func SocialMemoPacket(op Opcode, id uint32, memo string) []byte {
	w := NewWriter(op)
	w.U32(id)
	w.Text(memo)
	return w.Bytes()
}
```

</details>

## speed

[internal/proto/speed.go](../../internal/proto/speed.go#L1)

<details><summary>Types and codecs</summary>

```go
type CharacterSpeed struct {
	ID           uint32
	Attack, Move uint8
}
```

```go
func DecodeCharacterSpeed(p []byte) (m CharacterSpeed, err error) {
	r := NewReader(p)
	m.ID, m.Attack, m.Move = r.U32(), r.U8(), r.U8()
	return m, r.Err()
}
```

</details>

## stats

[internal/proto/stats.go](../../internal/proto/stats.go#L1)

<details><summary>Types and codecs</summary>

```go
type StatPoints [6]uint16
```

```go
func (d StatPoints) Total() uint32 {
	var total uint32
	for _, n := range d {
		total += uint32(n)
	}
	return total
}
```

```go
func StatPointsPacket(d StatPoints) []byte {
	w := NewWriter(SetStatPoint)
	for _, n := range d {
		w.U16(n)
	}
	return w.Bytes()
}
```

```go
func DecodeStatPoints(p []byte) (d StatPoints, err error) {
	r := NewReader(p)
	for i := range d {
		d[i] = r.U16()
	}
	return d, r.Err()
}
```

```go
type AdditionalStats struct {
	Str, Rec, Int, Wis, Dex, Luc             uint32
	MinAttack, MaxAttack, MinMagic, MaxMagic uint32
	Defense, Resistance                      uint32
}
```

```go
func DecodeAdditionalStats(p []byte) (s AdditionalStats, err error) {
	r := NewReader(p)
	s.Str, s.Rec, s.Int, s.Wis, s.Dex, s.Luc = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	s.MinAttack, s.MaxAttack, s.MinMagic, s.MaxMagic, s.Defense, s.Resistance = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	return s, r.Err()
}
```

```go
type StatReset struct {
	Result                       uint8
	Points                       uint16
	Str, Rec, Int, Wis, Dex, Luc uint16
}
```

```go
func DecodeStatReset(p []byte) (s StatReset, err error) {
	r := NewReader(p)
	s.Result = r.U8()
	if s.Result != 1 {
		return s, r.Err()
	}
	s.Points, s.Str, s.Rec, s.Int, s.Wis, s.Dex, s.Luc = r.U16(), r.U16(), r.U16(), r.U16(), r.U16(), r.U16(), r.U16()
	return s, r.Err()
}
```

```go
type MaxEnergy struct {
	ID    uint32
	Kind  uint8
	Value uint32
}
```

```go
func DecodeMaxEnergy(p []byte) (s MaxEnergy, err error) {
	r := NewReader(p)
	s.ID, s.Kind, s.Value = r.U32(), r.U8(), r.U32()
	return s, r.Err()
}
```

</details>

## summon rune

[internal/proto/summon_rune.go](../../internal/proto/summon_rune.go#L1)

<details><summary>Types and codecs</summary>

```go
type SummonRuneEvent struct {
	Action           uint8
	Caster, Offer    uint32
	Map, RemainingMS uint16
	Name, Message    string
}
```

```go
func DecodeSummonRune(p []byte) (v SummonRuneEvent, err error) {
	r := NewReader(p)
	v.Action = r.U8()
	switch v.Action {
	case SummonCast, SummonCastEnd:
		v.Caster, v.RemainingMS = r.U32(), r.U16()
	case SummonOffer:
		v.Offer, v.Caster, v.Map, v.RemainingMS = r.U32(), r.U32(), r.U16(), r.U16()
		v.Name = r.Fixed(21)
	case SummonResult:
		v.Message = r.Text()
	}
	return v, r.Err()
}
```

```go
func SummonAnswerPacket(id uint32, accept bool) []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonAnswer)
	w.U32(id)
	w.Bool(accept)
	return w.Bytes()
}
```

```go
func SummonCancelPacket() []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonCancel)
	return w.Bytes()
}
```

</details>

## target effects

[internal/proto/target_effects.go](../../internal/proto/target_effects.go#L1)

<details><summary>Types and codecs</summary>

```go
type TargetEffect struct {
	Skill        uint16
	Level        uint8
	Remaining    uint32
	HasRemaining bool
}
```

```go
type TargetEffectSnapshot struct {
	Kind    uint8
	ID      uint32
	Effects []TargetEffect
}
```

```go
func TargetEffectsQuery(kind uint8, id uint32) []byte {
	w := NewWriter(GXTargetEffectsQuery)
	w.U8(kind)
	w.U32(id)
	return w.Bytes()
}
```

```go
func DecodeTargetEffects(p []byte) (v TargetEffectSnapshot, err error) {
	r := NewReader(p)
	v.Kind, v.ID = r.U8(), r.U32()
	n := int(r.U8())
	if v.Kind != 1 && v.Kind != 2 || r.Remaining() != n*7 {
		return v, fmt.Errorf("invalid target effects snapshot")
	}
	v.Effects = make([]TargetEffect, n)
	for i := range v.Effects {
		v.Effects[i] = TargetEffect{Skill: r.U16(), Level: r.U8(), Remaining: r.U32(), HasRemaining: true}
	}
	return v, r.Err()
}
```

```go
func DecodeTargetEffectChange(p []byte) (v TargetEffectSnapshot, e TargetEffect, err error) {
	if len(p) != 10 {
		return v, e, fmt.Errorf("invalid target effect change")
	}
	r := NewReader(p)
	v.Kind, v.ID = r.U8(), r.U32()
	e.Skill, e.Level = r.U16(), r.U8()
	if v.Kind != 1 && v.Kind != 2 {
		return v, e, fmt.Errorf("invalid target effect kind")
	}
	return v, e, r.Err()
}
```

</details>

## target health

[internal/proto/target_health.go](../../internal/proto/target_health.go#L1)

<details><summary>Types and codecs</summary>

```go
type TargetHealth struct {
	User    uint32
	HP, Max uint32
}
```

```go
func DecodeTargetHealth(p []byte) (v TargetHealth, err error) {
	if len(p) != 14 {
		return v, fmt.Errorf("invalid GX target health length")
	}
	r := NewReader(p)
	v.User, v.HP, v.Max = r.U32(), r.U32(), r.U32()
	return v, r.Err()
}
```

</details>

## teleport stone

[internal/proto/teleport_stone.go](../../internal/proto/teleport_stone.go#L1)

<details><summary>Types and codecs</summary>

```go
type TeleportPoint struct {
	Saved   bool
	Map     uint16
	X, Y, Z float32
}
```

```go
type TeleportStoneResult struct {
	Action  uint8
	CastMS  uint16
	OK      bool
	Message string
	Points  [TeleportPointCount]TeleportPoint
}
```

```go
func TeleportStonePacket(action, slot uint8) []byte {
	w := NewWriter(GXTeleportStone)
	w.U8(action)
	w.U8(slot)
	return w.Bytes()
}
```

```go
func DecodeTeleportStone(p []byte) (out TeleportStoneResult, err error) {
	r := NewReader(p)
	out.Action, out.OK, out.Message = r.U8(), r.U8() != 0, r.Text()
	for i := range out.Points {
		out.Points[i] = TeleportPoint{Saved: r.U8() != 0, Map: r.U16(), X: r.F32(), Y: r.F32(), Z: r.F32()}
	}
	if out.Action == TeleportCasting {
		out.CastMS = r.U16()
	}
	return out, r.Err()
}
```

</details>

## transform

[internal/proto/transform.go](../../internal/proto/transform.go#L1)

<details><summary>Types and codecs</summary>

```go
type Transform struct {
	User uint32
	Mob  uint16
}
```

```go
func DecodeTransform(p []byte) (v Transform, err error) {
	if len(p) != 8 {
		return v, fmt.Errorf("invalid GX transform length")
	}
	r := NewReader(p)
	v.User, v.Mob = r.U32(), r.U16()
	return v, r.Err()
}
```

</details>

## vehicle

[internal/proto/vehicle.go](../../internal/proto/vehicle.go#L1)

<details><summary>Types and codecs</summary>

```go
type VehicleResult struct{ Result, On uint8 }
```

```go
func DecodeVehicleResult(p []byte) (VehicleResult, error) {
	r := NewReader(p)
	v := VehicleResult{Result: r.U8(), On: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.On > 1 {
		return v, fmt.Errorf("invalid vehicle result")
	}
	return v, nil
}
```

```go
type MountShape struct {
	ID    uint32
	Shape uint8
}
```

```go
func DecodeMountShape(p []byte) (MountShape, error) {
	r := NewReader(p)
	v := MountShape{ID: r.U32(), Shape: r.U8()}
	return v, r.Err()
}
```

```go
func UseVehiclePacket() []byte { return NewWriter(UseVehicle).Bytes() }
```

</details>

## warehouse

[internal/proto/warehouse.go](../../internal/proto/warehouse.go#L1)

<details><summary>Types and codecs</summary>

```go
type WarehouseList struct {
	Gold  uint32
	Items []InventoryItem
}
```

```go
func DecodeWarehouse(p []byte) (list WarehouseList, err error) {
	r := NewReader(p)
	list.Gold = r.U32()
	n := int(r.U8())
	if n > WarehouseSlots {
		return list, fmt.Errorf("warehouse: too many items %d", n)
	}
	if n > r.Remaining()/33 {
		return list, ErrShort
	}
	list.Items = make([]InventoryItem, n)
	seen := [WarehouseSlots]bool{}
	for i := range list.Items {
		m := &list.Items[i]
		m.Bag = WarehouseBag
		m.Slot, m.Type, m.TypeID = r.U8(), r.U8(), r.U8()
		m.Quality = r.U16()
		for j := range m.Gems {
			m.Gems[j] = r.U8()
		}
		m.Count, m.Craft = r.Count(), r.Fixed(21)
		if int(m.Slot) >= WarehouseSlots || seen[m.Slot] {
			return list, fmt.Errorf("warehouse: invalid/duplicate slot %d", m.Slot)
		}
		seen[m.Slot] = true
	}
	return list, r.Err()
}
```

```go
func DecodeWarehouseMove(p []byte) (items [2]InventoryItem, money uint32, hasMoney bool, err error) {
	r := NewReader(p)
	items = [2]InventoryItem{readItemSlot(r), readItemSlot(r)}
	if err := r.Err(); err != nil {
		return items, 0, false, err
	}
	for _, item := range items {
		if item.Bag == GuildBankBag && int(item.Slot) >= GuildBankSlots {
			return items, 0, false, fmt.Errorf("guild bank: invalid slot %d", item.Slot)
		}
	}
	if (items[0].Bag == WarehouseBag && items[1].Bag != WarehouseBag) || (items[0].Bag == GuildBankBag && items[1].Bag != GuildBankBag) {
		money, hasMoney = r.U32(), true
	}
	return items, money, hasMoney, r.Err()
}
```

</details>

## weather

[internal/proto/weather.go](../../internal/proto/weather.go#L1)

<details><summary>Types and codecs</summary>

```go
type Weather struct {
	MapID uint16
	Type  uint8
}
```

```go
func DecodeWeather(p []byte) (Weather, error) {
	r := NewReader(p)
	v := Weather{r.U16(), r.U8()}
	if err := r.Err(); err != nil {
		return Weather{}, err
	}
	if v.Type > WeatherSnow || r.Remaining() != 0 {
		return Weather{}, fmt.Errorf("invalid weather snapshot")
	}
	return v, nil
}
```

</details>

## world

[internal/proto/world.go](../../internal/proto/world.go#L1)

<details><summary>Types and codecs</summary>

```go
func FactionPacket(faction uint8) []byte {
	w := NewWriter(AccountFaction)
	w.U8(faction)
	return w.Bytes()
}
```

```go
type NewCharacter struct {
	Slot, Race, Mode, Hair, Face, Height, Class, Gender uint8
	Name                                                string
}
```

```go
func CreateCharacterPacket(c NewCharacter) []byte {
	w := NewWriter(CreateCharacter)
	for _, v := range []uint8{c.Slot, c.Race, c.Mode, c.Hair, c.Face, c.Height, c.Class, c.Gender} {
		w.U8(v)
	}
	name := EncodeText(c.Name)
	if len(name) > 18 {
		name = []byte(textwire.Prefix(string(name), 18))
	}
	w.Raw(append(name, 0))
	return w.Bytes()
}
```

```go
func idPacket(op Opcode, id uint32) []byte {
	w := NewWriter(op)
	w.U32(id)
	return w.Bytes()
}
```

```go
func DeleteCharacterPacket(id uint32) []byte { return idPacket(DeleteCharacter, id) }
```

```go
func RestoreCharacterPacket(id uint32) []byte { return idPacket(RestoreCharacter, id) }
```

```go
func SelectCharacterPacket(id uint32) []byte  { return idPacket(SelectCharacter, id) }
```

```go
func NpcTalkPacket(id uint32) []byte          { return idPacket(NpcTalk, id) }
```

```go
func MobAttackPacket(id uint32) []byte        { return idPacket(CharacterMobAutoAttack, id) }
```

```go
func UserAttackPacket(id uint32) []byte       { return idPacket(CharacterUserAutoAttack, id) }
```

```go
func LogoutPacket() []byte    { return NewWriter(Logout).Bytes() }
```

```go
func MapLoadedPacket() []byte { return NewWriter(CharacterEnteredMap).Bytes() }
```

```go
func MovePacket(motion uint8, dir uint16, x, y, z float32) []byte {
	for _, v := range []float32{x, y, z} {
		if v != v || v > 3.4e38 || v < -3.4e38 {
			return nil
		}
	}
	w := NewWriter(CharacterMove)
	w.U8(motion)
	w.U16(dir)
	w.F32(x)
	w.F32(y)
	w.F32(z)
	return w.Bytes()
}
```

```go
func ChatPacket(channel uint8, admin bool, text string) []byte {
	ops := [...]Opcode{ChatNormal, ChatWorld, ChatGuild, ChatParty, ChatShout}
	admins := [...]Opcode{ChatNormalAdmin, ChatWorldAdmin, 0xF104, 0xF105}
	var op Opcode
	switch {
	case channel == ChannelRaid:
		op = ChatRaid
	case admin && int(channel) < len(admins):
		op = admins[channel]
	case !admin && int(channel) < len(ops):
		op = ops[channel]
	default:
		return nil
	}
	w := NewWriter(op)
	w.Text(text)
	return w.Bytes()
}
```

```go
func WhisperPacket(target, text string) []byte {
	if target == "" || text == "" || len(EncodeText(target)) > 20 {
		return nil
	}
	w := NewWriter(ChatWhisper)
	w.Fixed(target, 21, true)
	w.Text(text)
	return w.Bytes()
}
```

```go
type Faction struct{ Faction uint8 }
```

```go
func DecodeFaction(p []byte) (m Faction, err error) {
	r := NewReader(p)
	m.Faction = r.U8()
	r.U8() // reserved byte
	return m, r.Err()
}
```

```go
type CharacterSlot struct {
	Slot                                          uint8
	ID                                            uint32
	Unk                                           uint32
	Level                                         uint16
	Race, Mode, Hair, Face, Height, Class, Gender uint8
	MapID                                         uint16
	Str, Rec, Int, Wis, Dex, Luc                  uint16
	MaxHP, MaxMP, MaxSP                           uint16
	EquipType, EquipTypeID                        [8]uint8
	Cloak                                         [6]uint8 // gem bytes of the equipped cloak: its emblem
	Name                                          string
	PendingDelete, RenamePending                  bool
}
```

```go
func DecodeCharacterSlot(p []byte) (m CharacterSlot, err error) {
	r := NewReader(p)
	m.Slot, m.ID = r.U8(), r.U32()
	if m.ID == 0 {
		return m, r.Err()
	}
	m.Unk, m.Level = r.U32(), r.U16()
	m.Race, m.Mode, m.Hair, m.Face, m.Height, m.Class, m.Gender = r.U8(), r.U8(), r.U8(), r.U8(), r.U8(), r.U8(), r.U8()
	m.MapID = r.U16()
	m.Str, m.Rec, m.Int, m.Wis, m.Dex, m.Luc = r.U16(), r.U16(), r.U16(), r.U16(), r.U16(), r.U16()
	m.MaxHP, m.MaxMP, m.MaxSP = r.U16(), r.U16(), r.U16()
	for i := range m.EquipType {
		m.EquipType[i] = r.U8()
	}
	for i := range m.EquipTypeID {
		m.EquipTypeID[i] = r.U8()
	}
	m.Name = r.Fixed(19)
	m.PendingDelete, m.RenamePending = r.Bool(), r.Bool()
	return m, r.Err()
}
```

```go
type IDResult struct {
	Result uint8
	ID     uint32
}
```

```go
func DecodeIDResult(p []byte) (m IDResult, err error) {
	r := NewReader(p)
	m.Result, m.ID = r.U8(), r.U32()
	return m, r.Err()
}
```

```go
func DecodeU8(p []byte) (uint8, error) { r := NewReader(p); v := r.U8(); return v, r.Err() }
```

```go
func DecodeU32(p []byte) (uint32, error) { r := NewReader(p); v := r.U32(); return v, r.Err() }
```

```go
type Details struct {
	StatPoint, SkillPoint        uint16
	MaxHP, MaxMP, MaxSP          uint32
	Dir                          uint16
	PrevExp, NextExp, Exp, Money uint32
	X, Y, Z                      float32
	Kills, Deaths, Wins, Losses  uint32
	Guild                        string
}
```

```go
func DecodeDetails(p []byte) (m Details, err error) {
	r := NewReader(p)
	m.StatPoint, m.SkillPoint = r.U16(), r.U16()
	m.MaxHP, m.MaxMP, m.MaxSP = r.U32(), r.U32(), r.U32()
	m.Dir = r.U16()
	m.PrevExp, m.NextExp, m.Exp, m.Money = r.U32(), r.U32(), r.U32(), r.U32()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	m.Kills, m.Deaths, m.Wins, m.Losses = r.U32(), r.U32(), r.U32(), r.U32()
	m.Guild = r.Fixed(25)
	return m, r.Err()
}
```

```go
type UserEnter struct {
	ID      uint32
	Marker  uint8 // 1 first load, 2 admin
	Unk     uint16
	X, Y, Z float32
	GuildID uint32
	Vehicle uint32
}
```

```go
func DecodeUserEnter(p []byte) (m UserEnter, err error) {
	r := NewReader(p)
	m.ID, m.Marker, m.Unk = r.U32(), r.U8(), r.U16()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	m.GuildID, m.Vehicle = r.U32(), r.U32()
	return m, r.Err()
}
```

```go
type UserMove struct {
	ID      uint32
	Motion  uint8
	Dir     uint16
	X, Y, Z float32
}
```

```go
func DecodeUserMove(p []byte) (m UserMove, err error) {
	r := NewReader(p)
	m.ID, m.Motion, m.Dir = r.U32(), r.U8(), r.U16()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}
```

```go
type MobSpawn struct {
	Country      uint8
	CountryKnown bool
	ID           uint32
	New          bool
	MobID        uint16
	X, Y, Z      float32
	HeightKnown  bool
}
```

```go
func DecodeMobEnter(p []byte) (m MobSpawn, err error) {
	r := NewReader(p)
	m.ID, m.New, m.MobID, m.X = r.U32(), r.Bool(), r.U16(), r.F32()
	m.HeightKnown = len(p) >= 2 && Opcode(binary.LittleEndian.Uint16(p[:2])) == GXMobEnter
	if m.HeightKnown {
		m.Y = r.F32()
	}
	m.Z = r.F32()
	if m.HeightKnown && r.Remaining() > 0 {
		if r.Remaining() != 2 {
			return m, fmt.Errorf("invalid GX mob metadata length")
		}
		m.Country = r.U8()
		height := r.U8()
		if m.Country > 2 || height > 1 {
			return m, fmt.Errorf("invalid GX mob metadata")
		}
		m.CountryKnown = true
		m.HeightKnown = height == 1
	}
	return m, r.Err()
}
```

```go
type MobMotion struct {
	ID          uint32
	Motion      uint8
	X, Y, Z     float32
	HeightKnown bool
	DurationMS  uint32
}
```

```go
func DecodeMobMove(p []byte) (m MobMotion, err error) {
	r := NewReader(p)
	m.ID, m.Motion, m.X = r.U32(), r.U8(), r.F32()
	m.HeightKnown = len(p) >= 2 && Opcode(binary.LittleEndian.Uint16(p[:2])) == GXMobMove
	if m.HeightKnown {
		m.Y = r.F32()
	}
	m.Z = r.F32()
	if m.HeightKnown {
		m.DurationMS = r.U32()
	}
	return m, r.Err()
}
```

```go
type NpcSpawn struct {
	Summoned bool
	ID       uint32
	Type     uint8
	TypeID   uint16
	X, Y, Z  float32
	Angle    uint16
}
```

```go
func DecodeNpcEnter(p []byte) (m NpcSpawn, err error) {
	r := NewReader(p)
	m.ID, m.Type, m.TypeID = r.U32(), r.U8(), r.U16()
	m.X, m.Y, m.Z, m.Angle = r.F32(), r.F32(), r.F32(), r.U16()
	if r.Remaining() > 0 {
		m.Summoned = r.Bool()
	}
	return m, r.Err()
}
```

```go
type NpcMotion struct {
	ID      uint32
	Motion  uint8
	X, Y, Z float32
}
```

```go
func DecodeNpcMove(p []byte) (n NpcMotion, err error) {
	r := NewReader(p)
	n.ID, n.Motion, n.X, n.Y, n.Z = r.U32(), r.U8(), r.F32(), r.F32(), r.F32()
	return n, r.Err()
}
```

```go
type NpcAttack struct {
	Result         uint8
	Source, Target uint32
	HP             uint16
}
```

```go
func DecodeNpcAttack(p []byte) (a NpcAttack, err error) {
	r := NewReader(p)
	a.Result, a.Source, a.Target, a.HP = r.U8(), r.U32(), r.U32(), r.U16()
	return a, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

```go
func DecodeChatByID(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.SenderID, m.Text = r.U32(), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
func DecodeChatByName(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Sender, m.Text = r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
func DecodeWhisper(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Direction, m.Sender, m.Text = r.U8(), r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type MapPosition struct {
	ID      uint32
	MapID   uint16
	X, Y, Z float32
}
```

```go
func DecodeMapPosition(p []byte) (m MapPosition, err error) {
	r := NewReader(p)
	m.ID, m.MapID, m.X, m.Y, m.Z = r.U32(), r.U16(), r.F32(), r.F32(), r.F32()
	return m, r.Err()
}
```

```go
func CastInterruptingMotion(motion uint8) bool {
	return motion&0x3f != 0 && motion&MotionFlag80 == 0
}
```

</details>
