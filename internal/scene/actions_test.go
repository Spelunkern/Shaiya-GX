package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"testing"
)

func TestSkillActionsFollowNativeCommonBranches(t *testing.T) {
	if SkillCastAction(1, 3) != 88 || SkillCastAction(0, 1) != 85 || SkillCastAction(0, 4) != 91 {
		t.Fatal("casting ready action")
	}
	for _, tc := range []struct {
		anim                   uint16
		attack, effect, weapon uint8
		want                   int
	}{
		{100, 1, 3, 7, 100}, {110, 1, 1, 1, 110}, {0, 1, 1, 1, 35},
		{1, 3, 3, 13, 90}, {0, 3, 1, 13, 87}, {0, 0, 4, 1, 93},
	} {
		if got := SkillAction(tc.anim, tc.attack, tc.effect, tc.weapon, 0); got != tc.want {
			t.Errorf("%+v: %d", tc, got)
		}
	}
}

func TestCastCanBeInterruptedWithoutReleasingDeath(t *testing.T) {
	c := &Character{locked: true, castLeft: 2, clip: &Clip{ani: &model.ANI{Start: 0, End: 300}}}
	c.Update(.5)
	if !c.locked || c.castLeft != 1.5 {
		t.Fatal("cast released before its duration")
	}
	c.CancelCast()
	if c.locked || c.castLeft != 0 {
		t.Fatal("movement did not interrupt casting")
	}
	c.locked, c.hold = true, true
	c.CancelCast()
	if !c.locked || !c.hold {
		t.Fatal("cast interruption released death")
	}
}

func TestDeathHoldsLastFrame(t *testing.T) {
	c := &Character{locked: true, hold: true, clip: &Clip{ani: &model.ANI{Start: 0, End: 30}}}
	c.Update(10)
	if c.Time >= 1 || c.Time < 0.99 {
		t.Fatal("death looped instead of holding its final frame")
	}
	c.ReleaseAction()
	if c.locked {
		t.Fatal("resurrection did not release death pose")
	}
}
