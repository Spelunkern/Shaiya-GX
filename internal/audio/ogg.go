package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/jfreymuth/oggvorbis"
)

// DecodeOGG decodes Ogg Vorbis into Oto's stereo signed 16-bit 44100 Hz stream.
// Resampling is bounded and incremental: only two input frames and one small
// decoder buffer are retained in addition to the final bounded PCM output.
func DecodeOGG(raw []byte) ([]byte, error) {
	r, err := oggvorbis.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("ogg: %w", err)
	}
	channels, rate := r.Channels(), r.SampleRate()
	length := r.Length()
	if (channels != 1 && channels != 2) || rate < 8000 || rate > 192000 || length < 1 || length > int64(maxPCMBytes) {
		return nil, fmt.Errorf("ogg: invalid stream format or length")
	}
	frames := (length*SampleRate + int64(rate) - 1) / int64(rate)
	if frames < 1 || frames*4 > maxPCMBytes {
		return nil, fmt.Errorf("ogg: decoded size out of range")
	}
	buffer := make([]float32, 4096)
	at, n := 0, 0
	var pending error
	readFrame := func() ([2]float64, error) {
		var frame [2]float64
		if at == n {
			if pending != nil {
				return frame, pending
			}
			n, pending = r.Read(buffer)
			at = 0
			if n == 0 {
				if pending == nil {
					pending = io.ErrNoProgress
				}
				return frame, pending
			}
			if n%channels != 0 {
				return frame, fmt.Errorf("ogg: incomplete sample frame")
			}
		}
		frame[0] = float64(buffer[at])
		frame[1] = frame[0]
		if channels == 2 {
			frame[1] = float64(buffer[at+1])
		}
		at += channels
		return frame, nil
	}
	a, err := readFrame()
	if err != nil {
		return nil, err
	}
	b, err := readFrame()
	if err == io.EOF {
		b = a
	} else if err != nil {
		return nil, err
	}
	out := make([]byte, int(frames)*4)
	source := int64(0)
	for i := int64(0); i < frames; i++ {
		pos := i * int64(rate)
		want := pos / SampleRate
		for source < want {
			a = b
			source++
			if source+1 < length {
				b, err = readFrame()
				if err != nil {
					return nil, fmt.Errorf("ogg: truncated samples: %w", err)
				}
			} else {
				b = a
			}
		}
		fraction := float64(pos%SampleRate) / SampleRate
		for c := 0; c < 2; c++ {
			v := (a[c]*(1-fraction) + b[c]*fraction) * 32768
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = 0
			}
			v = math.Max(-32768, math.Min(32767, v))
			binary.LittleEndian.PutUint16(out[int(i)*4+c*2:], uint16(int16(v)))
		}
	}
	return out, nil
}
