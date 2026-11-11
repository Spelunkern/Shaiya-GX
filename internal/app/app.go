// Package app wires the screens, the network session and the game state.
// Everything here runs on the main thread; network calls run in goroutines
// and post their results back through App.post.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
	img "github.com/Spelunkern/shaiya-go-client/internal/data/image"
	"github.com/Spelunkern/shaiya-go-client/internal/data/sdata"
	"github.com/Spelunkern/shaiya-go-client/internal/data/text"
	"github.com/Spelunkern/shaiya-go-client/internal/game"
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
	"github.com/Spelunkern/shaiya-go-client/internal/scene"
	"github.com/Spelunkern/shaiya-go-client/internal/session"
	"github.com/Spelunkern/shaiya-go-client/internal/ui"
)

type Config struct {
	LoginAddr string
	WorldPort int
	Version   uint32
}

type screen int

const (
	screenLogin screen = iota
	screenServers
	screenCharacters
	screenWorld
)

type App struct {
	cfg          Config
	arc          *archive.Archive
	sprites      *render.Sprites
	font         *ui.Font
	msg          text.SysMsg
	white        *render.Texture
	cache        map[string]*render.Texture
	assets       *scene.Assets
	scene3d      *render.Scene3D
	worlds       map[string]*scene.World
	skinner      *render.Skinner
	items        map[sdata.ItemKey]*sdata.Item
	actor        *scene.Character
	actorID      uint32
	login3       *scene.LoginScene
	login3Failed bool

	screen  screen
	login   *ui.Login
	servers *ui.ServerList
	chars   *ui.CharSelect

	sess  *session.Login
	world *session.World
	state *game.State

	clock   float64
	frameDT float64
	notice  charNotice

	posted chan func()
	Quit   bool
	status string
}

func New(cfg Config, arc *archive.Archive) (*App, error) {
	a := &App{cfg: cfg, arc: arc, cache: map[string]*render.Texture{}, posted: make(chan func(), 64),
		assets: scene.NewAssets(arc), worlds: map[string]*scene.World{}}
	var err error
	if a.scene3d, err = render.NewScene3D(); err != nil {
		return nil, err
	}
	if a.skinner, err = render.NewSkinner(); err != nil {
		return nil, err
	}
	if a.sprites, err = render.NewSprites(8192); err != nil {
		return nil, err
	}
	if a.font, err = ui.LoadFont(ui.DefaultFontPath(), 13); err != nil {
		return nil, err
	}
	if b, err := arc.ReadFile("sysmsg-uni.txt"); err == nil {
		a.msg = text.ParseSysMsg(b)
	} else {
		a.msg = text.SysMsg{}
	}
	a.white = render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	if a.login, err = ui.NewLogin(a); err != nil {
		return nil, err
	}
	if a.servers, err = ui.NewServerList(a, a.msg, a.white); err != nil {
		return nil, err
	}
	if a.chars, err = ui.NewCharSelect(a, a.msg); err != nil {
		return nil, err
	}
	a.loadSavedID()
	return a, nil
}

// Interface implements ui.Textures: data/interface/<name>, cached.
func (a *App) Interface(name string) (*render.Texture, error) {
	key := strings.ToLower(name)
	if t, ok := a.cache[key]; ok {
		return t, nil
	}
	b, err := a.arc.ReadFile("interface/" + name)
	if err != nil {
		return nil, err
	}
	t, err := img.Decode(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	tex := render.Upload(t)
	a.cache[key] = tex
	return tex, nil
}

func (a *App) post(f func()) { a.posted <- f }

// Update runs one frame of logic.
func (a *App) Update(in *platform.Input, w, h int, dt float64) {
	a.frameDT = dt
	for {
		select {
		case f := <-a.posted:
			f()
			continue
		default:
		}
		break
	}
	a.pumpWorld()
	switch a.screen {
	case screenLogin:
		switch a.login.Update(in, w, h, dt) {
		case ui.LoginSubmit:
			a.submitLogin()
		case ui.LoginQuit:
			a.Quit = true
		}
	case screenServers:
		switch a.servers.Update(in, w, h, dt) {
		case ui.ServerConnect:
			a.connectWorld()
		case ui.ServerBack:
			a.backToLogin("")
		}
	case screenCharacters, screenWorld:
		a.updateCharacters(in, w, h, dt)
	}
}

func (a *App) Draw(w, h int) {
	if a.screen == screenLogin || a.screen == screenServers {
		if ls := a.loginBackdrop(); ls != nil {
			ls.Update(a.frameDT)
			ls.Draw(a.scene3d, w, h)
		}
	}
	if a.screen == screenCharacters && a.state != nil {
		fury := a.state.Faction.Faction == 1
		a.drawWorld(a.selectWorld(fury), selectEye, selectTarget, w, h)
	}
	a.sprites.Begin(w, h)
	switch a.screen {
	case screenLogin:
		a.login.Draw(a.sprites, a.font, w, h)
	case screenServers:
		a.servers.Draw(a.sprites, a.font, w, h)
	case screenCharacters, screenWorld:
		a.drawCharacters(w, h)
	}
	a.sprites.End()
}

func (a *App) submitLogin() {
	id, pw := a.login.ID.Text, a.login.Password.Text
	if id == "" || pw == "" || a.login.Busy {
		return
	}
	a.login.Busy, a.login.Message = true, "Connecting..."
	a.saveID()
	go func() {
		l, err := session.Dial(a.cfg.LoginAddr, id, pw)
		a.post(func() {
			a.login.Busy = false
			if err != nil {
				a.login.Message = loginMessage(err)
				return
			}
			a.sess = l
			a.login.Message, a.login.Password.Text = "", ""
			a.servers.Servers, a.servers.Selected = l.Servers, -1
			if len(l.Servers) > 0 {
				a.servers.Selected = 0
			}
			a.screen = screenServers
		})
	}()
}

// NO VERIFICADO: the original maps each 0xA102 code to its own message.
func loginMessage(err error) string {
	if le, ok := err.(session.LoginError); ok {
		return fmt.Sprintf("Login failed (code %d).", le.Code)
	}
	return "Unable to connect to the login server."
}

func (a *App) backToLogin(msg string) {
	if a.world != nil {
		a.world.Close()
		a.world = nil
	}
	a.sess, a.state = nil, nil
	a.screen = screenLogin
	a.login.Message = msg
	a.login.Focus = 1
}

func (a *App) connectWorld() {
	if a.servers.Busy || a.sess == nil || a.servers.Selected < 0 {
		return
	}
	sv := a.servers.Servers[a.servers.Selected]
	a.servers.Busy = true
	l := a.sess
	go func() {
		ip, err := l.SelectServer(sv.ID, a.cfg.Version)
		var w *session.World
		if err == nil {
			w, err = l.DialWorld(ip, a.cfg.WorldPort)
		}
		a.post(func() {
			a.servers.Busy = false
			if err != nil {
				a.backToLogin("Unable to connect to the game server.")
				return
			}
			a.world, a.state = w, game.New()
			a.screen = screenCharacters
		})
	}()
}

// pumpWorld applies every packet received since the last frame.
func (a *App) pumpWorld() {
	if a.world == nil {
		return
	}
	for {
		select {
		case p, ok := <-a.world.Packets:
			if !ok {
				a.backToLogin("Disconnected from the server.")
				return
			}
			a.state.Apply(p)
			if proto.Op(p) == proto.CharacterSkillBar && a.state.Phase == game.PhaseLoading {
				a.world.Send(proto.MapLoadedPacket()) // map "loaded"
			}
		default:
			return
		}
	}
}

// ---- saved ID (LOGIN_ID_SAVE / LOGIN ID of the original) ----

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "shaiya-go", "login.txt")
}

func (a *App) loadSavedID() {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return
	}
	if id := strings.TrimSpace(string(b)); id != "" {
		a.login.ID.Text, a.login.SaveID.Checked, a.login.Focus = id, true, 1
	}
}

func (a *App) saveID() {
	p := configPath()
	if p == "" {
		return
	}
	if !a.login.SaveID.Checked {
		os.Remove(p)
		return
	}
	os.MkdirAll(filepath.Dir(p), 0o700)
	os.WriteFile(p, []byte(a.login.ID.Text), 0o600)
}
