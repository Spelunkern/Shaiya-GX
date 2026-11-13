package world

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/crypto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

func gmWireSession(t *testing.T, ch db.Character, admin bool) (*Session, <-chan []byte) {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() { a.Close(); b.Close() })
	packets := make(chan []byte, 512)
	go func() {
		defer close(packets)
		for {
			var header [2]byte
			if _, err := io.ReadFull(b, header[:]); err != nil {
				return
			}
			body := make([]byte, int(binary.LittleEndian.Uint16(header[:]))-2)
			if _, err := io.ReadFull(b, body); err != nil {
				return
			}
			packets <- crypto.ExpandedXOR(body)
		}
	}()
	logger, err := logx.New("gx_gm_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logger.Close() })
	s := &Session{log: logger.Tagged("gm"), logger: logger, char: &ch, isAdmin: admin, gmTurbo: admin, expanded: true, out: &connWriter{conn: a}, aesSend: crypto.NewAESCTR([16]byte{}, [16]byte{})}
	return s, packets
}

func gmResponse(t *testing.T, packets <-chan []byte) proto.GMState {
	t.Helper()
	for {
		select {
		case p, ok := <-packets:
			if !ok {
				t.Fatal("closed GM connection")
			}
			if proto.Op(p) == proto.GXGMState {
				g, e := proto.DecodeGMState(p)
				if e != nil {
					t.Fatal(e)
				}
				return g
			}
		case <-time.After(3 * time.Second):
			t.Fatal("missing GM response")
		}
	}
}

func TestGMCommandsRequireRoleAndToggleAuthoritativeState(t *testing.T) {
	s, packets := gmWireSession(t, db.Character{ID: 990020}, false)
	for _, command := range []string{"/turbo", "/iclear", "/getitem 25 1 200", "/mmake 1 3", "/nmake 1 1", "/cmove 1"} {
		if !s.handleGMCommand(command) {
			t.Fatal("GM command escaped to public chat")
		}
		g := gmResponse(t, packets)
		if g.GM || g.Turbo || s.gmTurbo {
			t.Fatal("non-GM acquired privileges")
		}
	}
	s.isAdmin, s.gmTurbo = true, true
	state := game.New()
	state.SelfID = s.char.ID
	for _, want := range []bool{false, true} {
		s.handleGMCommand("/turbo")
		g := gmResponse(t, packets)
		if !g.GM || g.Turbo != want || s.gmTurbo != want {
			t.Fatal("bad toggle", g)
		}
		// Exercise the exact server codec against the client state machine.
		s.gmReply("")
		p := <-packets
		if _, e := state.Apply(p); e != nil || state.GMTurbo != want || !state.Self.IsGM {
			t.Fatal("client GM state", e)
		}
	}
	for _, bad := range []string{"/turbo 1", "/getitem 25 1 -1", "/mmake 1 101", "/nmake 1", "/nmake 14 1", "/nmake 1001", "/nmake 1 1 2", "/cmove 65536"} {
		s.handleGMCommand(bad)
		gmResponse(t, packets)
	}
	if !s.gmTurbo {
		t.Fatal("malformed command changed state")
	}
	if s.handleGMCommand("ordinary chat") {
		t.Fatal("normal chat consumed")
	}
}

func TestGMGrantAndClearPersistOnlyOwnInventory(t *testing.T) {
	account := db.Account{UserUID: 1, UserID: "admin1"}
	id, err := db.CreateCharacter(account, db.NewCharacter{Slot: 4, Name: "GMCommandProbe", HP: 100, MP: 100, SP: 100, MapID: 60070})
	if err != nil {
		t.Fatal(err)
	}
	s, packets := gmWireSession(t, db.Character{ID: id, MapID: 60070, Level: 1, CurrentHP: 100}, true)
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.SetDefsForTest(defs) })
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() {
		pool.Exec(`DELETE FROM PS_GameData_CharItems WHERE CharID=? OR CharID=?`, id, id+1000000)
		pool.Exec(`DELETE FROM PS_GameData_Chars WHERE CharID=?`, id)
	})
	s.handleGMCommand("/getitem 25 1 200")
	reply := gmResponse(t, packets)
	if len(s.items) != 20 {
		t.Fatalf("200 apples should occupy 20 stacks, got %d: %+v", len(s.items), reply)
	}
	stored, err := db.CharacterItems(id)
	if err != nil || len(stored) != 20 {
		t.Fatal("grant persistence", len(stored), err)
	}
	sum := 0
	for _, it := range stored {
		sum += int(it.Count)
		if it.Count > 10 {
			t.Fatal("stack overflow")
		}
	}
	if sum != 200 {
		t.Fatal("wrong quantity", sum)
	}
	// A request that cannot fit must leave the existing stacks and DB intact.
	s.handleGMCommand("/getitem 25 1 30000")
	gmResponse(t, packets)
	stored, err = db.CharacterItems(id)
	if err != nil || len(stored) != 20 {
		t.Fatal("partial failed grant")
	}
	other := db.Item{Bag: 1, Slot: 23, Type: 25, TypeID: 1, Count: 1}
	if err = db.AddCharacterItem(id+1000000, &other); err != nil {
		t.Fatal(err)
	}
	equipped := db.Item{Bag: 0, Slot: 0, Type: 1, TypeID: 1, Count: 1}
	if err = db.AddCharacterItem(id, &equipped); err != nil {
		t.Fatal(err)
	}
	s.items = append(s.items, equipped)
	s.mounted = true
	s.handleGMCommand("/iclear")
	gmResponse(t, packets)
	stored, err = db.CharacterItems(id)
	if err != nil || len(stored) != 0 || len(s.items) != 0 || s.mounted {
		t.Fatal("clear failed", err)
	}
	remaining, err := db.CharacterItems(id + 1000000)
	if err != nil || len(remaining) != 1 {
		t.Fatal("clear affected another character", err)
	}
}

func TestGMSpawnsAreSeparatedAndVisibleImmediately(t *testing.T) {
	const mapID = 60071
	RegisterMapGrid(mapID, 200, nil)
	t.Cleanup(func() {
		shardsMu.Lock()
		delete(shards, mapID)
		shardsMu.Unlock()
		gridsMu.Lock()
		delete(grids, mapID)
		gridsMu.Unlock()
	})
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	var id uint16
	for key := range defs.Mobs {
		if id == 0 || key < id {
			id = key
		}
	}
	if id == 0 {
		t.Fatal("missing mob defs")
	}
	logger, err := logx.New("gx_gm_spawn_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logger.Close() })
	if err = loadNpcQuest(logger); err != nil {
		t.Fatal(err)
	}
	s, packets := gmWireSession(t, db.Character{ID: 990021, MapID: mapID, PosX: 100, PosY: 77, PosZ: 100}, true)
	s.player = RegisterPlayer(*s.char, nil, true, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	s.player.moveViewCell(100, 100)
	if err = s.gmSpawn(id, 12, 0); err != nil {
		t.Fatal(err)
	}
	gmResponse(t, packets)
	if err = s.gmSpawn(id, 12, 0); err != nil {
		t.Fatal(err)
	}
	gmResponse(t, packets)
	if !s.handleGMCommand("/nmake 1 1") {
		t.Fatal("NPC command not handled")
	}
	gmResponse(t, packets)
	entities := EntitiesOnMap(mapID)
	if len(entities.Mobs) != 24 || len(entities.NPCs) != 1 {
		t.Fatal("spawn count")
	}
	for i, m := range entities.Mobs {
		if m.CurrentHP == 0 || m.respawnDelay != 0 || !s.player.knows(viewMob, m.ID) {
			t.Fatal("spawn not live/visible")
		}
		for _, n := range entities.Mobs[:i] {
			if DistanceXZ(m.PosX, m.PosZ, n.PosX, n.PosZ) < 2 {
				t.Fatal("stacked mobs")
			}
		}
		if DistanceXZ(m.PosX, m.PosZ, entities.NPCs[0].PosX, entities.NPCs[0].PosZ) < 2 {
			t.Fatal("stacked NPC")
		}
	}
	n := entities.NPCs[0]
	if n.Type != 1 || n.TypeID != 1 || n.PosY != 77 || !s.player.knows(viewNpc, n.ID) {
		t.Fatal("NPC not at server height or missing view")
	}
}

func TestGMMoveUsesRelativeSafeDestinationAndNormalTeleport(t *testing.T) {
	const source, target = 972, 973
	for _, id := range []uint16{source, target} {
		RegisterMapGrid(id, 200, nil)
		t.Cleanup(func() {
			shardsMu.Lock()
			delete(shards, id)
			shardsMu.Unlock()
			gridsMu.Lock()
			delete(grids, id)
			gridsMu.Unlock()
		})
	}
	oldZone, hadZone := mapConfig.Zones[target]
	mapConfig.Zones[target] = ZoneConfig{CreateType: partyCreateType}
	t.Cleanup(func() {
		if hadZone {
			mapConfig.Zones[target] = oldZone
		} else {
			delete(mapConfig.Zones, target)
		}
	})
	sh := shard(target)
	sh.entities.SpawnAreas = []MapSpawnArea{{Boxes: []SpawnBox{
		{LowerX: 10, UpperX: 20, LowerY: 30, LowerZ: 10, UpperZ: 20},
		{LowerX: 170, UpperX: 180, LowerY: 75, LowerZ: 170, UpperZ: 180},
	}}}
	s, packets := gmWireSession(t, db.Character{ID: 990030, MapID: source, PosX: 180, PosY: 8, PosZ: 180}, true)
	if err := s.gmMove(target); err != nil {
		t.Fatal(err)
	}
	if s.char.MapID != target || s.char.PosX != 175 || s.char.PosZ != 175 || s.char.PosY != 75.9 {
		t.Fatalf("unsafe or unrelated destination: %+v", s.char)
	}
	state := game.New()
	state.SelfID = s.char.ID
	state.MapID = source
	for {
		select {
		case p := <-packets:
			if _, err := state.Apply(p); err != nil {
				t.Fatal(err)
			}
			if proto.Op(p) == proto.GXGMState {
				if state.MapID != target || state.Self.X != 175 || !state.Self.IsGM || !state.GMTurbo {
					t.Fatal("teleport did not reach client state")
				}
				if err := s.gmMove(974); err == nil || s.char.MapID != target {
					t.Fatal("unknown map accepted")
				}
				sh.entities.SpawnAreas = nil
				shard(source).entities.Portals = []MapPortal{{TargetMapID: target, TargetX: 35, TargetY: 40, TargetZ: 55}}
				if err := s.gmMove(target); err != nil {
					t.Fatal("incoming portal fallback", err)
				}
				gmResponse(t, packets)
				if s.char.PosX != 35 || s.char.PosY != 40 || s.char.PosZ != 55 {
					t.Fatal("wrong fallback position")
				}
				return
			}
		case <-time.After(3 * time.Second):
			t.Fatal("teleport packets missing")
		}
	}
}
