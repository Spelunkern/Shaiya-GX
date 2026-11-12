package proto

import "fmt"

type GatekeeperResult struct {
	Result uint8
	Money  uint32
}

func DecodeGatekeeperResult(p []byte) (GatekeeperResult, error) {
	r := NewReader(p)
	v := GatekeeperResult{Result: r.U8(), Money: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 {
		return v, fmt.Errorf("invalid gatekeeper result")
	}
	return v, nil
}
func GatekeeperPacket(id uint32, destination uint8) []byte {
	w := NewWriter(Gatekeeper)
	w.U32(id)
	w.U8(destination)
	return w.Bytes()
}
