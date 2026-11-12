package proto

import "fmt"

// Original handlers 0x5976d0, 0x597640 and 0x597a80 read 2+1,
// 4+4+4 and a count followed by 9-byte rows, respectively.
type GuildRankingNoticeInfo struct {
	Map  uint16
	Kind uint8
}
type GuildRankingPointsInfo struct{ Points, Maximum, Leader uint32 }
type GuildRankingEntry struct {
	Guild, Points uint32
	Rank          uint8
}

func DecodeGuildRankingNotice(p []byte) (GuildRankingNoticeInfo, error) {
	r := NewReader(p)
	v := GuildRankingNoticeInfo{Map: r.U16(), Kind: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("guild ranking notice: trailing bytes")
	}
	return v, nil
}
func DecodeGuildRankingPoints(p []byte) (GuildRankingPointsInfo, error) {
	r := NewReader(p)
	v := GuildRankingPointsInfo{Points: r.U32(), Maximum: r.U32(), Leader: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("guild ranking points: trailing bytes")
	}
	return v, nil
}
func DecodeGuildRankingList(p []byte) ([]GuildRankingEntry, error) {
	r := NewReader(p)
	count := int(r.U8())
	if r.Err() != nil {
		return nil, r.Err()
	}
	// Game.exe's handler has space for 40 rows; Rapture sends at most 30.
	if count > 40 {
		return nil, fmt.Errorf("guild ranking: too many rows %d", count)
	}
	if r.Remaining() != count*9 {
		return nil, fmt.Errorf("guild ranking: incorrect row length")
	}
	out := make([]GuildRankingEntry, 0, count)
	seen := make(map[uint32]bool, count)
	for i := 0; i < count; i++ {
		v := GuildRankingEntry{Guild: r.U32(), Points: r.U32(), Rank: r.U8()}
		if v.Guild == 0 || seen[v.Guild] {
			return nil, fmt.Errorf("guild ranking: duplicate or zero guild")
		}
		seen[v.Guild] = true
		out = append(out, v)
	}
	return out, r.Err()
}
