package sdata

import (
	"encoding/binary"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/itemstack"
	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
	"math"
)

// Item is one EP5 Item.SData definition (Parsec Shaiya/Item/ItemDefinition,
// EP5 layout). Only the fields the client uses are kept by name; the rest
// are skipped in order.
type Item struct {
	Name, Description                  string
	Special                            uint8
	Server                             uint8
	RecoveryHP, RecoverySP, RecoveryMP uint16
	Type, TypeID                       uint8
	Model, Icon                        uint8
	MinLevel                           uint16
	Country                            uint8
	Classes                            [6]uint8
	Grow                               uint8
	RequiredStats, Stats               [6]uint16
	Range, AttackTime, Attribute       uint8
	ReqOG, ReqIg                       uint8
	ReqVG                              uint16
	Slot                               uint8
	Quality                            uint16
	Attack, AttackAdd                  uint16
	Def, Resist                        uint16
	BuyPrice                           uint32
	SellPrice                          uint32
	Count                              uint16
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
func (c *cursor) u8() uint8    { return c.take(1)[0] }
func (c *cursor) u16() uint16  { return binary.LittleEndian.Uint16(c.take(2)) }
func (c *cursor) u32() uint32  { return binary.LittleEndian.Uint32(c.take(4)) }
func (c *cursor) f32() float32 { return math.Float32frombits(c.u32()) }
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

func latin1(b []byte) string { return textwire.Decode(b) }

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
			copy(it.Classes[:], c.take(6))
			it.Grow = c.u8()
			c.take(2) // Type2, Type3
			for i := range it.RequiredStats {
				it.RequiredStats[i] = c.u16()
			}
			it.ReqVG = c.u16()
			it.ReqOG = c.u8()
			it.ReqIg = c.u8()
			it.Range, it.AttackTime, it.Attribute = c.u8(), c.u8(), c.u8()
			it.Special = c.u8()
			it.Slot = c.u8()
			it.Quality = c.u16()
			it.Attack, it.AttackAdd, it.Def, it.Resist = c.u16(), c.u16(), c.u16(), c.u16()
			it.RecoveryHP, it.RecoverySP, it.RecoveryMP = c.u16(), c.u16(), c.u16()
			for i := range it.Stats {
				it.Stats[i] = c.u16()
			}
			c.take(2) // Speed, Exp
			it.BuyPrice, it.SellPrice = c.u32(), c.u32()
			c.take(4) // Grade, Drop
			it.Server = c.u8()
			it.Count = uint16(c.u8())
			if it.Count > 1 {
				it.Count = itemstack.Max
			}
			out[ItemKey{it.Type, it.TypeID}] = it
		}
	}
	if c.err == nil && c.off != len(b) {
		return out, fmt.Errorf("sdata: %d trailing bytes after items", len(b)-c.off)
	}
	return out, c.err
}
