package sdata

import (
	"reflect"
	"testing"
)

func TestNativeCashRoundTripAndOverflow(t *testing.T) {
	p := CashProduct{Index: 1, Bag: 4, Name: "Native 🌟", Code: "NATIVE", Description: "All 24 slots", Cost: 0}
	for i := range p.Items {
		p.Items[i] = CashItem{ItemID: 25001, Count: 255}
	}
	raw, err := EncodeCash([]CashProduct{p})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseCash(raw)
	if err != nil || !reflect.DeepEqual(got, []CashProduct{p}) {
		t.Fatal(got, err)
	}
	p.Items[23].Count = 256
	if _, err := EncodeCash([]CashProduct{p}); err == nil {
		t.Fatal("overflow accepted")
	}
}
