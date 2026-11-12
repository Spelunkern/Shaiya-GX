package model

import (
	"os"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func each(t *testing.T, ext string, fn func(string, []byte) error) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ok, bad := 0, 0
	a.Walk(func(p string, f *archive.File) {
		if !strings.HasSuffix(strings.ToLower(p), ext) || !strings.HasPrefix(p, "Character/") {
			return
		}
		b, err := a.Read(f)
		if err == nil {
			err = fn(p, b)
		}
		if err != nil {
			bad++
			if bad <= 8 {
				t.Errorf("%s: %v", p, err)
			}
			return
		}
		ok++
	})
	t.Logf("%s: %d ok, %d failed", ext, ok, bad)
}

func TestReal3DC(t *testing.T) {
	each(t, ".3dc", func(_ string, b []byte) error { _, err := Parse3DC(b); return err })
}
func TestRealANI(t *testing.T) {
	each(t, ".ani", func(_ string, b []byte) error { _, err := ParseANI(b); return err })
}
func TestRealMLT(t *testing.T) {
	each(t, ".mlt", func(_ string, b []byte) error { _, err := ParseMLT(b); return err })
}

func TestRealMON(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Monster/monster.MON")
	if err != nil {
		t.Fatal(err)
	}
	ms, err := ParseMON(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("empty MON")
	}
	t.Logf("%d MON records; first: %+v", len(ms), ms[0])
}

func TestRealNPCMON(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Npc/npc.MON")
	if err != nil {
		t.Fatal(err)
	}
	defs, err := ParseMON(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) <= 42 {
		t.Fatal("missing NPC model 42")
	}
	t.Logf("%d NPC visuals; 42: %s", len(defs), defs[42].Name)
}

func TestRealALT(t *testing.T) {
	each(t, ".alt", func(_ string, b []byte) error { _, err := ParseALT(b); return err })
}

func TestRealITM(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count := 0
	a.Walk(func(p string, f *archive.File) {
		if !strings.HasPrefix(p, "Item/") || !strings.HasSuffix(strings.ToLower(p), ".itm") {
			return
		}
		b, err := a.Read(f)
		if err != nil {
			t.Error(err)
			return
		}
		table, err := ParseITM(b)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			return
		}
		t.Logf("%s: %s %d models", p, table.Signature, len(table.Records))
		if p == "Item/19.ITM" {
			t.Logf("shield first: %+v; meshes=%v", table.Records[0], table.Meshes[:min(3, len(table.Meshes))])
		}
		count++
	})
	if count == 0 {
		t.Fatal("no ITM tables read")
	}
}

func TestRealVehicleMON(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	for _, race := range []string{"Hu", "El", "Vi", "De"} {
		b, err := a.ReadFile("Vehicle/Vehicle_" + race + "_01.MON")
		if err != nil {
			t.Fatal(err)
		}
		rows, err := ParseMON(b)
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range rows {
			t.Logf("%s %d %s animations=%v parts=%v height=%g", race, i, v.Name, v.Animations, v.Parts, v.Height)
		}
	}
}
