package platform

import (
	"os"
	"testing"
)

func TestWindowStatePersistence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	state, err := loadWindowState()
	if err != nil || state.Width != 0 {
		t.Fatal("first launch must use defaults", state, err)
	}
	w := &Window{rememberSize: true, normalSize: windowState{Width: 1376, Height: 824}, maximized: true}
	w.saveSize()
	state, err = loadWindowState()
	if err != nil || state != (windowState{Width: 1376, Height: 824, Maximized: true}) {
		t.Fatal("lost normal size or maximization", state, err)
	}
	w.rememberSize, w.normalSize.Width = false, 640
	w.saveSize()
	state, err = loadWindowState()
	if err != nil || state.Width != 1376 {
		t.Fatal("nonpersistent window overwrote desktop preferences", state, err)
	}
	path, err := windowStatePath()
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{`{"Width":0,"Height":768}`, `{"Width":1024,"Height":-1}`, `{"Width":999999,"Height":768}`, `broken`} {
		if err := os.WriteFile(path, []byte(invalid), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadWindowState(); err == nil {
			t.Fatal("accepted invalid preferences", invalid)
		}
	}
}
