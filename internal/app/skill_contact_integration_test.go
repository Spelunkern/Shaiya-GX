//go:build integration

package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

// Reuses the disposable fighter, real GL actor and muted device of the combat
// cue test; no second context, connection or simulated skill outcome is needed.
func checkPhysicalSkillContact(t *testing.T, a *App, send func([]byte), wait func(func() bool)) {
	t.Helper()
	const skillID = 57 // Double Damage, original fighter level three.
	st := a.state
	number := uint8(0)
	for _, k := range st.Skills {
		if k.ID == skillID {
			number = k.Number
		}
	}
	if number == 0 {
		send(proto.LearnSkillPacket(skillID, 1))
		wait(func() bool { return st.LastLearn.ID == skillID })
		if st.LastLearn.Result != 0 {
			t.Fatal("fixture could not learn Double Damage", st.LastLearn)
		}
		number = st.LastLearn.Number
	}
	mob := testworld.LivingMob(t, st, send, wait)
	a.wv.pos = render.V3{mob.X + 1, mob.Y, mob.Z}
	send(proto.MovePacket(0, 0, a.wv.pos[0], a.wv.pos[1], a.wv.pos[2]))
	send(proto.UseSkillPacket(number, mob.ID, false))
	deadline := time.Now().Add(5 * time.Second)
	for st.LastSkill.ID != skillID {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("physical skill result missing")
		}
		time.Sleep(time.Millisecond)
	}
	if st.LastSkill.Result == 3 {
		if len(a.wv.impactCues) != 0 || len(a.wv.impacts) != 0 {
			t.Fatal("miss created a physical impact")
		}
		t.Log("Double Damage missed; no impact queued (positive contact covered by the GL preview)")
		return
	}
	if st.LastSkill.Result > 1 || len(a.wv.impactCues) != 1 || len(a.wv.impacts) != 0 {
		t.Fatal("physical result did not wait for contact", st.LastSkill, len(a.wv.impactCues), len(a.wv.impacts))
	}
	cue := a.wv.impactCues[0]
	if cue.target != mob.ID || cue.actor != a.wv.self || cue.time <= 0 {
		t.Fatal("physical cue lost its real actor/target", cue)
	}
	pools, hp := st.Pools, mob.HP
	for i := 0; i < 180 && len(a.wv.impactCues) > 0; i++ {
		if a.wv.self.Time < cue.time && len(a.wv.impacts) != 0 {
			t.Fatal("physical effect appeared before ALT contact")
		}
		a.wv.self.Update(1.0 / 60)
		a.updateSkillImpactCues()
	}
	if len(a.wv.impactCues) != 0 || len(a.wv.impacts) != 1 || a.wv.self.Time < cue.time || st.Pools != pools || mob.HP != hp {
		t.Fatal("physical contact missing or changed confirmed gameplay")
	}
	a.updateSkillImpactCues()
	if len(a.wv.impacts) != 1 {
		t.Fatal("physical contact repeated its effect")
	}
	t.Logf("Double Damage confirmed by Rapture; one effect at ALT contact %.3fs, HP/pools unchanged during presentation", cue.time)
}
