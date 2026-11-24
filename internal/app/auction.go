package app

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type auctionList struct {
	page, selected int
	rows           []proto.AuctionRow
	more           bool
}

type auctionView struct {
	position                       ui.WindowPosition
	npc                            uint32
	sequence                       uint64
	sort                           int
	view, page, category, selected int
	exactType, exactID             uint8
	query, message, modal          string
	minLevel, maxLevel             string
	fields                         [3]string
	focus                          int
	duration                       uint8
	durationOpen                   bool
	source                         proto.InventoryItem
	token                          string
	rows                           []proto.AuctionRow
	more, pending                  bool
	nextRefresh                    float64
	subcategory                    int
	searchFaction                  uint8
	searchGrade                    *uint8
	other                          auctionList
	requestView                    int

	// Native EP5.4 controls (0x454590 search bar, 0x45a270 dispatcher).
	scroll                   int     // first visible row of the current page
	group, sub               int     // expanded category 1..5 and its entry 1..n; 0 none
	minIdx, maxIdx, grade    int     // dropdown indexes: level steps of 10, grade 7 = all
	unchecked                [2]bool // Light / Fury filters, both checked by default
	dropdown                 int     // open dropdown: 0 none, 1 min level, 2 max level, 3 grade
	confirm                  string  // action waiting for the native confirmation box
	detail                   int     // pane that opened the details window
	clickView, clickRow      int
	clickAt                  float64
	noticeSerial, noticeSeen int
}

func (v *auctionView) say(text string) {
	v.message = text
	if text != "" {
		v.noticeSerial++
	}
}

func (a *App) auctionBoard(npc uint32) bool {
	e := a.state.Entities[npc]
	return e != nil && e.NpcType == 1 && a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}].MerchantType == proto.AuctionBoard
}

func (a *App) requestAuction(req proto.AuctionRequest) {
	a.sendAuction(req, a.auction.view, a.auction.page)
}

func (a *App) sendAuction(req proto.AuctionRequest, view, page int) {
	v := &a.auction
	if a.world == nil || v.pending || v.npc == 0 {
		return
	}
	req.NPC, req.View, req.Page = v.npc, view, page
	a.auctionSequence++
	v.sequence = a.auctionSequence
	req.Sequence, req.Sort = v.sequence, v.sort
	if view == proto.AuctionBrowse {
		req.Category, req.Subcategory, req.Query = v.category, v.subcategory, v.query
		req.Faction, req.NameGrade = v.searchFaction, v.searchGrade
		minLevel, _ := strconv.Atoi(v.minLevel)
		maxLevel, _ := strconv.Atoi(v.maxLevel)
		if minLevel > 65535 || maxLevel > 65535 || maxLevel > 0 && minLevel > maxLevel {
			v.say(a.tr(12233))
			return
		}
		req.MinLevel, req.MaxLevel = minLevel, maxLevel
	}
	if req.Action != "post" {
		req.ItemType, req.ItemID = v.exactType, v.exactID
	}
	if packet := proto.AuctionPacket(req); packet != nil {
		if err := a.world.Send(packet); err != nil {
			a.backToLoginID(12010)
			return
		}
		v.pending = true
		v.requestView = view
		v.nextRefresh = a.clock + 10
	}
}

func (a *App) applyAuction(p []byte) {
	var r proto.AuctionReply
	v := &a.auction
	if proto.DecodeAuction(p, 12002, &r) != nil || !v.pending || v.npc == 0 || r.Sequence != v.sequence {
		return
	}
	v.pending = false
	v.nextRefresh = a.clock + 15
	if r.NPC == 0 {
		v.say(r.Error)
		v.nextRefresh = a.clock + 30
		return
	}
	if r.NPC != v.npc || r.View != v.requestView {
		return
	}
	if r.View != v.view {
		v.other.replace(r)
		if r.Error != "" {
			v.say(r.Error)
		}
		return
	}
	list := v.list()
	list.replace(r)
	v.setList(list)
	v.scroll = 0
	v.say(r.Message)
	if r.Error != "" {
		v.say(r.Error)
		return
	}
	if other := v.otherView(); other >= 0 {
		a.sendAuction(proto.AuctionRequest{}, other, v.other.page)
	}
}

func (l *auctionList) replace(r proto.AuctionReply) {
	var id int64
	if l.selected >= 0 && l.selected < len(l.rows) {
		id = l.rows[l.selected].ID
	}
	l.page, l.more, l.rows, l.selected = r.Page, r.More, r.Rows, -1
	for i, row := range l.rows {
		if id != 0 && row.ID == id {
			l.selected = i
			break
		}
	}
}

func (v *auctionView) list() auctionList {
	return auctionList{page: v.page, selected: v.selected, rows: v.rows, more: v.more}
}

func (v *auctionView) setList(l auctionList) {
	v.page, v.selected, v.rows, v.more = l.page, l.selected, l.rows, l.more
}

func (v *auctionView) otherView() int {
	first := auctionPair(v.view)
	if first < 0 {
		return -1
	}
	return first + (first + 1 - v.view)
}

func (v *auctionView) selectPane(view int) {
	if view != v.view {
		list := v.list()
		v.view = view
		v.setList(v.other)
		v.other = list
	}
}

func (a *App) beginAuction(npc uint32) {
	a.auction = auctionView{npc: npc, selected: -1, focus: -1, grade: 7}
	a.requestAuction(proto.AuctionRequest{})
}

func (v *auctionView) row() *proto.AuctionRow {
	if v.selected < 0 || v.selected >= len(v.rows) {
		return nil
	}
	return &v.rows[v.selected]
}

func (v *auctionView) actionAvailable(action string) bool {
	r := v.row()
	switch action {
	case "setup", "refresh":
		return true
	case "collectall", "clearwatch":
		return len(v.rows) > 0 || v.page > 0
	case "buy", "bid":
		return r != nil && r.Active && !r.Mine && (action == "buy" || !r.HighBidder)
	case "cancel":
		return r != nil && r.Active && r.Mine
	case "watch":
		return r != nil && r.Active
	default:
		return r != nil
	}
}

func (a *App) auctionAction(action string) {
	v := &a.auction
	if !v.actionAvailable(action) {
		return
	}
	r := v.row()
	switch action {
	case "setup":
		v.modal, v.focus, v.durationOpen = "setup", -1, false
		v.fields = [3]string{"", "500", "1000"}
		v.message = ""
		v.source = proto.InventoryItem{}
		a.inventory.open = true
	case "refresh", "collectall":
		if action == "refresh" {
			action = ""
		}
		a.requestAuction(proto.AuctionRequest{Action: action})
	case "buy", "cancel", "clearwatch":
		a.auctionAsk(action)
	case "exact":
		if r == nil {
			return
		}
		v.exactType, v.exactID, v.page, v.category = r.Type, r.TypeID, 0, 0
		v.query, v.minLevel, v.maxLevel = "", "", ""
		v.minIdx, v.maxIdx, v.grade = 0, 0, 7
		v.subcategory = 0
		v.searchFaction, v.searchGrade = proto.AuctionAnyFaction, nil
		a.requestAuction(proto.AuctionRequest{})
	case "details", "bid":
		v.modal, v.focus, v.detail = action, -1, v.view
		v.message = ""
		if action == "bid" {
			v.fields[0] = strconv.FormatUint(uint64(r.Price)*110/100, 10)
		}
	default:
		if r != nil {
			a.requestAuction(proto.AuctionRequest{Action: action, ID: r.ID, Version: r.Version})
		}
	}
}

// auctionAsk opens the native type-1 confirmation (0x4c4d10) with the
// SysMsg text used by 0x45a270 cases 0xc, 0xf and 0x16.
func (a *App) auctionAsk(action string) {
	v := &a.auction
	if a.auctionConfirm == nil {
		return
	}
	id, text := 7110, ""
	switch action {
	case "cancel":
		id = 7118
	case "clearwatch":
		id = 7129
	}
	text = strings.ReplaceAll(a.msg.Get(id), `\n`, "\n")
	if r := v.row(); r != nil && action == "buy" {
		text = strings.NewReplacer("<i>", r.Name, "<b>", ui.GoldText(r.Buyout)).Replace(text)
	}
	if text == "" {
		text = map[string]string{"buy": "Buy this item now?", "cancel": "Cancel this sale?", "clearwatch": "Erase the interest list?"}[action]
	}
	v.confirm = action
	a.auctionConfirm.OpenFull(text, true, a.font)
}

// auctionSubmit sends the actions that need the selected listing only.
func (a *App) auctionSubmit(action string) {
	v := &a.auction
	if action == "clearwatch" {
		a.requestAuction(proto.AuctionRequest{Action: action})
		return
	}
	r := v.row()
	if r == nil || !v.actionAvailable(action) {
		return
	}
	req := proto.AuctionRequest{Action: action, ID: r.ID, Version: r.Version}
	if action == "buy" {
		req.Amount = r.Buyout
	}
	a.requestAuction(req)
}

func (a *App) confirmAuction() {
	v := &a.auction
	req := proto.AuctionRequest{Action: v.modal}
	if v.modal == "setup" {
		count, _ := strconv.ParseUint(v.fields[0], 10, 16)
		start, _ := strconv.ParseUint(v.fields[1], 10, 32)
		buy, _ := strconv.ParseUint(v.fields[2], 10, 32)
		source, ok := a.state.Inventory[[2]uint8{v.source.Bag, v.source.Slot}]
		if !ok || source != v.source || count == 0 || count > uint64(source.Count) || start < 500 || start > 3800000000 || buy <= start || buy > 4000000000 {
			v.say(strings.ReplaceAll(a.msg.Get(7100), `\n`, " "))
			return
		}
		req.Action, req.Count, req.Start, req.Buyout = "post", uint16(count), uint32(start), uint32(buy)
		req.Item, req.Duration, req.Source = &source, v.duration, v.token
	} else {
		r := v.row()
		if r == nil || v.modal == "details" {
			a.closeAuctionDialog()
			return
		}
		req.ID, req.Version = r.ID, r.Version
		if req.Action == "bid" {
			amount, _ := strconv.ParseUint(v.fields[0], 10, 32)
			if amount < uint64(r.Price)*110/100 || amount >= uint64(r.Buyout) {
				v.say(a.msg.Get(7104))
				return
			}
			req.Amount = uint32(amount)
		}
	}
	a.closeAuctionDialog()
	a.requestAuction(req)
}

func (a *App) auctionSource() bool {
	v := &a.auction
	key := a.inventory.source
	item, ok := a.state.Inventory[key]
	if !ok || key[0] == 0 || a.inventory.pending {
		return false
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return false
	}
	v.source, v.token = item, fmt.Sprintf("%x", nonce)
	v.fields[0] = strconv.Itoa(int(item.Count))
	a.inventory.cancelItemPointer()
	return true
}

func editAuctionField(text *string, in *platform.Input, numeric bool) {
	if in.HasKey(platform.KeyDelete) {
		*text = ""
	}
	if in.HasKey(platform.KeyBackspace) && len(*text) > 0 {
		_, n := utf8.DecodeLastRuneInString(*text)
		*text = (*text)[:len(*text)-n]
	}
	limit := 31 // 0x6380 holds a 32-byte search string.
	if numeric {
		limit = 10
	}
	for _, r := range in.Chars {
		if len(*text)+utf8.RuneLen(r) <= limit && (numeric && r >= '0' && r <= '9' || !numeric && unicode.IsPrint(r)) {
			*text += string(r)
		}
	}
}

func auctionPair(view int) int {
	switch view {
	case proto.AuctionSelling, proto.AuctionBidding:
		return proto.AuctionSelling
	case proto.AuctionItems, proto.AuctionGold:
		return proto.AuctionItems
	default:
		return -1
	}
}
