package proto

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

// GXGMState is a GX-only server response: self ID:u32, GM:u8, turbo:u8,
// message byte length:u16, UTF-8 message. Requests use ordinary chat commands.
const GXGMState Opcode = 0x7F01

// GXVisibility: character ID:u32, concealed:u8. Sent only to authorized viewers.
const GXVisibility Opcode = 0x7F02

type GMState struct {
	ID        uint32
	GM, Turbo bool
	Message   string
}

func DecodeGMState(p []byte) (GMState, error) {
	if len(p) < 10 {
		return GMState{}, fmt.Errorf("short GX GM state")
	}
	n := int(binary.LittleEndian.Uint16(p[8:10]))
	if len(p) != 10+n || p[6] > 1 || p[7] > 1 || (p[6] == 0 && p[7] != 0) || !utf8.Valid(p[10:]) {
		return GMState{}, fmt.Errorf("invalid GX GM state")
	}
	return GMState{ID: binary.LittleEndian.Uint32(p[2:6]), GM: p[6] == 1, Turbo: p[7] == 1, Message: string(p[10:])}, nil
}
