package sdata

import "testing"

func FuzzNPCs(f *testing.F) {
	f.Add(make([]byte, 13*4))
	f.Fuzz(func(t *testing.T, b []byte) { ParseNPCs(b) })
}

func FuzzQuests(f *testing.F) {
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) { _, _, _ = ParseQuests(b) })
}
