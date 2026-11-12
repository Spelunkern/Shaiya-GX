package sdata

import "fmt"

type CashItem struct {
	ItemID uint32
	Count  uint8
}

// CashProduct is an original Cash.SData entry, including its 24 item slots.
type CashProduct struct {
	Index, Bag, Icon, Cost  uint32
	Items                   [24]CashItem
	Name, Code, Description string
}

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
			p.Items[j] = CashItem{c.u32(), c.u8()}
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
