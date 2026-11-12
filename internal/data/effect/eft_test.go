package effect

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func fixture(format string) []byte {
	b := bytes.NewBufferString(format)
	u := func(v uint32) { binary.Write(b, binary.LittleEndian, v) }
	f := func(v float32) { u(math.Float32bits(v)) }
	s := func(v string) { u(uint32(len(v))); b.WriteString(v) }
	u(1)
	s("mesh.3de")
	u(1)
	s("glow.dds")
	u(1)
	s("emitter")
	for _, v := range []uint32{0, 0, 0, 1, 5, 0, 2, 0, 0, 0} {
		u(v)
	}
	for _, v := range []float32{.03, 0, 0, 0, .1, 0, 0, 0} {
		f(v)
	}
	for i := 0; i < 15; i++ {
		f(float32(i))
	}
	u(1)
	u(0)
	u(0)
	f(1)
	f(2)
	f(3)
	f(.1)
	u(1)
	u(1)
	f(.5)
	u(2)
	if format == "EF3" {
		f(7)
		f(8)
	}
	u(1)
	f(1)
	f(.5)
	f(.25)
	f(1)
	f(.2)
	u(1)
	f(.8)
	f(.3)
	u(1)
	f(2)
	f(3)
	f(.4)
	u(0)
	u(0)
	u(0)
	u(0)
	u(1)
	u(0)
	u(1)
	s("sequence")
	u(1)
	u(0)
	f(.1)
	return b.Bytes()
}

func TestEFTLayoutsAndTruncation(t *testing.T) {
	for _, format := range []string{"EFT", "EF2", "EF3"} {
		b := fixture(format)
		l, err := ParseEFT(b)
		if err != nil {
			t.Fatal(format, err)
		}
		e := l.Effects[0]
		if e.Name != "emitter" || e.Position != [3]float32{6, 7, 8} || e.SrcBlend != 5 || e.DstBlend != 2 || e.Textures[0] != 0 || l.Sequences[0].Records[0].Time != .1 {
			t.Fatalf("%s: %+v", format, l)
		}
		if format == "EF3" && e.EF3Unk != [2]float32{7, 8} {
			t.Fatal("lost EF3 fields")
		}
		for n := 0; n < len(b); n++ {
			if _, err := ParseEFT(b[:n]); err == nil {
				t.Fatalf("%s accepted prefix %d", format, n)
			}
		}
		if _, err := ParseEFT(append(b, 0)); err == nil {
			t.Fatal("accepted trailing data")
		}
	}
}

func FuzzEFT(f *testing.F) {
	f.Add(fixture("EF3"))
	f.Add([]byte("EFT"))
	f.Fuzz(func(t *testing.T, b []byte) { ParseEFT(b) })
}

func TestRealEFT(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count, bad, emitters, sequences := 0, 0, 0, 0
	formats := map[string]int{}
	a.Walk(func(p string, _ *archive.File) {
		if !strings.HasSuffix(strings.ToLower(p), ".eft") {
			return
		}
		b, err := a.ReadFile(p)
		var l *Library
		if err == nil {
			l, err = ParseEFT(b)
		}
		if err != nil {
			bad++
			if bad <= 10 {
				t.Errorf("%s: %v", p, err)
			}
			return
		}
		count++
		formats[l.Format]++
		emitters += len(l.Effects)
		sequences += len(l.Sequences)
	})
	if count == 0 || bad > 0 {
		t.Fatalf("EFT parsed=%d failed=%d", count, bad)
	}
	t.Logf("%d libraries (%v), %d emitters, %d sequences", count, formats, emitters, sequences)
}

func TestRealCastCatalog(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	files := map[string]bool{}
	supported := 0
	for id := uint16(1); id <= 400; id++ {
		file := CastLibrary(id)
		if files[file] {
			continue
		}
		files[file] = true
		b, err := a.ReadFile(file)
		if err != nil {
			t.Fatal(file, err)
		}
		l, err := ParseEFT(b)
		if err != nil || len(l.Sequences) == 0 {
			t.Fatalf("cast library %s: %v", file, err)
		}
		seq := l.Sequences[min(1, len(l.Sequences)-1)]
		for _, r := range seq.Records {
			if r.Effect >= 0 && int(r.Effect) < len(l.Effects) && l.Effects[r.Effect].LinearSupported() {
				e := l.Effects[r.Effect]
				if e.UsesMesh() {
					if int(e.Mesh) >= len(l.Meshes) {
						t.Fatal("invalid mesh index", file)
					}
					name := path.Base(strings.ReplaceAll(l.Meshes[e.Mesh], `\`, "/"))
					raw, err := a.ReadFile("Effect/3DE/" + name)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := Parse3DE(raw); err != nil {
						t.Fatal(name, err)
					}
				}
				supported++
				break
			}
		}
	}
	t.Logf("%d cast libraries validated; %d have a supported hand emitter", len(files), supported)
}
