package sdata

import (
	"encoding/binary"
	"testing"
)

func TestItemTooltipFieldsUseEP5Offsets(t *testing.T) {
	// Parsec EP5 ItemDefinition, 81 bytes after its two length-prefixed strings.
	raw := make([]byte, 16+81)
	binary.LittleEndian.PutUint32(raw, 1)
	binary.LittleEndian.PutUint32(raw[4:], 1)
	p := raw[16:]
	p[0], p[1] = 1, 2
	p[7], p[11] = 1, 1
	p[13] = 2
	binary.LittleEndian.PutUint16(p[16:], 27)
	p[31] = 11
	p[32], p[33], p[34] = 24, 3, 2
	binary.LittleEndian.PutUint16(p[53:], 19)
	binary.LittleEndian.PutUint16(p[63:], 23)
	defs, err := ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	d := defs[ItemKey{1, 2}]
	if d.Classes != ([6]uint8{1, 0, 0, 0, 1, 0}) || d.Grow != 2 || d.RequiredStats[0] != 27 || d.Stats[0] != 19 || d.Stats[5] != 23 || d.ReqIg != 11 || d.Range != 24 || d.AttackTime != 3 || d.Attribute != 2 {
		t.Fatal("EP5 tooltip fields shifted", d)
	}
}
