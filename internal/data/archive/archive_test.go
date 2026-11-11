package archive

import "testing"

// 0x40af40: DJB2 over case-folded signed bytes.
func TestHashFoldsCase(t *testing.T) {
	if Hash("Login.TGA") != Hash("login.tga") {
		t.Fatal("hash must fold A-Z")
	}
	h := uint32(0x1505)
	for _, c := range "ab" {
		h = h*33 + uint32(c)
	}
	if Hash("AB") != h {
		t.Fatalf("hash %#x, want %#x", Hash("AB"), h)
	}
	if Hash("\xe9") == Hash("\xc9") {
		t.Fatal("bytes above 0x7f are not folded")
	}
}
