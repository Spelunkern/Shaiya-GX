package world

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/gm"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const gxGMState packet.Opcode = 0x7F01
const gmMaxSpawn = 100

func (s *Session) gmReply(message string) {
	if s.char == nil {
		return
	}
	w := packet.New(gxGMState)
	w.U32(s.char.ID)
	role, turbo := uint8(0), uint8(0)
	if s.isAdmin {
		role = 1
		if s.gmTurbo {
			turbo = 1
		}
	}
	w.U8(role)
	w.U8(turbo)
	w.U16(uint16(len(message)))
	w.Bytes([]byte(message))
	s.send(w.MustFinish(), "GX GM state")
}

func gmNumbers(args []string, bounds ...uint64) ([]uint64, error) {
	if len(args) != len(bounds) {
		return nil, fmt.Errorf("argument count")
	}
	out := make([]uint64, len(args))
	for i, a := range args {
		n, err := strconv.ParseUint(a, 10, 32)
		if err != nil || n == 0 || n > bounds[i] {
			return nil, fmt.Errorf("argument value")
		}
		out[i] = n
	}
	return out, nil
}

// Known commands are consumed even when refused; they never become public chat.
func (s *Session) handleGMCommand(text string) bool {
	command, args, known := gm.Parse(text)
	if !known {
		return false
	}
	if s.char == nil {
		return true
	}
	if !s.isAdmin {
		s.gmReply("GM: acceso denegado.")
		return true
	}
	if tradeFor(s.char.ID) != nil { // Inventory must not change underneath an exchange.
		s.gmReply("GM: termina el intercambio antes de usar comandos.")
		return true
	}
	var err error
	switch command {
	case "/turbo":
		if len(args) != 0 {
			s.gmReply("Uso: /turbo")
			break
		}
		s.gmTurbo = !s.gmTurbo
		state := "desactivado"
		if s.gmTurbo {
			state = "activado"
		}
		s.gmReply("GM: turbo " + state + ".")
	case "/iclear":
		if len(args) != 0 {
			s.gmReply("Uso: /iclear")
			break
		}
		err = s.gmClearInventory()
	case "/getitem":
		n, e := gmNumbers(args, 255, 255, 5*24*255)
		if e != nil {
			s.gmReply("Uso: /getitem tipo id cantidad")
			break
		}
		err = s.gmGiveItem(uint8(n[0]), uint8(n[1]), int(n[2]))
	case "/mmake":
		n, e := gmNumbers(args, 65535, gmMaxSpawn)
		if e != nil {
			s.gmReply("Uso: /mmake mobid cantidad (1-100)")
			break
		}
		err = s.gmSpawn(uint16(n[0]), int(n[1]), 0)
	case "/nmake":
		n, e := gmNumbers(args, npcTypeCount, 65535)
		if e != nil {
			s.gmReply("Uso: /nmake tipo id")
			break
		}
		err = s.gmSpawn(uint16(n[1]), 1, uint8(n[0]))
	case "/cmove":
		if len(args) != 1 {
			s.gmReply("Uso: /cmove mapid")
			break
		}
		n, e := strconv.ParseUint(args[0], 10, 16)
		if e != nil {
			s.gmReply("Uso: /cmove mapid")
			break
		}
		err = s.gmMove(uint16(n))
	}
	if err != nil {
		s.gmReply("GM: " + err.Error())
	}
	if s.log != nil {
		s.log.Printf("GX GM account_id=%d character_id=%d command=%q error=%v", s.accountID, s.char.ID, text, err)
	}
	return true
}

// Clear every item owned by this character, including equipped items; never
// the account warehouse or another character. One DELETE is atomic.
func (s *Session) gmClearInventory() error {
	ids := make([]int64, 0, len(s.items))
	for _, it := range s.items {
		ids = append(ids, it.UID)
	}
	if err := db.DeleteCharacterItems(s.char.ID, ids...); err != nil {
		return err
	}
	old := s.items
	s.items = nil
	s.dismount()
	for i := range old {
		s.send(buildRemoveItem(&old[i], true), "GM inventory clear")
	}
	s.equipmentChanged([]uint8{0, 1, 2, 3, 4, 5, 6, 7})
	s.sendStatPackets()
	s.gmReply(fmt.Sprintf("GM: inventario vaciado (%d objetos).", len(old)))
	return nil
}

// planGMGrant keeps the entire request atomic, including quantities above 255.
func planGMGrant(items []db.Item, def *db.ItemDef, quantity int) ([]db.Item, []db.Item, []db.Item, error) {
	if def == nil || quantity <= 0 || quantity > 5*24*255 {
		return nil, nil, nil, fmt.Errorf("objeto o cantidad invalida")
	}
	sim := append([]db.Item(nil), items...)
	tpl := db.Item{Type: def.Type, TypeID: def.TypeID, Quality: def.Quality, MakeType: "G"}
	var changed, added []db.Item
	stack := def.MaxStack()
	for quantity > 0 {
		if i := findStackable(sim, &tpl, stack); i >= 0 {
			n := min(quantity, int(stack-sim[i].Count))
			sim[i].Count += uint8(n)
			quantity -= n
			changed = append(changed, sim[i])
			continue
		}
		free := freeInventorySlots(sim)
		if len(free) == 0 {
			return nil, nil, nil, fmt.Errorf("no hay espacio para toda la cantidad")
		}
		it := tpl
		it.UID = db.NewItemUID()
		it.MakeTime = time.Now()
		it.Bag, it.Slot = free[0][0], free[0][1]
		it.Count = uint8(min(quantity, int(stack)))
		quantity -= int(it.Count)
		sim = append(sim, it)
		added = append(added, it)
	}
	return sim, changed, added, nil
}

func (s *Session) gmGiveItem(itemType, id uint8, quantity int) error {
	def := db.MustDefs().Item(itemType, id)
	sim, changed, added, err := planGMGrant(s.items, def, quantity)
	if err != nil {
		return err
	}
	if err = db.ApplyGMItemGrant(s.char.ID, changed, added); err != nil {
		return err
	}
	s.items = sim
	for i := range changed {
		s.send(buildAddItem(&changed[i]), "GM item stack")
	}
	for i := range added {
		s.send(buildAddItem(&added[i]), "GM item grant")
	}
	if s.player != nil {
		s.player.UpdateItems(s.items)
	}
	s.gmReply(fmt.Sprintf("GM: recibiste %d x %s (%d/%d).", quantity, def.Name, itemType, id))
	return nil
}

// Spawn positions form expanding rings; occupied points and blocked cells
// are rejected, including entities created by earlier commands.
func gmSpawnPositions(mapID uint16, x, z float32, count int, entities *MapEntities) ([][2]float32, error) {
	size, exists := mapSize(mapID)
	if !exists || size <= 0 {
		return nil, fmt.Errorf("mapa no disponible")
	}
	var result [][2]float32
	for ring := 1; ring <= 16 && len(result) < count; ring++ {
		radius := float64(3 * ring)
		points := max(6, 6*ring)
		for j := 0; j < points && len(result) < count; j++ {
			angle := 2 * math.Pi * float64(j) / float64(points)
			px, pz := x+float32(radius*math.Cos(angle)), z+float32(radius*math.Sin(angle))
			if px < 1 || pz < 1 || px >= size-1 || pz >= size-1 || !IsMapWalkable(mapID, px, pz) {
				continue
			}
			occupied := false
			for _, n := range entities.NPCs {
				if DistanceXZ(px, pz, n.PosX, n.PosZ) < 2 {
					occupied = true
					break
				}
			}
			if !occupied {
				for _, m := range entities.Mobs {
					if m.CurrentHP > 0 && DistanceXZ(px, pz, m.PosX, m.PosZ) < 2 {
						occupied = true
						break
					}
				}
			}
			if !occupied {
				for _, p := range result {
					if DistanceXZ(px, pz, p[0], p[1]) < 2 {
						occupied = true
						break
					}
				}
			}
			if !occupied {
				result = append(result, [2]float32{px, pz})
			}
		}
	}
	if len(result) != count {
		return nil, fmt.Errorf("no hay espacio libre cercano")
	}
	return result, nil
}

func (s *Session) gmSpawn(id uint16, count int, npcType uint8) error {
	if count < 1 || count > gmMaxSpawn {
		return fmt.Errorf("cantidad fuera de rango")
	}
	var nd *npcDef
	var md *db.MobDef
	if npcType != 0 {
		if count != 1 {
			return fmt.Errorf("nmake invoca exactamente un NPC")
		}
		nd = findNpcDef(npcType, id)
		if nd == nil {
			return fmt.Errorf("NPC desconocido; usa /nmake tipo id (1 1 = merchant 1)")
		}
	} else {
		md = db.MustDefs().Mobs[id]
		if md == nil {
			return fmt.Errorf("mob desconocido")
		}
	}
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	positions, err := gmSpawnPositions(s.char.MapID, s.char.PosX, s.char.PosZ, count, &sh.entities)
	if err != nil {
		sh.mu.Unlock()
		return err
	}
	var npcs []MapNpc
	var mobs []MapMob
	for _, pos := range positions {
		if npcType != 0 {
			n := MapNpc{ID: nextNpcID.Add(1) - 1, Type: nd.Type, TypeID: nd.TypeID, PosX: pos[0], PosY: s.char.PosY, PosZ: pos[1], cell: cellFor(pos[0], pos[1]), hasCell: true}
			sh.entities.NPCs = append(sh.entities.NPCs, n)
			npcs = append(npcs, n)
		} else {
			m := NewStaticMob(md, id, pos[0], pos[1], nil, 0)
			sh.entities.Mobs = append(sh.entities.Mobs, m)
			mobs = append(mobs, m)
		}
	}
	sh.mu.Unlock()
	for _, n := range npcs {
		for _, p := range PlayersViewing(s.char.MapID, n.cell) {
			viewEnter(p, viewNpc, n.ID, buildNpcEnter(&n), "GM NPC spawn")
		}
	}
	for _, m := range mobs {
		for _, p := range PlayersViewing(s.char.MapID, m.cell) {
			viewEnter(p, viewMob, m.ID, buildMobEnter(&m, false), "GM mob spawn")
		}
	}
	s.gmReply(fmt.Sprintf("GM: invocados %d (id %d).", count, id))
	return nil
}

func (s *Session) gmMove(mapID uint16) error {
	size, ok := mapSize(mapID)
	if !ok || size <= 0 || isInstanceZone(mapID) {
		return fmt.Errorf("mapa desconocido")
	}
	// The server owns bind heights, not terrain geometry. Choose the closest
	// bind to the equivalent relative X/Z rather than guessing an unsafe Y.
	x, z := s.char.PosX, s.char.PosZ
	if old, ok := mapSize(s.char.MapID); ok && old > 0 {
		x = x / old * size
		z = z / old * size
	}
	entities := EntitiesOnMap(mapID)
	var target *SpawnBox
	best := float32(math.MaxFloat32)
	for _, a := range entities.SpawnAreas {
		for i := range a.Boxes {
			b := a.Boxes[i]
			bx, bz := (b.LowerX+b.UpperX)/2, (b.LowerZ+b.UpperZ)/2
			d := DistanceXZ(x, z, bx, bz)
			if d < best && IsMapWalkable(mapID, bx, bz) {
				copy := b
				target = &copy
				best = d
			}
		}
	}
	if target == nil {
		// Dungeons can have no rebirth boxes; use an authored incoming portal
		// or gatekeeper destination, with its explicit server-side height.
		consider := func(px, py, pz float32) {
			d := DistanceXZ(x, z, px, pz)
			if d < best && px >= 0 && pz >= 0 && px < size && pz < size && IsMapWalkable(mapID, px, pz) {
				target = &SpawnBox{LowerX: px, UpperX: px, LowerY: py - .9, LowerZ: pz, UpperZ: pz}
				best = d
			}
		}
		for _, id := range shardIDs() {
			for _, portal := range EntitiesOnMap(id).Portals {
				if portal.TargetMapID == mapID {
					consider(portal.TargetX, portal.TargetY, portal.TargetZ)
				}
			}
		}
		for _, list := range npcDefs {
			for _, n := range list {
				if n.Type == npcTypeGatekeeper {
					for _, g := range n.Gates {
						if g.MapID == mapID {
							consider(g.X, g.Y, g.Z)
						}
					}
				}
			}
		}
	}
	if target == nil {
		return fmt.Errorf("mapa sin destino de llegada seguro")
	}
	b := target
	s.cast = nil
	s.autoAttack = false
	s.useNpc(0)
	s.applyTeleport(mapID, (b.LowerX+b.UpperX)/2, b.LowerY+.9, (b.LowerZ+b.UpperZ)/2, "GM cmove", true)
	if realMap(s.char.MapID) != mapID {
		return fmt.Errorf("destino no disponible")
	}
	s.gmReply(fmt.Sprintf("GM: mapa %d.", mapID))
	return nil
}
