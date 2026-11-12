package app

import (
	"reflect"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGemProtectionSlotsIgnoreLocalDefinitions(t *testing.T) {
	items := map[[2]uint8]proto.InventoryItem{
		{1, 9}: {Type: 44, Count: 1}, {2, 3}: {Type: 100, Count: 2},
		{1, 2}: {Type: 25, Count: 1}, {0, 1}: {Type: 25, Count: 1},
		{1, 4}: {Type: 30, Count: 1}, {1, 5}: {Type: 25, Count: 0},
	}
	want := [][2]uint8{{1, 2}, {1, 9}, {2, 3}}
	if got := gemProtectionSlots(items); !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate slots: got %v want %v", got, want)
	}
}

func TestGemRefreshMatchesBothEchoPositions(t *testing.T) {
	var refresh gemRefresh
	refresh.start([][2]uint8{{1, 2}, {1, 9}})
	packet := func(a, b [2]uint8) []byte {
		w := proto.NewWriter(proto.ItemBagToBag)
		for _, key := range [][2]uint8{a, b} {
			w.Raw([]byte{key[0], key[1]})
			w.Raw(make([]byte, 32))
		}
		return w.Bytes()
	}
	if refresh.acknowledge(packet([2]uint8{1, 2}, [2]uint8{1, 9})) || refresh.acknowledge([]byte{4, 2}) {
		t.Fatal("accepted an unrelated or truncated echo")
	}
	if !refresh.acknowledge(packet([2]uint8{1, 2}, [2]uint8{1, 2})) || len(refresh.pending) != 1 {
		t.Fatal("missing matching echo")
	}
	if refresh.acknowledge(packet([2]uint8{1, 2}, [2]uint8{1, 2})) {
		t.Fatal("duplicate echo acknowledged twice")
	}
	if !refresh.acknowledge(packet([2]uint8{1, 9}, [2]uint8{1, 9})) || len(refresh.pending) != 0 {
		t.Fatal("refresh did not finish")
	}
}
