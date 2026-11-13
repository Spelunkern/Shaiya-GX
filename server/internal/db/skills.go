package db

import (
	"database/sql"
	"fmt"
)

// CharacterSkill is one learned skill. Delay is the remaining cooldown in
// seconds as last saved (CharSkills.Delay).
type CharacterSkill struct {
	SkillID uint16
	Level   uint8
	Number  uint8
	Delay   int16
}

// CharacterSkills runs usp_Read_Char_Skills_R.
func CharacterSkills(charID uint32) ([]CharacterSkill, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT SkillID,SkillLevel,Number,[Delay] FROM PS_GameData_CharSkills WHERE CharID=@charid AND Del=0 ORDER BY Number ASC`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_Skills_R: %w", err)
	}
	defer rows.Close()
	var skills []CharacterSkill
	for rows.Next() {
		var s CharacterSkill
		var id int64
		if err := rows.Scan(&id, &s.Level, &s.Number, &s.Delay); err != nil {
			return nil, err
		}
		s.SkillID = clampU16(id)
		skills = append(skills, s)
	}
	return skills, rows.Err()
}

// AddCharacterSkill runs usp_Save_Char_Skill_Add_E.
func AddCharacterSkill(charID uint32, s CharacterSkill) error {
	return execSQL(GameData, `INSERT INTO PS_GameData_CharSkills(CharID,SkillID,SkillLevel,Number,[Delay]) VALUES(@charid,@skillid,@skilllevel,@number,@delay)`, sql.Named("charid", int32(charID)),
		sql.Named("skillid", int16(s.SkillID)), sql.Named("skilllevel", s.Level),
		sql.Named("number", s.Number), sql.Named("delay", s.Delay))
}

// SaveCharacterSkill runs usp_Save_Char_Skill_Mod_E (level and remaining
// cooldown).
func SaveCharacterSkill(charID uint32, s CharacterSkill) error {
	return execSQL(GameData, `UPDATE PS_GameData_CharSkills SET SkillLevel=@skilllevel,[Delay]=@delay WHERE CharID=@charid AND SkillID=@skillid`, sql.Named("charid", int32(charID)),
		sql.Named("skillid", int16(s.SkillID)), sql.Named("skilllevel", s.Level), sql.Named("delay", s.Delay))
}

// QuickSlot is one skill-bar entry.
type QuickSlot struct {
	QuickBar, QuickSlot, Bag uint8
	Number                   uint16
}

// QuickSlots runs usp_Read_Char_QuickSlots_R.
func QuickSlots(charID uint32) ([]QuickSlot, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT QuickBar,QuickSlot,Bag,Number FROM PS_GameData_CharQuickSlots WHERE CharID=@charid`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_QuickSlots_R: %w", err)
	}
	defer rows.Close()
	var slots []QuickSlot
	for rows.Next() {
		var q QuickSlot
		var number int64
		if err := rows.Scan(&q.QuickBar, &q.QuickSlot, &q.Bag, &number); err != nil {
			return nil, err
		}
		q.Number = clampU16(number)
		slots = append(slots, q)
	}
	return slots, rows.Err()
}

// SaveQuickSlots replaces the skill bar (usp_Save_Char_QuickSlot_Del_E then
// usp_Save_Char_QuickSlot_Add_E per entry, at most 128).
func SaveQuickSlots(charID uint32, slots []QuickSlot) error {
	if err := execSQL(GameData, `DELETE FROM PS_GameData_CharQuickSlots WHERE CharID=@charid`, sql.Named("charid", int32(charID))); err != nil {
		return err
	}
	for i, q := range slots {
		if i >= 128 {
			break
		}
		if err := execSQL(GameData, `INSERT INTO PS_GameData_CharQuickSlots(CharID,QuickBar,QuickSlot,Bag,Number) VALUES(@charid,@quickbar,@quickslot,@bag,@number)`, sql.Named("charid", int32(charID)),
			sql.Named("quickbar", q.QuickBar), sql.Named("quickslot", q.QuickSlot), sql.Named("bag", q.Bag),
			sql.Named("number", int16(q.Number))); err != nil {
			return err
		}
	}
	return nil
}

// AppliedBuff is a persisted active buff with its remaining seconds.
type AppliedBuff struct {
	SkillID     uint16
	Level       uint8
	LeftSeconds uint32
}

// AppliedBuffs runs usp_Read_Char_ApplySkills_R.
func AppliedBuffs(charID uint32) ([]AppliedBuff, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT SkillID,SkillLevel,LeftResetTime FROM PS_GameData_CharApplySkills WHERE CharID=@charid`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_ApplySkills_R: %w", err)
	}
	defer rows.Close()
	var buffs []AppliedBuff
	for rows.Next() {
		var b AppliedBuff
		var id int64
		var left int32
		if err := rows.Scan(&id, &b.Level, &left); err != nil {
			return nil, err
		}
		b.SkillID, b.LeftSeconds = clampU16(id), uint32(max(left, 0))
		buffs = append(buffs, b)
	}
	return buffs, rows.Err()
}

// SaveAppliedBuffs replaces the persisted buffs
// (usp_Save_Char_ApplySkill_Del_E with DeleteAll, then
// usp_Save_Char_ApplySkill_Add_E2 per buff).
func SaveAppliedBuffs(charID uint32, buffs []AppliedBuff) error {
	if err := execSQL(GameData, `DELETE FROM PS_GameData_CharApplySkills WHERE CharID=@charid AND (@deleteall=1 OR (SkillID=@skillid AND SkillLevel=@skilllevel))`, sql.Named("charid", int32(charID)),
		sql.Named("skillid", int16(0)), sql.Named("skilllevel", uint8(0)), sql.Named("deleteall", true)); err != nil {
		return err
	}
	for _, b := range buffs {
		if err := execSQL(GameData, `INSERT INTO PS_GameData_CharApplySkills(CharID,Skillid,SkillLevel,LeftResetTime) VALUES(@charid,@skillid,@skilllevel,@leftresettime)`, sql.Named("charid", int32(charID)),
			sql.Named("skillid", int16(b.SkillID)), sql.Named("skilllevel", b.Level),
			sql.Named("leftresettime", int32(b.LeftSeconds))); err != nil {
			return err
		}
	}
	return nil
}
