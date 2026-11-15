package world

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestParseRelicsShippedIni(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "data", "game", "Obelisk.ini"))
	if err != nil {
		t.Skip("Obelisk.ini not available:", err)
	}
	list, err := parseRelics(raw)
	if err != nil {
		t.Fatal(err)
	}
	perMap := map[uint16]int{}
	for _, r := range list {
		perMap[r.mapID]++
	}
	if perMap[0] != 7 || perMap[18] != 1 || perMap[30] != 1 || len(list) != 9 {
		t.Fatalf("relics per map = %v (total %d)", perMap, len(list))
	}
	first := list[0]
	if first.key != 1 || first.mobs != [3]uint16{52, 252, 152} || first.owner != relicNeutral ||
		first.x != 155.10 || first.z != 190.38 {
		t.Fatalf("first relic = %+v", *first)
	}
	if len(first.guards) == 0 || first.guards[0].mobs != [3]uint16{77, 69, 73} || first.guards[0].count != 1 {
		t.Fatalf("first relic guards = %+v", first.guards)
	}
	for _, r := range list {
		if r.mapID == 0 && r.key == 6 && r.owner != 1 {
			t.Fatalf("Obelisk_006 starts light, got %d", r.owner)
		}
		if r.mapID == 0 && r.key == 7 && r.owner != 2 {
			t.Fatalf("Obelisk_007 starts fury, got %d", r.owner)
		}
		if r.mapID == 18 && r.key != 18<<16+1 {
			t.Fatalf("zone 18 key = %#x", r.key)
		}
	}
}

func TestParseRelicsIgnoresBossSections(t *testing.T) {
	ini := []byte("[zone_0]\n[Obelisk_001] = (1,2,3),(4, 5, 6),0\n[Mob] = (1,2,3,4),(7,8,9),2\n" +
		"[bothmob_001, (19, 2, 604800, (19,1001))]  =  (229.43, 1, 1031.10),(1974, 1977)\n" +
		"[Mob] = (1,2,3,4),(10,11,12),1\n")
	list, err := parseRelics(ini)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || len(list[0].guards) != 1 {
		t.Fatalf("parsed %d relics, guards %d", len(list), len(list[0].guards))
	}
}

func TestRelicPacketLayouts(t *testing.T) {
	list := buildRelicList([]*relic{{key: 0x10002, owner: 1, x: 1, z: 2}, {key: 3, owner: 2}})
	// Frames carry a u16 length before the opcode.
	if len(list) != 2+3+2*13 || list[4] != 2 || list[5] != 2 || list[9] != 1 {
		t.Fatalf("0x2101 = % x", list)
	}
	if p := buildRelicOwner(0x10002, 2); len(p) != 2+7 || p[6] != 1 || p[8] != 2 {
		t.Fatalf("0x2102 = % x", p)
	}
	if p := buildRelicNotice(52, 0, 1); len(p) != 2+6 || p[4] != 52 || p[6] != 0 || p[7] != 1 {
		t.Fatalf("0x2103 = % x", p)
	}
}

func TestMobFriendly(t *testing.T) {
	light, fury, none := &db.MobDef{Country: 0}, &db.MobDef{Country: 1}, &db.MobDef{Country: 2}
	if !mobFriendly(light, 0) || mobFriendly(light, 1) || !mobFriendly(fury, 1) || mobFriendly(none, 0) || mobFriendly(nil, 0) {
		t.Fatal("mobFriendly mismatch")
	}
}
