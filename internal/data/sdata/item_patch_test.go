package sdata

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestNativeItemPatchPreservesUnrelatedBytes(t *testing.T) {
	b := binary.LittleEndian.AppendUint32(nil, 1)
	b = binary.LittleEndian.AppendUint32(b, 1)
	for _, s := range []string{"Original", "Old description"} {
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s)))
		b = append(b, []byte(s)...)
	}
	body := make([]byte, 81)
	body[0], body[1] = 100, 49
	body[28] = 200
	body[35] = 36
	body[36] = 1
	body[80] = 10
	b = append(b, body...)
	patches := map[ItemKey]ItemPatch{{Type: 100, TypeID: 49}: {ReqVG: 20, Description: "Adds 20 percentage points.", Stack: 255}}
	raw, err := PatchItemDefinitions(b, patches)
	if err != nil {
		t.Fatal(err)
	}
	old, _ := ParseItems(b)
	now, _ := ParseItems(raw)
	want := *old[ItemKey{100, 49}]
	want.ReqVG = 20
	want.Description = patches[ItemKey{100, 49}].Description
	if !reflect.DeepEqual(now[ItemKey{100, 49}], &want) {
		t.Fatal("unrelated parsed fields changed")
	}
	wantBody := append([]byte(nil), body...)
	wantBody[28] = 20
	wantBody[80] = 255
	if !bytes.Equal(raw[len(raw)-81:], wantBody) {
		t.Fatal("unknown native fields changed")
	}
	if _, err := PatchItemDefinitions(b, map[ItemKey]ItemPatch{{Type: 100, TypeID: 50}: {}}); err == nil {
		t.Fatal("missing definition accepted")
	}
}

func TestRealForgePatchPreservesEveryOtherDefinition(t *testing.T) {
	dir, candidate := os.Getenv("SHAIYA_DATA"), os.Getenv("SHAIYA_PATCHED_ITEMS")
	if dir == "" || candidate == "" {
		t.Skip("requires original and staged item data")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "Item/Item.SData"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(candidate)
	if err != nil {
		t.Fatal(err)
	}
	after, err := ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("../../../tools/data/forge-item-patches.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows[1:] {
		typ, _ := strconv.Atoi(r[0])
		id, _ := strconv.Atoi(r[1])
		vg, _ := strconv.Atoi(r[2])
		stack, _ := strconv.Atoi(r[3])
		it := before[ItemKey{uint8(typ), uint8(id)}]
		if it == nil {
			t.Fatal("missing item")
		}
		it.ReqVG = uint16(vg)
		it.Description = r[4]
		if stack > 1 {
			it.Count = 255
		}
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("staged native item patch changed unexpected fields")
	}
	t.Logf("all fields checked across %d definitions", len(after))
}
