package db

import (
	"os"
	"path/filepath"
	"testing"
)

// Include older integration tests in the same isolation boundary as the newer
// fixtures. No test is allowed to fall back to a developer's runtime database.
func TestMain(m *testing.M) {
	root, err := filepath.Abs("../..")
	if err != nil {
		panic(err)
	}
	temporary, err := os.MkdirTemp("", "gx-storage-tests-")
	if err != nil {
		panic(err)
	}
	os.Setenv("RAPTURE_DB_PATH", filepath.Join(temporary, "runtime.sqlite"))
	os.Setenv("RAPTURE_DB_SEED", filepath.Join(root, "data", "rapture.sqlite"))
	os.Setenv("RAPTURE_DATA_DIR", filepath.Join(root, "data"))
	code := m.Run()
	mu.Lock()
	for _, pool := range pools {
		pool.Close()
	}
	mu.Unlock()
	os.RemoveAll(temporary)
	os.Exit(code)
}
