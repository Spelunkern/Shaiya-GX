package db

import "fmt"

type SocialContact struct {
	ID         uint32
	Name, Memo string
	Class      uint8
}

func AddFriends(a, b Character) error {
	p, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	tx, err := p.BeginTx(c, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, pair := range [][2]Character{{a, b}, {b, a}} {
		var n int
		if err = tx.QueryRowContext(c, `SELECT COUNT(*) FROM PS_GameData_FriendChars WHERE CharID=? AND Refuse=0`, pair[0].ID).Scan(&n); err != nil {
			return err
		}
		if n >= 50 {
			return fmt.Errorf("friend list is full")
		}
		v := pair[1]
		if _, err = tx.ExecContext(c, `INSERT INTO PS_GameData_FriendChars(CharID,FriendID,FriendName,Family,Grow,Job,Memo,Refuse,CreateDate) VALUES(?,?,?,?,?,?,'',0,CURRENT_TIMESTAMP) ON CONFLICT(CharID,FriendID) DO UPDATE SET Refuse=0`, pair[0].ID, v.ID, v.Name, v.Race, v.Mode, v.Class); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func SocialContacts(charID uint32, blocked bool) ([]SocialContact, error) {
	p, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	query := `SELECT f.FriendID,f.FriendName,COALESCE(f.Memo,''),f.Job FROM PS_GameData_FriendChars f WHERE f.CharID=? AND f.Refuse=0 ORDER BY f.FriendID`
	if blocked {
		query = `SELECT BanID,BanName,COALESCE(Memo,''),0 FROM PS_GameData_BanChars WHERE CharID=? ORDER BY BanID`
	}
	rows, err := p.QueryContext(c, query, charID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SocialContact
	for rows.Next() {
		var v SocialContact
		if err = rows.Scan(&v.ID, &v.Name, &v.Memo, &v.Class); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func SocialCharacter(name string) (Character, error) {
	p, err := Pool(GameData)
	if err != nil {
		return Character{}, err
	}
	c, cancel := ctx()
	defer cancel()
	var ch Character
	err = p.QueryRowContext(c, `SELECT CharID,CharName,Family,Job,Grow FROM PS_GameData_Chars WHERE CharName=? COLLATE NOCASE AND Del=0`, name).Scan(&ch.ID, &ch.Name, &ch.Race, &ch.Class, &ch.Mode)
	return ch, err
}
func AddSocial(charID uint32, target Character, blocked bool) error {
	p, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	tx, err := p.BeginTx(c, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	table := "PS_GameData_FriendChars"
	if blocked {
		table = "PS_GameData_BanChars"
	}
	var count int
	if err = tx.QueryRowContext(c, `SELECT COUNT(*) FROM `+table+` WHERE CharID=?`, charID).Scan(&count); err != nil {
		return err
	}
	if count >= 50 {
		return fmt.Errorf("social list is full")
	}
	if blocked {
		_, err = tx.ExecContext(c, `INSERT OR IGNORE INTO PS_GameData_BanChars(CharID,BanID,BanName,Memo,BanDate) VALUES(?,?,?,'',CURRENT_TIMESTAMP)`, charID, target.ID, target.Name)
	} else {
		_, err = tx.ExecContext(c, `INSERT INTO PS_GameData_FriendChars(CharID,FriendID,FriendName,Family,Grow,Job,Memo,Refuse,CreateDate) VALUES(?,?,?,?,?,?,'',0,CURRENT_TIMESTAMP) ON CONFLICT(CharID,FriendID) DO UPDATE SET Refuse=0`, charID, target.ID, target.Name, target.Race, target.Mode, target.Class)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
func RemoveSocial(charID, target uint32, blocked bool) error {
	p, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	query := `DELETE FROM PS_GameData_FriendChars WHERE CharID=? AND FriendID=?`
	if blocked {
		query = `DELETE FROM PS_GameData_BanChars WHERE CharID=? AND BanID=?`
	}
	_, err = p.ExecContext(c, query, charID, target)
	return err
}
func SocialMemo(charID, target uint32, memo string, blocked bool) error {
	p, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	query := `UPDATE PS_GameData_FriendChars SET Memo=? WHERE CharID=? AND FriendID=?`
	if blocked {
		query = `UPDATE PS_GameData_BanChars SET Memo=? WHERE CharID=? AND BanID=?`
	}
	_, err = p.ExecContext(c, query, memo, charID, target)
	return err
}
func SocialBlocked(charID, sender uint32) bool {
	p, err := Pool(GameData)
	if err != nil {
		return false
	}
	c, cancel := ctx()
	defer cancel()
	var n int
	err = p.QueryRowContext(c, `SELECT COUNT(*) FROM PS_GameData_BanChars WHERE CharID=? AND BanID=?`, charID, sender).Scan(&n)
	return err == nil && n > 0
}
