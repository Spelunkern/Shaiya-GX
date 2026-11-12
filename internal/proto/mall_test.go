package proto

import (
	"bytes"
	"testing"
)

func TestMallPacketShapes(t *testing.T) {
	if p := MallBuyPacket("AP_1"); len(p) != 23 || Op(p) != PointBuy || !bytes.Equal(p[2:7], []byte{'A', 'P', '_', '1', 0}) {
		t.Fatal("buy shape", p)
	}
	if p := MallGiftPacket("AP_1", "Player"); len(p) != 44 || Op(p) != PointGiftSend || string(p[23:29]) != "Player" {
		t.Fatal("gift shape", p)
	}
	if p := MallClaimPacket(239); !bytes.Equal(p, []byte{7, 38, 239}) {
		t.Fatal("claim shape", p)
	}
	w := NewWriter(PointBuy)
	w.U8(0)
	w.U32(90)
	w.Fixed("AP_1", 21, true)
	w.U32(123)
	w.U32(10)
	w.U8(1)
	w.Raw([]byte{1, 3, 100, 251, 1})
	r, err := DecodeMallPurchase(w.Bytes())
	if err != nil || !r.HasDetails || r.Points != 90 || r.Cost != 10 || len(r.Items) != 1 || r.Items[0].TypeID != 251 {
		t.Fatal(r, err)
	}
	if _, err := DecodeMallPurchase(w.Bytes()[:len(w.Bytes())-1]); err == nil {
		t.Fatal("truncated purchase accepted")
	}
	reject := NewWriter(PointGiftSend)
	reject.U8(6)
	if r, err := DecodeMallGiftResult(reject.Bytes()); err != nil || r.Result != 6 || r.HasDetails {
		t.Fatal(r, err)
	}
}
