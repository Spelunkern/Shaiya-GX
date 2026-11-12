package game

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

// GuildState is populated only by server packets; roster batches can arrive
// before the country-wide guild directory at world entry.
type GuildState struct {
	ID                 uint32
	Name               string
	Rank               uint8
	Directory          map[uint32]proto.GuildInfo
	DirectoryReady     bool
	Ranking            map[uint32]proto.GuildRankingEntry
	RankingReady       bool
	RankingNotice      proto.GuildRankingNoticeInfo
	RankingPoints      proto.GuildRankingPointsInfo
	RankingPointsKnown bool
	Etin               uint32
	EtinKnown          bool
	Remark             string
	RemarkKnown        bool
	RemarkResult       uint8
	RemarkSerial       uint64
	Members            map[uint32]proto.GuildMember
	Applicants         map[uint32]proto.GuildCandidate
	Bank               map[uint8]proto.InventoryItem
	BankResult         proto.GuildBankResultInfo
	BankResultSerial   uint64
	ApplyResult        uint8
	ApplySerial        uint64
	JoinResult         uint8
	JoinSerial         uint64
	CreateResult       uint8
	CreateSerial       uint64
	CreateVote         proto.GuildCreateVote
	VoteSerial         uint64
	CreateFailure      uint8
	FailureSerial      uint64
	LeaveResult        uint8
	LeaveSerial        uint64
	DisbandResult      uint8
	DisbandSerial      uint64
	DisbandMaster      uint32
	DisbandVoteSerial  uint64
	DisbandedSerial    uint64
	InviteID           uint32
	InviteSerial       uint64
	LastReply          proto.PartyReply
	ReplySerial        uint64
}

func (s *State) clearVisibleGuild(id uint32) {
	if id == 0 {
		return
	}
	for _, e := range s.Entities {
		if e.Kind == KindUser && e.GuildID == id {
			e.GuildID = 0
		}
	}
}

func (s *State) setVisibleGuildMember(id, guild uint32) {
	if guild == 0 {
		return
	}
	if e := s.Entities[id]; e != nil && e.Kind == KindUser {
		e.GuildID = guild
	}
}

func (s *State) applyGuild(p []byte) error {
	g := &s.Guild
	switch proto.Op(p) {
	case proto.GuildEtin:
		v, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		g.Etin, g.EtinKnown = v, true
	case proto.GuildRemark:
		v, err := proto.DecodeGuildRemark(p)
		if err != nil {
			return err
		}
		g.RemarkResult = v.Result
		g.RemarkSerial++
		if v.Result == 1 {
			g.Remark, g.RemarkKnown = v.Remark, true
			if info, ok := g.Directory[g.ID]; ok {
				info.Remark = v.Remark
				g.Directory[g.ID] = info
			}
		}
	case proto.GuildRankingNotice:
		v, err := proto.DecodeGuildRankingNotice(p)
		if err != nil {
			return err
		}
		g.RankingNotice = v
	case proto.GuildRankingPoints:
		v, err := proto.DecodeGuildRankingPoints(p)
		if err != nil {
			return err
		}
		g.RankingPoints, g.RankingPointsKnown = v, true
	case proto.GuildRankingList:
		rows, err := proto.DecodeGuildRankingList(p)
		if err != nil {
			return err
		}
		g.Ranking = make(map[uint32]proto.GuildRankingEntry, len(rows))
		for _, row := range rows {
			g.Ranking[row.Guild] = row
			if info, ok := g.Directory[row.Guild]; ok {
				info.Rank, info.Points = row.Rank, row.Points
				g.Directory[row.Guild] = info
			}
		}
		g.RankingReady = true
	case proto.GuildBankList:
		items, err := proto.DecodeGuildBankList(p)
		if err != nil {
			return err
		}
		if g.Bank == nil {
			g.Bank = make(map[uint8]proto.InventoryItem)
		}
		for _, item := range items {
			g.Bank[item.Slot] = item
		}
	case proto.GuildBankPlaced, proto.GuildBankTaken:
		event, err := proto.DecodeGuildBankEvent(p)
		if err != nil {
			return err
		}
		if proto.Op(p) == proto.GuildBankTaken {
			delete(g.Bank, event.Item.Slot)
		} else {
			if g.Bank == nil {
				g.Bank = make(map[uint8]proto.InventoryItem)
			}
			g.Bank[event.Item.Slot] = event.Item
		}
	case proto.GuildBankResult:
		result, err := proto.DecodeGuildBankResult(p)
		if err != nil {
			return err
		}
		g.BankResult = result
		g.BankResultSerial++
	case proto.GuildListBegin:
		g.Directory = make(map[uint32]proto.GuildInfo)
		g.DirectoryReady = false
	case proto.GuildListEnd:
		g.DirectoryReady = true
	case proto.GuildList:
		list, err := proto.DecodeGuildList(p)
		if err != nil {
			return err
		}
		if g.Directory == nil {
			g.Directory = make(map[uint32]proto.GuildInfo)
		}
		for _, v := range list {
			g.Directory[v.ID] = v
			if v.ID == g.ID {
				g.Remark, g.RemarkKnown = v.Remark, true
			}
		}
	case proto.GuildNew:
		v, err := proto.DecodeGuildInfo(p)
		if err != nil {
			return err
		}
		if g.Directory == nil {
			g.Directory = make(map[uint32]proto.GuildInfo)
		}
		g.Directory[v.ID] = v
		if v.ID == g.ID {
			g.Remark, g.RemarkKnown = v.Remark, true
		}
	case proto.GuildDisbanded:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		delete(g.Directory, id)
		s.clearVisibleGuild(id)
	case proto.GuildOnlineList, proto.GuildOfflineList:
		members, err := proto.DecodeGuildMembers(p, proto.Op(p) == proto.GuildOnlineList)
		if err != nil {
			return err
		}
		if g.Members == nil {
			g.Members = make(map[uint32]proto.GuildMember)
		}
		for _, m := range members {
			g.Members[m.ID] = m
			s.setVisibleGuildMember(m.ID, g.ID)
			if m.ID == s.SelfID {
				g.Rank = m.Rank
			}
		}
	case proto.GuildMemberAdded:
		m, err := proto.DecodeGuildMemberAdded(p)
		if err != nil {
			return err
		}
		if g.Members == nil {
			g.Members = make(map[uint32]proto.GuildMember)
		}
		g.Members[m.ID] = m
		s.setVisibleGuildMember(m.ID, g.ID)
		if m.ID == s.SelfID {
			g.Rank = m.Rank
		}
	case proto.GuildApplicantList:
		applicants, err := proto.DecodeGuildApplicants(p)
		if err != nil {
			return err
		}
		if g.Applicants == nil {
			g.Applicants = make(map[uint32]proto.GuildCandidate)
		}
		for _, candidate := range applicants {
			g.Applicants[candidate.ID] = candidate
		}
	case proto.GuildApplicant:
		candidate, err := proto.DecodeGuildApplicant(p)
		if err != nil {
			return err
		}
		if g.Applicants == nil {
			g.Applicants = make(map[uint32]proto.GuildCandidate)
		}
		g.Applicants[candidate.ID] = candidate
	case proto.GuildApplicantRemoved:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		delete(g.Applicants, id)
	case proto.GuildApply:
		result, err := proto.DecodeU8(p)
		if err != nil {
			return err
		}
		g.ApplyResult = result
		g.ApplySerial++
	case proto.GuildStatus:
		r := proto.NewReader(p)
		mode, id := r.U8(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		if mode == 101 {
			g.DisbandedSerial++
			s.clearVisibleGuild(g.ID)
			g.ID, g.Name, g.Rank, g.Members = 0, "", 0, nil
			g.Bank = nil
			g.Etin, g.EtinKnown, g.Remark, g.RemarkKnown = 0, false, "", false
			g.Applicants = nil
			s.Details.Guild = ""
			return nil
		}
		if mode == 102 || mode == 103 {
			if e := s.Entities[id]; e != nil && e.Kind == KindUser && e.GuildID == g.ID {
				e.GuildID = 0
			}
		}
		m, ok := g.Members[id]
		if !ok {
			return nil
		}
		switch {
		case mode == 102 || mode == 103:
			delete(g.Members, id)
			return nil
		case mode == 104:
			m.Online = true
		case mode == 105:
			m.Online = false
		case mode > 200:
			m.Rank = mode - 200
		case mode > 0 && mode <= 100:
			m.Level = uint16(mode)
		}
		g.Members[id] = m
		if id == s.SelfID {
			g.Rank = m.Rank
		}
	case proto.GuildInvite:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		g.InviteID = id
		g.InviteSerial++
	case proto.GuildInviteAnswer:
		reply, err := proto.DecodePartyReply(p)
		if err != nil {
			return err
		}
		g.LastReply = reply
		g.ReplySerial++
	case proto.GuildJoinResult:
		v, err := proto.DecodeGuildJoin(p)
		if err != nil {
			return err
		}
		g.JoinResult = v.Result
		g.JoinSerial++
		if v.Result == 1 {
			g.ID, g.Name, g.Rank, g.Members = v.ID, v.Name, v.Rank, nil
			g.Etin, g.EtinKnown, g.Remark, g.RemarkKnown = 0, false, "", false
			if info, ok := g.Directory[v.ID]; ok {
				g.Remark, g.RemarkKnown = info.Remark, true
			}
			g.Bank = nil
			g.Applicants = nil
			s.Details.Guild = v.Name
			g.InviteID = 0
		}
	case proto.GuildCreate:
		v, remark, err := proto.DecodeGuildCreate(p)
		if err != nil {
			return err
		}
		g.CreateResult = v.Result
		g.CreateSerial++
		if v.Result == 1 {
			g.ID, g.Name, g.Rank, g.Members = v.ID, v.Name, v.Rank, nil
			g.Etin, g.EtinKnown, g.Remark, g.RemarkKnown = 0, false, remark, true
			g.Bank = nil
			g.Applicants = nil
			s.Details.Guild = v.Name
			g.InviteID = 0
		}
	case proto.GuildCreateAgree:
		vote, err := proto.DecodeGuildCreateVote(p)
		if err != nil {
			return err
		}
		g.CreateVote = vote
		g.VoteSerial++
	case proto.GuildCreateFailed:
		result, err := proto.DecodeGuildCreateFailure(p)
		if err != nil {
			return err
		}
		g.CreateFailure = result
		g.FailureSerial++
	case proto.GuildLeave:
		v, err := proto.DecodeU8(p)
		if err != nil {
			return err
		}
		g.LeaveResult = v
		g.LeaveSerial++
		if v == 1 {
			g.ID, g.Name, g.Rank, g.Members = 0, "", 0, nil
			g.Etin, g.EtinKnown, g.Remark, g.RemarkKnown = 0, false, "", false
			g.Bank = nil
			g.Applicants = nil
			s.Details.Guild = ""
		}
	case proto.GuildDisband:
		v, err := proto.DecodeU8(p)
		if err != nil {
			return err
		}
		g.DisbandResult = v
		g.DisbandSerial++
	case proto.GuildDisbandAgree:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		g.DisbandMaster = id
		g.DisbandVoteSerial++
	case proto.GuildExpel:
		reply, err := proto.DecodePartyReply(p)
		if err != nil {
			return err
		}
		if reply.Result == 1 && reply.ID == s.SelfID {
			g.ID, g.Name, g.Rank, g.Members = 0, "", 0, nil
			g.Etin, g.EtinKnown, g.Remark, g.RemarkKnown = 0, false, "", false
			g.Bank = nil
			g.Applicants = nil
			s.Details.Guild = ""
		}
	}
	return nil
}
