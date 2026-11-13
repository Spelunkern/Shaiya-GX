package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestSkillResourcesRejectBeforePacketAndPresentation(t *testing.T) {
	for _, sp := range []bool{false, true} {
		a := inputApp()
		a.msg = text.SysMsg{214: "<hp> is not enough", 417: "SP", 418: "MP"}
		d := sdata.Skill{MP: 10}
		expected := "MP is not enough"
		if sp {
			d = sdata.Skill{SP: 10}
			expected = "SP is not enough"
		}
		a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 12, Level: 1}: d}
		if p := a.learnedSkillPacket(proto.LearnedSkill{ID: 12, Level: 1}); p != nil {
			t.Fatal("rejected skill produced a packet", p)
		}
		if len(a.state.Chat) == 0 {
			t.Fatal("no localized feedback")
		}
		if got := a.state.Chat[len(a.state.Chat)-1].Text; got != expected {
			t.Fatal(got)
		}
		if a.wv.requestedSkill != nil {
			t.Fatal("predicted presentation started")
		}
	}
	a := inputApp()
	a.state.Pools.MP = 10
	if !a.skillResourcesAvailable(sdata.Skill{MP: 10}) {
		t.Fatal("exact resource budget rejected")
	}
	if !a.skillResourcesAvailable(sdata.Skill{SP: 1}) {
		t.Fatal("consume-current-pool convention changed")
	}
}

func TestSelfSelectionByPortraitAndF2(t *testing.T) {
	for _, portrait := range []bool{false, true} {
		a := inputApp()
		a.state.SelfID = 42
		a.state.Target = 7
		if portrait {
			a.updateHUDPointer(&platform.Input{Clicked: true, MouseX: 45, MouseY: 50}, 1024, 768)
		} else {
			a.updateCombat(&platform.Input{Keys: []platform.Key{platform.KeySelfTarget}}, 1024, 768)
		}
		if a.state.Target != 42 || a.state.PreviousTarget != 7 {
			t.Fatal(a.state.Target, a.state.PreviousTarget)
		}
	}
}
