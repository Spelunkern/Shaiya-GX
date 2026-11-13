package archive

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// PortableName preserves normal names and escapes legacy non-UTF8 bytes or
// characters forbidden by Windows. Extraction and lookup use the same spelling.
func PortableName(name string) string {
	invalid := !utf8.ValidString(name)
	var out strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		if invalid && c >= 128 || c < 32 || strings.ContainsRune(`<>:"|?*`, rune(c)) {
			fmt.Fprintf(&out, "%%%02X", c)
		} else {
			out.WriteByte(c)
		}
	}
	return out.String()
}

// OpenDirectory indexes loose assets once, preserving the original
// case-insensitive lookup and directory APIs used by the client.
func OpenDirectory(root string) (*Archive, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("assets: not a directory: %s", root)
	}
	a := &Archive{directory: root}
	var scan func(string, string) (*Dir, error)
	scan = func(full, name string) (*Dir, error) {
		d := &Dir{Name: name, fileByHash: map[uint32]*File{}, dirByHash: map[uint32]*Dir{}}
		entries, err := os.ReadDir(full)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			p := filepath.Join(full, entry.Name())
			if entry.Type()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("assets: symlink is not portable: %s", p)
			}
			if entry.IsDir() {
				child, err := scan(p, entry.Name())
				if err != nil {
					return nil, err
				}
				if old := d.dirByHash[Hash(child.Name)]; old != nil {
					return nil, fmt.Errorf("assets: ambiguous directories %q and %q in %s", old.Name, child.Name, full)
				}
				d.Dirs = append(d.Dirs, child)
				d.dirByHash[Hash(child.Name)] = child
				continue
			}
			st, err := entry.Info()
			if err != nil {
				return nil, err
			}
			if !st.Mode().IsRegular() || st.Size() > math.MaxUint32 {
				return nil, fmt.Errorf("assets: invalid file %s", p)
			}
			f := &File{Name: entry.Name(), Length: uint32(st.Size()), diskPath: p}
			if old := d.fileByHash[Hash(f.Name)]; old != nil {
				return nil, fmt.Errorf("assets: ambiguous names %q and %q in %s", old.Name, f.Name, full)
			}
			d.Files = append(d.Files, f)
			d.fileByHash[Hash(f.Name)] = f
			a.FileCount++
		}
		return d, nil
	}
	a.Root, err = scan(root, "")
	if err != nil {
		return nil, err
	}
	return a, nil
}
