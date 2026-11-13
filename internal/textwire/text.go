// Package textwire keeps GX network text valid UTF-8 within the original byte budgets.
package textwire

import (
	"bytes"
	"github.com/go-text/typesetting/segmenter"
	"golang.org/x/text/encoding/charmap"
	"strings"
	"unicode/utf8"
)

// Decode prefers UTF-8. Legacy original-client ANSI strings remain readable.
func Decode(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	if utf8.Valid(b) {
		return string(b)
	}
	out, _ := charmap.Windows1252.NewDecoder().Bytes(b)
	return string(out)
}
func Encode(s string) []byte { return []byte(strings.ToValidUTF8(s, "\uFFFD")) }

// Prefix never splits a code point or a grapheme (emoji sequences included).
func Prefix(s string, budget int) string {
	if budget <= 0 {
		return ""
	}
	if len(s) <= budget && utf8.ValidString(s) {
		return s
	}
	s = strings.ToValidUTF8(s, "\uFFFD")
	var seg segmenter.Segmenter
	seg.InitWithString(s)
	it := seg.GraphemeIterator()
	end := 0
	for it.Next() {
		g := it.Grapheme()
		n := len(string(g.Text))
		if end+n > budget {
			break
		}
		end += n
	}
	return s[:end]
}
func DropLast(s string) string {
	var seg segmenter.Segmenter
	seg.InitWithString(s)
	it := seg.GraphemeIterator()
	last := 0
	for it.Next() {
		last = it.Grapheme().Offset
	}
	return string([]rune(s)[:last])
}

// Graphemes returns user-perceived characters without separating combining marks or emoji.
func Graphemes(s string) []string {
	var seg segmenter.Segmenter
	seg.InitWithString(s)
	it := seg.GraphemeIterator()
	var out []string
	for it.Next() {
		out = append(out, string(it.Grapheme().Text))
	}
	return out
}
func DropFirst(s string) string {
	clusters := Graphemes(s)
	if len(clusters) == 0 {
		return ""
	}
	return s[len(clusters[0]):]
}
