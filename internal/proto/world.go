package proto

import "github.com/Spelunkern/Shaiya-GX/internal/textwire"

// Client -> server: FUN_00597f60 takes buffer + length.
// Each builder cites its writer in Game.exe.

// FactionPacket is 0x0109 (writer 0x598460, 3 bytes).
func FactionPacket(faction uint8) []byte {
	w := NewWriter(AccountFaction)
	w.U8(faction)
	return w.Bytes()
}

// NewCharacter is the 0x0102 request (writer 0x5980e0): eight bytes, then
// the name copied with lstrcpyA, length = strlen + 11 (terminator included).
type NewCharacter struct {
	Slot, Race, Mode, Hair, Face, Height, Class, Gender uint8
	Name                                                string
}

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

func idPacket(op Opcode, id uint32) []byte {
	w := NewWriter(op)
	w.U32(id)
	return w.Bytes()
}

// DeleteCharacterPacket 0x0103 (writer 0x598180), SelectCharacterPacket
// 0x0104 (0x5981b0), NpcTalkPacket 0x0E04 (0x599a30), MobAttackPacket
// 0x0503 (0x598b00), UserAttackPacket 0x0502 (0x598ab0): id:u32.
func DeleteCharacterPacket(id uint32) []byte { return idPacket(DeleteCharacter, id) }

// RestoreCharacterPacket is writer 0x59c240: opcode 0x010F and character ID.
func RestoreCharacterPacket(id uint32) []byte { return idPacket(RestoreCharacter, id) }
func SelectCharacterPacket(id uint32) []byte  { return idPacket(SelectCharacter, id) }
func NpcTalkPacket(id uint32) []byte          { return idPacket(NpcTalk, id) }
func MobAttackPacket(id uint32) []byte        { return idPacket(CharacterMobAutoAttack, id) }
func UserAttackPacket(id uint32) []byte       { return idPacket(CharacterUserAutoAttack, id) }

// LogoutPacket 0x0107 (writer 0x5981e0) and MapLoadedPacket 0x0201
// (writer 0x598210): opcode only.
func LogoutPacket() []byte    { return NewWriter(Logout).Bytes() }
func MapLoadedPacket() []byte { return NewWriter(CharacterEnteredMap).Bytes() }

// Motion bits of 0x0501 (0x58d760): the low bits are the motion, 0x80 and
// 0x40 are two flags the caller passes separately.
const (
	MotionFlag80 = 0x80
	MotionFlag40 = 0x40
)

// MovePacket is 0x0501 (writer 0x598980, 17 bytes). The original refuses
// to send non-finite coordinates; so does this.
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

// Chat channels of writer 0x599a60 (normal, trade, guild, party, shout);
// admin accounts use the 0xF1xx variants.
const (
	ChannelNormal uint8 = iota
	ChannelTrade
	ChannelGuild
	ChannelParty
	ChannelShout
)

// ChatPacket is writer 0x599a60: opcode by channel, len:u8 + text + NUL.
func ChatPacket(channel uint8, admin bool, text string) []byte {
	ops := [...]Opcode{ChatNormal, ChatWorld, ChatGuild, ChatParty, ChatShout}
	admins := [...]Opcode{ChatNormalAdmin, ChatWorldAdmin, 0xF104, 0xF105}
	var op Opcode
	switch {
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

// WhisperPacket is the 0x1102 request: target[21], followed by text.
func WhisperPacket(target, text string) []byte {
	if target == "" || text == "" || len(EncodeText(target)) > 20 {
		return nil
	}
	w := NewWriter(ChatWhisper)
	w.Fixed(target, 21, true)
	w.Text(text)
	return w.Bytes()
}

// ---- server -> client ----

// Faction is 0x0109: country and the highest creatable mode.
type Faction struct{ Faction, MaxMode uint8 }

func DecodeFaction(p []byte) (m Faction, err error) {
	r := NewReader(p)
	m.Faction, m.MaxMode = r.U8(), r.U8()
	return m, r.Err()
}

// CharacterSlot is one 0x0101 (handler 0x58d860; Rapture
// buildCharacterListSlot).
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
	Name                                          string
	PendingDelete, RenamePending                  bool
}

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

// IDResult is the shape of 0x0103/0x0104/0x010F replies: result:u8, id:u32.
type IDResult struct {
	Result uint8
	ID     uint32
}

func DecodeIDResult(p []byte) (m IDResult, err error) {
	r := NewReader(p)
	m.Result, m.ID = r.U8(), r.U32()
	return m, r.Err()
}

// DecodeU8 reads a single-byte result (0x0102 create).
func DecodeU8(p []byte) (uint8, error) { r := NewReader(p); v := r.U8(); return v, r.Err() }

// DecodeU32 reads a single id (0x0107, 0x0202, 0x0602, 0x0E02).
func DecodeU32(p []byte) (uint32, error) { r := NewReader(p); v := r.U32(); return v, r.Err() }

// Details is 0x0105 (handler 0x58dbd0; Rapture buildCharacterDetails).
type Details struct {
	StatPoint, SkillPoint        uint16
	MaxHP, MaxMP, MaxSP          uint32
	Dir                          uint16
	PrevExp, NextExp, Exp, Money uint32
	X, Y, Z                      float32
	Kills, Deaths, Wins, Losses  uint32
	Guild                        string
}

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

// UserEnter is the server's 0x0201: a user placed in view (or ourselves).
type UserEnter struct {
	ID      uint32
	Marker  uint8 // 1 first load, 2 admin
	Unk     uint16
	X, Y, Z float32
	GuildID uint32
	Vehicle uint32
}

func DecodeUserEnter(p []byte) (m UserEnter, err error) {
	r := NewReader(p)
	m.ID, m.Marker, m.Unk = r.U32(), r.U8(), r.U16()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	m.GuildID, m.Vehicle = r.U32(), r.U32()
	return m, r.Err()
}

// UserMove is the server's 0x0501.
type UserMove struct {
	ID      uint32
	Motion  uint8
	Dir     uint16
	X, Y, Z float32
}

func DecodeUserMove(p []byte) (m UserMove, err error) {
	r := NewReader(p)
	m.ID, m.Motion, m.Dir = r.U32(), r.U8(), r.U16()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}

// MobSpawn is 0x0601 (Rapture buildMobEnter): mobs carry only X and Z.
type MobSpawn struct {
	ID    uint32
	New   bool
	MobID uint16
	X, Z  float32
}

func DecodeMobEnter(p []byte) (m MobSpawn, err error) {
	r := NewReader(p)
	m.ID, m.New, m.MobID, m.X, m.Z = r.U32(), r.Bool(), r.U16(), r.F32(), r.F32()
	return m, r.Err()
}

// MobMotion is 0x0603.
type MobMotion struct {
	ID     uint32
	Motion uint8
	X, Z   float32
}

func DecodeMobMove(p []byte) (m MobMotion, err error) {
	r := NewReader(p)
	m.ID, m.Motion, m.X, m.Z = r.U32(), r.U8(), r.F32(), r.F32()
	return m, r.Err()
}

// NpcSpawn is 0x0E01.
type NpcSpawn struct {
	Summoned bool
	ID       uint32
	Type     uint8
	TypeID   uint16
	X, Y, Z  float32
	Angle    uint16
}

func DecodeNpcEnter(p []byte) (m NpcSpawn, err error) {
	r := NewReader(p)
	m.ID, m.Type, m.TypeID = r.U32(), r.U8(), r.U16()
	m.X, m.Y, m.Z, m.Angle = r.F32(), r.F32(), r.F32(), r.U16()
	if r.Remaining() > 0 {
		m.Summoned = r.Bool()
	}
	return m, r.Err()
}

// NpcMotion is 0x0E03: id, motion and server position, without facing.
type NpcMotion struct {
	ID      uint32
	Motion  uint8
	X, Y, Z float32
}

func DecodeNpcMove(p []byte) (n NpcMotion, err error) {
	r := NewReader(p)
	n.ID, n.Motion, n.X, n.Y, n.Z = r.U32(), r.U8(), r.F32(), r.F32(), r.F32()
	return n, r.Err()
}

// ChatByID is 0x1101-style (sender id + text); ChatByName is the
// 0x1102/0x1107 shape (name[21] + text).
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
}

func DecodeChatByID(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.SenderID, m.Text = r.U32(), r.Text()
	return m, r.Err()
}

func DecodeChatByName(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Sender, m.Text = r.Fixed(21), r.Text()
	return m, r.Err()
}

func DecodeWhisper(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Direction, m.Sender, m.Text = r.U8(), r.Fixed(21), r.Text()
	return m, r.Err()
}

// MapPosition is 0x020B: the map and position the character is placed at.
type MapPosition struct {
	ID      uint32
	MapID   uint16
	X, Y, Z float32
}

func DecodeMapPosition(p []byte) (m MapPosition, err error) {
	r := NewReader(p)
	m.ID, m.MapID, m.X, m.Y, m.Z = r.U32(), r.U16(), r.F32(), r.F32(), r.F32()
	return m, r.Err()
}
