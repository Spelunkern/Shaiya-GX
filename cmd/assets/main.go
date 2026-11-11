// Command assets inspects the original data.sah/data.saf.
//
//	assets -data "$SHAIYA_DATA" stats
//	assets -data "$SHAIYA_DATA" ls interface
//	assets -data "$SHAIYA_DATA" cat interface/login.tga > login.tga
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
	img "github.com/Spelunkern/shaiya-go-client/internal/data/image"
	"github.com/Spelunkern/shaiya-go-client/internal/data/world"
)

func main() {
	dataDir := flag.String("data", os.Getenv("SHAIYA_DATA"), "directory with data.sah and data.saf")
	flag.Parse()
	if *dataDir == "" || flag.NArg() == 0 {
		log.Fatal("usage: assets -data DIR stats | ls [DIR] | cat FILE | png FILE OUT.png | find SUBSTR")
	}
	a, err := archive.Open(*dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	switch flag.Arg(0) {
	case "stats":
		files, bytes := 0, int64(0)
		ext := map[string]int{}
		a.Walk(func(p string, f *archive.File) {
			files++
			bytes += int64(f.Length)
			if i := strings.LastIndexByte(p, '.'); i >= 0 {
				ext[strings.ToLower(p[i:])]++
			}
		})
		fmt.Printf("version %d, header count %d, %d files, %.1f MiB\n", a.Version, a.FileCount, files, float64(bytes)/(1<<20))
		keys := make([]string, 0, len(ext))
		for k := range ext {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return ext[keys[i]] > ext[keys[j]] })
		for _, k := range keys {
			fmt.Printf("  %-8s %d\n", k, ext[k])
		}
	case "ls":
		d, ok := a.LookupDir(flag.Arg(1))
		if !ok {
			log.Fatalf("no directory %q", flag.Arg(1))
		}
		for _, s := range d.Dirs {
			fmt.Printf("%s/\n", s.Name)
		}
		for _, f := range d.Files {
			fmt.Printf("%s\t%d\n", f.Name, f.Length)
		}
	case "cat":
		b, err := a.ReadFile(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		os.Stdout.Write(b)
	case "png":
		b, err := a.ReadFile(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		t, err := img.Decode(b)
		if err != nil {
			log.Fatal(err)
		}
		if t.RGBA == nil {
			log.Fatal("compressed DDS: png export supports TGA/BMP/uncompressed only")
		}
		out := image.NewNRGBA(image.Rect(0, 0, t.RGBA.W, t.RGBA.H))
		copy(out.Pix, t.RGBA.Pix)
		f, err := os.Create(flag.Arg(2))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, out); err != nil {
			log.Fatal(err)
		}
	case "wld":
		b, err := a.ReadFile(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		w, err := world.ParseWLD(b)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("dungeon=%v mapsize=%d layers=%d layout=%q sky=%q fog=%v %.0f..%.0f effects=%q (%d)\n",
			w.Dungeon, w.MapSize, len(w.Layers), w.Layout, w.Sky, w.FogColor, w.FogStart, w.FogEnd, w.EffectFile, len(w.Effects))
		for _, g := range []struct {
			name string
			g    world.Group
		}{{"buildings", w.Buildings}, {"shapes", w.Shapes}, {"trees", w.Trees}, {"grass", w.Grass},
			{"vaniA", w.VaniA}, {"vaniB", w.VaniB}, {"dungeons", w.Dungeons}, {"objects", w.Objects}} {
			fmt.Printf("%s: %d assets, %d instances\n", g.name, len(g.g.Assets), len(g.g.Instances))
			for i, n := range g.g.Assets {
				if i < 12 {
					fmt.Printf("   %s\n", n)
				}
			}
		}
		fmt.Printf("mani: %d assets %d instances\n", len(w.ManiAssets), len(w.Manis))
		for _, l := range w.Layers {
			fmt.Printf("layer %q tile=%.2f\n", l.Texture, l.TileSize)
		}
	case "dg":
		b, err := a.ReadFile(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		d, err := world.ParseDG(b)
		if err != nil {
			log.Fatal(err)
		}
		nodes, meshes, verts := 0, 0, 0
		d.Root.Walk(func(n *world.DGNode) {
			nodes++
			for _, g := range n.Groups {
				for _, m := range g.Meshes {
					meshes++
					verts += len(m.Vertices)
				}
			}
		})
		fmt.Printf("bounds=%v lightmaps=%d nodes=%d meshes=%d vertices=%d\n", d.Bounds, d.Lightmaps, nodes, meshes, verts)
		for _, t := range d.Textures {
			fmt.Println("  ", t)
		}
	case "smod":
		b, err := a.ReadFile(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		m, err := world.ParseSMOD(b)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("center=%v radius=%.1f box=%v meshes=%d\n", m.Center, m.Radius, m.ViewBox, len(m.Meshes))
		for _, x := range m.Meshes {
			fmt.Printf("   %s: %d vertices %d faces\n", x.Texture, len(x.Vertices), len(x.Faces))
		}
	case "find":
		needle := strings.ToLower(flag.Arg(1))
		a.Walk(func(p string, f *archive.File) {
			if strings.Contains(strings.ToLower(p), needle) {
				fmt.Printf("%s\t%d\n", p, f.Length)
			}
		})
	default:
		log.Fatalf("unknown command %q", flag.Arg(0))
	}
}
