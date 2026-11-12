package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestItemRepairMoneyOnlyAfterCompleteResponse(t *testing.T) {
	s := New()
	s.Details.Money = 1000
	w := proto.NewWriter(proto.ItemRepair)
	w.Raw([]byte{1, 2})
	w.U32(900)
	p := w.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || s.Details.Money != 1000 {
			t.Fatalf("partial repair packet %d", i)
		}
	}
	if _, err := s.Apply(p); err != nil || s.Details.Money != 900 {
		t.Fatal("confirmed repair money not applied", err)
	}
}
