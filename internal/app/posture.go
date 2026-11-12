package app

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

// Rapture retains auto-attack when a motion arrives. Stop it before sitting;
// the posture itself remains pending until the echoed 0x0506 arrives.
func (a *App) requestPosture(motion uint8) bool {
	if motion == proto.MotionSitDown {
		a.wv.sitAttack = 0
		if err := a.world.Send(proto.StopAttackPacket()); err != nil {
			a.backToLogin("Unable to stop attack before sitting.")
			return false
		}
	}
	if err := a.world.Send(proto.MotionPacket(motion)); err != nil {
		a.backToLogin("Unable to change sitting state.")
		return false
	}
	a.wv.sitPending = true
	a.wv.sitDeadline = a.clock + 30
	return true
}
