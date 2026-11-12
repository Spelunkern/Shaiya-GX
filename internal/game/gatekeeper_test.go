package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestGatekeeperMoneyRequiresCompleteServerResponse(t *testing.T) {
	s := New()
	s.Details.Money = 300
	w := proto.NewWriter(proto.Gatekeeper)
	w.U8(1)
	w.U32(200)
	p := w.Bytes()
	for n := 2; n < len(p); n++ {
		if _, err := s.Apply(p[:n]); err == nil || s.Details.Money != 300 {
			t.Fatalf("truncated gatekeeper result %d", n)
		}
	}
	if _, err := s.Apply(p); err != nil || s.Details.Money != 200 || s.GatekeeperSerial != 1 {
		t.Fatal("confirmed price not applied")
	}
}
