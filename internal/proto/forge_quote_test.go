package proto

import (
	"reflect"
	"testing"
)

func TestForgeQuoteRequestCodec(t *testing.T) {
	q := ForgeQuoteRequest{Serial: 42, Kind: ForgeQuoteEnchant, Item: [2]uint8{0, 2}, Lapis: [2]uint8{1, 4}, Material: [2]uint8{1, 5}, Hammer: [2]uint8{255, 255}, Count: 100}
	p := ForgeQuotePacket(q)
	got, err := DecodeForgeQuoteRequest(p)
	if err != nil || !reflect.DeepEqual(got, q) {
		t.Fatal(got, err)
	}
	for _, bad := range [][]byte{p[:len(p)-1], append(p, 0)} {
		if _, err := DecodeForgeQuoteRequest(bad); err == nil {
			t.Fatal("malformed quote accepted")
		}
	}
}
