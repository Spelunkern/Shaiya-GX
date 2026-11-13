package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooseAssetsKeepCaseInsensitiveLookupAndPrefix(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Interface", "Icons"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Interface", "Icons", "Atlas.png"), []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.FileCount != 1 {
		t.Fatal(a.FileCount)
	}
	b, err := a.ReadFile(`INTERFACE\icons\ATLAS.PNG`)
	if err != nil || string(b) != "abcdef" {
		t.Fatal(string(b), err)
	}
	b, err = a.ReadPrefix("interface/icons/atlas.png", 3)
	if err != nil || string(b) != "abc" {
		t.Fatal(string(b), err)
	}
	if _, err = a.ReadFile("../secret"); err == nil {
		t.Fatal("path traversal resolved")
	}
}
func TestLooseLegacyByteNameRemainsPortable(t *testing.T) {
	root := t.TempDir()
	name := "mesh\xb4\xf8.3do"
	portable := PortableName(name)
	if portable != "mesh%B4%F8.3do" {
		t.Fatal(portable)
	}
	os.WriteFile(filepath.Join(root, portable), []byte("mesh"), 0600)
	a, err := OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile(name)
	if err != nil || string(b) != "mesh" {
		t.Fatal(err)
	}
}
func TestLooseAssetsRejectAmbiguousNames(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("case-insensitive filesystem cannot create fixture")
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "same.png"), nil, 0600)
	os.WriteFile(filepath.Join(root, "SAME.PNG"), nil, 0600)
	if _, err := OpenDirectory(root); err == nil {
		t.Fatal("ambiguous names accepted")
	}
}
