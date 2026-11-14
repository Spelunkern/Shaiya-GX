package app

import (
	"bytes"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestBuffTimersFollowSnapshotsRefreshAndServerRemoval(t *testing.T) {
	a := inputApp()
	a.clock = 100
	a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{}
	a.npcSkillDefs[sdata.SkillKey{ID: 31, Level: 1}] = sdata.Skill{KeepTime: 2, FixRange: 1, Effect: 4}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 31, Level: 1}: {KeepTime: 2, FixRange: 1, Effect: 4}}
	apply := func(w *proto.Writer) {
		t.Helper()
		ev, err := a.state.Apply(w.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		a.confirmedBuffs(ev.Op)
	}
	w := proto.NewWriter(proto.CharacterActiveBuffs)
	w.U8(1)
	w.U32(12)
	w.U16(31)
	w.U8(100)
	w.U32(5)
	apply(w)
	if got := a.buffs.rows[0]; got.until != 105 || got.def.Effect != 4 || !got.timed {
		t.Fatal("snapshot ignored remaining or level-100 definition", got)
	}
	a.clock = 106
	if a.buffs.rows[0].seconds(a.clock) != 0 || len(a.state.Buffs) != 1 {
		t.Fatal("elapsed UI timer removed server state")
	}
	w = proto.NewWriter(proto.BuffAdd)
	w.U32(12)
	w.U16(31)
	w.U8(1)
	apply(w)
	if len(a.buffs.rows) != 1 || a.buffs.rows[0].until != 7306 {
		t.Fatal("refresh duplicated instance or lost SData hours", a.buffs)
	}
	if p := a.buffCancelPacket(0); !bytes.Equal(p, []byte{0x11, 0x05, 0xff, 31, 0, 0, 0}) {
		t.Fatalf("cancel must address SkillID, not instance: %x", p)
	}
	if len(a.state.Buffs) != 1 {
		t.Fatal("cancel request removed effect before server response")
	}
	w = proto.NewWriter(proto.BuffRemove)
	w.U32(12)
	apply(w)
	if len(a.buffs.rows) != 0 || len(a.state.Buffs) != 0 {
		t.Fatal("server removal left stale icon")
	}
}

func TestBuffPointerBlocksCameraAndCannotCancelDebuff(t *testing.T) {
	a := inputApp()
	a.buffs.rows = []buffBadge{{buff: proto.ActiveBuff{Instance: 99, Skill: 4}, def: sdata.Skill{Effect: 3}}}
	controls := a.worldControls(&platform.Input{MouseX: 641, MouseY: 109, RightPressed: true, RightDown: true, Down: "W"}, 1024, 768)
	if controls.RightPressed || controls.RightDown || controls.Down != "W" || a.buffCancelPacket(0) != nil {
		t.Fatal("buff click reached camera, swallowed movement or cancelled debuff")
	}
	controls = a.worldControls(&platform.Input{MouseX: 400, MouseY: 400, RightDown: true}, 1024, 768)
	if controls.RightDown {
		t.Fatal("right drag escaped buff pointer owner")
	}
}
