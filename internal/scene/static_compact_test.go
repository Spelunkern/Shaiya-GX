package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"os"
	"strings"
	"testing"
)

func TestRealStaticAssetAudit(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("assets")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	files, parts, adjacent, vertices, unique, rejected, indices := 0, 0, 0, 0, 0, 0, 0
	arc.Walk(func(name string, f *archive.File) {
		if !strings.HasSuffix(strings.ToLower(name), ".smod") {
			return
		}
		raw, err := arc.Read(f)
		if err != nil {
			t.Error(err)
			return
		}
		m, err := world.ParseSMOD(raw)
		if err != nil {
			rejected++
			return
		}
		files++
		for i, p := range m.Meshes {
			parts++
			vertices += len(p.Vertices)
			indices += len(p.Faces) * 3
			seen := map[world.Vertex]bool{}
			for _, v := range p.Vertices {
				seen[v] = true
			}
			unique += len(seen)
			if i > 0 && strings.EqualFold(m.Meshes[i-1].Texture, p.Texture) {
				adjacent++
			}
		}
	})
	t.Logf("SMOD files%d rejected%d parts%d adjacent same-material%d vertices%d exact unique%d; uint16 indices%d save%d bytes", files, rejected, parts, adjacent, vertices, unique, indices, indices*2)
}
