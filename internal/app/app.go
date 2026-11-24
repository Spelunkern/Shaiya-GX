// Package app wires the screens, the network session and the game state.
// Everything here runs on the main thread; network calls run in goroutines
// and post their results back through App.post.
package app

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/clientlog"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type Config struct {
	DisplayModes       func() []platform.DisplayMode
	CurrentDisplayMode func() platform.DisplayMode
	ApplyDisplayMode   func(platform.DisplayMode) error
	OverrideDisplay    bool
	RememberLayout     bool
	Volume             float64
	LoginAddr          string
	WorldPort          int
	Version            uint32
}

type screen int

const (
	screenLogin screen = iota
	screenServers
	screenCharacters
	screenWorld
)

type App struct {
	showCatalogIDs                      bool
	physicsClockAt                      time.Time
	collisionLines                      *render.DebugLines
	collisionVertices                   []render.DebugLineVertex
	missingInterfaces                   int
	tooltipMuted                        bool
	forgeQuoteSerial                    uint32
	npcSkillDefs                        map[sdata.SkillKey]sdata.Skill
	chatBubbles                         []chatBubble
	quickPotions                        quickPotionView
	evolution                           evolutionView
	revolver                            revolverView
	mapLabels                           mapLabelView
	skillSoundEvents                    map[skillSoundKey]float64
	loading                             mapLoading
	loadingScreen                       *ui.LoadingScreen
	loadingRulesLoaded                  bool
	mapScreens                          map[uint16]string
	randomLoading                       []string
	appliedDisplay                      platform.DisplayMode
	displayConfigured                   bool
	social                              socialView
	socialUI                            *ui.SocialPanel
	socialInviteUI                      *ui.Message
	appliedHideCloak, appliedHideHelmet bool
	combatNumbers                       []combatNumber
	buffLayout                          ui.BuffLayout
	gameMenu                            *ui.GameMenu
	menuOpen                            bool
	raidOnce                            bool
	raidReturn                          uint8
	menuConfirm                         *ui.Message
	menuIntent                          int
	menuPending                         bool
	menuDeadline                        float64
	deathUI                             *ui.DeathDialog
	guildMessage                        *ui.Message
	guildConfirm                        *ui.Message
	unionConfirm                        *ui.Message // confirm before dismantling a raid
	guildVoteMessage                    *ui.Message
	learnConfirm                        *ui.Message
	learnKey                            sdata.SkillKey
	guildDuelMessage                    *ui.Message
	guildDuel                           guildDuelView
	guildCreateUI                       *ui.GuildCreateDialog
	guildUI                             *ui.GuildPanel
	guildDirectoryUI                    *ui.GuildDirectory
	guildHouseUI                        *ui.GuildHouseWindow
	npcTalkUI                           *ui.NPCTalk
	remoteBank                          bool // Warehouse Call: the warehouse window opened from its buff.
	uiHidden                            bool // F11 hides the whole world UI (DAT_007198c4 cleared).
	tipTexture                          *render.Texture
	guildPointUI                        *ui.GuildPointWindow
	house                               houseView
	guild                               guildView
	playerContext                       playerContext
	playerMenuUI                        *ui.PlayerMenu
	duelUI                              *ui.ExchangePanel
	duelMessage                         *ui.Message
	duel                                duelView
	performance                         performanceHUD
	bank                                warehouseView
	bankUI                              *ui.Warehouse
	blacksmithUI                        *ui.Blacksmith
	emblemUI                            *ui.EmblemWindow
	emblem                              emblemState
	drop                                itemDrop
	gateUI                              *ui.Gatekeeper
	gateConfirm                         *ui.Message
	teleportStone                       teleportStoneView
	recordedPoints                      *ui.RecordedPoints
	summonRune                          summonRuneView
	summonMessage                       *ui.Message
	gemConfirm                          *ui.Message
	appearanceConfirm                   *ui.Message
	party                               partyView
	partyUI                             *ui.PartyPanel
	partyMessage                        *ui.Message
	vehicleMessage                      *ui.Message
	windowDrag                          windowDrag
	sound                               *audio.Engine
	soundErrors                         map[string]bool
	mapAudio                            mapAudioState
	cfg                                 Config
	arc                                 *archive.Archive
	sprites                             *render.Sprites
	actionBar                           *ui.ActionBar
	inventoryUI                         *ui.Inventory
	statusUI                            *ui.Status
	marketUI                            *ui.Market
	personalMarketUI                    *ui.PersonalMarket
	auction                             auctionView
	auctionSequence                     uint64
	auctionUI                           auctionWidgets
	auctionHover                        *proto.AuctionRow
	auctionConfirm                      *ui.Message
	pvpRewardUI                         *ui.PvPRewardPanel
	pvpRewardMessage                    *ui.Message
	pvpReward                           pvpRewardView
	mallUI                              *ui.Mall
	mallConfirm                         *ui.Message
	mall                                mallView
	quantityUI                          *ui.Quantity
	appearanceUI                        *ui.AppearanceEditor
	appearanceEdit                      *appearanceEdit
	skillBook                           *ui.SkillBook
	factionUI                           *ui.FactionScreen
	accountMessage                      *ui.Message
	account                             accountAction
	playerFeedback                      []playerFeedback
	characterSlotFile                   string
	restoreCharacterSlot                bool
	factionDeadline                     float64
	hud                                 *ui.HUD
	minimap                             *ui.Minimap
	bottom                              *ui.BottomBar
	subActionsUI                        *ui.SubActions
	subActionsOpen                      bool
	font                                *ui.Font
	tooltip                             []itemDetailLine
	tooltipFont                         *ui.Font
	tooltipHeadingFont                  *ui.Font
	chatTextFont                        *ui.Font
	tooltipItem                         bool
	tooltipDescription                  string
	tooltipExtra                        tooltipExtras
	tooltipSockets                      []tooltipSocket
	chatCache                           []game.ChatLine
	chatEntries                         []ui.ChatEntry
	chatFilter                          *text.WordFilter
	chatFilterApplied                   bool
	msg                                 text.SysMsg
	white                               *render.Texture
	cache                               map[string]*render.Texture
	assets                              *scene.Assets
	scene3d                             *render.Scene3D
	frameTarget                         *render.WaterTarget
	sceneGlow                           *render.SceneGlow
	sceneGlowFailed                     bool
	msaaTarget                          *render.MSAATarget
	fxaa                                *render.FXAA
	taa                                 *render.TemporalAA
	taaFailed                           bool
	aaMode                              int
	aaSamples                           []int
	aaSize                              [2]int
	aaContext                           [4]int
	msaaFailed, fxaaFailed              bool
	frameWidth, frameHeight             int
	physW, physH                        int // framebuffer size; the UI may work in a smaller-scale virtual window
	worlds                              map[string]*scene.World
	skinner                             *render.Skinner
	items                               map[sdata.ItemKey]*sdata.Item
	mallCatalog                         []cash.Product
	blessingBannerUntil                 float64
	blessingFeedbackActive              bool
	screenNotices                       []screenNotice
	actor                               *scene.Character
	actorID                             uint32
	actorAppearance                     scene.Appearance
	createActor                         *scene.Character
	createAppearance                    scene.Appearance
	createDeadline                      float64
	login3                              *scene.LoginScene
	login3Failed                        bool
	wv                                  *worldView
	dungeonMinimaps                     dungeonMinimapData
	hoverPicking                        bool
	hoveredMob                          uint32
	hoveredMobX, hoveredMobY            int
	lastMouseX                          int
	lastMouseY                          int

	screen   screen
	login    *ui.Login
	servers  *ui.ServerList
	chars    *ui.CharSelect
	create   *ui.Creation
	creating bool

	chatOpen                  bool
	chatInput                 ui.TextField
	chatCaret, recipientCaret ui.ChatCaret
	chatRecipientIndex        int
	chatUI                    *ui.ChatPanel
	chatHistory               []string
	chatHistoryIndex          int
	chatDraft                 string
	chatItems                 []chatItemDraft

	npcDefs           map[sdata.NPCKey]sdata.NPC
	questDefs         []sdata.Quest
	questDataLoaded   bool
	questMapCache     questMarkerCache
	questStartedAt    map[uint16]time.Time
	questSnapshotAt   map[uint16]time.Time
	questUI           *ui.QuestLog
	questConfirm      *ui.Message
	questItemOffer    itemQuestOffer
	questItemLinks    map[sdata.ItemKey]sdata.QuestItemLinks
	questLog          questLogView
	exchangeUI        *ui.ExchangePanel
	exchangeMoney     *ui.ExchangeMoney
	exchangeInvite    *ui.Message
	exchange          exchangeView
	npcQuestUI        *ui.NPCQuest
	npcQuest          npcQuestView
	shop              shopView
	personalMarket    personalMarketView
	blacksmith        blacksmithView
	gate              gatekeeperView
	vehicle           vehicleView
	gemLink           gemLinkView
	gemRemove         gemRemoveView
	enchant           enchantView
	compose           composeView
	attackVariant     uint8
	combatNotice      string
	combatNoticeUntil float64
	inventory         inventoryView
	status            statusView
	skills            skillsView
	buffs             buffsView
	targetEffects     targetEffectsView
	skillDefs         map[sdata.SkillKey]sdata.Skill

	sess  *session.Login
	world *session.World
	state *game.State

	clock    float64
	oxygenAt float64
	frameDT  float64

	posted chan func()
	Quit   bool
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
	fonts, err := ui.LoadFontSource(arc.ReadFile)
	if err != nil {
		return nil, err
	}
	if a.font, err = fonts.Load(13); err != nil {
		return nil, err
	}
	if a.tooltipFont, err = fonts.Load(14); err != nil {
		return nil, err
	}
	if a.tooltipHeadingFont, err = fonts.LoadEmphasis(14); err != nil {
		return nil, err
	}

	// Chat is larger than the 13px UI cell.
	if a.chatTextFont, err = fonts.Load(17); err != nil {
		return nil, err
	}
	a.chatTextFont.SetBorder(.5)

	if b, err := arc.ReadFile("sysmsg-uni.txt"); err == nil {
		a.msg = text.ParseSysMsg(b)
	} else {
		a.msg = text.SysMsg{}
	}
	ui.SetText(a.msg.Lookup)
	if b, err := arc.ReadFile("Filter.INI"); err == nil {
		a.chatFilter = text.ParseWordFilter(b)
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
	if a.create, err = ui.NewCharCreate(a, a.msg); err != nil {
		return nil, err
	}
	if a.accountMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.deathUI, err = ui.NewDeathDialog(a); err != nil {
		return nil, err
	}
	if a.hud, err = ui.NewHUD(a); err != nil {
		return nil, err
	}
	if a.hud.NumberFont, err = fonts.Load(12); err != nil {
		return nil, err
	}
	if a.minimap, err = ui.NewMinimap(a); err != nil {
		return nil, err
	}
	if a.bottom, err = ui.NewBottomBar(a); err != nil {
		return nil, err
	}
	if a.gameMenu, err = ui.NewGameMenu(a); err != nil {
		return nil, err
	}
	if cfg.DisplayModes != nil {
		a.gameMenu.SetDisplayModes(cfg.DisplayModes())
	}
	a.aaSamples = render.SupportedMSAASamples()
	a.gameMenu.SetAntialiasSamples(a.aaSamples)
	if a.menuConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.subActionsUI, err = ui.NewSubActions(a); err != nil {
		return nil, err
	}
	if a.chatUI, err = ui.NewChatPanel(a); err != nil {
		return nil, err
	}
	if a.playerMenuUI, err = ui.NewPlayerMenu(a); err != nil {
		return nil, err
	}
	var actionBarPlus *render.Texture
	if _, ok := a.arc.Lookup("interface/main_slot_plus.png"); ok {
		actionBarPlus, _ = a.Interface("main_slot_plus.png")
	}
	if a.actionBar, err = ui.NewActionBar(a, actionBarPlus); err != nil {
		return nil, err
	}
	if a.inventoryUI, err = ui.NewInventory(a); err != nil {
		return nil, err
	}
	a.inventoryUI.DockOrigin = a.inventoryDockOrigin
	if a.duelUI, err = ui.NewDuelPanel(a); err != nil {
		return nil, err
	}
	if a.duelMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.unionConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.auctionConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.learnConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildVoteMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildDuelMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.guildCreateUI, err = ui.NewGuildCreateDialog(a); err != nil {
		return nil, err
	}
	if a.guildUI, err = ui.NewGuildPanel(a); err != nil {
		return nil, err
	}
	if a.socialUI, err = ui.NewSocialPanel(a); err != nil {
		return nil, err
	}
	if a.socialInviteUI, err = ui.NewMessage(a); err != nil {
		return nil, err
	}

	if a.guildDirectoryUI, err = ui.NewGuildDirectory(a); err != nil {
		return nil, err
	}
	if a.guildCreateUI != nil && a.guildDirectoryUI != nil {
		a.guildCreateUI.Anchor = func(w, h int) (int, int) {
			x, y := a.guildDirectoryUI.Origin(w, h)
			return x + 0xa0, y + 0x8c
		}
	}

	if a.npcTalkUI, err = ui.NewNPCTalk(a); err != nil {
		return nil, err
	}
	if a.guildHouseUI, err = ui.NewGuildHouseWindow(a); err != nil {
		return nil, err
	}
	if a.guildPointUI, err = ui.NewGuildPointWindow(a); err != nil {
		return nil, err
	}
	if a.exchangeUI, err = ui.NewExchangePanel(a); err != nil {
		return nil, err
	}
	if a.exchangeMoney, err = ui.NewExchangeMoney(a); err != nil {
		return nil, err
	}
	if a.exchangeInvite, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.questUI, err = ui.NewQuestLog(a); err != nil {
		return nil, err
	}
	if a.npcQuestUI, err = ui.NewNPCQuest(a); err != nil {
		return nil, err
	}
	if a.questConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.gateUI, err = ui.NewGatekeeper(a); err != nil {
		return nil, err
	}
	if a.gateConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.summonMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	a.summonMessage.Movable = true
	if a.emblemUI, err = ui.NewEmblemWindow(a); err != nil {
		return nil, err
	}
	if a.blacksmithUI, err = ui.NewBlacksmith(a); err != nil {
		return nil, err
	}
	if a.appearanceConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.gemConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.bankUI, err = ui.NewWarehouse(a); err != nil {
		return nil, err
	}
	if a.statusUI, err = ui.NewStatus(a); err != nil {
		return nil, err
	}
	if a.marketUI, err = ui.NewMarket(a); err != nil {
		return nil, err
	}
	if a.personalMarketUI, err = ui.NewPersonalMarket(a); err != nil {
		return nil, err
	}
	if a.pvpRewardUI, err = ui.NewPvPRewardPanel(a); err != nil {
		return nil, err
	}
	if a.pvpRewardMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.mallUI, err = ui.NewMall(a, a.white); err != nil {
		return nil, err
	}
	if a.mallConfirm, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.quantityUI, err = ui.NewQuantity(a); err != nil {
		return nil, err
	}
	if a.appearanceUI, err = ui.NewAppearanceEditor(a); err != nil {
		return nil, err
	}
	if a.partyUI, err = ui.NewPartyPanel(a); err != nil {
		return nil, err
	}
	if a.vehicleMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.partyMessage, err = ui.NewMessage(a); err != nil {
		return nil, err
	}
	if a.skillBook, err = ui.NewSkillBook(a); err != nil {
		return nil, err
	}
	if a.factionUI, err = ui.NewFaction(a, a.msg); err != nil {
		return nil, err
	}
	if cfg.Volume > 0 {
		a.sound, err = audio.New(arc.ReadFile, cfg.Volume)
		if err != nil {
			log.Printf("audio unavailable: %v", err)
		}
		a.soundErrors = map[string]bool{}
	}
	a.loadingScreen = ui.NewLoadingScreen(a)
	a.loadSavedID()
	a.loadUILayout()
	a.loadClientSettings()
	a.gameMenu.Settings.Antialias = ui.NormalizeAntialias(a.gameMenu.Settings.Antialias, a.aaSamples)
	if a.gameMenu.Settings.DisplayMode == (platform.DisplayMode{}) && cfg.CurrentDisplayMode != nil {
		a.gameMenu.SetCurrentDisplayMode(cfg.CurrentDisplayMode())
	}
	a.applyClientSettings()
	if a.missingInterfaces > 0 {
		a.login.Message = "Interface resources missing or damaged. See diagnostic log."
	}
	return a, nil
}

// OptionalInterface allows interface panels to work with older data packs.
func (a *App) OptionalInterface(name string) *render.Texture {
	if _, ok := a.arc.Lookup("interface/" + name); !ok {
		return nil
	}
	t, _ := a.Interface(name)
	return t
}

// Interface implements ui.Textures: data/interface/<name>, cached.
func (a *App) Interface(name string) (*render.Texture, error) {
	key := strings.ToLower(name)
	if t, ok := a.cache[key]; ok {
		return t, nil
	}
	b, err := a.arc.ReadFile("interface/" + name)
	var t img.Texture
	if err == nil {
		t, err = img.DecodeResource("interface/"+name, b)
		if err == nil && key == "option_graphic.png" {
			prepareGraphicsMenuArt(t.RGBA)
		}
	}
	if err != nil {
		// Interface artwork is presentation, never authoritative game data.
		// Cache a visible substitute so an absent atlas cannot prevent startup
		// or flood the log on every frame. Keep normal control hit regions.
		clientlog.Warnf("interface/%s: %v; using placeholder", name, err)
		a.missingInterfaces++
		t = missingInterfaceTexture()
	}
	tex := render.Upload(t)
	a.cache[key] = tex
	return tex, nil
}

func (a *App) post(f func()) { a.posted <- f }

// Update runs one frame of logic.
func (a *App) Update(in *platform.Input, w, h int, dt float64) {
	started := time.Now()
	defer func() { a.performance.updateCPU = time.Since(started) }()
	a.frameDT = dt
	a.skills.heldActions = 0
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
	if a.state != nil {
		a.state.Blessing.Advance(dt)
	}
	loading := a.updateMapLoading()
	a.updatePendingNumbers()
	a.updateAudio()
	if loading {
		a.clock += dt
		return
	}
	// Below the 1024x768 design the whole UI shrinks in proportion: it works in
	// a virtual window of the design size and the pointer is converted to it.
	a.physW, a.physH = w, h
	if scale := a.uiScale(w, h); scale < 1 {
		virtual := *in
		virtual.MouseX, virtual.MouseY = int(float64(in.MouseX)/scale), int(float64(in.MouseY)/scale)
		if a.screen != screenWorld {
			virtual.MouseDX, virtual.MouseDY = in.MouseDX/scale, in.MouseDY/scale
		}
		in, w, h = &virtual, int(math.Round(float64(w)/scale)), int(math.Round(float64(h)/scale))
	}
	switch a.screen {
	case screenLogin:
		switch a.login.Update(in, w, h, dt) {
		case ui.LoginSubmit:
			a.playUISound("ui_login001.ogg") // native 0x4bc4a0: submitting the login
			a.submitLogin()
		case ui.LoginQuit:
			a.Quit = true
		}
	case screenServers:
		switch a.servers.Update(in, w, h, dt) {
		case ui.ServerConnect:
			a.playUI()
			a.connectWorld()
		case ui.ServerBack:
			a.playUI()
			a.backToLogin("")
		}
	case screenCharacters, screenWorld:
		a.updateCharacters(in, w, h, dt)
	}
}

func (a *App) Draw(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	if a.frameTarget == nil || a.frameWidth != w || a.frameHeight != h {
		if a.frameTarget != nil {
			a.frameTarget.Delete()
		}
		var err error
		a.frameTarget, err = render.NewWaterTarget(w, h)
		if err != nil {
			clientlog.Warnf("frame target: %v", err)
			a.drawFrame(w, h)
			return
		}
		a.frameWidth, a.frameHeight = w, h
	}
	previousDepth := a.scene3d.ReverseDepth
	a.scene3d.SetReverseDepth(true)
	defer a.scene3d.SetReverseDepth(previousDepth)
	a.frameTarget.Capture(render.V3{}, func() { a.drawFrame(w, h) })
	a.frameTarget.PresentBrightness(a.clientSettings().Brightness)
}

func (a *App) drawFrame(w, h int) {
	restoreAlpha := render.KeepSceneAlphaOpaque()
	defer restoreAlpha()
	a.sprites.InvalidateNameDepth()
	if a.performance.meter == nil {
		a.performance.meter = render.NewFrameMeter()
	}
	a.performance.meter.Begin()
	a.performance.sample(a.frameDT)
	render.ResetDrawCalls()
	started := time.Now()
	defer func() { a.performance.meter.End(); a.performance.drawCPU = time.Since(started) }()
	if a.loading.active && !a.loading.started {
		a.drawMapLoading(w, h)
		return
	}
	msaa := a.beginSceneAntialias(w, h)
	if a.screen == screenLogin || a.screen == screenServers {
		if ls := a.loginBackdrop(); ls != nil {
			ls.Update(a.frameDT)
			ls.Draw(a.scene3d, projectionAspect(w, h))
		}
	}
	if a.screen == screenWorld {
		a.drawWorldView(w, h)
	}
	if a.screen == screenCharacters && a.state != nil && (a.state.Phase == game.PhaseCharSelect || a.state.Phase == game.PhaseLoading) {
		fury := a.state.Faction.Faction == 1
		if a.creating {
			a.drawCreation(w, h)
		} else {
			a.drawWorld(a.selectWorld(fury), selectEye, selectTarget, w, h)
		}
	}
	// Resolve before the loading screen so it never draws into the MSAA target.
	a.endSceneAntialias(msaa)
	if a.loading.active {
		a.drawMapLoading(w, h)
		return
	}
	if a.scene3d.ReverseDepth && !a.sceneGlowFailed {
		if a.sceneGlow == nil {
			var err error
			a.sceneGlow, err = render.NewSceneGlow()
			if err != nil {
				clientlog.Warnf("default scene glow: %v", err)
				a.sceneGlowFailed = true
			}
		}
		if a.sceneGlow != nil {
			a.sceneGlow.Level = a.clientSettings().Glow
			if err := a.sceneGlow.Apply(); err != nil {
				clientlog.Warnf("default scene glow: %v", err)
				a.sceneGlowFailed = true
			}
		}
	}
	uiBefore := render.DrawCalls()
	a.physW, a.physH = w, h
	render.SetUIScale(1)
	if scale := a.uiScale(w, h); scale < 1 {
		render.SetUIScale(scale)
		w, h = int(math.Round(float64(w)/scale)), int(math.Round(float64(h)/scale))
	}
	a.sprites.Begin(w, h)
	if a.screen == screenWorld {
		a.drawSunFlare(w, h)
	}
	switch a.screen {
	case screenLogin:
		a.login.Draw(a.sprites, a.font, w, h)
	case screenServers:
		a.login.DrawLogo(a.sprites, w, h)
		a.servers.Draw(a.sprites, a.font, w, h)
	case screenCharacters, screenWorld:
		a.drawCharacters(w, h)
	}
	if a.accountMessage != nil {
		a.accountMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.partyMessage != nil {
		a.partyMessage.Draw(a.sprites, a.font, w, h)
		a.vehicleMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.exchangeInvite != nil {
		a.exchangeInvite.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.guildMessage != nil {
		a.guildMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.auctionConfirm != nil {
		a.auctionConfirm.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.unionConfirm != nil {
		a.unionConfirm.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.guildConfirm != nil {
		a.guildConfirm.Draw(a.sprites, a.font, w, h)
		if a.socialInviteUI != nil {
			a.socialInviteUI.Draw(a.sprites, a.font, w, h)
		}
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.guildDuelMessage != nil {
		a.guildDuelMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.duelMessage != nil {
		a.duelMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.pvpRewardMessage != nil {
		a.pvpRewardMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.guildCreateUI != nil {
		a.guildCreateUI.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.guildVoteMessage != nil {
		a.guildVoteMessage.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() && a.learnConfirm != nil {
		a.learnConfirm.Draw(a.sprites, a.font, w, h)
	}
	if a.screen == screenWorld && !a.worldUIHidden() {
		a.drawMallConfirm(w, h)
	}
	a.drawPerformance(w)
	a.sprites.End()
	if a.screen == screenWorld && a.wv != nil {
		a.wv.draws.UI = render.DrawCalls() - uiBefore
	}
}

func (a *App) submitLogin() {
	id, pw := a.login.ID.Text, a.login.Password.Text
	if id == "" || pw == "" || a.login.Busy {
		return
	}
	a.login.Busy, a.login.Message = true, a.tr(326)
	a.saveID()
	go func() {
		l, err := session.Dial(a.cfg.LoginAddr, id, pw)
		a.post(func() {
			a.login.Busy = false
			if err != nil {
				if _, rejected := err.(session.LoginError); !rejected {
					clientlog.Warnf("Login connection failed: %v", err)
				}
				a.login.Message = a.loginMessage(err)
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
func (a *App) loginMessage(err error) string {
	if le, ok := err.(session.LoginError); ok {
		return a.msg.Format(12277, map[string]string{"v": fmt.Sprint(le.Code)})
	}
	return a.tr(10019)
}

// backToLoginID is backToLogin with a SysMsg id (see tr).
func (a *App) backToLoginID(id int) { a.backToLogin(a.tr(id)) }

// tr is the SysMsg text for id: sysmsg-uni.txt first, then the embedded GX rows.
// User-visible text lives in those tables, never in code, so a language is
// one table.
func (a *App) tr(id int) string { return a.msg.Lookup(id) }

func (a *App) backToLogin(msg string) {
	a.loading = mapLoading{}
	a.menuOpen, a.menuPending = false, false
	a.buffs = buffsView{}
	a.targetEffects = targetEffectsView{}
	a.party = partyView{}
	if a.partyMessage != nil {
		a.partyMessage.Visible = false
	}
	a.releaseWorld()
	if a.world != nil {
		a.world.Close()
		a.world = nil
	}
	a.sess, a.state, a.uiHidden = nil, nil, false
	a.creating = false
	a.account = accountAction{}
	a.playerFeedback = a.playerFeedback[:0]
	if a.accountMessage != nil {
		a.accountMessage.Visible = false
	}
	a.chatOpen = false
	a.chatItems = nil
	if a.factionUI != nil {
		a.factionUI.Reset()
	}
	a.inventory = inventoryView{}
	a.teleportStone = teleportStoneView{}
	a.summonRune = summonRuneView{}
	if a.summonMessage != nil {
		a.summonMessage.Visible = false
	}
	a.skills = skillsView{}
	a.status = statusView{}
	if a.statusUI != nil {
		a.statusUI.Clear()
	}
	a.shop = shopView{}
	a.personalMarket = personalMarketView{selected: -1}
	a.auction = auctionView{}
	a.blacksmith = blacksmithView{}
	a.gate = gatekeeperView{}
	a.vehicle = vehicleView{}
	if a.vehicleMessage != nil {
		a.vehicleMessage.Visible = false
	}
	if a.gateConfirm != nil {
		a.gateConfirm.Visible = false
	}
	a.gemLink = gemLinkView{}
	a.gemRemove = gemRemoveView{}
	a.enchant = enchantView{}
	a.compose = composeView{}
	a.bank = warehouseView{}
	a.pvpReward = pvpRewardView{}
	if a.pvpRewardMessage != nil {
		a.pvpRewardMessage.Visible = false
	}
	a.questLog = questLogView{}
	a.exchange = exchangeView{}
	a.playerContext = playerContext{}
	a.duel = duelView{}
	a.guild = guildView{}
	a.guildDuel = guildDuelView{}
	if a.guildDuelMessage != nil {
		a.guildDuelMessage.Visible = false
	}
	if a.guildMessage != nil {
		a.guildMessage.Visible = false
	}
	if a.unionConfirm != nil {
		a.unionConfirm.Visible = false
	}
	if a.guildConfirm != nil {
		a.guildConfirm.Visible = false
	}
	if a.auctionConfirm != nil {
		a.auctionConfirm.Visible = false
	}
	if a.guildVoteMessage != nil {
		a.guildVoteMessage.Visible = false
	}
	if a.learnConfirm != nil {
		a.learnConfirm.Visible = false
	}
	if a.guildCreateUI != nil {
		a.guildCreateUI.Visible = false
	}
	if a.duelMessage != nil {
		a.duelMessage.Visible = false
	}
	if a.exchangeInvite != nil {
		a.exchangeInvite.Visible = false
	}
	a.npcQuest = npcQuestView{}
	a.questItemOffer = itemQuestOffer{}
	if a.questConfirm != nil {
		a.questConfirm.Visible = false
	}
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
				clientlog.Warnf("World connection failed: %v", err)
				a.backToLoginID(12007)
				return
			}
			a.playUISound("ui_login001.ogg") // native 0x4eb3c7: connected to the game server
			a.world, a.state = w, game.New()
			a.characterSlotFile = characterSlotPath(a.cfg, sv.ID, l.Result.UserUID)
			a.restoreCharacterSlot = true
			a.screen = screenCharacters
		})
	}()
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

// worldUIHidden reports the F11 state; the death dialog stays visible.
func (a *App) worldUIHidden() bool {
	return a.uiHidden && a.screen == screenWorld && a.state != nil && !a.state.Self.Dead
}

// uiScale is ui.UIScale: windows smaller than 1024x768 draw the UI shrunk.
func (a *App) uiScale(w, h int) float64 { return float64(ui.UIScale(w, h)) }
