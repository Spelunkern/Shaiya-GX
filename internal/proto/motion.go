package proto

import "fmt"

const (
	MotionJump       uint8 = 2 // Game.exe 0x4428b7 / 0x560e30, ALT 8.
	MotionEvadeLeft  uint8 = 10
	MotionEvadeRight uint8 = 11
	MotionBackstep   uint8 = 3 // Game.exe 0x560e30, ALT 13.
	MotionSitDown    uint8 = 1
	MotionStandUp    uint8 = 0
)

// Game.exe 0x560e30 swaps each adjacent pair of social motion IDs before
// selecting its ALT action. Posture uses native wire IDs 1/0.
func GestureAction(motion uint8) (int, bool) {
	if motion < 116 || motion > 125 {
		return 0, false
	}
	return int(motion ^ 1), true
}

type CharacterMotionResult struct {
	ID     uint32
	Motion uint8
}

func MotionPacket(motion uint8) []byte {
	w := NewWriter(CharacterMotion)
	w.U8(motion)
	return w.Bytes()
}

func DecodeCharacterMotion(p []byte) (CharacterMotionResult, error) {
	r := NewReader(p)
	m := CharacterMotionResult{ID: r.U32(), Motion: r.U8()}
	if r.Err() != nil {
		return CharacterMotionResult{}, r.Err()
	}
	if r.Remaining() != 0 {
		return CharacterMotionResult{}, fmt.Errorf("motion: trailing bytes")
	}
	return m, nil
}

// LocomotionAction maps the non-conflicting native one-shot movements.
// These wire IDs are distinct from the ALT indices.
func LocomotionAction(motion uint8) (int, bool) {
	switch motion {
	case MotionJump:
		return 8, true
	case MotionBackstep:
		return 13, true
	case MotionEvadeLeft:
		return 14, true
	case MotionEvadeRight:
		return 15, true
	}
	return 0, false
}
