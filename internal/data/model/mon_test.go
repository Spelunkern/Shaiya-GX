package model

import "testing"

func FuzzMON(f *testing.F) {
	f.Add([]byte("MO2\x00\x00\x00\x00"))
	f.Fuzz(func(t *testing.T, b []byte) { ParseMON(b) })
}
