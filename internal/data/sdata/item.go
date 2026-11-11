package sdata

import (
	"encoding/binary"
	"fmt"
)

// Item is one EP5 Item.SData definition (Parsec Shaiya/Item/ItemDefinition,
// EP5 layout). Only the fields the client uses are kept by name; the rest
// are skipped in order.
type Item struct {
	Name, Description string
	Type, TypeID      uint8
	Model, Icon       uint8
	MinLevel          uint16
	Country           uint8
	Slot              uint8
	Quality           uint16
	Attack, AttackAdd uint16
	Def, Resist       uint16
	BuyPrice          uint32
	SellPrice         uint32
	Count             uint8
}

type ItemKey struct{ Type, TypeID uint8 }

type cursor struct {
	b   []byte
	off int
	err error
}

func (c *cursor) take(n int) []byte {
	if c.err != nil || n < 0 || c.off+n > len(c.b) {
		if c.err == nil {
			c.err = fmt.Errorf("sdata: truncated at %d", c.off)
		}
		return make([]byte, max(n, 0))
	}
	p := c.b[c.off : c.off+n]
	c.off += n
	return p
}
func (c *cursor) u8() uint8   { return c.take(1)[0] }
func (c *cursor) u16() uint16 { return binary.LittleEndian.Uint16(c.take(2)) }
func (c *cursor) u32() uint32 { return binary.LittleEndian.Uint32(c.take(4)) }
func (c *cursor) count(min int) int {
	n := int(int32(c.u32()))
	if c.err == nil && (n < 0 || n*min > len(c.b)-c.off) {
		c.err = fmt.Errorf("sdata: bad count %d at %d", n, c.off-4)
		return 0
	}
	return n
}
func (c *cursor) str() string {
	b := c.take(c.count(1))
	for i, x := range b {
		if x == 0 {
			b = b[:i]
			break
		}
	}
	return latin1(b)
}

func latin1(b []byte) string {
	r := make([]rune, len(b))
	for i, x := range b {
		r[i] = rune(x)
	}
	return string(r)
}

// ParseItems reads Item.SData (decrypting it if needed).
func ParseItems(raw []byte) (map[ItemKey]*Item, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	out := map[ItemKey]*Item{}
	groups := c.count(4)
	for g := 0; g < groups && c.err == nil; g++ {
		n := c.count(60)
		for i := 0; i < n && c.err == nil; i++ {
			it := &Item{Name: c.str(), Description: c.str()}
			it.Type, it.TypeID, it.Model, it.Icon = c.u8(), c.u8(), c.u8(), c.u8()
			it.MinLevel = c.u16()
			it.Country = c.u8()
			c.take(6)  // class flags AttackFighter..DefenseMage
			c.take(3)  // Grow, Type2, Type3
			c.take(12) // ReqStr, Dex, Rec, Int, Wis, Luc (u16)
			c.take(4)  // ReqVg u16, ReqOg, ReqIg
			c.take(1)  // Range (u8 in EP5)
			c.take(3)  // AttackTime, Attrib, Special
			it.Slot = c.u8()
			it.Quality = c.u16()
			it.Attack, it.AttackAdd, it.Def, it.Resist = c.u16(), c.u16(), c.u16(), c.u16()
			c.take(2 * 9) // Hp, Sp, Mp, Str, Dex, Rec, Int, Wis, Luc
			c.take(2)     // Speed, Exp
			it.BuyPrice, it.SellPrice = c.u32(), c.u32()
			c.take(4) // Grade, Drop
			c.take(1) // Server
			it.Count = c.u8()
			out[ItemKey{it.Type, it.TypeID}] = it
		}
	}
	if c.err == nil && c.off != len(b) {
		return out, fmt.Errorf("sdata: %d trailing bytes after items", len(b)-c.off)
	}
	return out, c.err
}
