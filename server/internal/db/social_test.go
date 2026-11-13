package db

import "testing"

func TestSQLiteSocialPersistenceAndIndependentLists(t *testing.T) {
	p := sqliteFixture(t)
	rows, err := p.Query(`SELECT CharName FROM PS_GameData_Chars WHERE Del=0 ORDER BY CharID LIMIT 2`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	rows.Close()
	if len(names) != 2 {
		t.Fatal("fixture needs two characters")
	}
	a, err := SocialCharacter(names[0])
	if err != nil {
		t.Fatal(err)
	}
	b, err := SocialCharacter(names[1])
	if err != nil {
		t.Fatal(err)
	}
	if err = AddFriends(a, b); err != nil {
		t.Fatal(err)
	}
	if err = AddSocial(a.ID, b, true); err != nil {
		t.Fatal(err)
	}
	memo := "Tank's note; SELECT 1"
	if err = SocialMemo(a.ID, b.ID, memo, false); err != nil {
		t.Fatal(err)
	}
	found := false
	friends, err := SocialContacts(a.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range friends {
		if v.ID == b.ID {
			found = v.Memo == memo
		}
	}
	if !found || !SocialBlocked(a.ID, b.ID) {
		t.Fatal("friend memo or independent block lost")
	}
	if err = RemoveSocial(a.ID, b.ID, false); err != nil {
		t.Fatal(err)
	}
	if !SocialBlocked(a.ID, b.ID) {
		t.Fatal("deleting friend removed block")
	}
	friends, err = SocialContacts(b.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, v := range friends {
		if v.ID == a.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("friend acceptance did not persist both directions")
	}
	if err = RemoveSocial(a.ID, b.ID, true); err != nil {
		t.Fatal(err)
	}
	if SocialBlocked(a.ID, b.ID) {
		t.Fatal("unblock did not persist")
	}
}
