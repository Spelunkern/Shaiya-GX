package sdata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
)

// Cash.SData uses the original binary table layout, including byte quantities.
type CashItem = cash.Item
type CashProduct = cash.Product

func ParseCash(raw []byte) ([]CashProduct, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	n := c.count(16 + 24*5 + 12)
	products := make([]CashProduct, n)
	for i := range products {
		p := &products[i]
		p.Index, p.Bag, p.Icon, p.Cost = c.u32(), c.u32(), c.u32(), c.u32()
		for j := range p.Items {
			p.Items[j] = CashItem{ItemID: c.u32(), Count: uint16(c.u8())}
		}
		p.Name, p.Code, p.Description = c.str(), c.str(), c.str()
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("cash: %d trailing bytes", len(b)-c.off)
	}
	return products, nil
}

// EncodeCash writes an unencrypted native SData table. Encrypted originals
// remain supported by ParseCash; quantity overflow is rejected, never truncated.
func EncodeCash(products []CashProduct) ([]byte, error) {
	if err := cash.Validate(products); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	word := func(v uint32) { _ = binary.Write(&out, binary.LittleEndian, v) }
	word(uint32(len(products)))
	for _, p := range products {
		word(p.Index)
		word(p.Bag)
		word(p.Icon)
		word(p.Cost)
		for _, it := range p.Items {
			word(it.ItemID)
			out.WriteByte(byte(it.Count))
		}
		for _, value := range []string{p.Name, p.Code, p.Description} {
			word(uint32(len([]byte(value))))
			out.WriteString(value)
		}
	}
	return out.Bytes(), nil
}
