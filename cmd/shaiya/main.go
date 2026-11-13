// Command shaiya is the windowed client.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-gl/gl/v4.5-core/gl"

	"github.com/Spelunkern/Shaiya-GX/internal/app"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func init() { platform.LockMainThread() }

func main() {
	dataDir := flag.String("data", os.Getenv("SHAIYA_DATA"), "directory with data.sah and data.saf")
	login := flag.String("login", "127.0.0.1:30800", "login server address")
	worldPort := flag.Int("world-port", 30810, "world server port")
	width := flag.Int("w", 1024, "window width")
	height := flag.Int("h", 768, "window height")
	volume := flag.Float64("volume", 0.6, "master audio volume (0..1; screenshots are silent)")
	debug := flag.Bool("gldebug", false, "OpenGL debug context")
	shot := flag.String("screenshot", "", "render offscreen to this PNG and exit")
	shotScreen := flag.String("screenshot-screen", "login", "offscreen scene: login | servers | chars | create | faction | world-*")
	autoSelect := flag.Int("autoplay-select", -1, "with -autoplay: stop at character select on this slot")
	autoCast := flag.Int("autoplay-cast-slot", 0, "with -autoplay: activate slot 1..10 and wait for cast start")
	autoplay := flag.String("autoplay", "", "account to log in with (SHAIYA_PASSWORD) offscreen until in world, then save -screenshot")
	flag.Parse()
	if *dataDir == "" {
		if exe, err := os.Executable(); err == nil {
			dir := filepath.Dir(exe)
			if _, err := os.Stat(filepath.Join(dir, "data.sah")); err == nil {
				*dataDir = dir
			}
		}
	}
	if *autoCast < 0 || *autoCast > 10 {
		log.Fatal("-autoplay-cast-slot must be 0..10")
	}
	if *dataDir == "" {
		log.Fatal("set -data or SHAIYA_DATA to the folder with data.sah/data.saf")
	}
	if err := run(*dataDir, app.Config{LoginAddr: *login, WorldPort: *worldPort, Version: proto.ClientVersion, Volume: *volume}, *width, *height, *debug, *shot, *shotScreen, *autoplay, *autoSelect, *autoCast); err != nil {
		log.Fatal(err)
	}
}

func run(dataDir string, cfg app.Config, w, h int, debug bool, shot, shotScreen, autoUser string, autoSelect, autoCast int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	arc, err := archive.Open(dataDir)
	if err != nil {
		return err
	}
	defer arc.Close()
	restoreSize := true
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "w" || f.Name == "h" {
			restoreSize = false
		}
	})
	win, err := platform.Open(platform.Config{Title: "Shaiya GX", Width: w, Height: h, Debug: debug, VSync: true,
		Hidden: shot != "" || autoUser != "", RememberSize: true, RestoreSize: restoreSize})
	if err != nil {
		return err
	}
	defer win.Close()
	if err := render.Init(debug); err != nil {
		return err
	}
	if shot != "" || autoUser != "" {
		cfg.Volume = 0
	}
	cfg.RememberLayout = shot == "" && autoUser == ""
	a, err := app.New(cfg, arc)
	if err != nil {
		return err
	}
	defer a.Close()
	cursors := a.LoadCursors(win)
	defer cursors.Close()
	queue := win.InstallInput()
	var auto *app.Autoplay
	inWorld := false
	if autoUser != "" {
		auto = &app.Autoplay{User: autoUser, Password: os.Getenv("SHAIYA_PASSWORD"), StopAtSelect: autoSelect >= 0, Slot: autoSelect, CastSlot: uint8(autoCast), WaitResult: shotScreen == "world-projectile"}
	}
	frame := func(dt float64) {
		fw, fh := win.GetFramebufferSize()
		in := queue.Frame()
		if auto != nil && auto.Inject(a, &in, dt) {
			inWorld = true
		}
		a.Update(&in, fw, fh, dt)
		queue.SetMouseLook(a.MouseLookActive())
		gl.Viewport(0, 0, int32(fw), int32(fh))
		gl.ClearColor(0, 0, 0, 1)
		gl.ClearDepth(1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(fw, fh)
		cursors.Set(a.CursorIndex(fw, fh))
	}
	if auto != nil {
		start := platform.Time()
		last := start
		for win.Poll() && !inWorld && ctx.Err() == nil {
			now := platform.Time()
			if now-start > 30 {
				return fmt.Errorf("autoplay: not in world after 30 s")
			}
			frame(now - last)
			last = now
			win.SwapBuffers()
		}
		if ctx.Err() != nil {
			return nil
		}
		for i := 0; i < 30; i++ { // let the view fill in
			frame(1.0 / 60)
			win.SwapBuffers()
		}
		frame(1.0 / 60)
		gl.Finish()
		if shot != "" {
			fw, fh := win.GetFramebufferSize()
			return render.SavePNG(shot, fw, fh)
		}
		return nil
	}
	if shot != "" {
		if err := a.Preview(shotScreen); err != nil {
			return err
		}
		for i := 0; !a.SceneReady(); i++ {
			if i >= 240 {
				return fmt.Errorf("preview scene did not finish streaming")
			}
			frame(0) // Resource warmup must not change the animation time of a capture.
		}
		for i := 0; i < 3; i++ {
			frame(.25)
		}

		gl.Finish()
		fw, fh := win.GetFramebufferSize()
		if err := render.SavePNG(shot, fw, fh); err != nil {
			return err
		}
		fmt.Println("saved", shot)
		return nil
	}
	last := platform.Time()
	for win.Poll() && !a.Quit && ctx.Err() == nil {
		now := platform.Time()
		frame(now - last)
		last = now
		win.SwapBuffers()
	}
	return nil
}
