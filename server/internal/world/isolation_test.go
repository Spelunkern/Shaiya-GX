package world

import (
	"os"
	"path/filepath"
	"testing"
)

// Every integration test in this package must use an isolated runtime, even
// when the developer has server/.local/runtime.env and a personal database.
func TestMain(m *testing.M) {
	root, err := filepath.Abs("../..")
	if err != nil {
		panic(err)
	}
	temporary, err := os.MkdirTemp("", "gx-world-integration-")
	if err != nil {
		panic(err)
	}
	os.Setenv("RAPTURE_DB_PATH", filepath.Join(temporary, "runtime.sqlite"))
	os.Setenv("RAPTURE_DB_SEED", filepath.Join(root, "data", "rapture.sqlite"))
	os.Setenv("RAPTURE_DATA_DIR", filepath.Join(root, "data"))
	code := m.Run()
	os.RemoveAll(temporary)
	os.Exit(code)
}
