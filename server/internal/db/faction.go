package db

import (
	"database/sql"
	"errors"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

// FactionNone is reported while the account has not chosen a faction yet.
const FactionNone = 2

// ReadMaxGrow runs PS_GameData.dbo.usp_Read_User_MaxGrow_R. The original
// procedure always reports MaxGrow 3; without a row the account has no
// faction yet.
func ReadMaxGrow(userUID int32) (country, maxGrow uint8, err error) {
	pool, err := Pool(GameData)
	if err != nil {
		return 0, 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var countryValue int16
	var maxGrowValue int32
	err = pool.QueryRowContext(c, `SELECT Country,3 as MaxGrow FROM PS_GameData_UserMaxGrow WHERE ServerID=@serverid AND UserUID=@useruid`,
		sql.Named("serverid", config.ServerID), sql.Named("useruid", userUID)).Scan(&countryValue, &maxGrowValue)
	if errors.Is(err, sql.ErrNoRows) {
		return FactionNone, 3, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return uint8(countryValue), uint8(maxGrowValue), nil
}

// SaveCountry runs PS_GameData.dbo.usp_Save_Char_Country_E, inserting the
// UserMaxGrow row the first time an account picks a faction.
func SaveCountry(userUID int32, country uint8) error {
	current, _, err := ReadMaxGrow(userUID)
	if err != nil {
		return err
	}
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	_, err = pool.ExecContext(c, `INSERT INTO PS_GameData_UserMaxGrow(ServerID,UserUID,Country,MaxGrow) VALUES(@serverid,@useruid,@country,1) ON CONFLICT(ServerID,UserUID) DO UPDATE SET Country=excluded.Country`,
		sql.Named("serverid", config.ServerID), sql.Named("useruid", userUID),
		sql.Named("country", int16(country)), sql.Named("insert", current == FactionNone))
	return err
}
