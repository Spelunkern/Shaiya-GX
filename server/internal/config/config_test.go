package config

import (
	"net/url"
	"path/filepath"
	"testing"
)

func TestSQLiteFileURIsHaveNoAuthority(t *testing.T) {
	for _, path := range []string{"/tmp/Portable copy/área.sqlite", "C:/Portable copy/rapture.sqlite", "Z:/tmp/Portable copy/rapture.sqlite", "//server/share/rapture.sqlite"} {
		uri := url.URL{Scheme: "file", Path: sqliteURIPath(path)}
		parsed, err := url.Parse(uri.String())
		if err != nil || parsed.Host != "" || parsed.Opaque != "" {
			t.Fatalf("path %q became a non-file authority: %s, %v", path, uri.String(), err)
		}
		if len(path) > 1 && path[1] == ':' && parsed.Path != "/"+path {
			t.Fatalf("drive path was lost: %s", uri.String())
		}
	}
}

func TestSQLiteDSNRetainsPathsAndConnectionSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Portable copy", "área.sqlite")
	t.Setenv("RAPTURE_DB_PATH", path)
	value, err := DSN("PS_GameData")
	if err != nil {
		t.Fatal(err)
	}
	uri, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	if uri.Host != "" || uri.Path != sqliteURIPath(filepath.ToSlash(path)) {
		t.Fatal("database path changed", value)
	}
	if uri.Query().Get("_txlock") != "immediate" || len(uri.Query()["_pragma"]) != 4 {
		t.Fatal("connection settings changed", value)
	}
}
