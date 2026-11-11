// Package archive reads the original data.sah index and data.saf payloads.
//
// Format (Parsec Shaiya/Data/Sah.cs): "SAH", version:i32, fileCount:i32,
// 40 padding bytes, then the root directory: name, fileCount:i32, files
// (name, offset:i64, length:i32, version:i32), subdirCount:i32, subdirs.
// Names are length:i32 + bytes (terminator included).
//
// Lookup follows the client (sah_loader_seh.cpp 0x40af40 / get_or_insert):
// each directory maps a case-folded signed-byte DJB2 hash of the name to
// its entry, and a later duplicate replaces an earlier one.
package archive

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	maxName    = 1024
	maxEntries = 1 << 20
)

type File struct {
	Name    string
	Offset  int64
	Length  uint32
	Version int32
}

type Dir struct {
	Name  string
	Files []*File
	Dirs  []*Dir

	fileByHash map[uint32]*File
	dirByHash  map[uint32]*Dir
}

// Archive is safe for concurrent reads.
type Archive struct {
	Version   int32
	FileCount int32
	Root      *Dir

	mu  sync.Mutex
	saf *os.File
}

// Hash is the client's name hash (0x40af40).
func Hash(name string) uint32 {
	h := uint32(0x1505)
	for i := 0; i < len(name); i++ {
		c := int32(int8(name[i]))
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		h = h*33 + uint32(c)
	}
	return h
}

// Open reads dir/data.sah and opens dir/data.saf.
func Open(dir string) (*Archive, error) {
	return OpenFiles(dir+"/data.sah", dir+"/data.saf")
}

func OpenFiles(sahPath, safPath string) (*Archive, error) {
	f, err := os.Open(sahPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	a := &Archive{}
	r := &reader{r: bufio.NewReaderSize(f, 1<<20)}
	sig := r.bytes(3)
	if r.err == nil && string(sig) != "SAH" {
		return nil, fmt.Errorf("archive: %s: bad signature %q", sahPath, sig)
	}
	a.Version, a.FileCount = r.i32(), r.i32()
	r.bytes(40)
	a.Root = r.dir(0)
	if r.err != nil {
		return nil, fmt.Errorf("archive: %s: %w", sahPath, r.err)
	}
	if a.saf, err = os.Open(safPath); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Archive) Close() error { return a.saf.Close() }

// Lookup resolves a path like "interface/login.tga" (either slash).
func (a *Archive) Lookup(path string) (*File, bool) {
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return nil, false
	}
	d := a.Root
	for _, p := range parts[:len(parts)-1] {
		if d = d.dirByHash[Hash(p)]; d == nil {
			return nil, false
		}
	}
	f, ok := d.fileByHash[Hash(parts[len(parts)-1])]
	return f, ok
}

// LookupDir resolves a directory path ("" is the root).
func (a *Archive) LookupDir(path string) (*Dir, bool) {
	d := a.Root
	for _, p := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		if d = d.dirByHash[Hash(p)]; d == nil {
			return nil, false
		}
	}
	return d, true
}

var ErrNotFound = errors.New("archive: not found")

// ReadFile returns the bytes of path.
func (a *Archive) ReadFile(path string) ([]byte, error) {
	f, ok := a.Lookup(path)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
	}
	return a.Read(f)
}

func (a *Archive) Read(f *File) ([]byte, error) {
	buf := make([]byte, f.Length)
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, err := a.saf.ReadAt(buf, f.Offset); err != nil && !(errors.Is(err, io.EOF) && f.Length == 0) {
		return nil, fmt.Errorf("archive: %s: %w", f.Name, err)
	}
	return buf, nil
}

// Walk visits every file with its full path.
func (a *Archive) Walk(fn func(path string, f *File)) {
	var walk func(prefix string, d *Dir)
	walk = func(prefix string, d *Dir) {
		for _, f := range d.Files {
			fn(prefix+f.Name, f)
		}
		for _, s := range d.Dirs {
			walk(prefix+s.Name+"/", s)
		}
	}
	walk("", a.Root)
}

type reader struct {
	r   *bufio.Reader
	err error
}

func (r *reader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	b := make([]byte, n)
	_, r.err = io.ReadFull(r.r, b)
	return b
}

func (r *reader) i32() int32 {
	b := r.bytes(4)
	if b == nil {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(b))
}

func (r *reader) i64() int64 {
	b := r.bytes(8)
	if b == nil {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(b))
}

func (r *reader) name() string {
	n := r.i32()
	if n < 0 || n > maxName {
		r.fail(fmt.Errorf("name length %d", n))
		return ""
	}
	b := r.bytes(int(n))
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

func (r *reader) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

func (r *reader) count() int {
	n := r.i32()
	if n < 0 || n > maxEntries {
		r.fail(fmt.Errorf("entry count %d", n))
		return 0
	}
	return int(n)
}

func (r *reader) dir(depth int) *Dir {
	if depth > 64 {
		r.fail(errors.New("directory nesting too deep"))
		return nil
	}
	d := &Dir{Name: r.name(), fileByHash: map[uint32]*File{}, dirByHash: map[uint32]*Dir{}}
	n := r.count()
	for i := 0; i < n && r.err == nil; i++ {
		f := &File{Name: r.name(), Offset: r.i64(), Length: uint32(r.i32()), Version: r.i32()}
		d.Files = append(d.Files, f)
		d.fileByHash[Hash(f.Name)] = f
	}
	n = r.count()
	for i := 0; i < n && r.err == nil; i++ {
		if s := r.dir(depth + 1); s != nil {
			d.Dirs = append(d.Dirs, s)
			d.dirByHash[Hash(s.Name)] = s
		}
	}
	if depth == 0 {
		d.Name = "" // the root's own name is not part of paths
	}
	return d
}
