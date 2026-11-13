package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Guilds live in PS_GameData: Guilds, GuildDetails and GuildChars
// (GuildLevel 0 is a pending applicant, 1 the master, 2 sub-masters, up to
// 9). The original game server reached them through the DB agent; Rapture
// calls the same procedures. usp_Read_Guild2_R (orders by a Guilds.Rank
// column the EP4.5 schema lacks) and usp_Save_GuildRemark_E (reads a
// missing GuildHouses table) fail on this database, so their statements
// are run as plain SQL.

// mirrorGuild is what Trig_Guilds_Insert/Update/Delete did through the
// linked server PS_USERDB01 (disabled, see tools/sqlserver.py prepare):
// PS_UserData.dbo.CreatedGuilds holds a copy of the Guilds row for server 1.
// The EP4.5 CreatedGuilds also has a NOT NULL UpdateDate the triggers never
// set; it gets the time of the change.
func mirrorGuild(guildID uint32) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, p, func(tx *sql.Tx) error {
		if _, e := tx.ExecContext(c, `DELETE FROM PS_UserData_CreatedGuilds WHERE ServerID=1 AND GuildID=?1`, int32(guildID)); e != nil {
			return e
		}
		_, e := tx.ExecContext(c, `INSERT INTO PS_UserData_CreatedGuilds(ServerID,GuildID,GuildName,MasterUserID,MasterCharID,MasterName,Country,TotalCount,GuildPoint,Del,CreateDate,UpdateDate,DeleteDate)
   SELECT 1,GuildID,GuildName,MasterUserID,MasterCharID,MasterName,Country,TotalCount,GuildPoint,Del,CreateDate,CURRENT_TIMESTAMP,DeleteDate FROM PS_GameData_Guilds WHERE GuildID=?1`, int32(guildID))
		return e
	})
}

// guildOfMember is the live guild of a character, for the mirror after
// procedures keyed by CharID.
func guildOfMember(charID uint32) uint32 {
	pool, err := Pool(GameData)
	if err != nil {
		return 0
	}
	c, cancel := ctx()
	defer cancel()
	var id int64
	_ = pool.QueryRowContext(c, `SELECT GuildID FROM PS_GameData_GuildChars WHERE CharID = ?1 AND Del = 0 LIMIT 1`, int32(charID)).Scan(&id)
	return uint32(id)
}

// GuildRow is one usp_Read_Guild2_R row.
type GuildRow struct {
	ID         uint32
	Name       string
	MasterName string
	Country    uint8
	Points     uint32
	Rank       uint8
	Etin       uint32
	Remark     string
	BuyHouse   uint8
	UseHouse   uint8
	EtinReturn uint32
	KeepEtin   uint32
}

// GuildCharRow is one usp_Read_GuildChar_R row.
type GuildCharRow struct {
	CharID  uint32
	Name    string
	Level   uint16
	Job     uint8
	GuildID uint32
	Rank    uint8 // GuildLevel
}

// LoadGuilds reads every live guild and its members.
func LoadGuilds() ([]GuildRow, []GuildCharRow, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT G.GuildID, G.GuildName, G.MasterName, G.Country, G.GuildPoint,
		COALESCE(H.Rank,31), COALESCE(H.Etin,0), COALESCE(H.Remark,''), COALESCE(H.BuyHouse,0), COALESCE(H.UseHouse,0),
		COALESCE(H.EtinReturnCnt,0), COALESCE(H.KeepEtin,0)
		FROM PS_GameData_Guilds G LEFT OUTER JOIN PS_GameData_GuildDetails H ON G.GuildID = H.GuildID WHERE G.Del = 0`)
	if err != nil {
		return nil, nil, fmt.Errorf("read guilds: %w", err)
	}
	var guilds []GuildRow
	for rows.Next() {
		var g GuildRow
		var id, points, etin, ret, keep int64
		if err := rows.Scan(&id, &g.Name, &g.MasterName, &g.Country, &points, &g.Rank, &etin, &g.Remark, &g.BuyHouse,
			&g.UseHouse, &ret, &keep); err != nil {
			rows.Close()
			return nil, nil, err
		}
		g.ID, g.Points, g.Etin = uint32(id), uint32(max(points, 0)), uint32(max(etin, 0))
		g.EtinReturn, g.KeepEtin = uint32(max(ret, 0)), uint32(max(keep, 0))
		g.Name, g.MasterName, g.Remark = strings.TrimSpace(g.Name), strings.TrimSpace(g.MasterName), strings.TrimSpace(g.Remark)
		guilds = append(guilds, g)
	}
	rows.Close()
	rows, err = pool.QueryContext(c, `SELECT B.CharID,B.CharName,B.Level,B.Job,A.GuildID,A.GuildLevel FROM PS_GameData_GuildChars A JOIN PS_GameData_Chars B ON A.CharID=B.CharID WHERE A.Del=0 AND (@guildid=0 OR A.GuildID=@guildid)`, sql.Named("guildid", int32(0)))
	if err != nil {
		return nil, nil, fmt.Errorf("usp_Read_GuildChar_R: %w", err)
	}
	defer rows.Close()
	var members []GuildCharRow
	for rows.Next() {
		var m GuildCharRow
		var charID, guildID int64
		var level int64
		if err := rows.Scan(&charID, &m.Name, &level, &m.Job, &guildID, &m.Rank); err != nil {
			return nil, nil, err
		}
		m.CharID, m.GuildID, m.Level, m.Name = uint32(charID), uint32(guildID), uint16(max(level, 0)), strings.TrimSpace(m.Name)
		members = append(members, m)
	}
	return guilds, members, rows.Err()
}

// MaxGuildID is usp_Read_MaxGuildID_R: the next free guild id.
func NextGuildID() (uint32, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var next int64
	err = pool.QueryRowContext(c, `SELECT COALESCE( MAX(GuildID),0 ) + 1 FROM PS_GameData_Guilds`).Scan(&next)
	return uint32(max(next, 1)), err
}

func execStatus(query string, args ...any) (int, error) {
	p, e := Pool(GameData)
	if e != nil {
		return 0, e
	}
	c, cancel := ctx()
	defer cancel()
	res, e := p.ExecContext(c, query, args...)
	if e != nil {
		return 0, e
	}
	n, e := res.RowsAffected()
	return int(n), e
}

// CreateGuild is usp_Create_Guild2_E (0 on success).
func CreateGuild(id uint32, name, masterUserID string, masterCharID uint32, masterName string, country uint8, remark string) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	e = withTx(c, p, func(tx *sql.Tx) error {
		var count int
		if e := tx.QueryRowContext(c, `SELECT COUNT(*) FROM PS_GameData_Guilds WHERE GuildName=?1 AND Del=0`, name).Scan(&count); e != nil {
			return e
		}
		if count > 0 {
			return fmt.Errorf("guild name already exists")
		}
		if _, e := tx.ExecContext(c, `INSERT INTO PS_GameData_Guilds(GuildID,GuildName,MasterUserID,MasterCharID,MasterName,Country,TotalCount,GuildPoint,CreateDate) VALUES(?1,?2,?3,?4,?5,?6,0,0,CURRENT_TIMESTAMP)`, int32(id), strings.TrimSpace(name), masterUserID, int32(masterCharID), strings.TrimSpace(masterName), country); e != nil {
			return e
		}
		_, e := tx.ExecContext(c, `INSERT INTO PS_GameData_GuildDetails(GuildID,Remark) VALUES(?1,?2)`, int32(id), strings.TrimSpace(remark))
		return e
	})
	if e != nil {
		return e
	}
	return mirrorGuild(id)
}

// AddGuildChar is usp_Create_GuildChar_E (rank 0 for an applicant).
func AddGuildChar(guildID, charID uint32, rank uint8) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	e = withTx(c, p, func(tx *sql.Tx) error {
		var count int
		if e := tx.QueryRowContext(c, `SELECT COUNT(*) FROM PS_GameData_GuildChars WHERE CharID=?1 AND Del=0`, int32(charID)).Scan(&count); e != nil {
			return e
		}
		if count > 0 {
			return fmt.Errorf("character already in a guild")
		}
		if _, e := tx.ExecContext(c, `INSERT INTO PS_GameData_GuildChars(GuildID,CharID,GuildLevel,JoinDate) VALUES(?1,?2,?3,CURRENT_TIMESTAMP)`, int32(guildID), int32(charID), rank); e != nil {
			return e
		}
		_, e := tx.ExecContext(c, `UPDATE PS_GameData_Guilds SET TotalCount=TotalCount+1 WHERE GuildID=?1`, int32(guildID))
		return e
	})
	if e != nil {
		return e
	}
	return mirrorGuild(guildID)
}

// SaveGuildCharRank is usp_Save_GuildChar_E (rank 1 also moves the master).
func SaveGuildCharRank(charID uint32, rank uint8) error {
	guildID := guildOfMember(charID)
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	e = withTx(c, p, func(tx *sql.Tx) error {
		if rank == 1 {
			if _, e := tx.ExecContext(c, `UPDATE PS_GameData_Guilds SET MasterCharID=?1,MasterUserID=(SELECT UserID FROM PS_GameData_Chars WHERE CharID=?1 AND Del=0),MasterName=(SELECT CharName FROM PS_GameData_Chars WHERE CharID=?1 AND Del=0) WHERE GuildID=?2`, int32(charID), int32(guildID)); e != nil {
				return e
			}
		}
		_, e := tx.ExecContext(c, `UPDATE PS_GameData_GuildChars SET GuildLevel=?2 WHERE CharID=?1 AND Del=0`, int32(charID), rank)
		return e
	})
	if e != nil {
		return e
	}
	return mirrorGuild(guildID)
}

// DeleteGuildChar is usp_Delete_GuildChar_E.
func DeleteGuildChar(guildID, charID uint32) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	e = withTx(c, p, func(tx *sql.Tx) error {
		r, e := tx.ExecContext(c, `UPDATE PS_GameData_GuildChars SET Del=1,LeaveDate=CURRENT_TIMESTAMP WHERE GuildID=?1 AND CharID=?2 AND Del=0`, int32(guildID), int32(charID))
		if e != nil {
			return e
		}
		n, e := r.RowsAffected()
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(c, `UPDATE PS_GameData_Guilds SET TotalCount=MAX(0,TotalCount-?2) WHERE GuildID=?1`, int32(guildID), n)
		return e
	})
	if e != nil {
		return e
	}
	return mirrorGuild(guildID)
}

// DeleteGuild is usp_Delete_Guild_E.
func DeleteGuild(guildID uint32) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	e = withTx(c, p, func(tx *sql.Tx) error {
		for _, q := range []string{`UPDATE PS_GameData_Guilds SET Del=1,DeleteDate=CURRENT_TIMESTAMP,TotalCount=0 WHERE GuildID=?1 AND Del=0`, `UPDATE PS_GameData_GuildChars SET Del=1,LeaveDate=CURRENT_TIMESTAMP WHERE GuildID=?1 AND Del=0`} {
			if _, e := tx.ExecContext(c, q, int32(guildID)); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	return mirrorGuild(guildID)
}

// SaveGuildRemark stores the notice in GuildDetails (usp_Save_GuildRemark_E's
// update or insert).
func SaveGuildRemark(guildID uint32, remark string) error {
	return execSQL(GameData, `INSERT INTO PS_GameData_GuildDetails(GuildID,Remark) VALUES(?1,?2) ON CONFLICT(GuildID) DO UPDATE SET Remark=excluded.Remark`, int32(guildID), remark)
}

// GuildJoinRemainHours is usp_Read_Char_GuildJoinDate_R: hours left of the
// 72 hours after the character's last guild join.
func GuildJoinRemainHours(charID uint32) (int, error) {
	p, e := Pool(GameData)
	if e != nil {
		return 0, e
	}
	c, cancel := ctx()
	defer cancel()
	var hours int
	e = p.QueryRowContext(c, `SELECT MAX(0,72-CAST((julianday(strftime('%Y-%m-%d %H:00:00','now'))-julianday(strftime('%Y-%m-%d %H:00:00',JoinDate)))*24 AS INTEGER)) FROM PS_GameData_GuildChars WHERE CharID=?1 ORDER BY JoinDate DESC LIMIT 1`, int32(charID)).Scan(&hours)
	if errors.Is(e, sql.ErrNoRows) {
		return 0, nil
	}
	return hours, e
}

// SaveGuildRanking is usp_Save_GuildInfo_E2 (the DB agent's 0x0B16 batch):
// guild points, rank and the house fields, then the CreatedGuilds copy.
func SaveGuildRanking(g GuildRow) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	e = withTx(c, p, func(tx *sql.Tx) error {
		if _, e := tx.ExecContext(c, `UPDATE PS_GameData_Guilds SET GuildPoint=?2 WHERE GuildID=?1`, int32(g.ID), int32(g.Points)); e != nil {
			return e
		}
		_, e := tx.ExecContext(c, `INSERT INTO PS_GameData_GuildDetails(GuildID,UseHouse,BuyHouse,Rank,Etin,EtinReturnCnt,KeepEtin,Remark) VALUES(?1,?2,?3,?4,?5,?6,?7,?8)
   ON CONFLICT(GuildID) DO UPDATE SET UseHouse=excluded.UseHouse,BuyHouse=excluded.BuyHouse,Rank=excluded.Rank,Etin=excluded.Etin,EtinReturnCnt=excluded.EtinReturnCnt,KeepEtin=excluded.KeepEtin,Remark=excluded.Remark`, int32(g.ID), g.UseHouse, g.BuyHouse, g.Rank, int32(g.Etin), int32(g.EtinReturn), int32(g.KeepEtin), strings.TrimSpace(g.Remark))
		return e
	})
	if e != nil {
		return e
	}
	return mirrorGuild(g.ID)
}

// ReadGodBless is usp_Read_GodBless_R (ps_dbAgent CWorld::DBKillCountLoad):
// the light and dark country kill counts, 0 without a row.
func ReadGodBless() (light, dark uint32, err error) {
	pool, err := Pool(GameData)
	if err != nil {
		return 0, 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var l, d sql.NullInt64
	err = pool.QueryRowContext(c, `SELECT GodBless_Light, GodBless_Dark FROM PS_GameData_WorldInfo`).Scan(&l, &d)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("usp_Read_GodBless_R: %w", err)
	}
	return uint32(l.Int64), uint32(d.Int64), nil
}

// ErrNoWorldInfo is usp_Save_GodBless_E's -1: PS_GameData.dbo.WorldInfo
// has no row yet (the world calendar creates it).
var ErrNoWorldInfo = errors.New("WorldInfo has no row")

// SaveGodBless is usp_Save_GodBless_E (CWorld::DBKillCountSave): 1 saved,
// -1 without a WorldInfo row.
func SaveGodBless(light, dark uint32) error {
	n, e := execStatus(`UPDATE PS_GameData_WorldInfo SET GodBless_Light=?1,GodBless_Dark=?2`, int32(light), int32(dark))
	if e == nil && n != 1 {
		return ErrNoWorldInfo
	}
	return e
}

// SaveWorldInfo is usp_Save_WorldInfo_E (the DB agent's world calendar
// save), which creates the WorldInfo row when missing.
func SaveWorldInfo(lastWorldTime int32) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, p, func(tx *sql.Tx) error {
		r, e := tx.ExecContext(c, `UPDATE PS_GameData_WorldInfo SET LastWorldTime=?1`, lastWorldTime)
		if e != nil {
			return e
		}
		n, e := r.RowsAffected()
		if e != nil {
			return e
		}
		if n > 0 {
			return nil
		}
		_, e = tx.ExecContext(c, `INSERT INTO PS_GameData_WorldInfo(LastWorldTime,GodBless_Light,GodBless_Dark) VALUES(?1,0,0)`, lastWorldTime)
		return e
	})
}

// ReadWorldInfo is usp_Read_WorldInfo_E: the packed world calendar base
// (0 without a row).
func ReadWorldInfo() (int32, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var v sql.NullInt64
	err = pool.QueryRowContext(c, `SELECT LastWorldTime FROM PS_GameData_WorldInfo LIMIT 1`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("usp_Read_WorldInfo_E: %w", err)
	}
	return int32(v.Int64), nil
}
