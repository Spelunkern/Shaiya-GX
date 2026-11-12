//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestCombatCuesAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_COMBAT") != "1" || os.Getenv("SHAIYA_AUDIO_TEST") != "1" {
		t.Skip("requires disposable combat fixture, GL and audio device (muted)")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "weapon audio probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	st, world, send, wait := testworld.Enter(t)
	mob := testworld.LivingMob(t, st, send, wait)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.sound, err = audio.New(arc.ReadFile, 0)
	if err != nil {
		t.Fatal(err)
	}
	a.soundErrors = map[string]bool{}
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	if a.wv.self == nil {
		t.Fatal("fixture player model did not load")
	}
	a.wv.pos = render.V3{mob.X + 1, mob.Y, mob.Z}
	send(proto.MovePacket(0, 0, a.wv.pos[0], a.wv.pos[1], a.wv.pos[2]))
	send(proto.MobAttackPacket(mob.ID))
	deadline := time.Now().Add(8 * time.Second)
	for len(a.wv.attackSounds) < 2 {
		a.pumpWorld()
		a.updateAudio()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("confirmed weapon hit did not queue its swing/contact", st.LastAttack)
		}
		time.Sleep(time.Millisecond)
	}
	send(proto.StopAttackPacket())
	result, pools := st.LastAttack, st.Pools
	if result.Source != st.SelfID || result.Target != mob.ID || result.HP == 0 || result.Result > 1 {
		t.Fatal("audio was not tied to confirmed positive damage", result)
	}
	swing, hit := a.wv.attackSounds[0], a.wv.attackSounds[1]
	if swing.entity != st.SelfID || hit.entity != mob.ID || swing.time <= 0 || hit.time <= 0 || swing.actor != a.wv.self || hit.actor != a.wv.self {
		t.Fatal("weapon audio lost its actor/contact anchor", swing, hit)
	}
	for i := 0; i < 120 && len(a.wv.attackSounds) > 0; i++ {
		a.wv.self.Update(1.0 / 60)
		a.updateAudio()
	}
	if len(a.wv.attackSounds) != 0 || st.Pools != pools {
		t.Fatal("weapon cues repeated or changed gameplay pools")
	}
	t.Logf("Rapture hit queued %s at %.3fs and %s at ALT contact %.3fs; both drained muted, pools unchanged", swing.file, swing.time, hit.file, hit.time)
	checkPhysicalSkillContact(t, a, send, wait)
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}
