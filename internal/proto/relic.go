package proto

import "fmt"

// Relic (obelisk) packets. Addresses refer to an EP5.4 Game.exe build of
// 4,000,768 bytes (dispatcher branch 0x5f2eff..0x5f3037):
//
//   - 0x2101 (0x5e71c0 -> CNetWork 0x5996c0): count:u8, then per relic
//     key:u32 (map<<16 + relic number), owner:u8, x:f32, z:f32. The list
//     replaces the map's relics.
//   - 0x2102 (0x5e7230 -> 0x598540): key:u32, owner:u8 for a known relic.
//   - 0x2103 (0x5e7280 -> 0x5959a0): relic mob:u16, old owner:u8, new
//     owner:u8; the client names the destroyed relic in a notice.
//
// Owners: 0 neutral, 1 light, 2 fury; 101-103 mark boss portals, which
// the client draws with portal.tga.
const (
	RelicList   Opcode = 0x2101
	RelicOwner  Opcode = 0x2102
	RelicNotice Opcode = 0x2103
)

type Relic struct {
	Key   uint32
	Owner uint8
	X, Z  float32
}

func DecodeRelicList(p []byte) ([]Relic, error) {
	r := NewReader(p)
	n := int(r.U8())
	out := make([]Relic, 0, n)
	for range n {
		out = append(out, Relic{Key: r.U32(), Owner: r.U8(), X: r.F32(), Z: r.F32()})
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("relic list: %d trailing bytes", r.Remaining())
	}
	return out, nil
}

type RelicOwnerChange struct {
	Key   uint32
	Owner uint8
}

func DecodeRelicOwner(p []byte) (RelicOwnerChange, error) {
	r := NewReader(p)
	v := RelicOwnerChange{Key: r.U32(), Owner: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("relic owner: %d trailing bytes", r.Remaining())
	}
	return v, nil
}

type RelicDestroyed struct {
	MobID              uint16
	OldOwner, NewOwner uint8
}

func DecodeRelicNotice(p []byte) (RelicDestroyed, error) {
	r := NewReader(p)
	v := RelicDestroyed{MobID: r.U16(), OldOwner: r.U8(), NewOwner: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("relic notice: %d trailing bytes", r.Remaining())
	}
	return v, nil
}
