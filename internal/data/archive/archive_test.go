package archive

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestReadPrefixKeepsArchiveEntryIntact(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.saf")
	if err := os.WriteFile(file, []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}
	handle, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	entry := &File{Name: "model.smod", Length: 6}
	a := &Archive{saf: handle, Root: &Dir{fileByHash: map[uint32]*File{Hash(entry.Name): entry}}}
	prefix, err := a.ReadPrefix("MODEL.SMOD", 2)
	if err != nil || string(prefix) != "ab" || entry.Length != 6 {
		t.Fatal("prefix altered archive entry", string(prefix), err)
	}
	all, err := a.ReadPrefix("model.smod", 100)
	if err != nil || string(all) != "abcdef" {
		t.Fatal("prefix beyond file", string(all), err)
	}
	if _, err := a.ReadPrefix("model.smod", -1); err == nil {
		t.Fatal("negative prefix")
	}
}
