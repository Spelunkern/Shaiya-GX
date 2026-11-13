package db

import "testing"

func TestGMGrantRollsBackAllChanges(t *testing.T) {
	p := sqliteFixture(t)
	item := Item{Bag: 1, Slot: 0, Type: 25, TypeID: 1, Count: 2}
	if err := AddCharacterItem(990100, &item); err != nil {
		t.Fatal(err)
	}
	changed := item
	changed.Count = 10
	added := item
	added.Slot = 1 // Reusing the UID must fail after the UPDATE.
	if err := ApplyGMItemGrant(990100, []Item{changed}, []Item{added}); err == nil {
		t.Fatal("duplicate UID accepted")
	}
	var count int
	if err := p.QueryRow(`SELECT [Count] FROM PS_GameData_CharItems WHERE CharID=990100 AND ItemUID=?`, item.UID).Scan(&count); err != nil || count != 2 {
		t.Fatal("partial grant persisted", count, err)
	}
	if err := ApplyGMItemGrant(990101, []Item{changed}, nil); err == nil {
		t.Fatal("another character's item updated")
	}
	if err := ApplyGMItemGrant(990100, []Item{changed}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(990999,'gm_role_probe','local',0,255,0,0,0,'N',0)`); err != nil {
		t.Fatal(err)
	}
	a, err := AccountByUID(990999)
	if err != nil || a.Admin || a.AdminLevel != 255 {
		t.Fatal("role must use Admin independently of legacy level", a.Admin, err)
	}
	if _, err = p.Exec(`UPDATE PS_UserData_Users_Master SET Admin=1 WHERE UserUID=990999`); err != nil {
		t.Fatal(err)
	}
	a, err = AccountByUID(990999)
	if err != nil || !a.Admin {
		t.Fatal("GM flag not read", err)
	}
}
