package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func sqliteFixture(t *testing.T) *sql.DB {
	t.Helper()
	seed := os.Getenv("RAPTURE_TEST_SEED")
	if seed == "" {
		seed = os.Getenv("RAPTURE_DB_SEED")
		if seed == "" {
			_, file, _, _ := runtime.Caller(0)
			seed = filepath.Join(filepath.Dir(file), "../../data/rapture.sqlite")
		}
	}
	t.Setenv("RAPTURE_DB_PATH", filepath.Join(t.TempDir(), "runtime.sqlite"))
	t.Setenv("RAPTURE_DB_SEED", seed)
	reset := func() {
		mu.Lock()
		defer mu.Unlock()
		for _, p := range pools {
			p.Close()
		}
		pools = map[string]*sql.DB{}
	}
	reset()
	t.Cleanup(reset)
	p, e := Pool(GameData)
	if e != nil {
		t.Fatal(e)
	}
	return p
}

func TestSQLiteLifecycle(t *testing.T) {
	p := sqliteFixture(t)
	seed, _ := os.ReadFile(os.Getenv("RAPTURE_DB_SEED"))
	before := sha256.Sum256(seed)
	_, e := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(990001,'sqlite_probe','localpass',0,0,0,0,0,'N',1000)`)
	if e != nil {
		t.Fatal(e)
	}
	a, e := AccountByUID(990001)
	if e != nil {
		t.Fatal(e)
	}
	if r, e := TryGameLogin(a.UserID, "wrong", 123, "127.0.0.1"); e != nil || r.Status != -1 {
		t.Fatal("bad password", r, e)
	}
	if r, e := TryGameLogin(a.UserID, "localpass", 123, "127.0.0.1"); e != nil || r.Status != 0 || r.UserUID != 990001 {
		t.Fatal("login", r, e)
	}
	if e = SaveCountry(a.UserUID, 0); e != nil {
		t.Fatal(e)
	}
	if country, grow, e := ReadMaxGrow(a.UserUID); e != nil || country != 0 || grow != 3 {
		t.Fatal("faction", country, grow, e)
	}
	id, e := CreateCharacter(a, NewCharacter{Slot: 4, Name: "SQLiteProbe", MapID: 1, PosX: 578, PosY: 78, PosZ: 1762, HP: 100, MP: 80, SP: 70, Str: 14, Dex: 12, Rec: 9, Int: 8, Wis: 7, Luc: 15})
	if e != nil {
		t.Fatal("create", e)
	}
	chars, e := CharactersForAccount(uint32(a.UserUID))
	if e != nil || len(chars) != 1 {
		t.Fatal("list", len(chars), e)
	}
	ch, e := CharacterDetail(uint32(a.UserUID), id)
	if e != nil || ch.SkillPoint != 5 {
		t.Fatal("detail", ch, e)
	}
	ch.PosX += 3
	ch.Money = 123
	ch.CurrentHP = 90
	if e = SaveCharacter(&ch); e != nil {
		t.Fatal("save", e)
	}
	it := Item{Bag: 1, Slot: 2, Type: 1, TypeID: 1, Count: 1, Quality: 100, MakeTime: time.Now()}
	if e = AddCharacterItem(id, &it); e != nil {
		t.Fatal("item add", e)
	}
	if got, e := CharacterItems(id); e != nil || len(got) != 1 || got[0].MakeTime.IsZero() {
		t.Fatal("items", got, e)
	}
	it.Slot = 3
	if e = SaveCharacterItem(id, &it); e != nil {
		t.Fatal("item save", e)
	}
	if e = AddCharacterSkill(id, CharacterSkill{SkillID: 1, Level: 1, Number: 1}); e != nil {
		t.Fatal("skill", e)
	}
	if e = SaveCharacterSkill(id, CharacterSkill{SkillID: 1, Level: 2, Delay: 3}); e != nil {
		t.Fatal("skill save", e)
	}
	if got, e := CharacterSkills(id); e != nil || len(got) != 1 || got[0].Level != 2 {
		t.Fatal("skills", got, e)
	}
	if e = SaveQuickSlots(id, []QuickSlot{{QuickBar: 0, QuickSlot: 1, Bag: 1, Number: 2}}); e != nil {
		t.Fatal("quickslots", e)
	}
	if got, e := QuickSlots(id); e != nil || len(got) != 1 {
		t.Fatal("quickslot read", got, e)
	}
	if e = SaveAppliedBuffs(id, []AppliedBuff{{SkillID: 1, Level: 1, LeftSeconds: 30}}); e != nil {
		t.Fatal("buffs", e)
	}
	if got, e := AppliedBuffs(id); e != nil || len(got) != 1 {
		t.Fatal("buff read", got, e)
	}
	if e = AddCharacterQuest(id, CharQuest{QuestID: 1}); e != nil {
		t.Fatal("quest", e)
	}
	if e = SaveCharacterQuest(id, CharQuest{QuestID: 1, Counts: [3]uint8{1, 2, 3}}); e != nil {
		t.Fatal("quest save", e)
	}
	if got, e := CharacterQuests(id); e != nil || len(got) != 1 {
		t.Fatal("quests", got, e)
	}
	if e = FinishCharacterQuest(id, 1, 1); e != nil {
		t.Fatal("quest finish", e)
	}
	if got, e := FinishedQuests(id); e != nil || len(got) != 1 {
		t.Fatal("finished quests", got, e)
	}
	if e = DeleteCharacterItems(id, it.UID); e != nil {
		t.Fatal("item delete", e)
	}
	if n, e := SpendPoints(uint32(a.UserUID), id, 200, "probe", "", 1); e != nil || n != 800 {
		t.Fatal("points", n, e)
	}
	if _, e = SpendPoints(uint32(a.UserUID), id, 900, "probe", "", 1); e != ErrNotEnoughPoints {
		t.Fatal("overdraft", e)
	}
	if e = EnableRename(id); e != nil {
		t.Fatal("rename enable", e)
	}
	if ok, e := RenameCharacter(uint32(a.UserUID), id, "SQLiteProbe", "SQLiteRenamed"); e != nil || !ok {
		t.Fatal("rename", ok, e)
	}
	if ok, e := DeleteCharacter(uint32(a.UserUID), id); e != nil || !ok {
		t.Fatal("delete", ok, e)
	}
	if ok, e := RestoreCharacter(uint32(a.UserUID), id); e != nil || !ok {
		t.Fatal("restore", ok, e)
	}
	if e = TryGameLogout(a.UserUID, 123); e != nil {
		t.Fatal("logout", e)
	}
	var health string
	if e = p.QueryRow("PRAGMA integrity_check").Scan(&health); e != nil || health != "ok" {
		t.Fatal(health, e)
	}
	after, _ := os.ReadFile(os.Getenv("RAPTURE_DB_SEED"))
	if sha256.Sum256(after) != before {
		t.Fatal("runtime changed the seed")
	}
}

func TestSQLiteDefinitions(t *testing.T) {
	sqliteFixture(t)
	defs, e := Definitions()
	if e != nil {
		t.Fatal(e)
	}
	if len(defs.Items) == 0 || len(defs.Skills) == 0 || len(defs.Mobs) == 0 || len(defs.Exp) == 0 {
		t.Fatal("empty definitions")
	}
	t.Logf("items=%d skills=%d mobs=%d products=%d", len(defs.Items), len(defs.Skills), len(defs.Mobs), len(defs.Products))
}

func TestSQLiteGuildAndWarehouse(t *testing.T) {
	p := sqliteFixture(t)
	const uid = 990002
	if _, e := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(990002,'sqlite_guild','test',0,0,0,0,0,'N',1000)`); e != nil {
		t.Fatal(e)
	}
	a, e := AccountByUID(uid)
	if e != nil {
		t.Fatal(e)
	}
	id, e := CreateCharacter(a, NewCharacter{Slot: 0, Name: "SQLiteGuild", MapID: 1, HP: 100, MP: 100, SP: 100})
	if e != nil {
		t.Fatal(e)
	}
	guild, e := NextGuildID()
	if e != nil {
		t.Fatal(e)
	}
	if e = CreateGuild(guild, "SQLiteGuild", a.UserID, id, "SQLiteGuild", 0, "notice"); e != nil {
		t.Fatal("create guild", e)
	}
	if e = AddGuildChar(guild, id, 1); e != nil {
		t.Fatal("add member", e)
	}
	if n, e := GuildJoinRemainHours(id); e != nil || n < 71 || n > 72 {
		t.Fatal("guild join", n, e)
	}
	if e = SaveGuildCharRank(id, 2); e != nil {
		t.Fatal("guild rank", e)
	}
	if e = SaveGuildRemark(guild, "updated"); e != nil {
		t.Fatal("guild notice", e)
	}
	if e = SaveGuildRanking(GuildRow{ID: guild, Points: 42, Rank: 3, Etin: 200, Remark: "ranked"}); e != nil {
		t.Fatal("ranking", e)
	}
	gs, ms, e := LoadGuilds()
	if e != nil || len(gs) == 0 || len(ms) == 0 {
		t.Fatal("guild read", e)
	}
	it := Item{Slot: 4, Type: 1, TypeID: 1, Count: 1, Quality: 10}
	if e = AddStoredItem(uid, &it); e != nil {
		t.Fatal("warehouse", e)
	}
	if got, e := StoredItems(uid); e != nil || len(got) != 1 {
		t.Fatal("warehouse read", got, e)
	}
	it.Slot = 5
	if e = SaveStoredItem(uid, &it); e != nil {
		t.Fatal("warehouse save", e)
	}
	if e = DeleteStoredItems(uid, it.UID); e != nil {
		t.Fatal("warehouse delete", e)
	}
	it.UID = 0
	if e = AddGuildStoredItem(guild, &it); e != nil {
		t.Fatal("guild warehouse", e)
	}
	if got, e := GuildStoredItems(guild); e != nil || len(got) != 1 {
		t.Fatal("guild warehouse read", got, e)
	}
	if e = SaveGuildStoredItem(guild, &it); e != nil {
		t.Fatal("guild warehouse save", e)
	}
	if e = DeleteGuildStoredItems(guild, it.UID); e != nil {
		t.Fatal("guild warehouse delete", e)
	}
	if e = SaveWorldInfo(123456); e != nil {
		t.Fatal("world calendar", e)
	}
	if e = SaveGodBless(100, 200); e != nil {
		t.Fatal("blessing", e)
	}
	if light, dark, e := ReadGodBless(); e != nil || light != 100 || dark != 200 {
		t.Fatal("blessing read", light, dark, e)
	}
	if v, e := ReadWorldInfo(); e != nil || v != 123456 {
		t.Fatal("world calendar read", v, e)
	}
	if e = DeleteGuildChar(guild, id); e != nil {
		t.Fatal("guild leave", e)
	}
	if e = DeleteGuild(guild); e != nil {
		t.Fatal("guild delete", e)
	}
}

func TestSQLiteGiftsAndGuardedBalance(t *testing.T) {
	p := sqliteFixture(t)
	for uid := 990020; uid <= 990021; uid++ {
		if _, e := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?1,?2,'local',0,0,0,0,0,'N',100)`, uid, fmt.Sprintf("gift%d", uid)); e != nil {
			t.Fatal(e)
		}
	}
	a, e := AccountByUID(990021)
	if e != nil {
		t.Fatal(e)
	}
	id, e := CreateCharacter(a, NewCharacter{Slot: 0, Name: "SQLiteGift", MapID: 1, HP: 100, MP: 100, SP: 100})
	if e != nil {
		t.Fatal(e)
	}
	product := &Product{Code: "sqlite_test", Items: []ProductItem{{Type: 1, TypeID: 1, Count: 1}}}
	if result, uid, e := StoreGift("SQLiteGift", 990020, product); e != nil || result != GiftOK || uid != 990021 {
		t.Fatal("gift", result, uid, e)
	}
	entries, e := GiftBox(990021)
	if e != nil || len(entries) != 1 {
		t.Fatal("gift box", entries, e)
	}
	it := &Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}
	if ok, e := ClaimGift(990021, id, entries[0].Slot, it); e != nil || !ok {
		t.Fatal("claim", ok, e)
	}
	if ok, e := ClaimGift(990021, id, entries[0].Slot, it); e != nil || ok {
		t.Fatal("duplicate claim", ok, e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, e := SpendPoints(990020, id, 80, "sqlite_test", "SQLiteGift", 2); results <- e })
	}
	wg.Wait()
	close(results)
	wins := 0
	for e := range results {
		if e == nil {
			wins++
		} else if e != ErrNotEnoughPoints {
			t.Fatal(e)
		}
	}
	if n, e := Points(990020); e != nil || n != 20 || wins != 1 {
		t.Fatal("balance was charged twice", n, wins, e)
	}
}

func TestDistributedSQLiteSnapshot(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	seed := filepath.Join(filepath.Dir(file), "../../data/rapture.sqlite")
	manifestFile := filepath.Join(filepath.Dir(seed), "rapture.manifest.json")
	raw, e := os.ReadFile(manifestFile)
	if e != nil {
		t.Fatal(e)
	}
	var manifest struct {
		Tables map[string]int64
		SHA256 string
	}
	if e = json.Unmarshal(raw, &manifest); e != nil {
		t.Fatal(e)
	}
	raw, e = os.ReadFile(seed)
	if e != nil {
		t.Fatal(e)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != manifest.SHA256 {
		t.Fatal("distributed seed checksum does not match the manifest")
	}
	p, e := sql.Open("sqlite", "file:"+filepath.ToSlash(seed)+"?mode=ro")
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	for table, want := range manifest.Tables {
		var count int64
		if e = p.QueryRow(`SELECT COUNT(*) FROM "` + strings.ReplaceAll(table, `"`, `""`) + `"`).Scan(&count); e != nil || count != want {
			t.Fatal(table, count, want, e)
		}
	}
}
