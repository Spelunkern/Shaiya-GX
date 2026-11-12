package effect

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"math"
	"os"
	"strings"
	"testing"
)

func meshFixture() []byte {
	b := new(bytes.Buffer)
	u := func(v uint32) { binary.Write(b, binary.LittleEndian, v) }
	f := func(v float32) { u(math.Float32bits(v)) }
	u(4)
	b.WriteString("test")
	u(3)
	for i := 0; i < 3; i++ {
		f(float32(i))
		f(0)
		f(0)
		u(0xffffffff)
		f(float32(i) / 2)
		f(1)
	}
	u(1)
	binary.Write(b, binary.LittleEndian, []uint16{0, 1, 2})
	u(10)
	u(2)
	for _, key := range []uint32{0, 10} {
		u(key)
		for i := 0; i < 3; i++ {
			f(float32(i))
			f(float32(key))
			f(0)
			f(.5)
			f(1)
		}
	}
	return b.Bytes()
}
func Test3DE(t *testing.T) {
	b := meshFixture()
	m, err := Parse3DE(b)
	if err != nil {
		t.Fatal(err)
	}
	if m.Texture != "test" || len(m.Indices) != 3 || m.Vertices[0].Bone != -1 || m.Frames[1].Vertices[2].Position != [3]float32{2, 10, 0} {
		t.Fatalf("bad mesh %+v", m)
	}
	for n := 0; n < len(b); n++ {
		if _, err := Parse3DE(b[:n]); err == nil {
			t.Fatalf("accepted truncated prefix %d", n)
		}
	}
	if withTail, err := Parse3DE(append(b, 0)); err != nil || len(withTail.Trailing) != 1 {
		t.Fatal("lost unused trailing bytes", err)
	}
	// First face starts after string, vertex count/records and face count.
	bad := append([]byte(nil), b...)
	binary.LittleEndian.PutUint16(bad[88:], 3)
	if _, err := Parse3DE(bad); err == nil {
		t.Fatal("accepted invalid index")
	}
}
func Fuzz3DE(f *testing.F) {
	f.Add(meshFixture())
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) { Parse3DE(b) })
}
func TestReal3DE(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count, vertices, frames, rejected := 0, 0, 0, 0
	// These assets do not match Parsec's 3DE layout. Keep exact identities:
	// an empty placeholder and files with invalid indices/records or truncation.
	known := map[string]string{
		"effect/3de/004.3de":            "e94ab471e00cb86550d5b90a9d1db48d1c36df903f4ee9dbe766b6b37e6fe6a8",
		"effect/3de/deathknight007.3de": "52af003a64d233101e6e3132da3523af710187d7e7bab94c39b077825ff1e523",
		"effect/3de/test1.3de":          "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"effect/3de/undead5-02.3de":     "f89b34d35212a600476fdd06844991ae81023e3f0b714c130d7e46c85f09284b",
	}
	a.Walk(func(p string, _ *archive.File) {
		if !strings.HasSuffix(strings.ToLower(p), ".3de") {
			return
		}
		b, err := a.ReadFile(p)
		if err != nil {
			t.Error(err)
			return
		}
		m, err := Parse3DE(b)
		hash := fmt.Sprintf("%x", sha256.Sum256(b))
		if expected, ok := known[strings.ToLower(p)]; ok && hash == expected && err != nil {
			rejected++
			t.Logf("known non-3DE layout %s: %v", p, err)
			return
		}
		if err != nil {
			t.Errorf("%s: %v", p, err)
			return
		}
		count++
		vertices += len(m.Vertices)
		frames += len(m.Frames)
	})
	if count == 0 {
		t.Fatal("no 3DE meshes")
	}
	if rejected != len(known) {
		t.Fatalf("expected %d known rejected assets, got %d", len(known), rejected)
	}
	t.Logf("%d meshes, %d vertices, %d frames; %d known non-3DE layouts rejected", count, vertices, frames, rejected)
}

// The detailed cast sequence of Ice Arrow contains an authored 3DE particle
// as well as the billboard. Keep both available to the live loader.
func TestRealIceArrowHandMesh(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	raw, err := a.ReadFile(CastLibrary(37))
	if err != nil {
		t.Fatal(err)
	}
	l, err := ParseEFT(raw)
	if err != nil {
		t.Fatal(err)
	}
	meshes, billboards := 0, 0
	for _, r := range l.Sequences[1].Records {
		e := l.Effects[r.Effect]
		if !e.LinearSupported() {
			t.Fatal("unsupported Ice Arrow hand physics")
		}
		if e.Mesh < 0 {
			billboards++
			continue
		}
		raw, err := a.ReadFile("Effect/3DE/" + l.Meshes[e.Mesh])
		if err != nil {
			t.Fatal(err)
		}
		m, err := Parse3DE(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Frames) != 0 || len(m.Vertices) != 4 || len(m.Indices) != 6 {
			t.Fatal("unexpected hand mesh")
		}
		meshes++
	}
	if meshes != 1 || billboards != 1 {
		t.Fatalf("meshes=%d billboards=%d", meshes, billboards)
	}
}
