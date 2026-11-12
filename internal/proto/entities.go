package proto

import "fmt"

// AppearanceChange is the server confirmation 0x0226 (reader 0x58fc50).
type AppearanceChange struct {
	ID                         uint32
	Hair, Face, Height, Gender uint8
}

// ChangeAppearancePacket requests a change using a consumable inventory slot.
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

func DecodeAppearanceChange(p []byte) (m AppearanceChange, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Hair, m.Face, m.Height, m.Gender = r.U8(), r.U8(), r.U8(), r.U8()
	return m, r.Err()
}

// UserShape is 0x0303 (handler 0x58e360, Rapture buildCharacterShapeGuild).
// Position arrives separately in 0x0201/0x0501.
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

// EquipmentChange is 0x0507 (0x58f980; Rapture buildCharacterEquipment).
type EquipmentChange struct {
	ID                          uint32
	Slot, Type, TypeID, Enchant uint8
	Cloak                       [6]uint8
}

func DecodeEquipment(p []byte) (m EquipmentChange, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Slot, m.Type, m.TypeID, m.Enchant = r.U8(), r.U8(), r.U8(), r.U8()
	for i := range m.Cloak {
		m.Cloak[i] = r.U8()
	}
	return m, r.Err()
}
