package sdata

import "testing"

func FuzzMonsters(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) { ParseMonsters(b) })
}
