// Package proto holds the Shaiya EP5.4 opcodes and one codec per packet.
// Payloads are opcode:u16 + body, little-endian, serialized field by field.
// Each codec cites the client handler (dispatcher fun_0059cf70 in
// shaiya-client-decomp network/dispatcher.cpp) and the Rapture builder or
// handler it was cross-checked with. See docs/protocolo.md.
package proto

import (
	"encoding/binary"
	"errors"
	"math"
)

var ErrShort = errors.New("proto: short packet")

// Reader decodes a body; the first error sticks and later reads return zero.
type Reader struct {
	b   []byte
	off int
	err error
}

// NewReader starts after the opcode.
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}

func (r *Reader) take(n int) []byte {
	if r.err != nil || n < 0 || r.off+n > len(r.b) {
		r.err = ErrShort
		return nil
	}
	p := r.b[r.off : r.off+n]
	r.off += n
	return p
}

func (r *Reader) Err() error     { return r.err }
func (r *Reader) Remaining() int { return len(r.b) - r.off }

func (r *Reader) U8() uint8 {
	if p := r.take(1); p != nil {
		return p[0]
	}
	return 0
}
func (r *Reader) Bool() bool { return r.U8() != 0 }
func (r *Reader) U16() uint16 {
	if p := r.take(2); p != nil {
		return binary.LittleEndian.Uint16(p)
	}
	return 0
}
func (r *Reader) U32() uint32 {
	if p := r.take(4); p != nil {
		return binary.LittleEndian.Uint32(p)
	}
	return 0
}
func (r *Reader) I32() int32   { return int32(r.U32()) }
func (r *Reader) F32() float32 { return math.Float32frombits(r.U32()) }
func (r *Reader) U64() uint64 {
	if p := r.take(8); p != nil {
		return binary.LittleEndian.Uint64(p)
	}
	return 0
}

// Bytes returns a copy of n bytes.
func (r *Reader) Bytes(n int) []byte { return append([]byte(nil), r.take(n)...) }

// Fixed reads a null-padded string of n bytes (client ANSI codepage,
// mapped 1:1 to Latin-1).
func (r *Reader) Fixed(n int) string { return Latin1(r.take(n)) }

// Text reads length:u8 (terminator included) + text, the chat shape.
func (r *Reader) Text() string { return Latin1(r.take(int(r.U8()))) }

// Latin1 decodes a null-terminated single-byte string.
func Latin1(b []byte) string {
	rs := make([]rune, 0, len(b))
	for _, c := range b {
		if c == 0 {
			break
		}
		rs = append(rs, rune(c))
	}
	return string(rs)
}

// EncodeLatin1 maps runes above 0xFF to '?'.
func EncodeLatin1(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, c := range s {
		if c <= 0xFF {
			out = append(out, byte(c))
		} else {
			out = append(out, '?')
		}
	}
	return out
}

// Writer builds a payload starting with its opcode.
type Writer struct{ b []byte }

func NewWriter(op Opcode) *Writer {
	w := &Writer{b: make([]byte, 0, 64)}
	w.U16(uint16(op))
	return w
}

func (w *Writer) U8(v uint8)   { w.b = append(w.b, v) }
func (w *Writer) U16(v uint16) { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *Writer) U32(v uint32) { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *Writer) F32(v float32) {
	w.U32(math.Float32bits(v))
}
func (w *Writer) Bool(v bool) {
	if v {
		w.U8(1)
	} else {
		w.U8(0)
	}
}
func (w *Writer) Raw(p []byte) { w.b = append(w.b, p...) }

// Fixed writes s null-padded (or truncated) to exactly n bytes, always
// leaving at least one terminator when keepNul is set.
func (w *Writer) Fixed(s string, n int, keepNul bool) {
	b := EncodeLatin1(s)
	max := n
	if keepNul {
		max = n - 1
	}
	if len(b) > max {
		b = b[:max]
	}
	w.b = append(w.b, b...)
	w.b = append(w.b, make([]byte, n-len(b))...)
}

// Text writes length:u8 (terminator included) + text + NUL, max 127 chars.
func (w *Writer) Text(s string) {
	b := EncodeLatin1(s)
	if len(b) > 127 {
		b = b[:127]
	}
	w.U8(uint8(len(b) + 1))
	w.b = append(append(w.b, b...), 0)
}

func (w *Writer) Bytes() []byte { return w.b }

// Op returns the opcode of a payload.
func Op(payload []byte) Opcode {
	if len(payload) < 2 {
		return 0
	}
	return Opcode(binary.LittleEndian.Uint16(payload))
}
