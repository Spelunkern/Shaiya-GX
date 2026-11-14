package sdata

import (
	"encoding/binary"
	"fmt"
)

// ItemPatch changes only the specified native definition fields. All other
// record bytes, including unknown fields, are preserved exactly.
type ItemPatch struct {
	ReqVG       uint16
	Description string
	Stack       uint8
}

func PatchItemDefinitions(raw []byte, patches map[ItemKey]ItemPatch) ([]byte, error) {
	plain, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: plain}
	groups := c.count(4)
	out := binary.LittleEndian.AppendUint32(nil, uint32(groups))
	found := map[ItemKey]bool{}
	for g := 0; g < groups && c.err == nil; g++ {
		count := c.count(60)
		out = binary.LittleEndian.AppendUint32(out, uint32(count))
		for i := 0; i < count && c.err == nil; i++ {
			start := c.off
			c.str()
			nameEnd := c.off
			c.str()
			descEnd := c.off
			body := append([]byte(nil), c.take(81)...)
			if c.err != nil {
				return nil, c.err
			}
			key := ItemKey{Type: body[0], TypeID: body[1]}
			out = append(out, plain[start:nameEnd]...)
			if patch, ok := patches[key]; ok {
				found[key] = true
				out = binary.LittleEndian.AppendUint32(out, uint32(len([]byte(patch.Description))))
				out = append(out, []byte(patch.Description)...)
				binary.LittleEndian.PutUint16(body[28:30], patch.ReqVG)
				if patch.Stack != 0 {
					body[80] = patch.Stack
				}
			} else {
				out = append(out, plain[nameEnd:descEnd]...)
			}
			out = append(out, body...)
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(plain) {
		return nil, fmt.Errorf("Item.SData trailing data")
	}
	for key := range patches {
		if !found[key] {
			return nil, fmt.Errorf("missing patched item %d/%d", key.Type, key.TypeID)
		}
	}
	if _, err := ParseItems(out); err != nil {
		return nil, err
	}
	return out, nil
}
