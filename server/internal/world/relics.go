package world

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Relics (obelisks) follow ps_game.exe. The addresses below refer to the
// EP4.5 ps_game.exe build of 1,581,568 bytes, not the PDB build cited in
// docs/original-parity.md; the routines are identified by their behaviour.
//
//   - CGameData's Obelisk.ini loader (0x0042de40): [zone_<map>] then
//     [Obelisk_<n>] = (x,y,z),(neutral, light, fury mob),owner (0 neutral,
//     1 light, 2 fury) and its guards [Mob] = (x,y,z,radius),(neutral,
//     light, fury mob),count. The key is map<<16 + n (0x0042e812).
//   - The zone update (0x004306c0) spawns the owner's relic mob after its
//     respawn time and, while the relic stands, the owner's guards around
//     their point (a zero mob ID has no guard for that owner).
//   - The relic's death (0x004a1fb1 -> 0x00430cb0): its guards are removed,
//     the killer's country owns it, 0x2102 (key, owner) goes to the map and
//     0x2103 (relic mob, old owner, new owner) to everyone; the client says
//     "<relic> is destroyed!" (462) or "Destoyed <relic>." (463).
//   - Entering a map (0x0041c891 -> 0x004316e0) sends 0x2101: count, then
//     per relic key, owner, x, z. Nothing is sent for maps without relics.
//
// Owners on the wire: 0 neutral, 1 light, 2 fury (country + 1). Ownership
// is not saved; a world start restores the ini owners, as the original.

const (
	relicListOpcode   packet.Opcode = 0x2101
	relicOwnerOpcode  packet.Opcode = 0x2102
	relicNoticeOpcode packet.Opcode = 0x2103
	relicNeutral      uint8         = 0
)

type relicGuard struct {
	x, y, z, radius float32
	mobs            [3]uint16
	count           int
}

type relic struct {
	mapID     uint16
	key       uint32
	x, y, z   float32
	mobs      [3]uint16
	initial   uint8
	owner     uint8
	standing  bool
	guards    []relicGuard
	guardMobs []uint32
}

// relicLink ties a map mob to its relic; guards follow the relic's owner.
type relicLink struct {
	relic *relic
	guard *relicGuard
}

var relics = struct {
	sync.Mutex
	byMap map[uint16][]*relic
}{byMap: map[uint16][]*relic{}}

func parseRelicTuple(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// relicTuples splits "(a,b,c),(d,e,f),g" into its groups.
func relicTuples(value string) [][]string {
	var out [][]string
	for value = strings.TrimSpace(value); value != ""; value = strings.TrimSpace(value) {
		if value[0] == '(' {
			end := strings.IndexByte(value, ')')
			if end < 0 {
				return nil
			}
			out = append(out, parseRelicTuple(value[:end+1]))
			value = strings.TrimPrefix(strings.TrimSpace(value[end+1:]), ",")
			continue
		}
		next := strings.IndexByte(value, ',')
		if next < 0 {
			out = append(out, []string{value})
			break
		}
		out = append(out, []string{strings.TrimSpace(value[:next])})
		value = value[next+1:]
	}
	return out
}

func relicFloats(parts []string, n int) ([]float32, bool) {
	if len(parts) < n {
		return nil, false
	}
	out := make([]float32, n)
	for i := range n {
		v, err := strconv.ParseFloat(parts[i], 32)
		if err != nil {
			return nil, false
		}
		out[i] = float32(v)
	}
	return out, true
}

func relicMobIDs(parts []string) ([3]uint16, bool) {
	var ids [3]uint16
	if len(parts) < 3 {
		return ids, false
	}
	for i := range 3 {
		v, err := strconv.ParseUint(parts[i], 10, 16)
		if err != nil {
			return ids, false
		}
		ids[i] = uint16(v)
	}
	return ids, true
}

// parseRelics reads the zone/obelisk/guard entries; portal and boss
// sections end the current relic so their [Mob] lines are not taken.
func parseRelics(raw []byte) ([]*relic, error) {
	var out []*relic
	zone := -1
	var current *relic
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := scanner.Text()
		if i := strings.Index(text, "//"); i >= 0 {
			text = text[:i]
		}
		text = strings.TrimSpace(text)
		if !strings.HasPrefix(text, "[") {
			continue
		}
		end := strings.IndexByte(text, ']')
		if end < 0 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(text[1:end]))
		_, value, hasValue := strings.Cut(text[end+1:], "=")
		switch {
		case strings.HasPrefix(name, "zone_"):
			n, err := strconv.Atoi(strings.TrimPrefix(name, "zone_"))
			if err != nil {
				return nil, fmt.Errorf("line %d: bad zone %q", line, name)
			}
			zone, current = n, nil
		case strings.HasPrefix(name, "obelisk_") && hasValue && zone >= 0:
			n, err := strconv.Atoi(strings.TrimPrefix(name, "obelisk_"))
			if err != nil {
				return nil, fmt.Errorf("line %d: bad obelisk %q", line, name)
			}
			t := relicTuples(value)
			pos, okPos := relicFloats(first(t, 0), 3)
			mobs, okMobs := relicMobIDs(first(t, 1))
			owner, err := strconv.Atoi(strings.Join(first(t, 2), ""))
			if !okPos || !okMobs || err != nil || owner < 0 || owner > 2 {
				return nil, fmt.Errorf("line %d: bad obelisk entry", line)
			}
			current = &relic{mapID: uint16(zone), key: uint32(zone)<<16 + uint32(n), x: pos[0], y: pos[1], z: pos[2],
				mobs: mobs, initial: uint8(owner), owner: uint8(owner)}
			out = append(out, current)
		case name == "mob" && hasValue:
			if current == nil {
				continue
			}
			t := relicTuples(value)
			pos, okPos := relicFloats(first(t, 0), 4)
			mobs, okMobs := relicMobIDs(first(t, 1))
			count, err := strconv.Atoi(strings.Join(first(t, 2), ""))
			if !okPos || !okMobs || err != nil || count < 0 {
				return nil, fmt.Errorf("line %d: bad guard entry", line)
			}
			current.guards = append(current.guards, relicGuard{x: pos[0], y: pos[1], z: pos[2], radius: pos[3], mobs: mobs, count: count})
		default:
			current = nil
			if !strings.HasPrefix(name, "obelisk_") {
				zone = -1
			}
		}
	}
	return out, scanner.Err()
}

func first(t [][]string, i int) []string {
	if i < len(t) {
		return t[i]
	}
	return nil
}

// setDef turns a mob into another definition (a relic or guard variant).
func (m *MapMob) setDef(def *db.MobDef, id uint16) {
	m.MobID = id
	if def == nil {
		return
	}
	m.Level, m.MaxHP, m.MoneyMin, m.MoneyMax = def.Level, max(def.MaxHP, 1), def.MoneyMin, def.MoneyMax
	m.Exp, m.Attack, m.AI, m.ChaseRange = def.Exp, def.Attack, def.AI, def.ChaseRange
	m.def = def
}

func relicMob(defs *db.Defs, id uint16, x, y, z float32, area *MobArea, link *relicLink) (MapMob, bool) {
	def := defs.Mobs[id]
	if def == nil {
		return MapMob{}, false
	}
	m := NewStaticMob(def, id, x, z, area, def.RespawnSeconds)
	m.PosY, m.homeY = y, y
	m.spawnMinY, m.spawnMaxY, m.spawnHeightKnown = y, y, true
	m.relic = link
	return m, true
}

// LoadRelics reads Obelisk.ini and places the relics and their guards on
// the loaded maps (after LoadSvmaps).
func LoadRelics(log *logx.Logger) error {
	raw, err := os.ReadFile(filepath.Join(gameDataDir(), "Obelisk.ini"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	list, err := parseRelics(raw)
	if err != nil {
		return fmt.Errorf("Obelisk.ini: %w", err)
	}
	defs := db.MustDefs()
	placed := 0
	relics.Lock()
	defer relics.Unlock()
	for _, r := range list {
		if !hasGrid(r.mapID) {
			continue
		}
		var mobs []MapMob
		if id := r.mobs[r.owner]; id != 0 {
			if m, ok := relicMob(defs, id, r.x, r.y, r.z, nil, &relicLink{relic: r}); ok {
				m.placeRespawn(r.mapID)
				r.standing = true
				mobs = append(mobs, m)
			}
		}
		for gi := range r.guards {
			g := &r.guards[gi]
			area := MobArea{LowerX: g.x - g.radius, LowerZ: g.z - g.radius, UpperX: g.x + g.radius, UpperZ: g.z + g.radius}
			id := g.mobs[r.owner]
			group := &mobGroup{count: g.count}
			for range g.count {
				// A guard without a mob for this owner waits dead for an owner that has one.
				spawnID := id
				if spawnID == 0 {
					spawnID = firstRelicMob(g.mobs)
				}
				m, ok := relicMob(defs, spawnID, g.x, g.y, g.z, &area, &relicLink{relic: r, guard: g})
				if !ok {
					continue
				}
				m.group = group
				if id == 0 || !r.standing {
					m.CurrentHP, m.diedAt, m.corpseGone = 0, time.Now(), true
				} else {
					m.placeRespawn(r.mapID)
					group.alive++
				}
				mobs = append(mobs, m)
			}
		}
		AddEntities(r.mapID, MapEntities{Mobs: mobs})
		relics.byMap[r.mapID] = append(relics.byMap[r.mapID], r)
		placed++
	}
	log.Printf("GX Server World relics loaded=%d placed=%d", len(list), placed)
	return nil
}

func firstRelicMob(ids [3]uint16) uint16 {
	for _, id := range ids {
		if id != 0 {
			return id
		}
	}
	return 0
}

// prepareRespawn turns a relic mob or guard into the owner's variant;
// false keeps it down (no mob for this owner, or a guard of a fallen relic).
// The shard lock is held.
func (l *relicLink) prepareRespawn(m *MapMob) bool {
	relics.Lock()
	defer relics.Unlock()
	r := l.relic
	mobs := r.mobs
	if l.guard != nil {
		if !r.standing {
			return false
		}
		mobs = l.guard.mobs
	}
	id := mobs[r.owner]
	if id == 0 {
		return false
	}
	if id != m.MobID {
		m.setDef(db.MustDefs().Mobs[id], id)
	}
	if l.guard == nil {
		r.standing = true
	}
	return true
}

func buildRelicList(list []*relic) []byte {
	w := packet.New(relicListOpcode)
	w.U8(uint8(len(list)))
	for _, r := range list {
		w.U32(r.key)
		w.U8(r.owner)
		w.F32(r.x)
		w.F32(r.z)
	}
	return w.MustFinish()
}

func buildRelicOwner(key uint32, owner uint8) []byte {
	w := packet.New(relicOwnerOpcode)
	w.U32(key)
	w.U8(owner)
	return w.MustFinish()
}

func buildRelicNotice(mobID uint16, oldOwner, newOwner uint8) []byte {
	w := packet.New(relicNoticeOpcode)
	w.U16(mobID)
	w.U8(oldOwner)
	w.U8(newOwner)
	return w.MustFinish()
}

// relicListFor is 0x2101 for a map, nil without relics.
func relicListFor(mapID uint16) []byte {
	relics.Lock()
	defer relics.Unlock()
	list := relics.byMap[realMap(mapID)]
	if len(list) == 0 || mapID != realMap(mapID) {
		return nil
	}
	return buildRelicList(list[:min(len(list), 100)])
}

// relicDestroyed is the relic's death: the killer's country takes it.
func relicDestroyed(killer *ConnectedPlayer, mapID uint16, m *MapMob, log logPrinter) {
	l := m.relic
	if l == nil || l.guard != nil {
		return
	}
	relics.Lock()
	r := l.relic
	r.standing = false
	oldOwner, newOwner := r.owner, r.owner
	if killer != nil {
		newOwner = killer.Faction() + 1
	}
	r.owner = newOwner
	key := r.key
	relics.Unlock()
	removeRelicGuards(mapID, r)
	if killer == nil {
		return // an offline caster's damage over time: no country takes it
	}
	if log != nil {
		log.Printf("GX Server World relic destroyed key=%#x mob_id=%d owner %d -> %d by character_id=%d",
			key, m.MobID, oldOwner, newOwner, killer.CharacterID)
	}
	broadcast(PlayersOnMap(mapID), buildRelicOwner(key, newOwner), "0x2102 relic owner", nil)
	broadcast(AllPlayers(), buildRelicNotice(m.MobID, oldOwner, newOwner), "0x2103 relic destroyed", nil)
}

// removeRelicGuards takes the relic's guards out (their corpses leave the
// view on the next corpse sweep); they return for the new owner once the
// relic stands again.
func removeRelicGuards(mapID uint16, r *relic) {
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := range s.entities.Mobs {
		m := &s.entities.Mobs[i]
		if m.relic == nil || m.relic.relic != r || m.relic.guard == nil || m.CurrentHP == 0 {
			continue
		}
		m.CurrentHP, m.Effects, m.aggro, m.AggroTarget = 0, nil, nil, 0
		m.diedAt, m.corpseGone = now.Add(-mobCorpseDelay), false
		if g := m.group; g != nil {
			g.alive--
			g.respawnAt = now.Add(m.respawnDelay)
		}
	}
}

// stationary: towers (relics, their pillar guards) have no walk or chase
// step (NormalStep and ChaseStep 0) and never leave their spot; chaseStep's
// minimum of 1 must not move them.
func (m *MapMob) stationary() bool {
	return m.def != nil && m.def.ChaseStep == 0 && m.def.NormalStep == 0
}

// mobFriendly: a mob of the attacker's country (relics, their guards) cannot
// be attacked.
func mobFriendly(def *db.MobDef, faction uint8) bool {
	return def != nil && def.Country < 2 && def.Country == faction
}
