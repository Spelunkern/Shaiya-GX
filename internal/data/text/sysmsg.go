// Package text reads the client's message tables.
package text

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"unicode/utf16"
)

// SysMsg is sysmsg-uni.txt: UTF-16LE lines `id<TAB><TAB>"text"`, read by
// the client's message lookup (FUN_0041d4a0).
type SysMsg map[int]string

func ParseSysMsg(b []byte) SysMsg {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		b = b[2:]
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	m := SysMsg{}
	sc := bufio.NewScanner(bytes.NewReader([]byte(string(utf16.Decode(u)))))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		i := strings.IndexAny(line, " \t")
		if i < 0 {
			continue
		}
		id, err := strconv.Atoi(line[:i])
		if err != nil {
			continue
		}
		v := strings.TrimSpace(line[i:])
		v = strings.TrimPrefix(v, "\"")
		v = strings.TrimSuffix(v, "\"")
		m[id] = strings.TrimSpace(strings.ReplaceAll(v, `\n`, "\n"))
	}
	return m
}

// Get returns the message or "" when missing.
func (m SysMsg) Get(id int) string { return m[id] }
