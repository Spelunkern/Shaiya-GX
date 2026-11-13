// Package db owns the portable SQLite store. Original EP table/column names
// are retained; database names become table prefixes in the single file.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"io"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	UserData = "PS_UserData"
	GameData = "PS_GameData"
	GameDefs = "PS_GameDefs"
	Billing  = "PS_Billing"
)
const Timeout = 10 * time.Second

var (
	mu    sync.Mutex
	pools = map[string]*sql.DB{}
)

// Initialize copies the versioned seed once. A process lock and atomic rename
// prevent login/world from observing a partially copied database at startup.
func Initialize() error {
	path := config.DatabasePath()
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	lock := path + ".init"
	deadline := time.Now().Add(Timeout)
	for {
		if err := os.Mkdir(lock, 0700); err == nil {
			break
		} else if !os.IsExist(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("database initialization lock timed out: %s", lock)
		}
		time.Sleep(25 * time.Millisecond)
	}
	defer os.Remove(lock)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	seed, err := os.Open(config.DatabaseSeed())
	if err != nil {
		return fmt.Errorf("open SQLite seed: %w", err)
	}
	defer seed.Close()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".rapture-init-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = io.Copy(tmp, seed); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Pool shares a single connection per process; SQLite coordinates the two
// server processes through WAL and a bounded busy timeout.
func Pool(database string) (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()
	if p := pools["sqlite"]; p != nil {
		return p, nil
	}
	if err := Initialize(); err != nil {
		return nil, err
	}
	dsn, err := config.DSN(database)
	if err != nil {
		return nil, err
	}
	p, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	p.SetMaxOpenConns(1)
	c, cancel := ctx()
	defer cancel()
	var version int
	if err = p.QueryRowContext(c, "PRAGMA user_version").Scan(&version); err == nil && version != 1 {
		err = fmt.Errorf("unsupported SQLite schema version %d", version)
	}
	if err != nil {
		p.Close()
		return nil, err
	}
	pools["sqlite"] = p
	return p, nil
}
func Open(databases ...string) error {
	for _, d := range databases {
		if _, err := Pool(d); err != nil {
			return err
		}
	}
	return nil
}
func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), Timeout)
}
func longCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}
