package proto

import "fmt"

type VehicleResult struct{ Result, On uint8 }

func DecodeVehicleResult(p []byte) (VehicleResult, error) {
	r := NewReader(p)
	v := VehicleResult{Result: r.U8(), On: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.On > 1 {
		return v, fmt.Errorf("invalid vehicle result")
	}
	return v, nil
}

type MountShape struct {
	ID    uint32
	Shape uint8
}

func DecodeMountShape(p []byte) (MountShape, error) {
	r := NewReader(p)
	v := MountShape{ID: r.U32(), Shape: r.U8()}
	return v, r.Err()
}
func UseVehiclePacket() []byte { return NewWriter(UseVehicle).Bytes() }
