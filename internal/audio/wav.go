// Package audio decodes original sound assets and owns bounded playback.
package audio

import (
	"encoding/binary"
	"fmt"
	"math"
)

const SampleRate = 44100
const maxPCMBytes = 64 << 20

// DecodeWAV normalizes RIFF PCM/IEEE float to stereo signed 16-bit at 44100 Hz.
func DecodeWAV(b []byte) ([]byte, error) {
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, fmt.Errorf("wav: invalid RIFF/WAVE")
	}
	end := int64(binary.LittleEndian.Uint32(b[4:])) + 8
	if end > int64(len(b)) || end < 12 {
		return nil, fmt.Errorf("wav: truncated RIFF")
	}
	var format, data []byte
	for off := int64(12); off < end; {
		if end-off < 8 {
			return nil, fmt.Errorf("wav: truncated chunk header")
		}
		n := int64(binary.LittleEndian.Uint32(b[off+4:]))
		if n > end-off-8 {
			return nil, fmt.Errorf("wav: truncated chunk")
		}
		switch string(b[off : off+4]) {
		case "fmt ":
			format = b[off+8 : off+8+n]
		case "data":
			data = b[off+8 : off+8+n]
		}
		off += 8 + n + (n & 1)
	}
	if len(format) < 16 || len(data) == 0 {
		return nil, fmt.Errorf("wav: missing format or data")
	}
	u16 := binary.LittleEndian.Uint16
	kind, channels := u16(format), int(u16(format[2:]))
	rate := int(binary.LittleEndian.Uint32(format[4:]))
	block, bits := int(u16(format[12:])), int(u16(format[14:]))
	if kind == 0xfffe && len(format) >= 40 {
		kind = u16(format[24:])
	}
	if (channels != 1 && channels != 2) || rate < 8000 || rate > 192000 || bits%8 != 0 || block != channels*(bits/8) || block == 0 {
		return nil, fmt.Errorf("wav: invalid stream format")
	}
	if kind != 1 && kind != 3 || kind == 3 && bits != 32 || kind == 1 && bits != 8 && bits != 16 && bits != 24 && bits != 32 {
		return nil, fmt.Errorf("wav: unsupported encoding %d/%d bits", kind, bits)
	}
	if len(data)%block != 0 {
		return nil, fmt.Errorf("wav: incomplete sample frame")
	}
	frames := len(data) / block
	outputFrames := (int64(frames)*SampleRate + int64(rate) - 1) / int64(rate)
	if outputFrames*4 > maxPCMBytes || outputFrames == 0 {
		return nil, fmt.Errorf("wav: decoded size out of range")
	}
	sample := func(frame, channel int) float64 {
		if channels == 1 {
			channel = 0
		}
		p := data[frame*block+channel*bits/8:]
		if kind == 3 {
			v := float64(math.Float32frombits(binary.LittleEndian.Uint32(p)))
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return 0
			}
			return v * 32768
		}
		switch bits {
		case 8:
			return float64(int(p[0])-128) * 256
		case 16:
			return float64(int16(u16(p)))
		case 24:
			v := int32(p[0]) | int32(p[1])<<8 | int32(p[2])<<16
			return float64(v<<8>>8) / 256
		default:
			return float64(int32(binary.LittleEndian.Uint32(p))) / 65536
		}
	}
	out := make([]byte, int(outputFrames)*4)
	for i := int64(0); i < outputFrames; i++ {
		pos := i * int64(rate)
		a := int(pos / SampleRate)
		next := min(a+1, frames-1)
		t := float64(pos%SampleRate) / SampleRate
		for channel := 0; channel < 2; channel++ {
			value := sample(a, channel)*(1-t) + sample(next, channel)*t
			value = math.Max(-32768, math.Min(32767, value))
			binary.LittleEndian.PutUint16(out[int(i)*4+channel*2:], uint16(int16(value)))
		}
	}
	return out, nil
}
