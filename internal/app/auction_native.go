package app

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Native EP5.4 auction window. Geometry follows Game.exe: layout 0x459e70
// (window 0x2e4 x 600, tabs, close), tab layouts 0x454590/0x453140/0x4536a0/
// 0x453c00, draw 0x45a970 with the per-tab routines 0x459020/0x456ac0/
// 0x457540/0x457fd0 and input 0x45ac40/0x455e40.
const (
	auctionWidth  = 0x2e4
	auctionPixels = 600
	auctionStride = 0x33
	auctionPage   = proto.AuctionPageSize
)

func auctionHeight(int) int { return auctionPixels }

var (
	auctionTabX      = [4]int{0x43, 0xe2, 0x18a, 0x229}
	auctionTabViews  = [4]int{proto.AuctionBrowse, proto.AuctionSelling, proto.AuctionItems, proto.AuctionWatch}
	auctionFrames    = [4]string{"auction_objectsearch", "auction_progress", "auction_completio", "auction_interest_list"}
	auctionRed       = [4]float32{1, 0, 0, 1}
	auctionYellow    = [4]float32{1, 1, 0, 1}
	auctionOrange    = [4]float32{1, .5, 0, 1}
	auctionBlue      = [4]float32{0, 0, 1, 1}
	auctionSelection = [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 96.0 / 255}
	auctionRowsUV    = [5]int{0, 1, 2, 2, 3}
)

// auctionGroups maps the five original categories (7011+ sub-entry texts) onto
// GX's item families, by position of the original lists.
var auctionGroups = [5]struct {
	msg int
	gx  []int
}{
	{7011, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	{7021, []int{1, 2, 3, 5, 6, 7, 4}},
	{7028, []int{2, 1, 3}},
	{7031, []int{1, 2}},
	{7033, []int{0, 2, 1}},
}

func auctionTab(view int) int {
	switch view {
	case proto.AuctionSelling, proto.AuctionBidding:
		return 1
	case proto.AuctionItems, proto.AuctionGold:
		return 2
	case proto.AuctionWatch:
		return 3
	}
	return 0
}

func (a *App) auctionPanel(w, h int) (int, int, int) {
	x, y := w/2-0x172, h/2-300
	if h == 768 {
		y += 20
	}
	x, y = a.auction.position.Origin(w, h, auctionWidth, auctionPixels, x, y)
	return x, y, auctionWidth
}

func (a *App) auctionTex(name string) *render.Texture {
	if a.assets == nil {
		return nil
	}
	return a.assets.Texture("interface/" + name + ".png")
}

type auctionWidgets struct {
	btn    map[string]*ui.Button
	scroll map[string]*ui.PanelScroll
}

// rowButton is a 0x5289a0 widget: the art cell is drawn whole and only its
// top-left hit box answers the pointer. Rows of the atlas are the states.
func (a *App) rowButton(id, file string, col, cw, ch, ox, oy, x, y, hw, hh int, rows [5]int) *ui.Button {
	u := &a.auctionUI
	if u.btn == nil {
		u.btn = map[string]*ui.Button{}
	}
	b := u.btn[id]
	if b == nil {
		b = &ui.Button{}
		u.btn[id] = b
	}
	tex := a.auctionTex(file)
	b.Tex, b.OriginX, b.OriginY, b.OffX, b.OffY = tex, ox, oy, x, y
	b.DrawW, b.DrawH, b.HitW, b.HitH = cw, ch, hw, hh
	if tex != nil {
		for st, row := range rows {
			b.UV[st] = ui.UV{U0: float32(col*cw) / float32(tex.W), V0: float32(row*ch) / float32(tex.H),
				U1: float32((col+1)*cw) / float32(tex.W), V1: float32((row+1)*ch) / float32(tex.H)}
		}
	}
	return b
}

// colButton is a 0x528830 widget whose states are columns of a 32px cell.
func (a *App) colButton(id, file string, art, ox, oy, x, y int) *ui.Button {
	u := &a.auctionUI
	if u.btn == nil {
		u.btn = map[string]*ui.Button{}
	}
	b := u.btn[id]
	if b == nil {
		b = &ui.Button{}
		u.btn[id] = b
	}
	tex := a.auctionTex(file)
	b.Tex, b.OriginX, b.OriginY, b.OffX, b.OffY = tex, ox, oy, x, y
	b.DrawW, b.DrawH, b.HitW, b.HitH = art, art, art, art
	if tex != nil {
		for st, col := range [5]int{0, 1, 2, 2, 0} {
			b.UV[st] = ui.UV{U0: float32(col*32) / float32(tex.W), V0: 0,
				U1: float32(col*32+art) / float32(tex.W), V1: float32(art) / float32(tex.H)}
		}
	}
	return b
}

func (a *App) auctionScroller(id string, x, y, h int) *ui.PanelScroll {
	u := &a.auctionUI
	if u.scroll == nil {
		u.scroll = map[string]*ui.PanelScroll{}
	}
	tex := a.auctionTex("scrollbar")
	if tex == nil {
		return nil
	}
	s := u.scroll[id]
	if s == nil {
		n := ui.NewPanelScroll(tex, x, y, h)
		s = &n
		u.scroll[id] = s
	}
	return s
}

type auctionControl struct {
	b  *ui.Button
	on bool
	do func()
}

func (a *App) auctionClose() {
	a.auction = auctionView{}
	a.shop.npcID = 0
	if a.world != nil {
		_ = a.world.Send(proto.NpcTalkPacket(0))
	}
}

func (a *App) auctionSwitch(view int) {
	v := &a.auction
	if view == v.view || view == auctionPair(v.view) && auctionPair(view) >= 0 && auctionTab(view) == auctionTab(v.view) {
		return
	}
	v.view, v.other = view, auctionList{selected: -1}
	v.setList(auctionList{selected: -1})
	v.sort, v.scroll, v.dropdown, v.focus = 0, 0, 0, -1
	a.requestAuction(proto.AuctionRequest{})
}

func (a *App) auctionSearch() {
	v := &a.auction
	if v.pending {
		return
	}
	v.page, v.selected, v.scroll, v.exactType, v.focus, v.dropdown = 0, -1, 0, 0, -1, 0
	v.minLevel, v.maxLevel = "", ""
	if v.minIdx > 0 {
		v.minLevel = strconv.Itoa(v.minIdx * 10)
	}
	if v.maxIdx > 0 {
		v.maxLevel = strconv.Itoa(v.maxIdx * 10)
	}
	v.category, v.subcategory = v.group, 0
	if v.group > 0 && v.sub > 0 {
		v.subcategory = auctionGroups[v.group-1].gx[v.sub-1]
	}
	// EP5.4 0x1C01 uses the unrestricted selector when the boxes agree.
	switch v.unchecked {
	case [2]bool{false, true}:
		v.searchFaction = proto.AuctionLight
	case [2]bool{true, false}:
		v.searchFaction = proto.AuctionFury
	default:
		v.searchFaction = proto.AuctionAnyFaction
	}
	v.searchGrade = nil
	if v.grade < 7 {
		grade := uint8(v.grade)
		if grade == 6 {
			grade = 7
		}
		v.searchGrade = &grade
	}
	a.requestAuction(proto.AuctionRequest{})
}

func auctionGroupY(group, i int) int {
	if group >= 0 && i > group {
		return 0x160 + i*0x19
	}
	return 0x84 + i*0x19
}

// auctionControls lists the native widgets of the current tab with their
// enabled state (0x455e40/0x454fe0/0x455380/0x455780).
func (a *App) auctionControls(ox, oy int) []auctionControl {
	v := &a.auction
	var c []auctionControl
	c = append(c, auctionControl{a.colButton("close", "talk1_close_button", 22, ox, oy, 0x2c5, 6), true, a.auctionClose})
	tab := auctionTab(v.view)
	for i := range 4 {
		rows := auctionRowsUV
		if i == tab {
			rows = [5]int{1, 1, 1, 1, 1}
		}
		view := auctionTabViews[i]
		c = append(c, auctionControl{a.rowButton("tab"+strconv.Itoa(i), "auction_top_manual_button", i, 128, 64, ox, oy, auctionTabX[i], 0x14, 118, 30, rows),
			true, func() { a.auctionSwitch(view) }})
	}
	button := func(id, file string, col, x, y int, on bool, do func()) {
		c = append(c, auctionControl{a.rowButton(id, file, col, 128, 32, ox, oy, x, y, 84, 28, auctionRowsUV), on, do})
	}
	act := func(action string) func() { return func() { a.auctionAction(action) } }
	pane := func(view int, action string) func() {
		return func() { v.selectPane(view); a.auctionAction(action) }
	}
	switch v.view {
	case proto.AuctionBrowse:
		c = append(c,
			auctionControl{a.rowButton("find", "auction_buy_search_botton", 0, 64, 32, ox, oy, 0x293, 0x43, 47, 22, auctionRowsUV), true, a.auctionSearch},
			auctionControl{a.rowButton("prev", "market_button", 0, 32, 32, ox, oy, 0x17f, 0x1f7, 23, 23, auctionRowsUV), v.page > 0, func() { a.auctionPage(-1) }},
			auctionControl{a.rowButton("next", "market_button", 1, 32, 32, ox, oy, 0x1db, 0x1f7, 23, 23, auctionRowsUV), v.more, func() { a.auctionPage(1) }})
		for i := range 5 {
			group := i + 1
			c = append(c, auctionControl{a.rowButton("cat"+strconv.Itoa(i), "auction_completion", i, 128, 32, ox, oy, 0x12, auctionGroupY(v.group-1, i), 128, 20, auctionRowsUV), true,
				func() {
					v.dropdown = 0
					if v.group == group {
						v.group = 0
					} else {
						v.group = group
					}
					v.sub = 0
				}})
		}
		button("bid", "auction_objectsearch_button", 0, 0x146, 0x233, v.actionAvailable("bid"), act("bid"))
		button("buy", "auction_objectsearch_button", 1, 0x1aa, 0x233, v.actionAvailable("buy"), act("buy"))
		button("watch", "auction_objectsearch_button", 2, 0x20e, 0x233, v.actionAvailable("watch"), act("watch"))
		button("exact", "auction_objectsearch_button", 3, 0x272, 0x233, v.actionAvailable("exact"), act("exact"))
	case proto.AuctionSelling, proto.AuctionBidding:
		selling, bidding := v.view == proto.AuctionSelling, v.view == proto.AuctionBidding
		button("setup", "auction_progress_button", 0, 0x15, 0x231, true, pane(proto.AuctionSelling, "setup"))
		button("cancel", "auction_progress_button", 1, 0x79, 0x231, selling && v.actionAvailable("cancel"), act("cancel"))
		button("buy", "auction_objectsearch_button", 1, 0x216, 0x231, bidding && v.actionAvailable("buy"), act("buy"))
		button("watch", "auction_objectsearch_button", 2, 0x27a, 0x231, bidding && v.actionAvailable("watch"), act("watch"))
	case proto.AuctionItems, proto.AuctionGold:
		items, gold := v.view == proto.AuctionItems, v.view == proto.AuctionGold
		other := v.other
		allItems, allGold := items && v.actionAvailable("collectall") || !items && len(other.rows) > 0 || !items && other.page > 0,
			gold && v.actionAvailable("collectall") || !gold && len(other.rows) > 0 || !gold && other.page > 0
		button("get", "auction_completion_button", 0, 0x15, 0x231, items && v.actionAvailable("collect"), act("collect"))
		button("getall", "auction_completion_button", 1, 0x79, 0x231, allItems, pane(proto.AuctionItems, "collectall"))
		button("gold", "auction_completion_button", 2, 0x216, 0x231, gold && v.actionAvailable("collect"), act("collect"))
		button("goldall", "auction_completion_button", 3, 0x27a, 0x231, allGold, pane(proto.AuctionGold, "collectall"))
	default:
		button("refresh", "auction_interestlist_button", 0, 0x15, 0x231, true, act("refresh"))
		button("view", "auction_interestlist_button", 1, 0x79, 0x231, v.actionAvailable("details"), act("details"))
		button("delete", "auction_interestlist_button", 2, 0x216, 0x231, v.actionAvailable("unwatch"), act("unwatch"))
		button("deleteall", "auction_interestlist_button", 3, 0x27a, 0x231, v.actionAvailable("clearwatch"), act("clearwatch"))
	}
	return c
}

func (a *App) auctionPage(delta int) {
	v := &a.auction
	page := v.page + delta
	if page < 0 || delta > 0 && !v.more {
		return
	}
	v.page, v.selected, v.scroll = page, -1, 0
	a.requestAuction(proto.AuctionRequest{})
}

// paneList returns the list shown by a paired tab for view.
func (v *auctionView) paneList(view int) auctionList {
	if view == v.view {
		return v.list()
	}
	return v.other
}

// pageScroll turns a paired list's page into the scrollbar position the
// original shows for its whole result list.
func pageScroll(l auctionList) (count, offset int) {
	count = auctionPage * (l.page + 1)
	if l.more {
		count += auctionPage
	}
	return count, l.page * auctionPage
}

func (a *App) updateAuction(in *platform.Input, w, h int) bool {
	v := &a.auction
	e := a.state.Entities[v.npc]
	if e == nil || a.state.Self.Dead || a.shop.npcID != v.npc {
		*v = auctionView{}
		return false
	}
	if a.auctionConfirm != nil && a.auctionConfirm.Visible {
		if result := a.auctionConfirm.Update(in, w, h); result != ui.MessageNone && v.confirm != "" {
			action := v.confirm
			v.confirm = ""
			if result == ui.MessageOK {
				a.auctionSubmit(action)
			}
		}
		return true
	}
	if v.noticeSerial != v.noticeSeen && a.auctionConfirm != nil {
		v.noticeSeen = v.noticeSerial
		if v.message != "" {
			a.auctionConfirm.OpenFull(v.message, false, a.font)
			return true
		}
	}
	dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
	if dx*dx+dz*dz > 100 || in.HasKey(platform.KeyEscape) && v.modal == "" && v.dropdown == 0 && v.focus < 0 {
		a.auctionClose()
		return true
	}
	if v.pending && a.clock >= v.nextRefresh {
		v.pending = false
		v.say(a.tr(12235))
		v.nextRefresh = a.clock + 15
	}
	if !v.pending && v.modal == "" && v.focus < 0 && v.dropdown == 0 && a.clock >= v.nextRefresh {
		a.requestAuction(proto.AuctionRequest{})
	}
	if v.focus >= 0 && !v.pending {
		if v.modal == "setup" || v.modal == "bid" {
			editAuctionField(&v.fields[v.focus], in, true)
			if in.HasKey(platform.KeyEnter) {
				a.confirmAuction()
			}
		} else if v.focus == 0 {
			editAuctionField(&v.query, in, false)
			if in.HasKey(platform.KeyEnter) {
				a.auctionSearch()
			}
		}
	}
	if in.HasKey(platform.KeyEscape) {
		if v.modal != "" {
			v.modal, v.focus = "", -1
			return true
		}
		v.dropdown, v.focus = 0, -1
	}
	ox, oy, _ := a.auctionPanel(w, h)
	rx, ry := in.MouseX-ox, in.MouseY-oy
	if v.modal != "" {
		a.updateAuctionDialog(in, w, h)
		return true
	}
	inside := rx >= 0 && ry >= 0 && rx < auctionWidth && ry < auctionPixels
	if !inside {
		return v.focus >= 0 && !in.Pressed && !in.Clicked && (len(in.Chars) > 0 || len(in.Keys) > 0)
	}
	controls := a.auctionControls(ox, oy)
	clicked := -1
	for i, c := range controls {
		c.b.Disabled = !c.on || v.pending && i != 0
		if c.b.Update(in) && clicked < 0 {
			clicked = i
		}
	}
	if clicked >= 0 {
		controls[clicked].do()
		return true
	}
	if v.pending {
		return true
	}
	a.updateAuctionLists(in, ox, oy, rx, ry)
	return true
}

func inRect(x, y, rx, ry, w, h int) bool { return x >= rx && y >= ry && x < rx+w && y < ry+h }

// updateAuctionLists handles scrollbars, wheel, row picking and the search bar.
func (a *App) updateAuctionLists(in *platform.Input, ox, oy, rx, ry int) {
	v := &a.auction
	switch v.view {
	case proto.AuctionBrowse:
		a.updateAuctionSearchBar(in, rx, ry)
		visible := 7
		last := max(0, len(v.rows)-visible)
		if s := a.auctionScroller("browse", 0x2c8, 0x7c, 0x16c); s != nil {
			v.scroll = max(0, min(last, s.Update(in, ox, oy, len(v.rows), visible, v.scroll)))
		}
		if in.Wheel != 0 && inRect(rx, ry, 0x9b, 0x7c, 0x22d, 0x16c) {
			switch {
			case in.Wheel > 0 && v.scroll > 0:
				v.scroll--
			case in.Wheel > 0:
				a.auctionPage(-1)
			case v.scroll < last:
				v.scroll++
			default:
				a.auctionPage(1)
			}
		}
		if in.Pressed && inRect(rx, ry, 0x9b, 0x7c, 0x22d, 0x16c) && v.dropdown == 0 {
			a.pickAuctionRow(proto.AuctionBrowse, v.scroll+(ry-0x7c)/auctionStride, 0x7c)
		}
	case proto.AuctionWatch:
		a.updateAuctionPaged(in, ox, oy, rx, ry, proto.AuctionWatch, "watchpage", 0x2c8, 0x7c, 0x1ad, 0x9b, 0x22d, 0x7c)
	default:
		first := auctionPair(v.view)
		a.updateAuctionPaged(in, ox, oy, rx, ry, first, "p0", 0x167, 0x7a, 0x1a3, 0xe, 0x159, 0x7f)
		a.updateAuctionPaged(in, ox, oy, rx, ry, first+1, "p1", 0x2cb, 0x7a, 0x1a3, 0x175, 0x153, 0x7f)
	}
	if v.view == proto.AuctionBrowse || v.view == proto.AuctionWatch {
		if in.Pressed && ry >= 0x62 && ry < 0x78 && rx >= 0xa0 && rx < 0x2c8 {
			sort := 1
			for i, x := range [...]int{0xa0, 0x180, 0x1bc, 0x250} {
				if rx >= x {
					sort = i + 1
				}
			}
			if v.sort == sort {
				sort = -sort
			}
			v.sort, v.page, v.scroll = sort, 0, 0
			a.requestAuction(proto.AuctionRequest{})
		}
	}
}

// updateAuctionPaged drives one 8-row list: its page scrollbar, wheel and row
// picking. Pages replace the scrolling the original does over the full list.
func (a *App) updateAuctionPaged(in *platform.Input, ox, oy, rx, ry, view int, id string, sx, sy, sh, left, width, top int) {
	v := &a.auction
	l := v.paneList(view)
	count, offset := pageScroll(l)
	if s := a.auctionScroller(id, sx, sy, sh); s != nil {
		if n := s.Update(in, ox, oy, count, auctionPage, offset); n != offset {
			a.auctionPageOf(view, l.page+map[bool]int{true: 1, false: -1}[n > offset])
			return
		}
	}
	if in.Wheel != 0 && inRect(rx, ry, left, sy, width, sh) {
		a.auctionPageOf(view, l.page+map[bool]int{true: -1, false: 1}[in.Wheel > 0])
		return
	}
	if in.Pressed && inRect(rx, ry, left, top, width, auctionPage*auctionStride) {
		a.pickAuctionRow(view, (ry-top)/auctionStride, top)
	}
}

func (a *App) auctionPageOf(view, page int) {
	v := &a.auction
	l := v.paneList(view)
	if page < 0 || page > l.page && !l.more {
		return
	}
	v.selectPane(view)
	v.page, v.selected, v.scroll = page, -1, 0
	a.requestAuction(proto.AuctionRequest{})
}

func (a *App) pickAuctionRow(view, index, top int) {
	v := &a.auction
	l := v.paneList(view)
	if index < 0 || index >= len(l.rows) {
		return
	}
	v.selectPane(view)
	double := v.clickView == view+1 && v.clickRow == index && a.clock-v.clickAt < .4
	v.selected = index
	v.clickView, v.clickRow, v.clickAt = view+1, index, a.clock
	if double {
		v.clickView = 0
		a.auctionAction("details")
	}
}

// updateAuctionSearchBar covers the Light/Fury checks, the three dropdowns,
// the text field and the category sub-list (0x455e40).
func (a *App) updateAuctionSearchBar(in *platform.Input, rx, ry int) {
	v := &a.auction
	if !in.Pressed {
		return
	}
	if v.dropdown != 0 {
		arrow := [3][2]int{{0xc1, 0x42}, {0x10a, 0x42}, {0x1b8, 0x42}}[v.dropdown-1]
		left := arrow[0] - 0x14
		if v.dropdown == 3 {
			left = arrow[0] - 0x78
		}
		if inRect(rx, ry, left, arrow[1]+0x14, arrow[0]+0x14-left, 0x78) {
			if i := (ry - (arrow[1] + 0x19)) / 0xf; ry >= arrow[1]+0x19 && i < 8 {
				switch v.dropdown {
				case 1:
					v.minIdx = i
				case 2:
					v.maxIdx = i
				default:
					v.grade = i
				}
				v.dropdown = 0
				return
			}
		}
	}
	v.dropdown = 0
	for i, x := range [2]int{0x23, 0x56} {
		if inRect(rx, ry, x, 0x43, 0x2f, 16) {
			v.unchecked[i] = !v.unchecked[i]
			return
		}
	}
	for i, r := range [3][4]int{{0xc1 - 0x14, 0x42, 0x28, 0x14}, {0x10a - 0x14, 0x42, 0x28, 0x14}, {0x1b8 - 100, 0x42, 0x78, 0x14}} {
		if inRect(rx, ry, r[0], r[1], r[2], r[3]) {
			v.dropdown, v.focus = i+1, -1
			return
		}
	}
	if inRect(rx, ry, 0x1d1, 0x40, 0xbe, 0x16) {
		v.focus = 0
		return
	}
	v.focus = -1
	if v.group > 0 {
		n := len(auctionGroups[v.group-1].gx)
		top := 0x84 + (v.group-1)*0x19 + 0x1c
		if inRect(rx, ry, 0x12, top, 0x82, n*0x16) {
			v.sub = (ry-top)/0x16 + 1
		}
	}
}

func auctionLeft(ends int64, now time.Time) (string, [4]float32) {
	left := ends - now.Unix()
	if left <= 0 {
		return "0", auctionRed
	}
	hours := left / 3600
	var s string
	switch {
	case hours >= 24:
		s = fmt.Sprintf("%dD", hours/24)
	case hours == 0:
		return fmt.Sprintf("%dm", left/60), auctionRed
	default:
		s = fmt.Sprintf("%dH", hours)
	}
	switch {
	case hours > 12:
		return s, auctionYellow
	case hours > 1:
		return s, auctionOrange
	}
	return s, auctionRed
}

func auctionRGB(c [3]uint8) [4]float32 {
	return [4]float32{float32(c[0]) / 255, float32(c[1]) / 255, float32(c[2]) / 255, 1}
}

func (a *App) auctionText(s string, x, y int, c [4]float32) {
	a.font.Draw(a.sprites, s, x, y, c, 0)
}

func (a *App) auctionRight(s string, right, y int, c [4]float32) {
	a.font.Draw(a.sprites, s, right-a.font.Width(s), y, c, 0)
}

func (a *App) auctionAmount(value uint32, right, y int, c [4]float32) {
	a.auctionRight(ui.GoldText(value), right, y, c)
}

func (a *App) auctionGoldAt(value uint32, x, y int) {
	a.auctionText(ui.GoldText(value), x, y, auctionRGB(ui.GoldRGB(value)))
}

func (a *App) auctionGold(value uint32, right, y int) {
	a.auctionAmount(value, right, y, auctionRGB(ui.GoldRGB(value)))
}

// fitText follows the 0x46/0x82 pixel name limits of the row routines.
func (a *App) fitText(s string, limit int) string {
	if a.font.Width(s) <= limit {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && a.font.Width(string(r)) > limit {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

func (a *App) auctionNameColor(r proto.AuctionRow) [4]float32 {
	if d := a.itemsItem(r.Type, r.TypeID); d != nil {
		return auctionRGB(ui.ItemNameRGB(d.NameGrade()))
	}
	return white
}

func (a *App) auctionIcon(r proto.AuctionRow, x, y int) {
	if d := a.itemsItem(r.Type, r.TypeID); d != nil {
		a.drawItemIcon(d, x, y, 32, white, &proto.InventoryItem{Type: r.Type, TypeID: r.TypeID, Count: r.Count, Quality: r.Quality, Gems: r.Gems, Craft: r.Craft})
		if a.lastMouseX >= x && a.lastMouseX < x+32 && a.lastMouseY >= y && a.lastMouseY < y+32 {
			a.auctionHover = &r
		}
	}
}

func (a *App) auctionLeftText(ends int64, x, y int) {
	s, c := auctionLeft(ends, time.Now())
	a.auctionText(s, x, y, c)
}

// auctionBids draws the "Bid now / Buy now" pair shared by every list.
func (a *App) auctionBids(r proto.AuctionRow, labelX, bidY, buyY, right int) {
	a.auctionText(a.msg.Get(7041), labelX, bidY, white)
	a.auctionText(a.msg.Get(7042), labelX, buyY, auctionRed)
	a.auctionGold(r.Price, right, bidY)
	a.auctionAmount(r.Buyout, right, buyY, auctionRed)
}

func (a *App) auctionLevel(r proto.AuctionRow) string {
	if r.Type == 30 {
		return "0"
	}
	return strconv.Itoa(int(r.Level))
}

func (a *App) drawAuctionBrowseRows(ox, oy int, rows []proto.AuctionRow, first, visible, selected, nameY int, markers bool) {
	for i := range visible {
		idx := first + i
		if idx >= len(rows) {
			return
		}
		r, by := rows[idx], oy+i*auctionStride
		if idx == selected {
			a.auctionBox(ox+0x9b, by+0x7c, 0x22d, 0x2e, auctionSelection)
		}
		a.auctionIcon(r, ox+0xa6, by+0x87)
		a.auctionText(r.Name, ox+0xdb, by+nameY, a.auctionNameColor(r))
		if markers && r.Mine {
			a.auctionText(a.msg.Get(7054), ox+0x168, by+0x9b, white)
		} else if markers && r.HighBidder {
			a.auctionText(a.msg.Get(7055), ox+0x168, by+0x9b, white)
		}
		a.auctionText(a.auctionLevel(r), ox+0x1b4, by+0x8e, white)
		a.auctionLeftText(r.Ends, ox+0x1fc, by+0x8e)
		a.auctionBids(r, ox+0x229, by+0x87, by+0x9b, ox+0x2c1)
	}
}

// drawAuctionPane draws one half of the progress tab: left selling list or
// right bidding list. Offsets are the 0x456ac0 constants.
func (a *App) drawAuctionPane(ox, oy int, l auctionList, right, active bool) {
	iconX, nameX, timeX, labelX, rightX, hl := 0x18, 0x3d, 0x99, 0xc4, 0x15e, [2]int{0xe, 0x159}
	bidY, buyY := 0x90, 0xa4
	if right {
		iconX, nameX, timeX, labelX, rightX, hl = 0x17c, 0x1a2, 0x200, 0x22b, 0x2c4, [2]int{0x175, 0x153}
		bidY, buyY = 0x8b, 0x9f
	}
	for i := range min(auctionPage, len(l.rows)) {
		r, by := l.rows[i], oy+i*auctionStride
		if active && i == l.selected {
			a.auctionBox(ox+hl[0], by+0x7f, hl[1], 0x2e, auctionSelection)
		}
		a.auctionIcon(r, ox+iconX, by+0x8b)
		a.auctionText(a.fitText(r.Name, 0x46), ox+nameX, by+0x94, a.auctionNameColor(r))
		a.auctionLeftText(r.Ends, ox+timeX, by+0x94)
		a.auctionBids(r, ox+labelX, by+bidY, by+buyY, ox+rightX)
	}
}

func auctionItemOutcome(reason string) (int, [4]float32) {
	switch reason {
	case "Expired":
		return 7049, auctionBlue
	case "Canceled":
		return 7056, auctionBlue
	}
	return 7046, auctionYellow
}

func (a *App) drawAuctionEnded(ox, oy int, items, gold auctionList, view int) {
	for i := range min(auctionPage, len(items.rows)) {
		r, by := items.rows[i], oy+i*auctionStride
		if view == proto.AuctionItems && i == items.selected {
			a.auctionBox(ox+0xe, by+0x7f, 0x159, 0x2e, auctionSelection)
		}
		msg, c := auctionItemOutcome(r.Reason)
		a.auctionText(a.msg.Get(msg), ox+0x24, by+0x94, c)
		a.auctionLeftText(r.Ends, ox+0x7b, by+0x94)
		a.auctionIcon(r, ox+0xaa, by+0x8b)
		a.auctionText(a.fitText(r.Name, 0x82), ox+0xce, by+0x94, a.auctionNameColor(r))
	}
	for i := range min(auctionPage, len(gold.rows)) {
		r, by := gold.rows[i], oy+i*auctionStride
		if view == proto.AuctionGold && i == gold.selected {
			a.auctionBox(ox+0x175, by+0x7f, 0x153, 0x2e, auctionSelection)
		}
		a.auctionLeftText(r.Ends, ox+0x1ea, by+0x94)
		if r.Reason == "Bid refund" {
			a.auctionText(a.msg.Get(7047), ox+0x19a, by+0x94, auctionBlue)
			a.auctionText(a.msg.Get(7043), ox+0x22b, by+0x94, white)
			a.auctionGold(r.Gold, ox+0x2c4, by+0x94)
			continue
		}
		a.auctionText(a.msg.Get(7048), ox+0x19a, by+0x94, auctionYellow)
		a.auctionText(a.msg.Get(7044), ox+0x22b, by+0x8b, white)
		a.auctionText(a.msg.Get(7045), ox+0x22b, by+0x9f, white)
		a.auctionGold(r.Price, ox+0x2c4, by+0x8b)
		a.auctionGold(r.Gold, ox+0x2c4, by+0x9f)
	}
}

func (a *App) drawAuctionArrow(x, y int, hover bool) {
	tex := a.auctionTex("option_graphic_resol")
	if tex == nil {
		return
	}
	col := 0
	if hover {
		col = 1
	}
	a.sprites.Draw(tex, float32(x), float32(y), 20, 20, float32(col*32)/float32(tex.W), 0, float32(col*32+20)/float32(tex.W), 20/float32(tex.H), white)
}

func (a *App) auctionGradeText(grade int) (string, [4]float32) {
	idx := grade
	switch grade {
	case 6:
		idx = 7
	case 7:
		idx = 10
	}
	return a.msg.Get(7000 + grade), auctionRGB(ui.ItemNameRGB(uint8(idx)))
}

func (a *App) drawAuctionSearchBar(ox, oy int) {
	v := &a.auction
	if tex := a.auctionTex("chat_check"); tex != nil {
		for i, x := range [2]int{0x23, 0x56} {
			if !v.unchecked[i] {
				a.sprites.Draw(tex, float32(ox+x), float32(oy+0x43), 16, 16, 0, 0, 1, 1, white)
			}
		}
	}
	hover := func(x, y, w, h int) bool { return inRect(a.lastMouseX-ox, a.lastMouseY-oy, x, y, w, h) }
	arrows := [3][2]int{{0xc1, 0x42}, {0x10a, 0x42}, {0x1b8, 0x42}}
	for i, p := range arrows {
		left, width := p[0]-0x14, 0x28
		if i == 2 {
			left, width = p[0]-100, 0x78
		}
		a.drawAuctionArrow(ox+p[0], oy+p[1], v.dropdown == i+1 || hover(left, p[1], width, 0x14))
	}
	a.auctionText(strconv.Itoa(v.minIdx*10), ox+0xaf, oy+0x46, white)
	a.auctionText(strconv.Itoa(v.maxIdx*10), ox+0xf8, oy+0x46, white)
	text, color := a.auctionGradeText(v.grade)
	a.auctionText(text, ox+0x154, oy+0x46, color)
	query := v.query
	if v.focus == 0 && int(a.clock*2)%2 == 0 {
		query += "_"
	}
	a.auctionText(query, ox+0x1d1, oy+0x47, white)
	if v.group > 0 {
		group := auctionGroups[v.group-1]
		top := oy + 0x84 + (v.group-1)*0x19
		for i := range len(group.gx) {
			c := white
			if v.sub == i+1 {
				c = [4]float32{1, 204.0 / 255, 51.0 / 255, 1}
				if tex := a.auctionTex("market_button"); tex != nil {
					a.sprites.Draw(tex, float32(ox+0x16), float32(top+0x16+i*0x16), 32, 32, .5, 0, 1, .25, white)
				}
			}
			a.auctionText(a.msg.Get(group.msg+i), ox+0x30, top+0x1c+i*0x16, c)
		}
	}
}

func (a *App) drawAuctionDropdown(ox, oy int) {
	v := &a.auction
	if v.dropdown == 0 {
		return
	}
	arrow := [3][2]int{{0xc1, 0x42}, {0x10a, 0x42}, {0x1b8, 0x42}}[v.dropdown-1]
	x, y := ox+arrow[0], oy+arrow[1]
	selected := [3]int{v.minIdx, v.maxIdx, v.grade}[v.dropdown-1]
	if v.dropdown == 3 {
		a.auctionBox(x-0x74, y+0x14, 0x86, 0x82, [4]float32{.28, .28, .28, .5})
	} else {
		a.auctionBox(x-0x1a, y+0x14, 0x2d, 0x82, [4]float32{.28, .28, .28, .5})
	}
	for i := range 8 {
		c := white
		if selected == i {
			c = [4]float32{1, 0, 1, 1}
		}
		switch {
		case v.dropdown == 3:
			text, grade := a.auctionGradeText(i)
			if selected != i {
				c = grade
			}
			a.auctionText(text, x-100, y+0x19+i*0xf, c)
		default:
			a.auctionText(strconv.Itoa(i*10), x-0x12, y+0x19+i*0xf, c)
		}
	}
}

func (a *App) drawAuction(w, h int) {
	v := &a.auction
	ox, oy, _ := a.auctionPanel(w, h)
	a.clearTooltipUnderPanel(ox, oy, auctionWidth, auctionPixels)
	a.auctionHover = nil
	tab := auctionTab(v.view)
	if tex := a.auctionTex(auctionFrames[tab]); tex != nil {
		a.sprites.Draw(tex, float32(ox), float32(oy), 1024, 1024, 0, 0, 1, 1, white)
	} else {
		a.auctionBox(ox, oy, auctionWidth, auctionPixels, [4]float32{.07, .06, .05, 1})
	}
	for _, c := range a.auctionControls(ox, oy) {
		c.b.Disabled = !c.on
		c.b.Draw(a.sprites)
	}
	money := a.state.Details.Money
	right := [4]int{0xe3, 0x1e0, 0x1e0, 0x1ce}[tab]
	a.auctionGold(money, ox+right, oy+0x23c)
	switch v.view {
	case proto.AuctionBrowse:
		a.drawAuctionSearchBar(ox, oy)
		a.drawAuctionBrowseRows(ox, oy, v.rows, v.scroll, 7, v.selected, 0x87, true)
		a.drawAuctionScroll("browse", 0x2c8, 0x7c, 0x16c, ox, oy, len(v.rows), 7, v.scroll)
	case proto.AuctionWatch:
		a.drawAuctionBrowseRows(ox, oy, v.rows, 0, auctionPage, v.selected, 0x8e, false)
		count, offset := pageScroll(v.list())
		a.drawAuctionScroll("watchpage", 0x2c8, 0x7c, 0x1ad, ox, oy, count, auctionPage, offset)
	case proto.AuctionSelling, proto.AuctionBidding:
		first := auctionPair(v.view)
		for i := range 2 {
			l := v.paneList(first + i)
			a.drawAuctionPane(ox, oy, l, i == 1, first+i == v.view)
			count, offset := pageScroll(l)
			a.drawAuctionScroll("p"+strconv.Itoa(i), [2]int{0x167, 0x2cb}[i], 0x7a, 0x1a3, ox, oy, count, auctionPage, offset)
		}
	default:
		items, gold := v.paneList(proto.AuctionItems), v.paneList(proto.AuctionGold)
		a.drawAuctionEnded(ox, oy, items, gold, v.view)
		for i, l := range [2]auctionList{items, gold} {
			count, offset := pageScroll(l)
			a.drawAuctionScroll("p"+strconv.Itoa(i), [2]int{0x167, 0x2cb}[i], 0x7a, 0x1a3, ox, oy, count, auctionPage, offset)
		}
	}
	a.drawAuctionDropdown(ox, oy)
	if v.modal != "" {
		a.drawAuctionDialog(w, h)
	}
	if r := a.auctionHover; r != nil && v.modal == "" {
		if d := a.itemsItem(r.Type, r.TypeID); d != nil {
			a.drawItemDetails(d, &proto.InventoryItem{Type: r.Type, TypeID: r.TypeID, Count: r.Count, Quality: r.Quality, Gems: r.Gems, Craft: r.Craft}, r.Count, a.lastMouseX+16, a.lastMouseY+16, w, h, "")
		}
	}
}

func (a *App) drawAuctionScroll(id string, x, y, height, ox, oy, count, visible, offset int) {
	if s := a.auctionScroller(id, x, y, height); s != nil {
		s.Draw(a.sprites, ox, oy, count, visible, offset)
	}
}

func (a *App) auctionBox(x, y, w, h int, c [4]float32) {
	a.sprites.Draw(a.white, float32(x), float32(y), float32(w), float32(h), 0, 0, 1, 1, c)
}
