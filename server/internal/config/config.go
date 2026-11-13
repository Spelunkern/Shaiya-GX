// Package config resolves runtime settings from the environment, falling
// back to .local/runtime.env for local overrides. SQLite needs no credentials.
package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const ServerID = 1

// LoadRuntimeEnv reads KEY=VALUE lines from .local/runtime.env into the
// process environment without overriding variables that are already set.
func LoadRuntimeEnv() error {
	// Explicit data/database overrides keep the caller's working directory.
	// Otherwise standalone binaries find their own package even when launched
	// from another directory (including a double-click inside build/).
	if os.Getenv("RAPTURE_DATA_DIR") == "" && os.Getenv("RAPTURE_DB_PATH") == "" {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		if resolved, err := filepath.EvalSymlinks(executable); err == nil {
			executable = resolved
		}
		directory := filepath.Dir(executable)
		for _, root := range []string{directory, filepath.Dir(directory)} {
			if info, err := os.Stat(filepath.Join(root, "data", "rapture.sqlite")); err == nil && !info.IsDir() {
				if err = os.Chdir(root); err != nil {
					return fmt.Errorf("enter server directory: %w", err)
				}
				break
			}
		}
	}
	LoadRuntimeEnvFrom(".")
	return nil
}

// LoadRuntimeEnvFrom is LoadRuntimeEnv for the checkout at root.
func LoadRuntimeEnvFrom(root string) {
	defer resolvePaths(root)
	file, err := os.Open(filepath.Join(root, ".local", "runtime.env"))
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
		if !ok || key == "" || strings.HasPrefix(key, "#") {
			continue
		}
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, value)
		}
	}
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// DatabasePath is the mutable runtime copy, never the versioned seed.
func DatabasePath() string { return env("RAPTURE_DB_PATH", filepath.Join("runtime", "rapture.sqlite")) }
func DatabaseSeed() string { return env("RAPTURE_DB_SEED", filepath.Join(DataDir(), "rapture.sqlite")) }

// DSN applies connection settings to every connection, including both servers.
func DSN(_ string) (string, error) {
	path, err := filepath.Abs(DatabasePath())
	if err != nil {
		return "", err
	}
	u := url.URL{Scheme: "file", Path: sqliteURIPath(filepath.ToSlash(path))}
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(10000)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(FULL)")
	q.Add("_pragma", "foreign_keys(ON)")
	q.Set("_txlock", "immediate")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// DataDir holds map data (data/maps/*.svmap).
func DataDir() string { return env("RAPTURE_DATA_DIR", "data") }

// PublicIP is the world address sent to clients after server selection.
func PublicIP() string { return env("RAPTURE_PUBLIC_IP", "127.0.0.1") }

func LoginListen() string { return env("RAPTURE_LOGIN_LISTEN", "127.0.0.1:30800") }
func WorldListen() string { return env("RAPTURE_WORLD_LISTEN", "127.0.0.1:30810") }

// Launchers and direct binaries resolve storage/data against their checkout.
func resolvePaths(root string) {
	for key, value := range map[string]string{"RAPTURE_DATA_DIR": DataDir(), "RAPTURE_DB_PATH": DatabasePath(), "RAPTURE_DB_SEED": DatabaseSeed()} {
		if !filepath.IsAbs(value) {
			if absolute, err := filepath.Abs(filepath.Join(root, value)); err == nil {
				os.Setenv(key, absolute)
			}
		}
	}
}

// A Windows drive is part of the URI path, never its authority. Without the
// leading slash net/url emits file://C:/..., which SQLite rejects.
func sqliteURIPath(path string) string {
	if len(path) >= 2 && path[1] == ':' {
		return "/" + path
	}
	return path
}
