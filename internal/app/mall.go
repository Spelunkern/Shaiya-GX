package app

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// The gift claim packet gives only two slots: the original item definition
// supplies the initial quality and maximum stack after server confirmation.
func (a *App) confirmedMall(packet []byte) {
	s := a.state
	if proto.Op(packet) == a.mall.pending {
		a.mall.pending = 0
	}
	switch proto.Op(packet) {
	case proto.CashPoint:
		a.questMessage(fmt.Sprintf("Item Mall points: %d", s.Mall.Points))
	case proto.PointBuy:
		result := s.Mall.LastPurchase
		messages := [...]string{"Item Mall purchase completed.", "Item Mall product not found.", "Not enough Item Mall points.", "Not enough inventory slots."}
		message := "Item Mall purchase failed."
		if int(result.Result) < len(messages) {
			message = messages[result.Result]
		}
		a.questMessage(message)
	case proto.PointGiftSend:
		result := s.Mall.LastGift
		messages := map[uint8]string{0: "Gift sent.", 1: "Character not found.", 2: "Cannot gift to the same account.", 3: "Gift box is full.", 4: "Not enough gift box slots.", 5: "Unable to store gift.", 6: "Product cannot be gifted.", 10: "Not enough Item Mall points."}
		message := messages[result.Result]
		if message == "" {
			message = "Gift failed."
		}
		a.questMessage(message)
	case proto.PointStoredItemGet:
		claim := s.Mall.PendingClaim
		if claim == nil {
			return
		}
		def := a.itemsItem(claim.Gift.Type, claim.Gift.TypeID)
		if def == nil {
			log.Printf("mall: missing item definition %d/%d after confirmed gift claim", claim.Gift.Type, claim.Gift.TypeID)
			a.questMessage("Gift claimed; reconnect to refresh inventory.")
			return
		}
		if s.ResolveMallClaim(min(def.Quality, 4999), max(def.Count, 1)) {
			a.questMessage("Gift received in inventory.")
		}
	}
}

// Native wire requests are exposed while the mall artwork is integrated.
// Purchases and claims wait for Rapture's replies before changing state.
func (a *App) mallCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/mall") {
		return false
	}
	if len(fields) == 1 {
		a.openMall()
		return true
	}
	var packet []byte
	if len(fields) >= 2 && strings.EqualFold(fields[1], "catalog") {
		page := 1
		if len(fields) > 3 {
			a.questMessage("Use /mall catalog [page].")
			return true
		}
		if len(fields) == 3 {
			var err error
			page, err = strconv.Atoi(fields[2])
			if err != nil || page < 1 {
				a.questMessage("Use /mall catalog [page].")
				return true
			}
		}
		if a.mallCatalog == nil {
			if err := a.loadMallCatalog(); err != nil {
				log.Printf("Item Mall catalog: %v", err)
				a.questMessage("Item Mall catalog unavailable.")
				return true
			}
		}
		const pageSize = 5
		pages := max(1, (len(a.mallCatalog)+pageSize-1)/pageSize)
		if page > pages {
			a.questMessage(fmt.Sprintf("Item Mall catalog has %d pages.", pages))
			return true
		}
		a.questMessage(fmt.Sprintf("Item Mall catalog %d/%d:", page, pages))
		for _, product := range a.mallCatalog[(page-1)*pageSize : min(page*pageSize, len(a.mallCatalog))] {
			a.questMessage(fmt.Sprintf("%s: %s (%d points)", product.Code, product.Name, product.Cost))
		}
		return true
	} else if len(fields) == 2 && strings.EqualFold(fields[1], "points") {
		packet = proto.MallPointsPacket()
	} else if len(fields) == 3 && strings.EqualFold(fields[1], "buy") && len(proto.EncodeText(fields[2])) <= 20 {
		packet = proto.MallBuyPacket(fields[2])
	} else if len(fields) == 4 && strings.EqualFold(fields[1], "gift") && len(proto.EncodeText(fields[2])) <= 20 && len(proto.EncodeText(fields[3])) <= 20 {
		packet = proto.MallGiftPacket(fields[2], fields[3])
	} else if len(fields) == 3 && strings.EqualFold(fields[1], "claim") {
		slot, err := strconv.ParseUint(fields[2], 10, 8)
		if err == nil && slot < 240 && a.state.Mall.PendingClaim == nil {
			if _, ok := a.state.Mall.Stored[uint8(slot)]; ok {
				packet = proto.MallClaimPacket(uint8(slot))
			}
		}
	} else if len(fields) == 2 && strings.EqualFold(fields[1], "gifts") {
		if !a.state.Mall.StoredReady {
			a.questMessage("Gift box is not available yet.")
			return true
		}
		if len(a.state.Mall.Stored) == 0 {
			a.questMessage("Gift box is empty.")
			return true
		}
		for slot := 0; slot < 240; slot++ {
			if gift, ok := a.state.Mall.Stored[uint8(slot)]; ok {
				name := fmt.Sprintf("%d/%d", gift.Type, gift.TypeID)
				if def := a.itemsItem(gift.Type, gift.TypeID); def != nil {
					name = def.Name
				}
				a.questMessage(fmt.Sprintf("Gift %d: %s x%d", slot, name, gift.Count))
			}
		}
		return true
	}
	if packet == nil {
		a.questMessage("Use /mall catalog [page], points, buy CODE, gift CODE NAME, gifts, or claim SLOT.")
		return true
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to send Item Mall request.")
	}
	return true
}
