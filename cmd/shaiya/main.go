// Command shaiya is the windowed client.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-gl/gl/v4.5-core/gl"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
	img "github.com/Spelunkern/shaiya-go-client/internal/data/image"
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

func init() { platform.LockMainThread() }

func main() {
	dataDir := flag.String("data", os.Getenv("SHAIYA_DATA"), "directory with data.sah and data.saf")
	width := flag.Int("w", 1024, "window width")
	height := flag.Int("h", 768, "window height")
	debug := flag.Bool("gldebug", false, "OpenGL debug context")
	shot := flag.String("screenshot", "", "render one frame offscreen to this PNG and exit")
	flag.Parse()
	if err := run(*dataDir, *width, *height, *debug, *shot); err != nil {
		log.Fatal(err)
	}
}

func run(dataDir string, w, h int, debug bool, shot string) error {
	if dataDir == "" {
		return fmt.Errorf("set -data or SHAIYA_DATA to the folder with data.sah/data.saf")
	}
	arc, err := archive.Open(dataDir)
	if err != nil {
		return err
	}
	defer arc.Close()
	win, err := platform.Open(platform.Config{Title: "Shaiya", Width: w, Height: h, Debug: debug, VSync: true, Hidden: shot != ""})
	if err != nil {
		return err
	}
	defer win.Close()
	if err := render.Init(debug); err != nil {
		return err
	}
	sprites, err := render.NewSprites(4096)
	if err != nil {
		return err
	}
	load := func(path string) (*render.Texture, error) {
		b, err := arc.ReadFile(path)
		if err != nil {
			return nil, err
		}
		t, err := img.Decode(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		return render.Upload(t), nil
	}
	// Placeholder screen until the login layout from 0x4b8ff0 is ported.
	panel, err := load("interface/login.tga")
	if err != nil {
		return err
	}
	logo, err := load("interface/shaiyalogo01.TGA")
	if err != nil {
		return err
	}
	frame := func() {
		fw, fh := win.GetFramebufferSize()
		gl.Viewport(0, 0, int32(fw), int32(fh))
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		sprites.Begin(fw, fh)
		sprites.DrawImage(logo, float32(fw-logo.W)/2, 40)
		sprites.DrawImage(panel, float32(fw-panel.W)/2, float32(fh-panel.H)/2+120)
		sprites.End()
	}
	if shot != "" {
		frame()
		gl.Finish()
		fw, fh := win.GetFramebufferSize()
		return render.SavePNG(shot, fw, fh)
	}
	for win.Poll() {
		frame()
		win.SwapBuffers()
	}
	return nil
}
