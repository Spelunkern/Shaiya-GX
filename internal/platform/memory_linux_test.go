package platform

import "testing"

func TestResidentBytesReadsProcessMemory(t *testing.T) {
	n, err := ResidentBytes()
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("resident memory reported zero")
	}
}
