package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
)

func TestDetectInfoDisplaysConfirmedValues(t *testing.T) {
	a := inputApp()
	packets := make(chan []byte, 1)
	a.world = &session.World{Packets: packets}
	w := proto.NewWriter(proto.SkillDetectInfo)
	w.U32(4)
	w.U16(57)
	w.U8(3)
	packets <- w.Bytes()
	a.pumpWorld()
	if len(a.state.Chat) != 1 || a.state.Chat[0].Text != "Detection: level 57, Ultimate mode, defense element Wind I." {
		t.Fatalf("detection result: %+v", a.state.Chat)
	}
}
