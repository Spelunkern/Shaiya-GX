package app

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Native dialogs: bid 0x45bfa0/0x45b730, listing 0x45c9d0/0x45ba20 and
// details 0x45cfb0/0x45dbf0. Each is its own centred window.
type auctionBounds struct {
	x, y, width, height int
}

func (b auctionBounds) contains(x, y int) bool {
	return x >= b.x && y >= b.y && x < b.x+b.width && y < b.y+b.height
}

// source is the item slot of the listing dialog, drawn in its art.
func (b auctionBounds) source() auctionBounds {
	return auctionBounds{b.x + 24, b.y + 45, 34, 34}
}

func auctionDialogBox(modal string, w, h int) auctionBounds {
	if modal == "details" {
		return auctionBounds{w/2 - 0x80, h/2 - 0x100, 256, 512}
	}
	return auctionBounds{w/2 - 0x80, h/2 - 0x80, 256, 256}
}

var auctionDialogArt = map[string]string{"bid": "auction_tender", "setup": "auction_commission", "details": "auction_detailview"}

func (a *App) closeAuctionDialog() {
	v := &a.auction
	v.modal, v.focus, v.dropdown = "", -1, 0
}

func (a *App) auctionDialogControls(ox, oy int) []auctionControl {
	v := &a.auction
	var c []auctionControl
	closeX := map[string]int{"bid": 0xea, "setup": 0xe6, "details": 0xe8}[v.modal]
	c = append(c, auctionControl{a.colButton("dclose", "talk1_close_button", 22, ox, oy, closeX, 2), true, a.closeAuctionDialog})
	button := func(id, file string, col, x, y, hw, hh int, on bool, do func()) {
		c = append(c, auctionControl{a.rowButton(id, file, col, 128, 32, ox, oy, x, y, hw, hh, auctionRowsUV), on, do})
	}
	input := func(id string, y, field int) {
		c = append(c, auctionControl{a.rowButton(id, "auction_tender_enter_button", 0, 64, 32, ox, oy, 0xcd, y, 42, 20, auctionRowsUV), true, func() { v.focus = field }})
	}
	closeButton := func(x, y int) {
		button("dback", "auction_commission_button", 1, x, y, 84, 28, true, a.closeAuctionDialog)
	}
	r := v.row()
	switch v.modal {
	case "bid":
		button("dbid", "auction_objectsearch_button", 0, 0x1c, 0xb5, 84, 28, true, a.confirmAuction)
		button("dbuy", "auction_objectsearch_button", 1, 0x8c, 0xb5, 84, 28, v.actionAvailable("buy"), func() { a.closeAuctionDialog(); a.auctionAction("buy") })
		button("dwatch", "auction_objectsearch_button", 2, 0x1c, 0xd8, 84, 28, v.actionAvailable("watch"), func() { a.closeAuctionDialog(); a.auctionAction("watch") })
		closeButton(0x8c, 0xd8)
		input("dinput0", 0x8b, 0)
	case "setup":
		input("dinput1", 0x57, 1)
		input("dinput2", 0x71, 2)
		button("dstart", "auction_commission_button", 0, 0x1c, 0xd8, 84, 28, true, a.confirmAuction)
		closeButton(0x8c, 0xd8)
	default:
		on := r != nil
		pane := v.detail
		switch pane {
		case proto.AuctionBrowse, proto.AuctionWatch:
			button("dbid", "auction_objectsearch_button", 0, 0x1f, 0x1b5, 84, 28, on && v.actionAvailable("bid"), func() { v.modal = ""; v.focus = -1; a.auctionAction("bid") })
			button("dbuy", "auction_objectsearch_button", 1, 0x8d, 0x1b5, 84, 28, on && v.actionAvailable("buy"), func() { a.closeAuctionDialog(); a.auctionAction("buy") })
			if pane == proto.AuctionBrowse {
				button("dwatch", "auction_objectsearch_button", 2, 0x1f, 0x1d8, 84, 28, on && v.actionAvailable("watch"), func() { a.closeAuctionDialog(); a.auctionAction("watch") })
			} else {
				button("ddelete", "auction_interestlist_button", 2, 0x1f, 0x1d8, 84, 28, on, func() { a.closeAuctionDialog(); a.auctionAction("unwatch") })
			}
		case proto.AuctionBidding:
			button("dbuy", "auction_objectsearch_button", 1, 0x8d, 0x1b5, 84, 28, on && v.actionAvailable("buy"), func() { a.closeAuctionDialog(); a.auctionAction("buy") })
		case proto.AuctionItems:
			button("dget", "auction_completion_button", 0, 0x1f, 0x1d8, 84, 28, on, func() { a.closeAuctionDialog(); a.auctionAction("collect") })
		case proto.AuctionGold:
			button("dgold", "auction_completion_button", 2, 0x1f, 0x1d8, 84, 28, on, func() { a.closeAuctionDialog(); a.auctionAction("collect") })
		}
		closeButton(0x8d, 0x1d9)
	}
	return c
}

func (a *App) updateAuctionDialog(in *platform.Input, w, h int) {
	v := &a.auction
	if v.pending {
		return
	}
	b := auctionDialogBox(v.modal, w, h)
	ctrls := a.auctionDialogControls(b.x, b.y)
	clicked := -1
	for i, c := range ctrls {
		c.b.Disabled = !c.on
		if c.b.Update(in) && clicked < 0 {
			clicked = i
		}
	}
	if clicked >= 0 {
		ctrls[clicked].do()
		return
	}
	if v.modal == "setup" && in.Pressed {
		rx, ry := in.MouseX-b.x, in.MouseY-b.y
		if v.dropdown == 4 {
			v.dropdown = 0
			if inRect(rx, ry, 0x63, 0x9f, 0x66, 0x46) {
				if i := (ry - 0xa4) / 0xf; ry >= 0xa4 && i < 4 {
					v.duration = uint8(i)
				}
				return
			}
		}
		if inRect(rx, ry, 0xb5, 0x8b, 0x14, 0x14) {
			v.dropdown = 4
		}
	}
}

func auctionDuration(duration uint8, hour, day string) string {
	hours := proto.AuctionHours(duration)
	if hours < 25 {
		return fmt.Sprintf("%d %s(%dH)", hours, hour, hours)
	}
	return fmt.Sprintf("%d %s(%dD)", hours/24, day, hours/24)
}

// dialogValue is a 0x4ce790 number at (100, y): the typed digits while the
// field has focus, the grouped amount otherwise.
func (a *App) dialogValue(b auctionBounds, y int, text string, focused bool) {
	v, _ := strconv.ParseUint(text, 10, 32)
	if focused {
		if int(a.clock*2)%2 == 0 {
			text += "_"
		}
		a.auctionText(text, b.x+100, b.y+y, white)
		return
	}
	s := ui.GoldText(uint32(v))
	a.auctionText(s, b.x+100, b.y+y, auctionRGB(ui.GoldRGB(uint32(v))))
}

func (a *App) auctionDialogItem(b auctionBounds, typ, id uint8, inst *proto.InventoryItem) {
	d := a.itemsItem(typ, id)
	if d == nil {
		return
	}
	a.drawItemIcon(d, b.x+0x19, b.y+0x2e, 32, white, inst)
	a.auctionText(d.Name, b.x+0x3e, b.y+0x2e, auctionRGB(ui.ItemNameRGB(d.NameGrade())))
}

func (a *App) drawAuctionDialog(w, h int) {
	v := &a.auction
	b := auctionDialogBox(v.modal, w, h)
	if tex := a.auctionTex(auctionDialogArt[v.modal]); tex != nil {
		a.sprites.Draw(tex, float32(b.x), float32(b.y), float32(tex.W), float32(tex.H), 0, 0, 1, 1, white)
	} else {
		a.auctionBox(b.x, b.y, b.width, b.height, [4]float32{.07, .06, .05, 1})
	}
	r := v.row()
	switch v.modal {
	case "bid":
		if r != nil {
			a.auctionDialogItem(b, r.Type, r.TypeID, &proto.InventoryItem{Type: r.Type, TypeID: r.TypeID, Count: r.Count, Quality: r.Quality, Gems: r.Gems, Craft: r.Craft})
			a.auctionGoldAt(r.Price, b.x+100, b.y+0x5a)
			a.auctionText(ui.GoldText(r.Buyout), b.x+100, b.y+0x74, auctionRed)
		}
		a.dialogValue(b, 0x90, v.fields[0], v.focus == 0)
	case "setup":
		if v.source.Type != 0 {
			a.auctionDialogItem(b, v.source.Type, v.source.TypeID, &v.source)
		}
		a.dialogValue(b, 0x5a, v.fields[1], v.focus == 1)
		a.dialogValue(b, 0x78, v.fields[2], v.focus == 2)
		hour, day := a.msg.Get(7036), a.msg.Get(7037)
		a.auctionText(auctionDuration(v.duration, hour, day), b.x+100, b.y+0x90, white)
		start, _ := strconv.ParseUint(v.fields[1], 10, 32)
		a.auctionGoldAt(proto.AuctionDeposit(uint32(start), v.duration), b.x+100, b.y+0xac)
	default:
		if r != nil {
			a.drawAuctionDetails(b, *r, w, h)
		}
	}
	for _, c := range a.auctionDialogControls(b.x, b.y) {
		c.b.Disabled = !c.on
		c.b.Draw(a.sprites)
	}
	if v.modal == "setup" {
		a.drawAuctionArrow(b.x+0xb5, b.y+0x8b, inRect(a.lastMouseX-b.x, a.lastMouseY-b.y, 0xb5, 0x8b, 0x14, 0x14) || v.dropdown == 4)
		if v.dropdown == 4 {
			a.auctionBox(b.x+0x63, b.y+0x9f, 0x66, 0x46, [4]float32{.28, .28, .28, .5})
			for i := range 4 {
				c := white
				if int(v.duration) == i {
					c = [4]float32{1, 0, 1, 1}
				}
				a.auctionText(auctionDuration(uint8(i), a.msg.Get(7036), a.msg.Get(7037)), b.x+100, b.y+0xa4+i*0xf, c)
			}
		}
	}
}

// drawAuctionDetails draws 0x45dbf0: item, tooltip, sockets and the closing
// date and prices, by the pane that opened it.
func (a *App) drawAuctionDetails(b auctionBounds, r proto.AuctionRow, w, h int) {
	v := &a.auction
	inst := &proto.InventoryItem{Type: r.Type, TypeID: r.TypeID, Count: r.Count, Quality: r.Quality, Gems: r.Gems, Craft: r.Craft}
	def := a.itemsItem(r.Type, r.TypeID)
	if def != nil {
		a.drawItemIcon(def, b.x+0x70, b.y+0x20, 32, white, inst)
		// The shared tooltip carries the sockets and fits the 205px text box.
		a.drawItemDetails(def, inst, r.Count, b.x+0x17, b.y+0x53, w, h, "")
		a.drawSharedTooltipAt(w, h, &[2]int{b.x + 0x17, b.y + 0x53})
		a.setTooltip(tooltipContent{})
	}
	t := time.Unix(r.Ends, 0)
	date := fmt.Sprintf("%d/%d %d:%d", int(t.Month()), t.Day(), t.Hour(), t.Minute())
	_, color := auctionLeft(r.Ends, time.Now())
	label := a.msg.Get(7053)
	if v.detail == proto.AuctionItems || v.detail == proto.AuctionGold {
		label = a.msg.Get(7050)
	}
	a.auctionText(label, b.x+0xe, b.y+0x172, white)
	a.auctionText(date, b.x+0x5e, b.y+0x172, color)
	switch v.detail {
	case proto.AuctionItems:
	case proto.AuctionGold:
		a.auctionText(a.msg.Get(7045), b.x+0xe, b.y+0x181, white)
		a.auctionText(ui.GoldText(r.Gold), b.x+0x5e, b.y+0x181, auctionRGB(ui.GoldRGB(r.Gold)))
		if r.Reason != "Bid refund" {
			a.auctionText(a.msg.Get(7051), b.x+0xe, b.y+0x190, white)
			a.auctionText(ui.GoldText(r.Price/50), b.x+0x5e, b.y+0x190, auctionRGB(ui.GoldRGB(r.Price/50)))
		}
	default:
		a.auctionText(a.msg.Get(7041), b.x+0xe, b.y+0x181, white)
		a.auctionText(ui.GoldText(r.Price), b.x+0x5e, b.y+0x181, auctionRGB(ui.GoldRGB(r.Price)))
		a.auctionText(a.msg.Get(7042), b.x+0xe, b.y+0x190, white)
		a.auctionText(ui.GoldText(r.Buyout), b.x+0x5e, b.y+0x190, auctionRed)
	}
}
