package control

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"os"
	"path/filepath"
	"strings"
)

// SQLite is embedded; database availability follows the runtime file.
func DatabaseRunning() bool {
	path := config.DatabasePath()
	if !filepath.IsAbs(path) {
		path = filepath.Join(RepoRoot(), path)
	}
	_, err := os.Stat(path)
	return err == nil
}
func EnsureDatabase() error {
	config.LoadRuntimeEnvFrom(RepoRoot())
	return db.Open(db.UserData, db.GameData, db.GameDefs)
}

// No external database process needs to be stopped.
func StopDatabase() {}

// lastLogLine returns the last non-empty line of logs/<name>.log.
func lastLogLine(name string) string {
	data, err := os.ReadFile(filepath.Join(RepoRoot(), "logs", name+".log"))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
