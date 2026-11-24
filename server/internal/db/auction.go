package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

var (
	ErrAuctionChanged   = errors.New("The auction changed. Refresh and try again.")
	ErrAuctionInvalid   = errors.New("This auction action is unavailable.")
	ErrAuctionFunds     = errors.New("You do not have enough gold.")
	ErrAuctionItem      = errors.New("This item cannot be listed.")
	ErrAuctionWatchFull = errors.New("Your interest list is full.")
)

type AuctionListing struct {
	Seller, Bidder                   uint32
	SellerName, Name, State, Request string
	Category                         int
	Level                            uint16
	Item                             Item
	ID, Version, Ends                int64
	Price, Buyout, Deposit           uint32
}

type AuctionClaim struct {
	ID   int64
	Item Item
	Gold uint32
}

func auctionTx(run func(context.Context, *sql.Tx) error) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error { return run(c, tx) })
}

func auctionClaim(c context.Context, tx *sql.Tx, id int64, owner uint32, gold int64, reason string, now int64) error {
	_, err := tx.ExecContext(c, `INSERT INTO GX_AuctionClaims(Auction,Owner,Gold,Reason,Ends) VALUES(?,?,?,?,?)`, id, owner, gold, reason, now+14*86400)
	return err
}

func settleAuction(c context.Context, tx *sql.Tx, a AuctionListing, buyer uint32, price uint32, state string, now int64) error {
	owner, reason := buyer, state
	if buyer == 0 {
		owner = a.Seller
	}
	if err := auctionClaim(c, tx, a.ID, owner, 0, reason, now); err != nil {
		return err
	}
	if buyer != 0 {
		net := int64(price) + int64(a.Deposit) - int64(price)/50
		if err := auctionClaim(c, tx, a.ID, a.Seller, net, state, now); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(c, `UPDATE GX_Auctions SET State=?,Version=Version+1 WHERE ID=? AND State='active' AND Version=?`, state, a.ID, a.Version)
	if err != nil {
		return err
	}
	return requireOne(result)
}

func auctionRefund(c context.Context, tx *sql.Tx, a AuctionListing, now int64) error {
	if a.Bidder == 0 {
		return nil
	}
	return auctionClaim(c, tx, a.ID, a.Bidder, int64(a.Price), "Bid refund", now)
}

const auctionFields = "ID,Seller,Bidder,Price,Buyout,Deposit,Ends,State,Version"

func scanAuction(scan func(...any) error, a *AuctionListing) error {
	return scan(&a.ID, &a.Seller, &a.Bidder, &a.Price, &a.Buyout, &a.Deposit, &a.Ends, &a.State, &a.Version)
}

func ExpireAuctions() error {
	return auctionTx(func(c context.Context, tx *sql.Tx) error {
		now := time.Now().Unix()
		rows, err := tx.QueryContext(c, "SELECT "+auctionFields+" FROM GX_Auctions WHERE State='active' AND Ends<=? ORDER BY Ends LIMIT 32", now)
		if err != nil {
			return err
		}
		var due []AuctionListing
		for rows.Next() {
			var a AuctionListing
			if err := scanAuction(rows.Scan, &a); err != nil {
				rows.Close()
				return err
			}
			due = append(due, a)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, a := range due {
			state := "Expired"
			if a.Bidder != 0 {
				state = "Won by bid"
			}
			if err := settleAuction(c, tx, a, a.Bidder, a.Price, state, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func PostAuction(a AuctionListing, change CharacterChange) error {
	return auctionTx(func(c context.Context, tx *sql.Tx) error {
		item, err := json.Marshal(a.Item)
		if err != nil {
			return err
		}
		if err = commitGameplayTx(c, tx, []CharacterChange{change}, nil); err != nil {
			return err
		}
		_, err = tx.ExecContext(c, `INSERT INTO GX_Auctions(Seller,Request,SellerName,Name,Category,ItemType,TypeID,Level,Item,ItemUID,Price,Buyout,Deposit,Ends) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			a.Seller, a.Request, a.SellerName, a.Name, a.Category, a.Item.Type, a.Item.TypeID, a.Level, item, a.Item.UID, a.Price, a.Buyout, a.Deposit, a.Ends)
		return err
	})
}

func ChangeAuction(owner uint32, req proto.AuctionRequest, change *CharacterChange) error {
	return auctionTx(func(c context.Context, tx *sql.Tx) error {
		now := time.Now().Unix()
		if req.Action == "clearwatch" {
			_, err := tx.ExecContext(c, "DELETE FROM GX_AuctionWatch WHERE Owner=?", owner)
			return err
		}
		if req.Action == "unwatch" {
			_, err := tx.ExecContext(c, "DELETE FROM GX_AuctionWatch WHERE Owner=? AND Auction=?", owner, req.ID)
			return err
		}
		var a AuctionListing
		row := tx.QueryRowContext(c, "SELECT "+auctionFields+" FROM GX_Auctions WHERE ID=?", req.ID)
		if err := scanAuction(row.Scan, &a); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAuctionChanged
			}
			return err
		}
		if a.State != "active" || a.Ends <= now || a.Version != req.Version {
			return ErrAuctionChanged
		}
		switch req.Action {
		case "watch":
			var n, exists int
			err := tx.QueryRowContext(c, "SELECT COUNT(*),COALESCE(MAX(Auction=?),0) FROM GX_AuctionWatch WHERE Owner=?", a.ID, owner).Scan(&n, &exists)
			if err != nil || exists != 0 {
				return err
			}
			if n >= 10 {
				return ErrAuctionWatchFull
			}
			_, err = tx.ExecContext(c, "INSERT INTO GX_AuctionWatch(Owner,Auction) VALUES(?,?)", owner, a.ID)
			return err
		case "cancel":
			if owner != a.Seller {
				return ErrAuctionInvalid
			}
			if err := auctionRefund(c, tx, a, now); err != nil {
				return err
			}
			return settleAuction(c, tx, a, 0, 0, "Canceled", now)
		case "bid", "buy":
			if owner == a.Seller || change == nil || change.Before.ID != owner {
				return ErrAuctionInvalid
			}
			amount := req.Amount
			if req.Action == "bid" {
				if owner == a.Bidder || uint64(amount) < uint64(a.Price)*110/100 || amount >= a.Buyout {
					return ErrAuctionInvalid
				}
			} else {
				amount = a.Buyout

			}
			if change.Before.Money < amount || change.Before.Money-change.After.Money != amount {
				return ErrAuctionInvalid
			}
			if err := commitGameplayTx(c, tx, []CharacterChange{*change}, nil); err != nil {
				return err
			}
			if err := auctionRefund(c, tx, a, now); err != nil {
				return err
			}
			if req.Action == "buy" {
				return settleAuction(c, tx, a, owner, a.Buyout, "Bought", now)
			}
			_, err := tx.ExecContext(c, "UPDATE GX_Auctions SET Bidder=?,Price=?,Version=Version+1 WHERE ID=?", owner, amount, a.ID)
			return err
		}
		return ErrAuctionInvalid
	})
}

func AuctionClaims(owner uint32, view int, id int64) ([]AuctionClaim, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT c.ID,c.Gold,a.Item FROM GX_AuctionClaims c JOIN GX_Auctions a ON a.ID=c.Auction WHERE c.Owner=? AND c.Claimed=0 AND (c.Gold>0)=? AND (c.ID=? OR ?=0) ORDER BY c.ID LIMIT 101`, owner, view == proto.AuctionGold, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuctionClaim
	for rows.Next() {
		var claim AuctionClaim
		var gold int64
		var item []byte
		if err := rows.Scan(&claim.ID, &gold, &item); err != nil {
			return nil, err
		}
		if gold > 4294967295 {
			return nil, ErrAuctionInvalid
		}
		claim.Gold = uint32(gold)
		if err := json.Unmarshal(item, &claim.Item); err != nil {
			return nil, err
		}
		out = append(out, claim)
	}
	return out, rows.Err()
}

func CollectAuction(claim AuctionClaim, change CharacterChange) error {
	return auctionTx(func(c context.Context, tx *sql.Tx) error {
		res, err := tx.ExecContext(c, "UPDATE GX_AuctionClaims SET Claimed=1 WHERE ID=? AND Owner=? AND Gold=? AND Claimed=0", claim.ID, change.Before.ID, claim.Gold)
		if err != nil {
			return err
		}
		if requireOne(res) != nil {
			return ErrAuctionChanged
		}
		if claim.Gold == 0 {
			result, err := tx.ExecContext(c, "UPDATE GX_Auctions SET ItemUID=NULL WHERE ItemUID=? AND ID=(SELECT Auction FROM GX_AuctionClaims WHERE ID=?)", claim.Item.UID, claim.ID)
			if err != nil {
				return err
			}
			if err := requireOne(result); err != nil {
				return err
			}
		}
		return commitGameplayTx(c, tx, []CharacterChange{change}, nil)
	})
}

func ListAuctions(owner uint32, req proto.AuctionRequest) (proto.AuctionReply, error) {
	reply := proto.AuctionReply{Sequence: req.Sequence, NPC: req.NPC, View: req.View, Page: req.Page}
	pool, err := Pool(GameData)
	if err != nil {
		return reply, err
	}
	c, cancel := ctx()
	defer cancel()
	now := time.Now().Unix()
	q := `SELECT a.ID,a.Version,a.Name,a.SellerName,a.Item,a.Level,a.Price,a.Buyout,a.Ends,a.State,a.Seller,a.Bidder,0,'' FROM GX_Auctions a WHERE a.State='active' AND a.Ends>?`
	args := []any{now}
	switch req.View {
	case proto.AuctionBrowse:
		if !req.ValidSearchFilters() {
			return reply, ErrAuctionInvalid
		}
		if req.Category > 0 {
			q += " AND a.Category=?"
			args = append(args, req.Category)
		}
		if req.Subcategory > 0 {
			types := proto.AuctionSubcategories[req.Category][req.Subcategory-1].Types
			q += " AND a.ItemType IN (" + strings.TrimSuffix(strings.Repeat("?,", len(types)), ",") + ")"
			for _, itemType := range types {
				args = append(args, itemType)
			}
		}
		if req.ItemType != 0 {
			q += " AND a.ItemType=? AND a.TypeID=?"
			args = append(args, req.ItemType, req.ItemID)
		}
		if req.Query != "" {
			term := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(req.Query)
			q += " AND (a.Name LIKE ? ESCAPE '\\' OR a.SellerName LIKE ? ESCAPE '\\')"
			args = append(args, "%"+term+"%", "%"+term+"%")
		}
		if req.MinLevel > 0 {
			q += " AND a.Level>=?"
			args = append(args, req.MinLevel)
		}
		if req.MaxLevel > 0 {
			q += " AND a.Level<=?"
			args = append(args, req.MaxLevel)
		}
		if req.Faction != proto.AuctionAnyFaction || req.NameGrade != nil {
			q += " AND EXISTS (SELECT 1 FROM PS_GameDefs_Items d WHERE d.ItemID=a.ItemType*1000+a.TypeID"
			switch req.Faction {
			case proto.AuctionLight:
				q += " AND (d.Type=30 OR d.Country NOT BETWEEN 3 AND 5)"
			case proto.AuctionFury:
				q += " AND (d.Type=30 OR d.Country>2)"
			}
			if req.NameGrade != nil {
				// The native market compares the low byte of ReqDex (item name grade).
				q += " AND (d.ReqDex & 255)=?"
				args = append(args, *req.NameGrade)
			}
			q += ")"
		}
	case proto.AuctionSelling:
		q += " AND a.Seller=?"
		args = append(args, owner)
	case proto.AuctionBidding:
		q += " AND a.Bidder=?"
		args = append(args, owner)
	case proto.AuctionWatch:
		q = `SELECT a.ID,a.Version,a.Name,a.SellerName,a.Item,a.Level,a.Price,a.Buyout,a.Ends,a.State,a.Seller,a.Bidder,0,'' FROM GX_AuctionWatch w JOIN GX_Auctions a ON a.ID=w.Auction WHERE w.Owner=?`
		args = []any{owner}
	case proto.AuctionItems, proto.AuctionGold:
		q = `SELECT c.ID,a.Version,a.Name,a.SellerName,a.Item,a.Level,a.Price,a.Buyout,c.Ends,a.State,a.Seller,a.Bidder,c.Gold,c.Reason FROM GX_AuctionClaims c JOIN GX_Auctions a ON a.ID=c.Auction WHERE c.Owner=? AND c.Claimed=0 AND (c.Gold>0)=? ORDER BY c.ID DESC`
		args = []any{owner, req.View == proto.AuctionGold}
	default:
		return reply, ErrAuctionInvalid
	}
	if req.View != proto.AuctionItems && req.View != proto.AuctionGold {
		order := [...]string{"a.ID", "a.Name COLLATE NOCASE", "a.Level", "a.Ends", "a.Price"}
		sort := req.Sort
		direction := " ASC"
		if sort <= 0 {
			sort = -sort
			direction = " DESC"
		}
		if sort >= len(order) {
			return reply, ErrAuctionInvalid
		}
		q += " ORDER BY " + order[sort] + direction
		if sort != 0 {
			q += ", a.ID DESC"
		}
	}
	rows, err := pool.QueryContext(c, q+" LIMIT ? OFFSET ?", append(args, proto.AuctionPageSize+1, req.Page*proto.AuctionPageSize)...)
	if err != nil {
		return reply, err
	}
	defer rows.Close()
	for rows.Next() {
		var r proto.AuctionRow
		var raw []byte
		var seller, bidder uint32
		var state string
		if err := rows.Scan(&r.ID, &r.Version, &r.Name, &r.Seller, &raw, &r.Level, &r.Price, &r.Buyout, &r.Ends, &state, &seller, &bidder, &r.Gold, &r.Reason); err != nil {
			return reply, err
		}
		if len(reply.Rows) == proto.AuctionPageSize {
			reply.More = true
			break
		}
		var item Item
		if err := json.Unmarshal(raw, &item); err != nil {
			return reply, err
		}
		r.Type, r.TypeID, r.Count, r.Quality, r.Gems, r.Craft = item.Type, item.TypeID, item.Count, item.Quality, item.Gems, item.CraftName
		r.Active, r.Mine, r.HighBidder = state == "active" && r.Ends > now, seller == owner, bidder == owner
		if r.Reason == "" && !r.Active {
			r.Reason = state
			if state == "active" {
				r.Reason = "Expired"
			}
		}
		reply.Rows = append(reply.Rows, r)
	}
	return reply, rows.Err()
}
