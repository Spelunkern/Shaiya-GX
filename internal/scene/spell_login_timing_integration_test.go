//go:build integration

package scene

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOriginalSpellReleaseAndLoginWaterGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "release and water", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	// Concentration's main target column starts at one second. The
	// native release is a different sequence and must already emit before it.
	release, err := LoadEffect(assets, "Effect/IN_mc_yellow.EFT", 0)
	if err != nil {
		t.Fatal(err)
	}
	target, err := LoadEffect(assets, "Effect/DA_target_concentration.EFT", 0)
	if err != nil {
		t.Fatal(err)
	}
	for range 18 {
		release.Update(1.0 / 60)
		target.Update(1.0 / 60)
	}
	particles := func(e *Effect) int {
		n := 0
		for _, emitter := range e.emitters {
			n += len(emitter.particles)
		}
		return n
	}
	if particles(release) == 0 || target.emitters[0].starts != 1 || len(target.emitters[0].particles) != 0 {
		t.Fatalf("native early release/late target phases: particles %d/%d", particles(release), particles(target))
	}
	for range 54 {
		target.Update(1.0 / 60)
	}
	if len(target.emitters[0].particles) == 0 {
		t.Fatal("authored later target phase never emitted")
	}
	login, err := LoadLoginScene(assets)
	if err != nil {
		t.Fatal(err)
	}
	defer login.World.Close()
	for _, asset := range login.World.vanis {
		if asset.frameMilliseconds != 33 || asset.frameAt(.2) != 6 {
			t.Fatal("native VANI pose clock incorrect")
		}
		for _, part := range asset.parts {
			var wrap int32
			gl.GetTextureParameteriv(part.texture.ID, gl.TEXTURE_WRAP_S, &wrap)
			if wrap != gl.REPEAT {
				t.Fatal("VANI animated UVs are clamped")
			}
		}
	}
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	login.Camera.Skip()
	login.World.Update(.2)
	gl.Viewport(0, 0, 1024, 768)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	login.Draw(r, 1024.0/768)
	var depthWrite bool
	gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthWrite)
	if !depthWrite {
		t.Fatal("water leaked disabled depth writes into world rendering")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("login GL error %#x", code)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = render.SavePNG(filepath.Join(dir, "login-water-native.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	eye := render.V3{436, 339, 433}
	r.SetCameraView(eye, render.V3{436, 328, 463}, render.V3{0, 1, 0})
	for range 120 {
		login.World.Update(1.0 / 60)
		login.World.PrepareMapEffects(r, eye)
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	r.Begin()
	login.World.Draw(r)
	login.World.RenderMapEffects(r)
	r.End()
	for _, placed := range login.World.mapFX.instances {
		if placed.sequence == 16 && (placed.instance == nil || placed.instance.Skipped != 0) {
			t.Fatal("fountain skipped original water emitters")
		}
	}
	if err = render.SavePNG(filepath.Join(dir, "login-fountain-native.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	t.Log("Concentration emits release at 0.3 s, main target column after 1 s; original login UVs repeat at 33 ms per frame")
}
