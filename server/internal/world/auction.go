package world

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func (s *Session) auctionNPC(id uint32) bool {
	if id == 0 || id != s.usingNpc || s.char == nil || (s.dead || s.char.CurrentHP == 0) {
		return false
	}
	n, ok := npcOnMap(s.char.MapID, id)
	if !ok || n.Type != npcTypeShop {
		return false
	}
	def := findNpcDef(n.Type, n.TypeID)
	dx, dy, dz := n.PosX-s.char.PosX, n.PosY-s.char.PosY, n.PosZ-s.char.PosZ
	return def != nil && def.ShopType == proto.AuctionBoard && dx*dx+dy*dy+dz*dz <= 100
}

func (s *Session) auctionReply(reply proto.AuctionReply) {
	if frame := proto.AuctionPacket(reply); frame != nil {
		w := packet.New(packet.Opcode(proto.GXAuction))
		w.Bytes(frame[2:])
		s.send(w.MustFinish(), "GX auction")
	}
}

func (s *Session) handleAuction(p []byte) {
	var req proto.AuctionRequest
	if proto.DecodeAuction(p, 1024, &req) != nil || req.Page < 0 || req.Page > 10000 || !req.ValidSearchFilters() ||
		req.Sort < -4 || req.Sort > 4 || req.MinLevel < 0 || req.MaxLevel < 0 || req.MinLevel > 65535 || req.MaxLevel > 65535 || req.MaxLevel > 0 && req.MinLevel > req.MaxLevel ||
		req.View < proto.AuctionBrowse || req.View > proto.AuctionWatch || len(req.Query) > 80 || len(req.Source) > 64 {
		return
	}
	if !s.auctionContact || !s.auctionNPC(req.NPC) {
		s.auctionReply(proto.AuctionReply{Sequence: req.Sequence, Error: "Approach an auction board to use the market."})
		return
	}
	reply := proto.AuctionReply{Sequence: req.Sequence, NPC: req.NPC, View: req.View, Page: req.Page}
	if req.Action != "" && (tradeFor(s.char.ID) != nil || s.teleportStoneDueling()) {
		reply.Error = "Finish your trade before using the market."
		s.auctionReply(reply)
		return
	}
	var err error
	var message string
	switch req.Action {
	case "":
	case "post":
		err = s.postAuction(req)
	case "bid", "buy":
		err = s.payAuction(req)
	case "collect", "collectall":
		message, err = s.collectAuction(req)
	case "cancel", "watch", "unwatch", "clearwatch":
		err = db.ChangeAuction(s.char.ID, req, nil)
	default:
		err = db.ErrAuctionInvalid
	}
	actionErr := err
	req.Query = strings.TrimSpace(req.Query)
	reply, err = db.ListAuctions(s.char.ID, req)
	if err == nil && reply.Page > 0 && len(reply.Rows) == 0 {
		req.Page--
		reply, err = db.ListAuctions(s.char.ID, req)
	}
	if err != nil {
		actionErr = err
	}
	if message == "" && actionErr == nil && req.Action != "" {
		message = "Market updated."
	}
	reply.Message = message
	if actionErr != nil {
		reply.Error = "The market could not complete this request."
		switch {
		case errors.Is(actionErr, db.ErrAuctionInvalid), errors.Is(actionErr, db.ErrAuctionWatchFull), errors.Is(actionErr, db.ErrAuctionChanged), errors.Is(actionErr, db.ErrAuctionFunds), errors.Is(actionErr, db.ErrAuctionItem):
			reply.Error = actionErr.Error()
		default:
			s.logErr("auction", actionErr)
		}
	}
	s.auctionReply(reply)
}

func (s *Session) postAuction(req proto.AuctionRequest) error {
	if req.Item == nil {
		return db.ErrAuctionInvalid
	}
	i := findItem(s.items, req.Item.Bag, req.Item.Slot)
	if i < 0 || req.Item.Bag == 0 || req.Count == 0 || req.Source == "" || req.Start < 500 || req.Start > 3800000000 ||
		req.Buyout <= req.Start || req.Buyout > 4000000000 || proto.AuctionHours(req.Duration) == 0 {
		return db.ErrAuctionInvalid
	}
	item := s.items[i]
	def := db.MustDefs().Item(item.Type, item.TypeID)
	source := proto.InventoryItem{Bag: item.Bag, Slot: item.Slot, Type: item.Type, TypeID: item.TypeID, Quality: item.Quality, Gems: item.Gems, Count: item.Count, Craft: item.CraftName}
	if *req.Item != source || req.Count > item.Count {
		return db.ErrAuctionChanged
	}
	if def == nil || def.ReqOg != 0 || realType(item.Type) == 27 || req.Count > def.MaxStack() {
		return db.ErrAuctionItem
	}
	deposit := proto.AuctionDeposit(req.Start, req.Duration)
	if deposit > s.char.Money {
		return db.ErrAuctionFunds
	}
	item.Count = req.Count
	whole := item.Count == s.items[i].Count
	if !whole {
		item.UID = db.NewItemUID()
	}
	listing := db.AuctionListing{
		Seller: s.char.ID, SellerName: s.char.Name, Name: def.Name, Item: item, Request: req.Source,
		Category: proto.AuctionCategory(realType(item.Type), item.Type), Level: def.ReqLevel,
		Price: req.Start, Buyout: req.Buyout, Deposit: deposit, Ends: time.Now().Unix() + 3600*proto.AuctionHours(req.Duration),
	}
	var err error
	if !s.inventoryOperationWithCommit(func() {
		if whole {
			s.items = slices.Delete(s.items, i, i+1)
		} else {
			s.items[i].Count -= item.Count
		}
		left := item
		if !whole {
			left = s.items[i]
		}
		s.char.Money -= deposit
		s.send(buildRemoveItem(&left, whole), "auction item escrow")
		s.send(buildSetMoney(s.char.Money), "auction deposit")
	}, func(change db.CharacterChange) error {
		err = db.PostAuction(listing, change)
		return err
	}) && err == nil {
		return db.ErrAuctionInvalid
	}
	return err
}

func (s *Session) payAuction(req proto.AuctionRequest) error {
	if req.Amount == 0 {
		return db.ErrAuctionInvalid
	}
	if req.Amount > s.char.Money {
		return db.ErrAuctionFunds
	}
	var err error
	if !s.inventoryOperationWithCommit(func() {
		s.char.Money -= req.Amount
		s.send(buildSetMoney(s.char.Money), "auction payment")
	}, func(change db.CharacterChange) error {
		err = db.ChangeAuction(s.char.ID, req, &change)
		return err
	}) && err == nil {
		return db.ErrAuctionInvalid
	}
	return err
}

func (s *Session) collectAuction(req proto.AuctionRequest) (string, error) {
	if req.View != proto.AuctionItems && req.View != proto.AuctionGold || req.Action == "collect" && req.ID <= 0 {
		return "", db.ErrAuctionInvalid
	}
	id := req.ID
	if req.Action == "collectall" {
		id = 0
	}
	claims, err := db.AuctionClaims(s.char.ID, req.View, id)
	if err != nil {
		return "", err
	}
	more := len(claims) > 100
	claims = claims[:min(len(claims), 100)]
	done := 0
	for _, claim := range claims {
		var pos [2]uint8
		var mergeLimit uint16
		if claim.Gold == 0 {
			free := freeInventorySlots(s.items)
			if len(free) == 0 {
				mergeLimit = maxStackOf(claim.Item.Type, claim.Item.TypeID)
				if !wouldFit(s.items, &claim.Item, claim.Item.Count, mergeLimit) {
					continue
				}
			} else {
				pos = free[0]
			}
		} else if uint64(s.char.Money)+uint64(claim.Gold) > db.MoneyCap {
			continue
		}
		if s.inventoryOperationWithCommit(func() {
			if mergeLimit != 0 {
				packets, _ := planGiveItem(&s.items, &claim.Item, claim.Item.Count, mergeLimit)
				for _, p := range packets {
					s.send(p, "auction item merged")
				}
			} else if claim.Gold == 0 {
				item := claim.Item
				item.Bag, item.Slot = pos[0], pos[1]
				s.items = append(s.items, item)
				s.send(buildAddItem(&item), "auction item claimed")
			} else {
				s.char.Money += claim.Gold
				s.send(buildSetMoney(s.char.Money), "auction gold claimed")
			}
		}, func(change db.CharacterChange) error {
			return db.CollectAuction(claim, change)
		}) {
			done++
		}
	}
	if done < len(claims) {
		return fmt.Sprintf("Collected %d of %d. Remaining claims are available.", done, len(claims)), nil
	}
	if more {
		return fmt.Sprintf("Collected %d. More claims are available.", done), nil
	}
	if done == 0 {
		return "No available claims.", nil
	}
	return fmt.Sprintf("Collected %d.", done), nil
}
