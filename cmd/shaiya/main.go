// Command shaiya is the windowed client.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-gl/gl/v4.5-core/gl"

	"github.com/Spelunkern/shaiya-go-client/internal/app"
	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

func init() { platform.LockMainThread() }

func main() {
	dataDir := flag.String("data", os.Getenv("SHAIYA_DATA"), "directory with data.sah and data.saf")
	login := flag.String("login", "127.0.0.1:30800", "login server address")
	worldPort := flag.Int("world-port", 30810, "world server port")
	width := flag.Int("w", 1024, "window width")
	height := flag.Int("h", 768, "window height")
	debug := flag.Bool("gldebug", false, "OpenGL debug context")
	shot := flag.String("screenshot", "", "render offscreen to this PNG and exit")
	shotScreen := flag.String("screenshot-screen", "login", "screen for -screenshot: login | servers")
	autoplay := flag.String("autoplay", "", "account to log in with (SHAIYA_PASSWORD) offscreen until in world, then save -screenshot")
	flag.Parse()
	if *dataDir == "" {
		log.Fatal("set -data or SHAIYA_DATA to the folder with data.sah/data.saf")
	}
	if err := run(*dataDir, app.Config{LoginAddr: *login, WorldPort: *worldPort, Version: proto.ClientVersion}, *width, *height, *debug, *shot, *shotScreen, *autoplay); err != nil {
		log.Fatal(err)
	}
}

func run(dataDir string, cfg app.Config, w, h int, debug bool, shot, shotScreen, autoUser string) error {
	arc, err := archive.Open(dataDir)
	if err != nil {
		return err
	}
	defer arc.Close()
	win, err := platform.Open(platform.Config{Title: "Shaiya", Width: w, Height: h, Debug: debug, VSync: true, Hidden: shot != "" || autoUser != ""})
	if err != nil {
		return err
	}
	defer win.Close()
	if err := render.Init(debug); err != nil {
		return err
	}
	a, err := app.New(cfg, arc)
	if err != nil {
		return err
	}
	queue := win.InstallInput()
	var auto *app.Autoplay
	inWorld := false
	if autoUser != "" {
		auto = &app.Autoplay{User: autoUser, Password: os.Getenv("SHAIYA_PASSWORD")}
	}
	frame := func(dt float64) {
		fw, fh := win.GetFramebufferSize()
		in := queue.Frame()
		if auto != nil && auto.Inject(a, &in, dt) {
			inWorld = true
		}
		a.Update(&in, fw, fh, dt)
		gl.Viewport(0, 0, int32(fw), int32(fh))
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		a.Draw(fw, fh)
	}
	if auto != nil {
		start := platform.Time()
		last := start
		for win.Poll() && !inWorld {
			now := platform.Time()
			if now-start > 30 {
				return fmt.Errorf("autoplay: not in world after 30 s")
			}
			frame(now - last)
			last = now
			win.SwapBuffers()
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
		frame(0.75) // caret visible
		gl.Finish()
		fw, fh := win.GetFramebufferSize()
		if err := render.SavePNG(shot, fw, fh); err != nil {
			return err
		}
		fmt.Println("saved", shot)
		return nil
	}
	last := platform.Time()
	for win.Poll() && !a.Quit {
		now := platform.Time()
		frame(now - last)
		last = now
		win.SwapBuffers()
	}
	return nil
}
