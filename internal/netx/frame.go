package netx

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// MaxFrame bounds a frame; the length prefix is u16 and counts itself.
const MaxFrame = 0xFFFF

// ReadFrame reads one frame and returns its payload (opcode + body).
func ReadFrame(r *bufio.Reader) ([]byte, error) {
	var head [2]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return nil, err
	}
	n := int(binary.LittleEndian.Uint16(head[:]))
	if n < 4 {
		return nil, fmt.Errorf("netx: frame length %d", n)
	}
	p := make([]byte, n-2)
	if _, err := io.ReadFull(r, p); err != nil {
		return nil, err
	}
	return p, nil
}

// AppendFrame prefixes payload with its frame length.
func AppendFrame(dst, payload []byte) ([]byte, error) {
	if len(payload)+2 > MaxFrame || len(payload) < 2 {
		return dst, fmt.Errorf("netx: payload length %d", len(payload))
	}
	dst = binary.LittleEndian.AppendUint16(dst, uint16(len(payload)+2))
	return append(dst, payload...), nil
}
