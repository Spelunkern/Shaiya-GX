package app

import (
	"fmt"
	"log"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type mallView struct {
	open     bool
	products [4][]ui.MallProduct
	modal    ui.MallAction
	code     string
	target   ui.TextField
	pending  proto.Opcode
	deadline float64
}

func (a *App) loadMallCatalog() error {
	if a.mallCatalog != nil {
		return nil
	}
	b, err := a.arc.ReadFile("Cash.SData")
	if err != nil {
		return err
	}
	catalog, err := sdata.ParseCash(b)
	if err != nil {
		return err
	}
	a.mallCatalog = catalog
	return nil
}

func (a *App) openMall() {
	if a.mallUI == nil || a.state == nil || a.world == nil {
		return
	}
	if a.mall.open {
		a.mall.open = false
		return
	}
	if a.mall.pending != 0 {
		a.mall.open = true
		return
	}
	if err := a.prepareMall(); err != nil {
		log.Printf("Item Mall catalog: %v", err)
		a.questMessage("Item Mall catalog unavailable.")
		return
	}
	if err := a.world.Send(proto.MallPointsPacket()); err != nil {
		a.backToLogin("Unable to request Item Mall points.")
	}
}

func (a *App) prepareMall() error {
	if err := a.loadMallCatalog(); err != nil {
		return err
	}
	a.mall = mallView{open: true}
	a.mallUI.Reset()
	for _, p := range a.mallCatalog {
		if p.Bag == 0 {
			continue
		}
		category := min(int(p.Bag)-1, 3)
		product := ui.MallProduct{Name: p.Name, Code: p.Code, Description: p.Description, Cost: p.Cost}
		if a.assets != nil {
			if p.Icon == 0 {
				for _, entry := range p.Items {
					if entry.ItemID == 0 {
						continue
					}
					typ, id := uint8(entry.ItemID/1000), uint8(entry.ItemID%1000)
					if item := a.itemsItem(typ, id); item != nil {
						if file, uv, ok := ui.ItemIcon(typ, item.Icon); ok {
							product.Icon = a.assets.Texture("interface/" + file)
							product.UV = uv
						}
					}
					break
				}
			}
			if file, uv, ok := ui.MallIcon(p.Icon); ok {
				product.Icon = a.assets.Texture("interface/" + file)
				product.UV = uv
			}
		}
		// A product containing several different items has no one affinity.
		if a.assets != nil {
			entries := 0
			var definition *sdata.Item
			for _, entry := range p.Items {
				if entry.ItemID != 0 {
					entries++
					definition = a.itemsItem(uint8(entry.ItemID/1000), uint8(entry.ItemID%1000))
				}
			}
			if entries == 1 {
				product.Badge = a.itemBadge(definition, nil)
			}
		}
		a.mall.products[category] = append(a.mall.products[category], product)
	}
	return nil
}

func (a *App) updateMall(in *platform.Input, w, h int) bool {
	if !a.mall.open || a.mallUI == nil || a.state == nil {
		return false
	}
	if a.mall.pending != 0 && a.clock >= a.mall.deadline {
		a.mall.pending = 0
		a.questMessage("Item Mall request timed out.")
	}
	busy := a.mall.pending != 0
	if a.mallUI.GiftOpen {
		slot, _, captured := a.mallUI.UpdateGifts(in, w, h, a.state.Mall.Stored, a.clock, busy)
		if slot >= 0 && a.state.Mall.PendingClaim == nil {
			if err := a.world.Send(proto.MallClaimPacket(uint8(slot))); err != nil {
				a.backToLogin("Unable to claim gift.")
			} else {
				a.mall.pending, a.mall.deadline = proto.PointStoredItemGet, a.clock+10
			}
		}
		if !captured {
			x, y := a.mallUI.Origin(w, h)
			captured = panelPointer(in, x, y, ui.MallWidth, ui.MallHeight)
		}
		return captured
	}
	products := a.mall.products[a.mallUI.Category]
	action, captured := a.mallUI.Update(in, w, h, len(products), busy)
	switch action {
	case ui.MallClose:
		a.mall.open = false
	case ui.MallBuy, ui.MallGift:
		products = a.mall.products[a.mallUI.Category]
		if a.mallUI.Selected >= 0 && a.mallUI.Selected < len(products) && a.mallConfirm != nil {
			p := products[a.mallUI.Selected]
			a.mall.modal, a.mall.code = action, p.Code
			if action == ui.MallBuy {
				a.mallConfirm.Open(fmt.Sprintf("Buy %s for %d points?", p.Name, p.Cost), true)
			} else {
				a.mall.target = ui.TextField{Max: 20}
				a.mallConfirm.Open(fmt.Sprintf("Gift %s to character:", p.Name), true)
			}
		}
	}
	return captured
}

func (a *App) updateMallConfirm(in *platform.Input, w, h int) {
	if a.mallConfirm == nil || !a.mallConfirm.Visible {
		return
	}
	if a.mall.modal == ui.MallGift {
		a.mall.target.Edit(in)
	}
	answer := a.mallConfirm.Update(in, w, h)
	if answer == ui.MessageNone {
		return
	}
	if answer == ui.MessageOK {
		var packet []byte
		var op proto.Opcode
		if a.mall.modal == ui.MallBuy {
			packet, op = proto.MallBuyPacket(a.mall.code), proto.PointBuy
		} else if target := strings.TrimSpace(a.mall.target.Text); target != "" {
			packet, op = proto.MallGiftPacket(a.mall.code, target), proto.PointGiftSend
		} else {
			a.questMessage("Enter a character name for the gift.")
		}
		if packet != nil {
			if err := a.world.Send(packet); err != nil {
				a.backToLogin("Unable to send Item Mall request.")
			} else {
				a.mall.pending, a.mall.deadline = op, a.clock+10
			}
		}
	}
	a.mall.modal, a.mall.code = ui.MallNone, ""
}

func (a *App) drawMall(w, h int) {
	if !a.mall.open || a.mallUI == nil || a.state == nil {
		return
	}
	identity := ui.MallIdentity{}
	if slot, ok := a.selfSlot(); ok {
		identity.Name = slot.Name
		classes := [2][6]string{{"Fighter", "Defender", "Ranger", "Archer", "Mage", "Priest"}, {"Warrior", "Guardian", "Assassin", "Hunter", "Pagan", "Oracle"}}
		if slot.Class < 6 {
			faction := min(int(a.state.Faction.Faction), 1)
			identity.Class = classes[faction][slot.Class]
		}
		if slot.Mode < 4 {
			identity.Mode = [4]string{"Easy", "Normal", "Hard", "Ultimate"}[slot.Mode]
		}
	}
	if a.servers != nil && a.servers.Selected >= 0 && a.servers.Selected < len(a.servers.Servers) {
		identity.Server = a.servers.Servers[a.servers.Selected].Name
	}
	a.mallUI.Draw(a.sprites, a.font, w, h, a.mall.products[a.mallUI.Category], identity, a.state.Mall.Points, a.state.Mall.PointsReady, a.mall.pending != 0)
	if a.mallUI.GiftOpen {
		a.mallUI.DrawGifts(a.sprites, a.font, w, h, a.state.Mall.Stored, func(it proto.MallStoredItem) (*render.Texture, ui.UV, bool) {
			def := a.itemsItem(it.Type, it.TypeID)
			if def == nil {
				return nil, ui.UV{}, false
			}
			file, uv, ok := ui.ItemIcon(it.Type, def.Icon)
			if !ok {
				return nil, ui.UV{}, false
			}
			return a.assets.Texture("interface/" + file), uv, true
		}, func(it proto.MallStoredItem) ui.ItemBadge { return a.itemBadge(a.itemsItem(it.Type, it.TypeID), nil) })
	} else {
		products := a.mall.products[a.mallUI.Category]
		if index := a.mallUI.ProductAt(w, h, a.lastMouseX, a.lastMouseY, len(products)); index >= 0 {
			a.drawMallTooltip(products[index], w, h)
		}
	}
}

func (a *App) drawMallTooltip(p ui.MallProduct, w, h int) {
	if p.Description == "" || a.white == nil {
		return
	}
	lines := []string{p.Name}
	line := ""
	for _, word := range strings.Fields(p.Description) {
		next := word
		if line != "" {
			next = line + " " + word
		}
		if line != "" && a.font.Width(next) > 270 {
			lines = append(lines, line)
			line = word
		} else {
			line = next
		}
		if len(lines) >= 7 {
			break
		}
	}
	if line != "" && len(lines) < 8 {
		lines = append(lines, line)
	}
	height := 12 + 15*len(lines)
	left := max(0, min(a.lastMouseX+16, w-286))
	top := max(0, min(a.lastMouseY+16, h-height))
	a.sprites.Draw(a.white, float32(left), float32(top), 286, float32(height), 0, 0, 1, 1, [4]float32{.03, .03, .03, .96})
	for i, text := range lines {
		color := white
		if i == 0 {
			color = [4]float32{.95, .8, .45, 1}
		}
		a.font.Draw(a.sprites, text, left+8, top+6+i*15, color, 270)
	}
}
func (a *App) drawMallConfirm(w, h int) {
	if a.mallConfirm == nil || !a.mallConfirm.Visible {
		return
	}
	a.mallConfirm.Draw(a.sprites, a.font, w, h)
	if a.mall.modal == ui.MallGift {
		a.font.Draw(a.sprites, a.mall.target.Text+"_", w/2-100, h/2-35, white, 200)
	}
}
