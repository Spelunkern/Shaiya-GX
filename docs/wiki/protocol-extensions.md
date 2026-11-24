# Protocol: GX protocol extensions

> Generated from Go source by `tools/generate_server_reference.py`; do not edit by hand.

[All opcodes](server-opcodes.md) · [Wire conventions](server-protocol.md) · [Server architecture](server-architecture.md)

Client codecs operate on opcode + body; server packet.Writer emits length + opcode + body. Excerpts preserve the respective construction API. Go wire methods specify widths, not host struct alignment. `Count` is special: see the stack-width compatibility rules. Loops, conditions, helper calls and fixed string sizes are significant; do not flatten them into a single fixed-size struct.

## Opcode 7f01

**`0x7F01` — GXGMState**

Declarations: [proto.GXGMState](../../internal/proto/gm.go#L11).

Source notes: GXGMState is a GX-only server response: self ID:u32, GM:u8, turbo:u8, message byte length:u16, UTF-8 message. Map teleport requests use GXGMMapTeleport; other requests use chat commands.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [gmReply (server/internal/world/gm.go)](../../server/internal/world/gm.go#L22)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1279)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>gmReply — server/internal/world/gm.go</summary>

[Source definition](../../server/internal/world/gm.go#L22)

```go
func (s *Session) gmReply(message string) {
	if s.char == nil {
		return
	}
	w := packet.New(gxGMState)
	w.U32(s.char.ID)
	role, turbo := uint8(0), uint8(0)
	if s.isAdmin {
		role = 1
		if s.turboActive() {
			turbo = 1
		}
	}
	w.U8(role)
	w.U8(turbo)
	w.U16(uint16(len(message)))
	w.Bytes([]byte(message))
	s.send(w.MustFinish(), "GX GM state")
}
```

</details>

<details><summary>DecodeGMState — internal/proto/gm.go</summary>

[Source definition](../../internal/proto/gm.go#L22)

```go
func DecodeGMState(p []byte) (GMState, error) {
	if len(p) < 10 {
		return GMState{}, fmt.Errorf("short GX GM state")
	}
	n := int(binary.LittleEndian.Uint16(p[8:10]))
	if len(p) != 10+n || p[6] > 1 || p[7] > 1 || (p[6] == 0 && p[7] != 0) || !utf8.Valid(p[10:]) {
		return GMState{}, fmt.Errorf("invalid GX GM state")
	}
	return GMState{ID: binary.LittleEndian.Uint32(p[2:6]), GM: p[6] == 1, Turbo: p[7] == 1, Message: string(p[10:])}, nil
}
```

```go
type GMState struct {
	ID        uint32
	GM, Turbo bool
	Message   string
}
```

</details>

## Opcode 7f02

**`0x7F02` — GXVisibility**

Declarations: [proto.GXVisibility](../../internal/proto/gm.go#L14).

Source notes: GXVisibility: character ID:u32, concealed:u8. Sent only to authorized viewers.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [visibilityPacket (server/internal/world/visibility.go)](../../server/internal/world/visibility.go#L27)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1261)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>visibilityPacket — server/internal/world/visibility.go</summary>

[Source definition](../../server/internal/world/visibility.go#L27)

```go
func (p *ConnectedPlayer) visibilityPacket() []byte {
	w := packet.New(gxVisibility)
	w.U32(p.CharacterID)
	hidden := uint8(0)
	if p.hidden.Load() {
		hidden = 1
	}
	w.U8(hidden)
	return w.MustFinish()
}
```

</details>

## Opcode 7f03

**`0x7F03` — GXForgeQuote**

Declarations: [packet.GXForgeQuote](../../server/internal/packet/packet.go#L342), [proto.GXForgeQuote](../../internal/proto/forge_quote.go#L6).

Source notes: GXForgeQuote is a read-only preview; execution still validates live inventory.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L294)

**S→C serializers:** [handleForgeQuote (server/internal/world/forge_quote.go)](../../server/internal/world/forge_quote.go#L105)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>handleForgeQuote — server/internal/world/forge_quote.go</summary>

[Source definition](../../server/internal/world/forge_quote.go#L105)

```go
func (s *Session) handleForgeQuote(p []byte) {
	q, err := proto.DecodeForgeQuoteRequest(p)
	if err != nil {
		return
	}
	out := s.forgeQuote(q)
	w := packet.New(packet.GXForgeQuote)
	w.U32(out.Serial)
	w.U8(out.Kind)
	w.Bool(out.Valid)
	w.U32(out.Rate)
	w.U32(out.Gold)
	w.U8(out.Lapis)
	w.U32(out.Material)
	s.send(w.MustFinish(), "GX forge quote")
}
```

</details>

<details><summary>ForgeQuotePacket — internal/proto/forge_quote.go</summary>

[Source definition](../../internal/proto/forge_quote.go#L28)

```go
func ForgeQuotePacket(q ForgeQuoteRequest) []byte {
	w := NewWriter(GXForgeQuote)
	w.U32(q.Serial)
	w.U8(q.Kind)
	for _, k := range [][2]uint8{q.Item, q.Lapis, q.Material, q.Hammer} {
		w.Raw(k[:])
	}
	w.U8(q.Count)
	return w.Bytes()
}
```

```go
type ForgeQuoteRequest struct {
	Serial                        uint32
	Kind                          uint8
	Item, Lapis, Material, Hammer [2]uint8
	Count                         uint8
}
```

</details>

## Opcode 7f04

**`0x7F04` — GXQuestNPCs**

Declarations: [proto.GXQuestNPCs](../../internal/proto/quest_map.go#L9).

Source notes: GXQuestNPCs supplies map metadata, not visible/interactive world entities.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [questNPCFrames (server/internal/world/quest_map.go)](../../server/internal/world/quest_map.go#L50)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L282)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>questNPCFrames — server/internal/world/quest_map.go</summary>

[Source definition](../../server/internal/world/quest_map.go#L50)

```go
func questNPCFrames(mapID uint16, rows []proto.QuestNPC) [][]byte {
	// length/opcode 4 + map/flags/count 5 + 11 bytes per NPC.
	const batch = proto.QuestNPCChunkLimit
	var frames [][]byte
	for start := 0; ; {
		end := min(start+batch, len(rows))
		flags := uint8(0)
		if start == 0 {
			flags |= proto.QuestNPCBegin
		}
		if end == len(rows) {
			flags |= proto.QuestNPCEnd
		}
		w := packet.New(packet.Opcode(proto.GXQuestNPCs))
		w.U16(mapID)
		w.U8(flags)
		w.U16(uint16(end - start))
		for _, n := range rows[start:end] {
			w.U8(n.Type)
			w.U16(n.TypeID)
			w.F32(n.X)
			w.F32(n.Z)
		}
		frames = append(frames, w.MustFinish())
		if end == len(rows) {
			return frames
		}
		start = end
	}
}
```

</details>

## Opcode 7f05

**`0x7F05` — GXMobEnter**

Declarations: [proto.GXMobEnter](../../internal/proto/world.go#L257).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1293), [applyEntitiesPacket (internal/game/state_entities_packets.go)](../../internal/game/state_entities_packets.go#L124)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeMobEnter — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L272)

```go
func DecodeMobEnter(p []byte) (m MobSpawn, err error) {
	r := NewReader(p)
	m.ID, m.New, m.MobID, m.X = r.U32(), r.Bool(), r.U16(), r.F32()
	m.HeightKnown = len(p) >= 2 && Opcode(binary.LittleEndian.Uint16(p[:2])) == GXMobEnter
	if m.HeightKnown {
		m.Y = r.F32()
	}
	m.Z = r.F32()
	if m.HeightKnown && r.Remaining() > 0 {
		if r.Remaining() != 2 {
			return m, fmt.Errorf("invalid GX mob metadata length")
		}
		m.Country = r.U8()
		height := r.U8()
		if m.Country > 2 || height > 1 {
			return m, fmt.Errorf("invalid GX mob metadata")
		}
		m.CountryKnown = true
		m.HeightKnown = height == 1
	}
	return m, r.Err()
}
```

```go
type MobSpawn struct {
	Country      uint8
	CountryKnown bool
	ID           uint32
	New          bool
	MobID        uint16
	X, Y, Z      float32
	HeightKnown  bool
}
```

</details>

## Opcode 7f06

**`0x7F06` — GXMobMove**

Declarations: [proto.GXMobMove](../../internal/proto/world.go#L258).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1293), [applyEntitiesPacket (internal/game/state_entities_packets.go)](../../internal/game/state_entities_packets.go#L130)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeMobMove — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L304)

```go
func DecodeMobMove(p []byte) (m MobMotion, err error) {
	r := NewReader(p)
	m.ID, m.Motion, m.X = r.U32(), r.U8(), r.F32()
	m.HeightKnown = len(p) >= 2 && Opcode(binary.LittleEndian.Uint16(p[:2])) == GXMobMove
	if m.HeightKnown {
		m.Y = r.F32()
	}
	m.Z = r.F32()
	if m.HeightKnown {
		m.DurationMS = r.U32()
	}
	return m, r.Err()
}
```

```go
type MobMotion struct {
	ID          uint32
	Motion      uint8
	X, Y, Z     float32
	HeightKnown bool
	DurationMS  uint32
}
```

</details>

## Opcode 7f07

**`0x7F07` — GXPhysicsClock**

Declarations: [proto.GXPhysicsClock](../../internal/proto/world.go#L255).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendPhysicsClock (server/internal/world/player_movement.go)](../../server/internal/world/player_movement.go#L471)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1251)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendPhysicsClock — server/internal/world/player_movement.go</summary>

[Source definition](../../server/internal/world/player_movement.go#L471)

```go
func (s *Session) sendPhysicsClock() {
	if s.char == nil {
		return
	}
	w := packet.New(packet.Opcode(proto.GXPhysicsClock))
	w.U16(realMap(s.char.MapID))
	w.U64(math.Float64bits(mapgeometry.PhysicsSeconds()))
	s.send(w.MustFinish(), "GX physics clock")
}
```

</details>

## Opcode 7f08

**`0x7F08` — GXPlayerCorrection**

Declarations: [proto.GXPlayerCorrection](../../internal/proto/player_physics.go#L8).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [correctMovement (server/internal/world/player_movement.go)](../../server/internal/world/player_movement.go#L251)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1241)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>correctMovement — server/internal/world/player_movement.go</summary>

[Source definition](../../server/internal/world/player_movement.go#L251)

```go
func (s *Session) correctMovement(reason string) {
	m := &s.movement
	if s.log != nil {
		s.log.Debugf("movement correction map=%d reason=%s", s.char.MapID, reason)
	}
	w := packet.New(packet.Opcode(proto.GXPlayerCorrection))
	w.U32(s.char.ID)
	w.F32(s.char.PosX)
	w.F32(s.char.PosY)
	w.F32(s.char.PosZ)
	w.U8(m.motion)
	w.U16(m.direction)
	w.Bool(m.jumping)
	w.F32(m.jumpFall)
	w.F32(m.fall)
	w.F32(m.slope.Frames)
	s.send(w.MustFinish(), "movement correction: "+reason)
}
```

</details>

<details><summary>DecodePlayerCorrection — internal/proto/player_physics.go</summary>

[Source definition](../../internal/proto/player_physics.go#L20)

```go
func DecodePlayerCorrection(p []byte) (c PlayerCorrection, err error) {
	if len(p) != 34 || p[21] > 1 {
		return c, fmt.Errorf("invalid player physics correction")
	}
	r := NewReader(p)
	c.ID = r.U32()
	c.X, c.Y, c.Z = r.F32(), r.F32(), r.F32()
	c.Motion, c.Direction, c.Jumping = r.U8(), r.U16(), r.Bool()
	c.JumpFall, c.Fall = r.F32(), r.F32()
	c.SlopeFrames = r.F32()
	if c.SlopeFrames < 0 {
		return PlayerCorrection{}, fmt.Errorf("negative slope pressure")
	}
	for _, v := range []float32{c.X, c.Y, c.Z, c.JumpFall, c.Fall, c.SlopeFrames} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return PlayerCorrection{}, fmt.Errorf("non-finite physics correction")
		}
	}
	return c, r.Err()
}
```

```go
type PlayerCorrection struct {
	ID             uint32
	X, Y, Z        float32
	Motion         uint8
	Direction      uint16
	Jumping        bool
	JumpFall, Fall float32
	SlopeFrames    float32
}
```

</details>

## Opcode 7f09

**`0x7F09` — GXGMMapTeleport**

Declarations: [proto.GXGMMapTeleport](../../internal/proto/gm_map.go#L10).

Source notes: GXGMMapTeleport requests an exact map-image coordinate. The server owns Y.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L292)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>GMMapTeleportPacket — internal/proto/gm_map.go</summary>

[Source definition](../../internal/proto/gm_map.go#L12)

```go
func GMMapTeleportPacket(mapID uint16, u, v float32) []byte {
	w := NewWriter(GXGMMapTeleport)
	w.U16(mapID)
	w.F32(u)
	w.F32(v)
	return w.Bytes()
}
```

</details>

## Opcode 7f0a

**`0x7F0A` — GXOJLimits**

Declarations: [proto.GXOJLimits](../../internal/proto/oj_limits.go#L9).

Source notes: GXOJLimits publishes static equipment recreation limits from the server DB.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendOJLimits (server/internal/world/oj_limits.go)](../../server/internal/world/oj_limits.go#L32)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L271)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendOJLimits — server/internal/world/oj_limits.go</summary>

[Source definition](../../server/internal/world/oj_limits.go#L32)

```go
func (s *Session) sendOJLimits() {
	limits := equipmentOJLimits(db.MustDefs())
	for len(limits) > 0 {
		n := min(len(limits), 256)
		payload := proto.OJLimitsPacket(limits[:n])
		w := packet.New(packet.Opcode(proto.GXOJLimits))
		w.Bytes(payload[2:])
		s.send(w.MustFinish(), "GX equipment OJ limits")
		limits = limits[n:]
	}
}
```

</details>

<details><summary>OJLimitsPacket — internal/proto/oj_limits.go</summary>

[Source definition](../../internal/proto/oj_limits.go#L13)

```go
func OJLimitsPacket(limits []OJLimit) []byte {
	w := NewWriter(GXOJLimits)
	w.U16(uint16(len(limits)))
	for _, v := range limits {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Stats)
		w.U8(v.Value)
	}
	return w.Bytes()
}
```

```go
type OJLimit struct{ Type, TypeID, Stats, Value uint8 }
```

</details>

<details><summary>DecodeOJLimits — internal/proto/oj_limits.go</summary>

[Source definition](../../internal/proto/oj_limits.go#L24)

```go
func DecodeOJLimits(p []byte) ([]OJLimit, error) {
	if len(p) < 4 || Op(p) != GXOJLimits || len(p) != 4+4*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid OJ limits")
	}
	out := make([]OJLimit, 0, (len(p)-4)/4)
	for i := 4; i < len(p); i += 4 {
		v := OJLimit{p[i], p[i+1], p[i+2], p[i+3]}
		if v.Type == 0 || v.TypeID == 0 || v.Stats > 9 || v.Value > 99 || (v.Stats == 0) != (v.Value == 0) {
			return nil, fmt.Errorf("invalid OJ limit")
		}
		out = append(out, v)
	}
	return out, nil
}
```

</details>

## Opcode 7f0b

**`0x7F0B` — GXChestCloakFits**

Declarations: [proto.GXChestCloakFits](../../internal/proto/cloak_fits.go#L10).

Source notes: GXChestCloakFits supplies the native chest-dependent cloak fit. It is an additive cosmetic catalog; existing packet layouts and game rules are unchanged.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendChestCloakFits (server/internal/world/cloak_fits.go)](../../server/internal/world/cloak_fits.go#L28)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L254)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendChestCloakFits — server/internal/world/cloak_fits.go</summary>

[Source definition](../../server/internal/world/cloak_fits.go#L28)

```go
func (s *Session) sendChestCloakFits() {
	rows := chestCloakFits(db.MustDefs())
	for len(rows) > 0 {
		n := min(len(rows), 256)
		payload := proto.ChestCloakFitsPacket(rows[:n])
		w := packet.New(packet.Opcode(proto.GXChestCloakFits))
		w.Bytes(payload[2:])
		s.send(w.MustFinish(), "GX chest cloak fits")
		rows = rows[n:]
	}
}
```

</details>

<details><summary>ChestCloakFitsPacket — internal/proto/cloak_fits.go</summary>

[Source definition](../../internal/proto/cloak_fits.go#L14)

```go
func ChestCloakFitsPacket(rows []ChestCloakFit) []byte {
	w := NewWriter(GXChestCloakFits)
	w.U16(uint16(len(rows)))
	for _, v := range rows {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Fit)
	}
	return w.Bytes()
}
```

```go
type ChestCloakFit struct{ Type, TypeID, Fit uint8 }
```

</details>

<details><summary>DecodeChestCloakFits — internal/proto/cloak_fits.go</summary>

[Source definition](../../internal/proto/cloak_fits.go#L24)

```go
func DecodeChestCloakFits(p []byte) ([]ChestCloakFit, error) {
	if len(p) < 4 || Op(p) != GXChestCloakFits || len(p) != 4+3*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid chest cloak fits")
	}
	rows := make([]ChestCloakFit, 0, (len(p)-4)/3)
	for i := 4; i < len(p); i += 3 {
		v := ChestCloakFit{p[i], p[i+1], p[i+2]}
		if v.Type == 0 || v.TypeID == 0 || v.Fit > 7 {
			return nil, fmt.Errorf("invalid chest cloak fit")
		}
		rows = append(rows, v)
	}
	return rows, nil
}
```

</details>

## Opcode 7f0c

**`0x7F0C` — GXFallDamage**

Declarations: [proto.GXFallDamage](../../internal/proto/fall_damage.go#L7).

Source notes: GXFallDamage is an additive server notification. Health still uses 0x0521. No client-authored fall height or damage report is accepted by the server.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L914)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeFallDamage — internal/proto/fall_damage.go</summary>

[Source definition](../../internal/proto/fall_damage.go#L13)

```go
func DecodeFallDamage(p []byte) (FallDamage, error) {
	return decodeEnvironmentalDamage(p, "fall")
}
```

```go
type FallDamage = EnvironmentalDamage
```

</details>

## Opcode 7f0d

**`0x7F0D` — GXLadder**

Declarations: [proto.GXLadder](../../internal/proto/ladder.go#L7).

Source notes: GXLadder is additive. Requests contain guide index + mode; notifications also identify the actor. Native movement positions and flags retain their layouts.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L36)

**S→C serializers:** [buildLadderState (server/internal/world/ladders.go)](../../server/internal/world/ladders.go#L11)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L886)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildLadderState — server/internal/world/ladders.go</summary>

[Source definition](../../server/internal/world/ladders.go#L11)

```go
func buildLadderState(id uint32, index uint16, mode uint8) []byte {
	w := packet.New(packet.Opcode(proto.GXLadder))
	w.U32(id)
	w.U16(index)
	w.U8(mode)
	return w.MustFinish()
}
```

</details>

<details><summary>LadderPacket — internal/proto/ladder.go</summary>

[Source definition](../../internal/proto/ladder.go#L15)

```go
func LadderPacket(index uint16, mode uint8) []byte {
	w := NewWriter(GXLadder)
	w.U16(index)
	w.U8(mode)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeLadderState — internal/proto/ladder.go</summary>

[Source definition](../../internal/proto/ladder.go#L21)

```go
func DecodeLadderState(p []byte) (v LadderState, err error) {
	if len(p) != 9 {
		return v, fmt.Errorf("ladder state: expected 9 bytes, got %d", len(p))
	}
	r := NewReader(p)
	v.ID = r.U32()
	v.Index = r.U16()
	v.Mode = r.U8()
	if v.Mode > 3 {
		return v, fmt.Errorf("invalid ladder mode")
	}
	return v, r.Err()
}
```

```go
type LadderState struct {
	ID    uint32
	Index uint16
	Mode  uint8
}
```

</details>

## Opcode 7f0e

**`0x7F0E` — GXSkillFailure**

Declarations: [proto.GXSkillFailure](../../internal/proto/skill_failure.go#L6).

Source notes: GXSkillFailure supplies a native SysMsg ID without inventing native result codes.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendSkillMessage (server/internal/world/skill_range.go)](../../server/internal/world/skill_range.go#L50)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L880)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendSkillMessage — server/internal/world/skill_range.go</summary>

[Source definition](../../server/internal/world/skill_range.go#L50)

```go
func (s *Session) sendSkillMessage(skillID, message uint16) {
	w := packet.New(packet.Opcode(proto.GXSkillFailure))
	w.U16(skillID)
	w.U16(message)
	s.send(w.MustFinish(), "localized skill failure")
}
```

</details>

<details><summary>DecodeSkillFailure — internal/proto/skill_failure.go</summary>

[Source definition](../../internal/proto/skill_failure.go#L15)

```go
func DecodeSkillFailure(p []byte) (v SkillFailure, err error) {
	if len(p) != 6 {
		return v, fmt.Errorf("invalid GX skill failure length")
	}
	r := NewReader(p)
	v.Skill, v.Message = r.U16(), r.U16()
	return v, r.Err()
}
```

```go
type SkillFailure struct{ Skill, Message uint16 }
```

</details>

## Opcode 7f0f

**`0x7F0F` — GXTargetEffectsQuery**

Declarations: [proto.GXTargetEffectsQuery](../../internal/proto/target_effects.go#L11).

Source notes: GX requests a selected target's native snapshot; no native request is claimed.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L32)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>TargetEffectsQuery — internal/proto/target_effects.go</summary>

[Source definition](../../internal/proto/target_effects.go#L30)

```go
func TargetEffectsQuery(kind uint8, id uint32) []byte {
	w := NewWriter(GXTargetEffectsQuery)
	w.U8(kind)
	w.U32(id)
	return w.Bytes()
}
```

</details>

## Opcode 7f10

**`0x7F10` — GXVehicleState**

Declarations: [proto.GXVehicleState](../../internal/proto/mount_state.go#L12).

Source notes: Native requests and invitation messages. 0x021C is also the server's passenger attachment notification; direction determines its layout.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildVehicleState (server/internal/world/vehicle.go)](../../server/internal/world/vehicle.go#L359)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L432)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildVehicleState — server/internal/world/vehicle.go</summary>

[Source definition](../../server/internal/world/vehicle.go#L359)

```go
func buildVehicleState(v proto.VehicleState) []byte {
	raw := proto.VehicleStatePacket(v)
	w := packet.New(packet.Opcode(proto.GXVehicleState))
	w.Bytes(raw[2:])
	return w.MustFinish()
}
```

</details>

<details><summary>VehicleStatePacket — internal/proto/mount_state.go</summary>

[Source definition](../../internal/proto/mount_state.go#L31)

```go
func VehicleStatePacket(v VehicleState) []byte {
	w := NewWriter(GXVehicleState)
	w.U32(v.ID)
	w.U8(v.State)
	w.U8(v.Race)
	w.U8(v.Model)
	w.U8(v.Speed)
	w.U8(v.Seats)
	w.U32(v.Driver)
	w.U32(v.Passenger)
	w.U32(v.CastMS)
	w.U32(v.CooldownMS)
	return w.Bytes()
}
```

```go
type VehicleState struct {
	ID                               uint32
	State, Race, Model, Speed, Seats uint8
	Driver, Passenger                uint32
	CastMS, CooldownMS               uint32
}
```

</details>

<details><summary>DecodeVehicleState — internal/proto/mount_state.go</summary>

[Source definition](../../internal/proto/mount_state.go#L45)

```go
func DecodeVehicleState(p []byte) (VehicleState, error) {
	if len(p) != 27 || Op(p) != GXVehicleState {
		return VehicleState{}, fmt.Errorf("invalid vehicle state size")
	}
	r := NewReader(p)
	v := VehicleState{ID: r.U32(), State: r.U8(), Race: r.U8(), Model: r.U8(), Speed: r.U8(), Seats: r.U8(), Driver: r.U32(), Passenger: r.U32(), CastMS: r.U32(), CooldownMS: r.U32()}
	if v.State > VehiclePassenger || v.Race > 3 || (v.State >= VehicleDriver && (v.Speed < 4 || v.Speed > 5 || v.Seats < 1 || v.Seats > 2 || v.Driver == 0)) {
		return v, fmt.Errorf("invalid vehicle state")
	}
	return v, r.Err()
}
```

</details>

## Opcode 7f11

**`0x7F11` — GXSkillDash**

Declarations: [proto.GXSkillDash](../../internal/proto/skill_dash.go#L11).

Source notes: GXSkillDash is a server-authored Stinger trajectory. Native skill and move packets keep their layouts. Clients never submit dash destinations or speed.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildSkillDash (server/internal/world/skill_dash.go)](../../server/internal/world/skill_dash.go#L83)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1235)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildSkillDash — server/internal/world/skill_dash.go</summary>

[Source definition](../../server/internal/world/skill_dash.go#L83)

```go
func buildSkillDash(d proto.SkillDash) []byte {
	raw := proto.SkillDashPacket(d)
	w := packet.New(packet.Opcode(proto.GXSkillDash))
	w.Bytes(raw[2:])
	return w.MustFinish()
}
```

</details>

<details><summary>SkillDashPacket — internal/proto/skill_dash.go</summary>

[Source definition](../../internal/proto/skill_dash.go#L24)

```go
func SkillDashPacket(d SkillDash) []byte {
	w := NewWriter(GXSkillDash)
	w.U32(d.Source)
	for _, p := range [][3]float32{d.From, d.To} {
		for _, v := range p {
			w.F32(v)
		}
	}
	bits := math.Float64bits(d.Started)
	w.U32(uint32(bits))
	w.U32(uint32(bits >> 32))
	w.F32(d.Contact)
	if d.Stopped {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
type SkillDash struct {
	Stopped  bool
	Source   uint32
	From, To [3]float32
	Started  float64 // synchronized map physics seconds
	Contact  float32 // native ALT contact frame
}
```

</details>

<details><summary>DecodeSkillDash — internal/proto/skill_dash.go</summary>

[Source definition](../../internal/proto/skill_dash.go#L43)

```go
func DecodeSkillDash(p []byte) (d SkillDash, err error) {
	if len(p) != 43 || p[42] > 1 || Op(p) != GXSkillDash {
		return d, fmt.Errorf("invalid skill dash size")
	}
	r := NewReader(p)
	d.Source = r.U32()
	for i := range d.From {
		d.From[i] = r.F32()
	}
	for i := range d.To {
		d.To[i] = r.F32()
	}
	d.Started = math.Float64frombits(r.U64())
	d.Contact = r.F32()
	d.Stopped = r.Bool()
	if d.Source == 0 || math.IsNaN(d.Started) || math.IsInf(d.Started, 0) || d.Started < 0 || d.Contact < 10 || d.Contact > 30 {
		return SkillDash{}, fmt.Errorf("invalid skill dash metadata")
	}
	for _, v := range append(d.From[:], d.To[:]...) {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return SkillDash{}, fmt.Errorf("non-finite skill dash")
		}
	}
	if math.IsNaN(float64(d.Contact)) || d.Distance() > MaxSkillDashDistance {
		return SkillDash{}, fmt.Errorf("invalid skill dash range")
	}
	return d, r.Err()
}
```

</details>

## Opcode 7f12

**`0x7F12` — GXEnchantRules, Opcode**

Declarations: [proto.GXEnchantRules](../../internal/proto/enchant_rules.go#L9), [authwire.Opcode](../../internal/authwire/credentials.go#L12).

Source notes: GXEnchantRules publishes server enchant bonuses for authoritative tooltips.

**C→S receive adapters:** [frame (server/cmd/gx_login/main.go)](../../server/cmd/gx_login/main.go#L224)

**S→C serializers:** [sendEnchantRules (server/internal/world/enchant_rules.go)](../../server/internal/world/enchant_rules.go#L80)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L265)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendEnchantRules — server/internal/world/enchant_rules.go</summary>

[Source definition](../../server/internal/world/enchant_rules.go#L80)

```go
func (s *Session) sendEnchantRules() {
	// The legacy packet has one display bound. Quotes and attempts use
	// their equipment-specific cap on the server.
	payload := proto.EnchantRulesPacket(proto.EnchantRules{MaxLevel: max(enchantConfig.WeaponCap, enchantConfig.ArmorCap), WeaponBonus: weaponEnchantAttackBonus, ArmorBonus: armorEnchantAbsorption})
	w := packet.New(packet.Opcode(proto.GXEnchantRules))
	w.Bytes(payload[2:])
	s.send(w.MustFinish(), "GX enchant rules")
}
```

</details>

<details><summary>EnchantRulesPacket — internal/proto/enchant_rules.go</summary>

[Source definition](../../internal/proto/enchant_rules.go#L16)

```go
func EnchantRulesPacket(r EnchantRules) []byte {
	w := NewWriter(GXEnchantRules)
	w.U8(r.MaxLevel)
	for _, table := range [2][50]uint32{r.WeaponBonus, r.ArmorBonus} {
		for _, v := range table {
			w.U32(v)
		}
	}
	return w.Bytes()
}
```

```go
type EnchantRules struct {
	MaxLevel                uint8
	WeaponBonus, ArmorBonus [50]uint32
}
```

</details>

<details><summary>DecodeEnchantRules — internal/proto/enchant_rules.go</summary>

[Source definition](../../internal/proto/enchant_rules.go#L27)

```go
func DecodeEnchantRules(p []byte) (EnchantRules, error) {
	var r EnchantRules
	if len(p) != 403 || Op(p) != GXEnchantRules || p[2] > 49 {
		return r, fmt.Errorf("invalid enchant rules")
	}
	r.MaxLevel = p[2]
	off := 3
	for _, table := range []*[50]uint32{&r.WeaponBonus, &r.ArmorBonus} {
		for i := range table {
			v := binary.LittleEndian.Uint32(p[off:])
			off += 4
			if v > 2147483647 {
				return EnchantRules{}, fmt.Errorf("invalid enchant bonus")
			}
			table[i] = v
		}
	}
	return r, nil
}
```

</details>

<details><summary>Validate — internal/authwire/credentials.go</summary>

[Source definition](../../internal/authwire/credentials.go#L16)

```go
func Validate(user, password string) error {
	if len(user) < 1 || len(user) > MaxUser || len(password) < 1 || len(password) > MaxPassword || !utf8.ValidString(user) || !utf8.ValidString(password) || strings.ContainsRune(user, 0) || strings.ContainsRune(password, 0) {
		return errors.New("credentials exceed UTF-8 byte limits (18/128) or contain invalid text")
	}
	return nil
}
```

</details>

<details><summary>Encode — internal/authwire/credentials.go</summary>

[Source definition](../../internal/authwire/credentials.go#L22)

```go
func Encode(user, password string) ([]byte, error) {
	if err := Validate(user, password); err != nil {
		return nil, err
	}
	out := binary.LittleEndian.AppendUint16(nil, Opcode)
	out = append(out, byte(len(user)), byte(len(password)))
	out = append(out, user...)
	out = append(out, password...)
	return out, nil
}
```

</details>

<details><summary>Decode — internal/authwire/credentials.go</summary>

[Source definition](../../internal/authwire/credentials.go#L32)

```go
func Decode(payload []byte) (string, string, error) {
	if len(payload) < 4 || binary.LittleEndian.Uint16(payload) != Opcode {
		return "", "", errors.New("invalid credential extension")
	}
	u, p := int(payload[2]), int(payload[3])
	if len(payload) != 4+u+p {
		return "", "", errors.New("invalid credential lengths")
	}
	user, password := string(payload[4:4+u]), string(payload[4+u:])
	return user, password, Validate(user, password)
}
```

</details>

## Opcode 7f13

**`0x7F13` — GXTeleportStone**

Declarations: [proto.GXTeleportStone](../../internal/proto/teleport_stone.go#L4).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L83)

**S→C serializers:** [sendTeleportStone (server/internal/world/teleport_stone.go)](../../server/internal/world/teleport_stone.go#L51)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendTeleportStone — server/internal/world/teleport_stone.go</summary>

[Source definition](../../server/internal/world/teleport_stone.go#L51)

```go
func (s *Session) sendTeleportStone(action uint8, ok bool, message string, points [proto.TeleportPointCount]db.TeleportPoint) {
	w := packet.New(packet.Opcode(proto.GXTeleportStone))
	w.U8(action)
	w.Bool(ok)
	writeChatText(w, message)
	for _, p := range points {
		w.Bool(p.Saved)
		w.U16(p.Map)
		w.F32(p.X)
		w.F32(p.Y)
		w.F32(p.Z)
	}
	if action == proto.TeleportCasting {
		w.U16(uint16(teleportStoneCastDuration / time.Millisecond))
	}
	s.send(w.MustFinish(), "recorded points")
}
```

</details>

<details><summary>TeleportStonePacket — internal/proto/teleport_stone.go</summary>

[Source definition](../../internal/proto/teleport_stone.go#L27)

```go
func TeleportStonePacket(action, slot uint8) []byte {
	w := NewWriter(GXTeleportStone)
	w.U8(action)
	w.U8(slot)
	return w.Bytes()
}
```

</details>

## Opcode 7f14

**`0x7F14` — GXSummonRune**

Declarations: [proto.GXSummonRune](../../internal/proto/summon_rune.go#L4).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L81)

**S→C serializers:** [summonResult (server/internal/world/summon_rune.go)](../../server/internal/world/summon_rune.go#L58), [publishSummonCast (server/internal/world/summon_rune.go)](../../server/internal/world/summon_rune.go#L65), [tickPartySummon (server/internal/world/summon_rune.go)](../../server/internal/world/summon_rune.go#L134)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>summonResult — server/internal/world/summon_rune.go</summary>

[Source definition](../../server/internal/world/summon_rune.go#L58)

```go
func (s *Session) summonResult(message string) {
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	w.U8(proto.SummonResult)
	writeChatText(w, message)
	s.send(w.MustFinish(), "party summon result")
}
```

</details>

<details><summary>publishSummonCast — server/internal/world/summon_rune.go</summary>

[Source definition](../../server/internal/world/summon_rune.go#L65)

```go
func (s *Session) publishSummonCast(active bool) {
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	if active {
		w.U8(proto.SummonCast)
	} else {
		w.U8(proto.SummonCastEnd)
	}
	w.U32(s.char.ID)
	ms := uint16(0)
	if active {
		ms = uint16(summonCastTime / time.Millisecond)
	}
	w.U16(ms)
	s.broadcastNear(w.MustFinish(), "party summon cast", true)
}
```

</details>

<details><summary>tickPartySummon — server/internal/world/summon_rune.go</summary>

[Source definition](../../server/internal/world/summon_rune.go#L134)

```go
func (s *Session) tickPartySummon(now time.Time) {
	if s.summonCastEnd.IsZero() {
		return
	}
	if s.char == nil || s.player == nil || s.dead || s.char.CurrentHP == 0 || s.actionBlocked() || s.inCombat() {
		s.cancelPartySummon()
		return
	}
	if now.Before(s.summonCastEnd) {
		return
	}
	s.summonCastEnd = time.Time{}
	s.publishSummonCast(false)
	group, targets := partySummonTargets(s.player)
	if len(targets) == 0 || !teleportStoneField(s.char.MapID) {
		s.summonResult("There are no party members to summon.")
		return
	}

	offer := partySummonOffer{summonOfferSerial.Add(1), s.player, group, now.Add(summonOfferTime)}
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	w.U8(proto.SummonOffer)
	w.U32(offer.id)
	w.U32(s.char.ID)
	w.U16(realMap(s.char.MapID))
	w.U16(uint16(summonOfferTime / time.Millisecond))
	w.FixedString(s.char.Name, 21)
	frame := w.MustFinish()
	sent := false
	for _, p := range targets {
		if !p.IsAlive() || p.Faction() != s.faction {
			continue
		}
		p.mailMu.Lock()
		p.summonOffer = &offer
		p.mailMu.Unlock()
		_ = p.Send(frame, "party summon offer")
		sent = true
	}
	if !sent {
		s.summonResult("There are no party members to summon.")
	}
}
```

</details>

<details><summary>SummonAnswerPacket — internal/proto/summon_rune.go</summary>

[Source definition](../../internal/proto/summon_rune.go#L35)

```go
func SummonAnswerPacket(id uint32, accept bool) []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonAnswer)
	w.U32(id)
	w.Bool(accept)
	return w.Bytes()
}
```

</details>

<details><summary>SummonCancelPacket — internal/proto/summon_rune.go</summary>

[Source definition](../../internal/proto/summon_rune.go#L43)

```go
func SummonCancelPacket() []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonCancel)
	return w.Bytes()
}
```

</details>

## Opcode 7f15

**`0x7F15` — GXPartyInviteName**

Declarations: [packet.GXPartyInviteName](../../server/internal/packet/packet.go#L345), [proto.GXPartyInviteName](../../internal/proto/party.go#L107).

Source notes: GXPartyInviteName is the global /party <name> invitation request. GXPartyInviteName is the GX /party <name> request: the party invitation of 0x0B09 without the proximity or same-map requirement. The target still receives the native 0x0B09 prompt.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L128)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>PartyGlobalInvitePacket — internal/proto/party.go</summary>

[Source definition](../../internal/proto/party.go#L109)

```go
func PartyGlobalInvitePacket(name string) []byte {
	w := NewWriter(GXPartyInviteName)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
```

</details>

## Opcode 7f16

**`0x7F16` — GXSkillCooldown**

Declarations: [proto.GXSkillCooldown](../../internal/proto/skill_cooldown.go#L10).

Source notes: GXSkillCooldown tells the client that a skill's reuse timer has started by itself: skill uint16, remaining seconds uint16, total seconds uint16. Total is what a full timer bar represents; 0 keeps the current one. A Potential triggering, a cooldown shortened by a cooldown reduction and a reduction applied to a timer that is already running all use it.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendSkillCooldown (server/internal/world/cooldown_reduction.go)](../../server/internal/world/cooldown_reduction.go#L33)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L817)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendSkillCooldown — server/internal/world/cooldown_reduction.go</summary>

[Source definition](../../server/internal/world/cooldown_reduction.go#L33)

```go
func (s *Session) sendSkillCooldown(skillID, remaining, total uint16) {
	w := packet.New(packet.Opcode(proto.GXSkillCooldown))
	w.U16(skillID)
	w.U16(remaining)
	w.U16(total)
	s.send(w.MustFinish(), "GX skill reuse")
}
```

</details>

<details><summary>DecodeSkillCooldown — internal/proto/skill_cooldown.go</summary>

[Source definition](../../internal/proto/skill_cooldown.go#L14)

```go
func DecodeSkillCooldown(p []byte) (v SkillCooldown, err error) {
	if len(p) != 8 {
		return v, fmt.Errorf("invalid GX skill cooldown length")
	}
	r := NewReader(p)
	v.Skill, v.Seconds, v.Total = r.U16(), r.U16(), r.U16()
	return v, r.Err()
}
```

```go
type SkillCooldown struct{ Skill, Seconds, Total uint16 }
```

</details>

## Opcode 7f17

**`0x7F17` — GXField**

Declarations: [proto.GXField](../../internal/proto/field.go#L9).

Source notes: GXField announces a field left on the ground (Fire Bush's curtain of fire) to the users around it: owner uint32, skill uint16, level uint8, position x, y, z float32 and the seconds it lasts uint16. The client shows the field's area effect there until it ends. Traps are not announced: they are invisible.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [skillField (server/internal/world/trap.go)](../../server/internal/world/trap.go#L35)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L874)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>skillField — server/internal/world/trap.go</summary>

[Source definition](../../server/internal/world/trap.go#L35)

```go
func (s *Session) skillField(k db.CharacterSkill, def *db.SkillDef, stats CharacterStats, completed ...bool) {
	if def.TypeDetail != detailTrap && def.TypeDetail != detailFireBush {
		s.log.Printf("GX Server World rejected skill use: field detail %d not modeled skill_id=%d", def.TypeDetail, k.SkillID)
		return
	}
	ch := s.char
	s.startCooldown(k, def, len(completed) > 0 && completed[0])
	sp, mp := s.skillCosts(def, &stats)
	s.spendDelta(mp, sp)
	s.persist()
	s.send(buildSkillUsedSpMp(sp, mp), "0x050C skill used sp/mp")
	s.broadcastNear(buildSkillUseResult(packet.SkillUseNoTarget, 0, ch.ID, 0, k.SkillID, k.Level, 0, 0, 0), "0x0511 trap placed", true)
	t := placedTrap{mapID: ch.MapID, x: ch.PosX, y: ch.PosY, z: ch.PosZ, def: def,
		endsAt: time.Now().Add(time.Duration(def.KeepSeconds) * time.Second)}
	if def.TypeDetail == detailFireBush {
		t.nextHit = time.Now()
		// Everybody around sees the curtain of fire; a trap is invisible and is not announced.
		w := packet.New(packet.Opcode(proto.GXField))
		w.U32(ch.ID)
		w.U16(k.SkillID)
		w.U8(k.Level)
		w.F32(ch.PosX)
		w.F32(ch.PosY)
		w.F32(ch.PosZ)
		w.U16(uint16(min(def.KeepSeconds, 65535)))
		sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "GX field placed")
	}
	s.traps = append(s.traps, t)
	s.log.Printf("GX Server World trap placed skill_id=%d level=%d radius=%d seconds=%d", k.SkillID, k.Level, def.ApplyRange, def.KeepSeconds)
}
```

</details>

<details><summary>DecodeField — internal/proto/field.go</summary>

[Source definition](../../internal/proto/field.go#L19)

```go
func DecodeField(p []byte) (v Field, err error) {
	if len(p) != 23 {
		return v, fmt.Errorf("invalid GX field length")
	}
	r := NewReader(p)
	v.Owner, v.Skill, v.Level = r.U32(), r.U16(), r.U8()
	v.X, v.Y, v.Z = r.F32(), r.F32(), r.F32()
	v.Seconds = r.U16()
	return v, r.Err()
}
```

```go
type Field struct {
	Owner   uint32
	Skill   uint16
	Level   uint8
	X, Y, Z float32
	Seconds uint16
}
```

</details>

## Opcode 7f18

**`0x7F18` — GXTransform**

Declarations: [proto.GXTransform](../../internal/proto/transform.go#L9).

Source notes: GXTransform tells the users around a character that it looks like a monster (Degeneration) or is back to normal: character id uint32 and the monster type uint16, 0 when the transformation ends. The server also repeats it for every transformed character that enters a user's view.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildTransform (server/internal/world/degeneration.go)](../../server/internal/world/degeneration.go#L31)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L861)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildTransform — server/internal/world/degeneration.go</summary>

[Source definition](../../server/internal/world/degeneration.go#L31)

```go
func buildTransform(user uint32, mob uint16) []byte {
	w := packet.New(packet.Opcode(proto.GXTransform))
	w.U32(user)
	w.U16(mob)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeTransform — internal/proto/transform.go</summary>

[Source definition](../../internal/proto/transform.go#L16)

```go
func DecodeTransform(p []byte) (v Transform, err error) {
	if len(p) != 8 {
		return v, fmt.Errorf("invalid GX transform length")
	}
	r := NewReader(p)
	v.User, v.Mob = r.U32(), r.U16()
	return v, r.Err()
}
```

```go
type Transform struct {
	User uint32
	Mob  uint16
}
```

</details>

## Opcode 7f19

**`0x7F19` — GXTargetHealth**

Declarations: [proto.GXTargetHealth](../../internal/proto/target_health.go#L8).

Source notes: GXTargetHealth gives the HP of the user the client has selected (the native protocol only tells the HP of party members): character id, current and maximum HP, all uint32. The server repeats it whenever it changes.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendTargetHealth (server/internal/world/target_effects.go)](../../server/internal/world/target_effects.go#L90)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L836)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendTargetHealth — server/internal/world/target_effects.go</summary>

[Source definition](../../server/internal/world/target_effects.go#L90)

```go
func (s *Session) sendTargetHealth(p *ConnectedPlayer) {
	ch := p.Snapshot()
	p.mu.Lock()
	st := StatsFor(&ch, GetEquipmentStats(p.items))
	p.mu.Unlock()
	now := [3]uint32{ch.ID, st.CurrentHP, st.MaxHP}
	if now == s.targetHealthSent {
		return
	}
	s.targetHealthSent = now
	w := packet.New(packet.Opcode(proto.GXTargetHealth))
	w.U32(now[0])
	w.U32(now[1])
	w.U32(now[2])
	s.send(w.MustFinish(), "GX selected user HP")
}
```

</details>

<details><summary>DecodeTargetHealth — internal/proto/target_health.go</summary>

[Source definition](../../internal/proto/target_health.go#L15)

```go
func DecodeTargetHealth(p []byte) (v TargetHealth, err error) {
	if len(p) != 14 {
		return v, fmt.Errorf("invalid GX target health length")
	}
	r := NewReader(p)
	v.User, v.HP, v.Max = r.U32(), r.U32(), r.U32()
	return v, r.Err()
}
```

```go
type TargetHealth struct {
	User    uint32
	HP, Max uint32
}
```

</details>

## Opcode 7f1a

**`0x7F1A` — GXKillNotice**

Declarations: [proto.GXKillNotice](../../internal/proto/kill_notice.go#L7).

Source notes: GXKillNotice tells a user that it killed another player: the victim's name (21 bytes). The client says it in red with SysMsg 8, "Killed <t>".

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [notifyKill (server/internal/world/pvpdeath.go)](../../server/internal/world/pvpdeath.go#L240)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L830)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>notifyKill — server/internal/world/pvpdeath.go</summary>

[Source definition](../../server/internal/world/pvpdeath.go#L240)

```go
func (s *Session) notifyKill(killer *ConnectedPlayer) {
	w := packet.New(packet.Opcode(proto.GXKillNotice))
	w.FixedString(s.char.Name, 21)
	killer.Send(w.MustFinish(), "GX kill notice")
}
```

</details>

<details><summary>DecodeKillNotice — internal/proto/kill_notice.go</summary>

[Source definition](../../internal/proto/kill_notice.go#L9)

```go
func DecodeKillNotice(p []byte) (string, error) {
	if len(p) != 23 {
		return "", fmt.Errorf("invalid GX kill notice length")
	}
	r := NewReader(p)
	name := r.Fixed(21)
	return name, r.Err()
}
```

</details>

## Opcode 7f1b

**`0x7F1B` — GXBlessingSnapshot**

Declarations: [proto.GXBlessingSnapshot](../../internal/proto/blessing.go#L29).

Source notes: GXBlessingSnapshot carries the server's presentation data independently of the native count packets. Entry snapshots restore the timer without a burst.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [blessingSnapshotLocked (server/internal/world/killcount.go)](../../server/internal/world/killcount.go#L299)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L317)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>blessingSnapshotLocked — server/internal/world/killcount.go</summary>

[Source definition](../../server/internal/world/killcount.go#L299)

```go
func blessingSnapshotLocked(c uint8, entry bool) []byte {
	if c >= uint8(len(kcWorld)) {
		return nil
	}
	world := kcWorld[c]
	maximum := kcMaxKill()
	count := min(world.count, maximum)
	var remaining uint32
	if !world.resetAt.IsZero() {
		remaining = uint32(max(time.Until(world.resetAt).Milliseconds(), 0))
	}
	w := packet.New(packet.Opcode(proto.GXBlessingSnapshot))
	w.U8(c)
	var flags uint8
	if entry {
		flags = 1
	}
	w.U8(flags)
	w.U32(count)
	w.U32(maximum)
	w.U32(remaining)
	// The native tooltip selects by count. Actual stats keep debit hysteresis.
	index := kcTierFor(count)
	for typ := 1; typ < kcTypes; typ++ {
		var value uint16
		if index >= 0 {
			value = uint16(kcTiers[index].Status[typ])
		}
		w.U16(value)
	}
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeBlessingSnapshot — internal/proto/blessing.go</summary>

[Source definition](../../internal/proto/blessing.go#L38)

```go
func DecodeBlessingSnapshot(p []byte) (BlessingSnapshot, error) {
	r := NewReader(p)
	country, flags := r.U8(), r.U8()
	b := BlessingSnapshot{Country: country, Entry: flags == 1, Count: r.U32(), Maximum: r.U32(), RemainingMS: r.U32()}
	for i := range b.Bonuses {
		b.Bonuses[i] = r.U16()
	}
	if err := r.Err(); err != nil {
		return b, err
	}
	if country > 1 || flags > 1 || b.Maximum == 0 || b.Count > b.Maximum || b.RemainingMS > 600000 ||
		b.RemainingMS != 0 && b.Count != b.Maximum || r.Remaining() != 0 {
		return b, fmt.Errorf("invalid blessing snapshot")
	}
	return b, nil
}
```

```go
type BlessingSnapshot struct {
	Country                     uint8
	Entry                       bool
	Count, Maximum, RemainingMS uint32
	Bonuses                     [27]uint16
}
```

</details>

## Opcode 7f1c

**`0x7F1C` — GXWeather**

Declarations: [proto.GXWeather](../../internal/proto/weather.go#L5).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [updateWeather (server/internal/world/weather.go)](../../server/internal/world/weather.go#L30)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L311)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>updateWeather — server/internal/world/weather.go</summary>

[Source definition](../../server/internal/world/weather.go#L30)

```go
func (s *Session) updateWeather(now time.Time, force bool) {
	if s.char == nil || s.awaitingMapLoad || (!force && now.Before(s.weatherCheck)) {
		return
	}
	s.weatherCheck = now.Add(time.Second)
	current := weatherAt(mapConfig.Zones[realMap(s.char.MapID)], weatherStarted, now)
	if now.Before(s.weatherOverrideUntil) {
		current = s.weatherOverride
	}
	if current == s.lastWeather && !force {
		return
	}
	s.lastWeather = current
	w := packet.New(packet.Opcode(proto.GXWeather))
	w.U16(s.char.MapID)
	w.U8(current)
	s.send(w.MustFinish(), "GX local weather")
}
```

</details>

<details><summary>DecodeWeather — internal/proto/weather.go</summary>

[Source definition](../../internal/proto/weather.go#L18)

```go
func DecodeWeather(p []byte) (Weather, error) {
	r := NewReader(p)
	v := Weather{r.U16(), r.U8()}
	if err := r.Err(); err != nil {
		return Weather{}, err
	}
	if v.Type > WeatherSnow || r.Remaining() != 0 {
		return Weather{}, fmt.Errorf("invalid weather snapshot")
	}
	return v, nil
}
```

```go
type Weather struct {
	MapID uint16
	Type  uint8
}
```

</details>

## Opcode 7f1d

**`0x7F1D` — GXServerNotice**

Declarations: [proto.GXServerNotice](../../internal/proto/server_notice.go#L9).

Source notes: GXServerNotice carries a global announcement as native length-prefixed chat text.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [broadcastNotice (server/internal/world/notices.go)](../../server/internal/world/notices.go#L107)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L399)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>broadcastNotice — server/internal/world/notices.go</summary>

[Source definition](../../server/internal/world/notices.go#L107)

```go
func broadcastNotice(message string) {
	w := packet.New(packet.Opcode(proto.GXServerNotice))
	writeChatText(w, message)
	frame := w.MustFinish()
	for _, p := range AllPlayers() {
		p.Send(frame, "global server notice")
	}
}
```

</details>

<details><summary>DecodeServerNotice — internal/proto/server_notice.go</summary>

[Source definition](../../internal/proto/server_notice.go#L11)

```go
func DecodeServerNotice(payload []byte) (string, error) {
	if Op(payload) != GXServerNotice || len(payload) < 5 || len(payload) > 131 || payload[len(payload)-1] != 0 {
		return "", fmt.Errorf("invalid server notice")
	}
	r := NewReader(payload)
	text := r.Text()
	if err := r.Err(); err != nil {
		return "", err
	}
	if r.Remaining() != 0 || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("invalid server notice text")
	}
	return text, nil
}
```

</details>

## Opcode 7f1e

**`0x7F1E` — GXItemCooldownReset**

Declarations: [proto.GXItemCooldownReset](../../internal/proto/gm.go#L34).

Source notes: GXItemCooldownReset clears the recipient’s item and mount cooldowns.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [gmCure (server/internal/world/gm.go)](../../server/internal/world/gm.go#L621)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L250)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>gmCure — server/internal/world/gm.go</summary>

[Source definition](../../server/internal/world/gm.go#L621)

```go
func (s *Session) gmCure() error {
	if s.dead {
		return fmt.Errorf("you are dead")
	}
	s.removeEffectsInternal(func(b *ActiveBuff) bool { return b.Effect == typeEffectDebuff }, "GM cure", true)
	s.refillPools()
	s.persist()
	st := s.stats()
	s.send(buildCurrentHitpoints(&st), "0x0521 GM cure")
	for id := range s.skillCooldowns {
		s.sendSkillCooldown(id, 0, 0)
	}
	clear(s.skillCooldowns)
	clear(s.itemCooldowns)
	vehicleMu.Lock()
	delete(vehicleCooldowns, s.char.ID)
	vehicleMu.Unlock()
	s.send(packet.New(packet.Opcode(proto.GXItemCooldownReset)).MustFinish(), "GM item cooldown reset")
	s.persistSkillDelays()
	s.gmReply("GM: restored HP, MP and SP and cleared all debuffs and cooldowns.")
	return nil
}
```

</details>

## Opcode 7f1f

**`0x7F1F` — GXAuction**

Declarations: [proto.GXAuction](../../internal/proto/auction.go#L8).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L34)

**S→C serializers:** [auctionReply (server/internal/world/auction.go)](../../server/internal/world/auction.go#L28)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>auctionReply — server/internal/world/auction.go</summary>

[Source definition](../../server/internal/world/auction.go#L28)

```go
func (s *Session) auctionReply(reply proto.AuctionReply) {
	if frame := proto.AuctionPacket(reply); frame != nil {
		w := packet.New(packet.Opcode(proto.GXAuction))
		w.Bytes(frame[2:])
		s.send(w.MustFinish(), "GX auction")
	}
}
```

</details>

<details><summary>AuctionPacket — internal/proto/auction.go</summary>

[Source definition](../../internal/proto/auction.go#L80)

```go
func AuctionPacket(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil || len(body) > 12000 {
		return nil
	}
	w := NewWriter(GXAuction)
	w.Raw(body)
	return w.Bytes()
}
```

</details>

## Opcode 7f20

**`0x7F20` — GXLootNotice**

Declarations: [proto.GXLootNotice](../../internal/proto/loot_notice.go#L8).

Source notes: GXLootNotice tells a party or raid which member just received an item from a kill: the receiver's character id and the item type and id. The client says it with SysMsg 61, "<p> attained <i>".

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [announceLoot (server/internal/world/loot.go)](../../server/internal/world/loot.go#L177)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L823)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>announceLoot — server/internal/world/loot.go</summary>

[Source definition](../../server/internal/world/loot.go#L177)

```go
func (s *Session) announceLoot(it db.Item) {
	if s.player == nil {
		return
	}
	partyMu.Lock()
	var others []*ConnectedPlayer
	if pt := partyOf(s.player); pt != nil {
		for _, m := range pt.members {
			if m != s.player {
				others = append(others, m)
			}
		}
	}
	partyMu.Unlock()
	if len(others) == 0 {
		return
	}
	w := packet.New(packet.Opcode(proto.GXLootNotice))
	w.U32(s.char.ID)
	w.U8(it.Type)
	w.U8(it.TypeID)
	broadcast(others, w.MustFinish(), "GX group loot notice", s.logger)
}
```

</details>

<details><summary>DecodeLootNotice — internal/proto/loot_notice.go</summary>

[Source definition](../../internal/proto/loot_notice.go#L15)

```go
func DecodeLootNotice(p []byte) (LootNotice, error) {
	if len(p) != 8 {
		return LootNotice{}, fmt.Errorf("invalid GX loot notice length")
	}
	r := NewReader(p)
	n := LootNotice{Receiver: r.U32(), Type: r.U8(), TypeID: r.U8()}
	return n, r.Err()
}
```

```go
type LootNotice struct {
	Receiver     uint32
	Type, TypeID uint8
}
```

</details>

## Opcode 7f21

**`0x7F21` — GXFormAppearance**

Declarations: [proto.GXFormAppearance](../../internal/proto/player_forms.go#L6).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildFormAppearance (server/internal/world/player_forms.go)](../../server/internal/world/player_forms.go#L110)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L842)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildFormAppearance — server/internal/world/player_forms.go</summary>

[Source definition](../../server/internal/world/player_forms.go#L110)

```go
func buildFormAppearance(id uint32, look formLook) []byte {
	switch look.kind {
	case proto.FormMonsterCopy, proto.FormDegeneration:
		return buildTransform(id, look.mob)
	}
	w := packet.New(packet.Opcode(proto.GXFormAppearance))
	w.U32(id)
	w.U8(uint8(look.kind))
	switch look.kind {
	case proto.FormEvolution:
		w.U16(look.skill)
		w.U8(look.rank)
		for _, action := range evolutionActions(db.MustDefs().Mobs[look.mob]) {
			var id uint16
			if action.skill != nil {
				id = action.skill.SkillID
			}
			w.U16(id)
			w.U8(action.slot)
		}
	case proto.FormDisguise:
		w.Bytes(look.disguise.shape[2:])
	}
	return w.MustFinish()
}
```

</details>

<details><summary>DecodePlayerForm — internal/proto/player_forms.go</summary>

[Source definition](../../internal/proto/player_forms.go#L57)

```go
func DecodePlayerForm(p []byte) (v PlayerFormAppearance, err error) {
	r := NewReader(p)
	v.User, v.Kind = r.U32(), PlayerFormKind(r.U8())
	switch v.Kind {
	case FormNone:
		if r.Remaining() == 0 {
			return v, r.Err()
		}
	case FormEvolution:
		v.Skill, v.Rank = r.U16(), r.U8()
		for i := range v.Attacks {
			v.Attacks[i] = FormAttack{Skill: r.U16(), Slot: r.U8()}
		}
		if r.Remaining() == 0 {
			return v, r.Err()
		}
	case FormDisguise:
		if len(p) >= 9 && Op(p[7:]) == CharacterShape {
			v.Shape, err = DecodeUserShape(p[7:])
			if err == nil && v.Shape.ID != v.User {
				err = fmt.Errorf("invalid disguise character")
			}
			return v, err
		}
	}
	return v, fmt.Errorf("invalid player form")
}
```

```go
type PlayerFormKind uint8
```

```go
type PlayerFormAppearance struct {
	User    uint32
	Kind    PlayerFormKind
	Skill   uint16
	Rank    uint8
	Shape   UserShape
	Attacks [2]FormAttack
}
```

```go
type FormAttack struct {
	Skill uint16
	Slot  uint8
}
```

</details>

## Opcode 7f22

**`0x7F22` — GXEvolutionBar**

Declarations: [proto.GXEvolutionBar](../../internal/proto/player_forms.go#L7).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendEvolutionBar (server/internal/world/evolution.go)](../../server/internal/world/evolution.go#L57)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L855)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendEvolutionBar — server/internal/world/evolution.go</summary>

[Source definition](../../server/internal/world/evolution.go#L57)

```go
func (s *Session) sendEvolutionBar() {
	w := packet.New(packet.Opcode(proto.GXEvolutionBar))
	w.U32(s.evolutionGrant)
	for _, a := range s.evolutionActions {
		var id uint16
		var reach uint8
		var total, left uint32
		if a.skill != nil {
			id, reach = a.skill.SkillID, a.skill.AttackRange
			total = uint32(a.skill.ResetTimeSeconds) * 1000
			left = uint32(max(time.Until(s.evolutionReadyAt[id]).Milliseconds(), 0))
		}
		w.U16(id)
		w.U8(reach)
		w.U32(total)
		w.U32(left)
	}
	s.send(w.MustFinish(), "temporary monster skills")
}
```

</details>

<details><summary>DecodeEvolutionBar — internal/proto/player_forms.go</summary>

[Source definition](../../internal/proto/player_forms.go#L96)

```go
func DecodeEvolutionBar(p []byte) (v EvolutionBar, err error) {
	if len(p) != 28 {
		return v, fmt.Errorf("invalid evolution bar length")
	}
	r := NewReader(p)
	v.Generation = r.U32()
	for i := range v.Actions {
		a := &v.Actions[i]
		a.Skill, a.Range, a.CooldownMS, a.LeftMS = r.U16(), r.U8(), r.U32(), r.U32()
	}
	return v, r.Err()
}
```

```go
type EvolutionBar struct {
	Generation uint32
	Actions    [2]EvolutionAction
}
```

</details>

## Opcode 7f23

**`0x7F23` — GXEvolutionCast**

Declarations: [proto.GXEvolutionCast](../../internal/proto/player_forms.go#L8).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L28)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>EvolutionCastPacket — internal/proto/player_forms.go</summary>

[Source definition](../../internal/proto/player_forms.go#L109)

```go
func EvolutionCastPacket(generation uint32, cell uint8, target uint32) []byte {
	w := NewWriter(GXEvolutionCast)
	w.U32(generation)
	w.U8(cell)
	w.U32(target)
	return w.Bytes()
}
```

</details>

## Opcode 7f24

**`0x7F24` — GXFirecracker**

Declarations: [proto.GXFirecracker](../../internal/proto/firecracker.go#L9).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [useFirecracker (server/internal/world/firecracker.go)](../../server/internal/world/firecracker.go#L16)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>useFirecracker — server/internal/world/firecracker.go</summary>

[Source definition](../../server/internal/world/firecracker.go#L16)

```go
func (s *Session) useFirecracker(index int, def *db.ItemDef) bool {
	ch := s.char
	variant := proto.FirecrackerVariant(def.Type, def.TypeID)
	if variant == 0 || s.dead || ch.CurrentHP == 0 {
		return false
	}
	now := time.Now()
	delay := cooldownForReqIg(def.ReqIg)
	if delay > 0 && now.Sub(s.itemCooldowns[def.ReqIg]) < delay {
		return false
	}
	item, err := s.consumeOne(index)
	if err != nil {
		s.logErr("consume firecracker", err)
		return false
	}
	if delay > 0 {
		if s.itemCooldowns == nil {
			s.itemCooldowns = make(map[uint8]time.Time)
		}
		s.itemCooldowns[def.ReqIg] = now
	}
	s.send(buildUseItemResult(ch.ID, &item), "0x050A firecracker used")
	payload := proto.FirecrackerPacket(proto.Firecracker{Variant: variant, X: ch.PosX, Y: ch.PosY, Z: ch.PosZ, Name: ch.Name})
	w := packet.New(packet.Opcode(proto.GXFirecracker))
	w.Bytes(payload[2:])
	frame := w.MustFinish()
	s.send(frame, "local firecracker")
	for _, viewer := range PlayersNearExcluding(ch.MapID, ch.ID, ch.PosX, ch.PosZ, firecrackerRadius) {
		viewer.Send(frame, "local firecracker")
	}
	return true
}
```

</details>

<details><summary>FirecrackerPacket — internal/proto/firecracker.go</summary>

[Source definition](../../internal/proto/firecracker.go#L32)

```go
func FirecrackerPacket(v Firecracker) []byte {
	w := NewWriter(GXFirecracker)
	w.U8(v.Variant)
	w.F32(v.X)
	w.F32(v.Y)
	w.F32(v.Z)
	w.Text(v.Name)
	return w.Bytes()
}
```

```go
type Firecracker struct {
	Variant uint8
	X, Y, Z float32
	Name    string
}
```

</details>

## Opcode 7f25

**`0x7F25` — GXPlayerFeedback**

Declarations: [proto.GXPlayerFeedback](../../internal/proto/player_feedback.go#L6).

Source notes: GXPlayerFeedback carries a presentation event for an already visible player.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendPlayerFeedback (server/internal/world/player_feedback.go)](../../server/internal/world/player_feedback.go#L8)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendPlayerFeedback — server/internal/world/player_feedback.go</summary>

[Source definition](../../server/internal/world/player_feedback.go#L8)

```go
func (s *Session) sendPlayerFeedback(kind uint8) {
	p := proto.PlayerFeedbackPacket(s.char.ID, kind)
	w := packet.New(packet.Opcode(proto.GXPlayerFeedback))
	w.Bytes(p[2:])
	frame := w.MustFinish()
	if kind == proto.PlayerLevelFeedback {
		s.send(frame, "GX level feedback")
	}
	s.sendToViewers(frame, "GX player feedback")
}
```

</details>

<details><summary>PlayerFeedbackPacket — internal/proto/player_feedback.go</summary>

[Source definition](../../internal/proto/player_feedback.go#L14)

```go
func PlayerFeedbackPacket(id uint32, kind uint8) []byte {
	w := NewWriter(GXPlayerFeedback)
	w.U32(id)
	w.U8(kind)
	return w.Bytes()
}
```

</details>

## Opcode 7f26

**`0x7F26` — GXOxygen**

Declarations: [proto.GXOxygen](../../internal/proto/oxygen.go#L9).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendOxygen (server/internal/world/oxygen.go)](../../server/internal/world/oxygen.go#L19)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L896)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendOxygen — server/internal/world/oxygen.go</summary>

[Source definition](../../server/internal/world/oxygen.go#L19)

```go
func (s *Session) sendOxygen() {
	b := &s.oxygen
	status := proto.OxygenStatus{Phase: b.phase, RemainingMS: uint16(b.remaining.Milliseconds())}
	w := packet.New(packet.Opcode(proto.GXOxygen))
	w.Bytes(proto.OxygenPacket(status)[2:])
	s.send(w.MustFinish(), "GX oxygen")
}
```

</details>

<details><summary>OxygenPacket — internal/proto/oxygen.go</summary>

[Source definition](../../internal/proto/oxygen.go#L32)

```go
func OxygenPacket(v OxygenStatus) []byte {
	w := NewWriter(GXOxygen)
	w.U8(uint8(v.Phase))
	w.U16(v.RemainingMS)
	return w.Bytes()
}
```

```go
type OxygenStatus struct {
	Phase       OxygenPhase
	RemainingMS uint16
}
```

</details>

<details><summary>DecodeOxygen — internal/proto/oxygen.go</summary>

[Source definition](../../internal/proto/oxygen.go#L39)

```go
func DecodeOxygen(p []byte) (OxygenStatus, error) {
	r := NewReader(p)
	v := OxygenStatus{Phase: OxygenPhase(r.U8()), RemainingMS: r.U16()}
	if Op(p) != GXOxygen || r.Err() != nil || r.Remaining() != 0 ||
		v.Phase > OxygenRefilling || v.RemainingMS > uint16(OxygenCapacity.Milliseconds()) ||
		v.Phase == OxygenHidden && v.RemainingMS != 0 {
		return OxygenStatus{}, fmt.Errorf("invalid oxygen status")
	}
	return v, nil
}
```

```go
type OxygenPhase uint8
```

</details>

## Opcode 7f27

**`0x7F27` — GXDrowningDamage**

Declarations: [proto.GXDrowningDamage](../../internal/proto/oxygen.go#L10).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L902)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeDrowningDamage — internal/proto/oxygen.go</summary>

[Source definition](../../internal/proto/oxygen.go#L50)

```go
func DecodeDrowningDamage(p []byte) (EnvironmentalDamage, error) {
	return decodeEnvironmentalDamage(p, "drowning")
}
```

</details>

## Opcode 7f28

**`0x7F28` — GXPartyAssist**

Declarations: [proto.GXPartyAssist](../../internal/proto/party_assist.go#L4).

Source notes: GXPartyAssist shares selected targets with nearby members of the same group.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L30)

**S→C serializers:** [handlePartyAssist (server/internal/world/party_assist.go)](../../server/internal/world/party_assist.go#L8)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>handlePartyAssist — server/internal/world/party_assist.go</summary>

[Source definition](../../server/internal/world/party_assist.go#L8)

```go
func (s *Session) handlePartyAssist(frame []byte) {
	me := s.player
	if me == nil || len(frame) != 7 || !me.IsAlive() {
		return
	}
	r := proto.NewReader(frame)
	action, id := r.U8(), r.U32()
	switch action {
	case proto.PartyAssistSelection:
		if id != 0 && !me.knowsAssistTarget(id) {
			id = 0
		}
		me.mu.Lock()
		me.assistTarget = id
		me.mu.Unlock()
	case proto.PartyAssistRequest:
		ally := FindPlayerByID(id)
		if ally == nil || ally == me || !ally.IsAlive() || !samePartyIDs(me.CharacterID, id) ||
			!me.knows(viewUser, id) || !me.canSee(ally) {
			return
		}
		ally.mu.Lock()
		target, sameMap := ally.assistTarget, ally.mapID == s.char.MapID
		ally.mu.Unlock()
		if sameMap && (target == 0 || ally.knowsAssistTarget(target) && me.knowsAssistTarget(target)) {
			reply := packet.New(packet.Opcode(proto.GXPartyAssist))
			reply.U32(id)
			reply.U32(target)
			s.send(reply.MustFinish(), "GX party assist")
		}
	}
}
```

</details>

<details><summary>PartyAssistPacket — internal/proto/party_assist.go</summary>

[Source definition](../../internal/proto/party_assist.go#L11)

```go
func PartyAssistPacket(action uint8, id uint32) []byte {
	w := NewWriter(GXPartyAssist)
	w.U8(action)
	w.U32(id)
	return w.Bytes()
}
```

</details>

## Opcode 7f29

**`0x7F29` — GXGMModes**

Declarations: [proto.GXGMModes](../../internal/proto/gm.go#L39).

Source notes: GXGMModes tells a GM's own client which modes are active: self ID:u32, concealed:u8 (/char on), immune:u8 (/attack on), turbo:u8. Every byte is 0 or 1 and the modes require GM. Sent on entry and after every change.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendGMModes (server/internal/world/gm.go)](../../server/internal/world/gm.go#L43)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1271)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendGMModes — server/internal/world/gm.go</summary>

[Source definition](../../server/internal/world/gm.go#L43)

```go
func (s *Session) sendGMModes() {
	if s.char == nil || !s.isAdmin {
		return
	}
	b := func(v bool) uint8 {
		if v {
			return 1
		}
		return 0
	}
	w := packet.New(gxGMModes)
	w.U32(s.char.ID)
	w.U8(b(s.gmConcealed))
	w.U8(b(s.gmImmune))
	w.U8(b(s.turboActive()))
	s.send(w.MustFinish(), "GX GM modes")
}
```

</details>

<details><summary>DecodeGMModes — internal/proto/gm.go</summary>

[Source definition](../../internal/proto/gm.go#L46)

```go
func DecodeGMModes(p []byte) (GMModes, error) {
	if len(p) != 9 || p[6] > 1 || p[7] > 1 || p[8] > 1 {
		return GMModes{}, fmt.Errorf("invalid GX GM modes")
	}
	return GMModes{ID: binary.LittleEndian.Uint32(p[2:6]), Concealed: p[6] == 1, Immune: p[7] == 1, Turbo: p[8] == 1}, nil
}
```

```go
type GMModes struct {
	ID                       uint32
	Concealed, Immune, Turbo bool
}
```

</details>

## Opcode 7f2a

**`0x7F2A` — GXDrowningDeath**

Declarations: [proto.GXDrowningDeath](../../internal/proto/oxygen.go#L11).

Source notes: target u32: the target died of drowning

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [applyEnvironmentalDamage (server/internal/world/fall_damage.go)](../../server/internal/world/fall_damage.go#L27)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L908)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>applyEnvironmentalDamage — server/internal/world/fall_damage.go</summary>

[Source definition](../../server/internal/world/fall_damage.go#L27)

```go
func (s *Session) applyEnvironmentalDamage(amount uint32, opcode proto.Opcode, cause string) {
	// Environmental damage ignores GM immunity: "attack on" never stops drowning.
	if s.char == nil || s.dead {
		return
	}
	damage := min(s.char.CurrentHP, amount)
	if damage == 0 {
		return
	}
	s.cancelTravelCast()
	s.commitMovement()
	s.cancelMove()
	s.breakStealth()
	s.char.CurrentHP = satSub(s.char.CurrentHP, damage)
	s.persist()
	w := packet.New(packet.Opcode(opcode))
	w.U32(s.char.ID)
	w.U32(damage)
	frame := w.MustFinish()
	s.send(frame, "GX "+cause+" damage")
	s.sendToViewers(frame, "GX "+cause+" damage (view)")
	current := s.stats()
	s.send(buildCurrentHitpoints(&current), "current hp after "+cause)
	if current.CurrentHP > 0 {
		s.wake()
		return
	}
	s.damageLedger = nil
	state, death := buildCharacterDeathState(s.char.ID), buildCharacterDeath(s.char.ID, deathTypeMob, 0)
	s.send(state, cause+" death state")
	s.send(death, cause+" death")
	s.sendToViewers(state, cause+" death state (view)")
	s.sendToViewers(death, cause+" death (view)")
	if s.player != nil {
		s.player.SetAlive(false)
		duelLeave(s.player, 3, true)
		guildDuelLeave(s.player, nil, 3)
	}
	s.dead, s.autoAttack, s.cast = true, false, nil
	s.clearBuffsOnDeath()
	// Environmental deaths grant no PvP credit, item or XP loss.
	s.deathLoss = -1
	if opcode == proto.GXDrowningDamage {
		notice := packet.New(packet.Opcode(proto.GXDrowningDeath))
		notice.U32(s.char.ID)
		frame := notice.MustFinish()
		s.send(frame, "GX drowning death")
		s.sendToViewers(frame, "GX drowning death (view)")
	}
}
```

</details>

<details><summary>DecodeDrowningDeath — internal/proto/oxygen.go</summary>

[Source definition](../../internal/proto/oxygen.go#L55)

```go
func DecodeDrowningDeath(p []byte) (uint32, error) {
	r := NewReader(p)
	id := r.U32()
	if Op(p) != GXDrowningDeath || r.Err() != nil || r.Remaining() != 0 {
		return 0, fmt.Errorf("invalid drowning death")
	}
	return id, nil
}
```

</details>
