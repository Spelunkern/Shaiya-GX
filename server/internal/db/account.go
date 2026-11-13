package db

import (
	"database/sql"
	"errors"
)

// LoginResult is what PS_UserData.dbo.usp_Try_GameLogin_Taiwan selects.
type LoginResult struct {
	Status  int32
	UserUID int32
}

// TryGameLogin translates the original procedure into one SQLite transaction,
// including account status, the login log and UserLoginStatus.
func TryGameLogin(userID, password string, sessionID int64, userIP string) (LoginResult, error) {
	p, err := Pool(UserData)
	if err != nil {
		return LoginResult{}, err
	}
	c, cancel := ctx()
	defer cancel()
	result := LoginResult{Status: -3}
	err = withTx(c, p, func(tx *sql.Tx) error {
		var uid int32
		var status int32
		var leave int
		var matches bool
		e := tx.QueryRowContext(c, `SELECT UserUID,Status,Leave,Pw=?2 FROM PS_UserData_Users_Master WHERE UserID=?1`, userID, password).Scan(&uid, &status, &leave, &matches)
		if errors.Is(e, sql.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		result = LoginResult{Status: status, UserUID: uid}
		if leave != 0 {
			result.Status = -3
			return nil
		}
		if !matches {
			result.Status = -1
			return nil
		}
		if status >= 2 && status <= 6 {
			var expired bool
			e = tx.QueryRowContext(c, `SELECT datetime(BlockEndDate)<=CURRENT_TIMESTAMP FROM PS_UserData_Users_Block WHERE UserUID=?1`, uid).Scan(&expired)
			if e != nil && !errors.Is(e, sql.ErrNoRows) {
				return e
			}
			if expired {
				result.Status = 0
				if _, e = tx.ExecContext(c, `UPDATE PS_UserData_Users_Master SET Status=0 WHERE UserUID=?1`, uid); e != nil {
					return e
				}
			}
		}
		if result.Status != 0 && result.Status != 16 && result.Status != 32 && result.Status != 48 && result.Status != 64 && result.Status != 80 {
			return nil
		}
		_, e = tx.ExecContext(c, `INSERT INTO PS_UserData_UserLoginStatus(UserUID,UserID,LoginStatus,LoginTime,LogoutTime,LoginSession,LastPlayTime)
   VALUES(?1,?2,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,?3,0)
   ON CONFLICT(UserUID) DO UPDATE SET LoginStatus=1,LoginTime=CURRENT_TIMESTAMP,LogoutTime=CURRENT_TIMESTAMP,LoginSession=excluded.LoginSession,LastPlayTime=0`, uid, userID, sessionID)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(c, `INSERT INTO Rapture_LoginLog(UserUID,SessionID,UserIP,LogType) VALUES(?1,?2,?3,0)`, uid, sessionID, userIP)
		return e
	})
	return result, err
}

// Logout and the original two-minute online-point credit commit together.
func TryGameLogout(userUID int32, sessionID int64) error {
	p, err := Pool(UserData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, p, func(tx *sql.Tx) error {
		var loginSession int64
		var minutes int
		var loginTime string
		e := tx.QueryRowContext(c, `SELECT LoginSession,MAX(0,CAST((julianday('now')-julianday(LoginTime))*1440 AS INTEGER)),LoginTime FROM PS_UserData_UserLoginStatus WHERE UserUID=?1`, userUID).Scan(&loginSession, &minutes, &loginTime)
		if errors.Is(e, sql.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		if sessionID == loginSession {
			if _, e = tx.ExecContext(c, `INSERT INTO PS_UserData_UserOnlineTimeLog(UserUID,OldPoint,AddPoint,LoginTime,LogoutTime,SessionID)
    SELECT UserUID,COALESCE(Point,0),?2,?3,CURRENT_TIMESTAMP,?4 FROM PS_UserData_Users_Master WHERE UserUID=?1`, userUID, minutes, loginTime, sessionID); e != nil {
				return e
			}
			if _, e = tx.ExecContext(c, `UPDATE PS_UserData_Users_Master SET Point=COALESCE(Point,0)+?2 WHERE UserUID=?1`, userUID, minutes/2); e != nil {
				return e
			}
		}
		if _, e = tx.ExecContext(c, `UPDATE PS_UserData_UserLoginStatus SET LoginStatus=0,LogoutTime=CURRENT_TIMESTAMP,LogoutSession=?2,LastPlayTime=?3 WHERE UserUID=?1`, userUID, sessionID, minutes); e != nil {
			return e
		}
		_, e = tx.ExecContext(c, `INSERT INTO Rapture_LoginLog(UserUID,SessionID,LogType) VALUES(?1,?2,1)`, userUID, sessionID)
		return e
	})
}

// LoginDecision maps the procedure's Status to the client's login result
// byte and permission (auth status), as recovered from the original
// ps_login (CUser login handling; see Eternity src/auth/login_protocol.cpp).
// Permission 11 is a regular account; 1..5 are the admin tiers.
func LoginDecision(login LoginResult) (result uint8, permission uint8) {
	permission = 11
	if login.UserUID == 0 {
		return 1, permission
	}
	switch login.Status {
	case -3:
		result = 1
	case -2:
		result = 2
	case -1:
		result = 3
	case 1:
		result = 9
	case 2:
		result = 4
	case 6:
		result = 5
	case 0x11:
		result = 10
	case 0x21:
		result = 11
	case 0x31:
		result = 12
	case 0x41:
		result = 13
	case 0x51:
		result = 14
	case 0x80:
		result = 6
	case 0x100:
		result = 7
	case 0x200:
		result = 8
	default:
		switch {
		case login.Status&2 != 0:
			result = 10
		case login.Status&1 != 0:
			result = 11
		case (uint32(login.Status)>>7)&7 != 0:
			result = 6
		default:
			permission = uint8((login.Status >> 4) & 7)
			if permission == 0 {
				permission = 11
			}
		}
	}
	return result, permission
}

// Account is the account state the world server needs.
type Account struct {
	UserUID    int32
	UserID     string
	Status     int16
	Admin      bool
	AdminLevel uint8
	Point      int32
	Country    uint8 // 0 light, 1 dark, 2 none chosen yet
	MaxGrow    uint8
}

var ErrNotFound = errors.New("not found")

// AccountByUID reads Users_Master plus the realm's UserMaxGrow row
// (PS_GameData.dbo.usp_Read_User_MaxGrow_R).
func AccountByUID(userUID int32) (Account, error) {
	pool, err := Pool(UserData)
	if err != nil {
		return Account{}, err
	}
	c, cancel := ctx()
	defer cancel()
	var a Account
	err = pool.QueryRowContext(c, `SELECT UserUID, UserID, Status, Admin, AdminLevel, Point
		FROM PS_UserData_Users_Master WHERE UserUID = ?1 AND Leave = 0`, userUID).
		Scan(&a.UserUID, &a.UserID, &a.Status, &a.Admin, &a.AdminLevel, &a.Point)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, err
	}
	a.Country, a.MaxGrow, err = ReadMaxGrow(userUID)
	return a, err
}
