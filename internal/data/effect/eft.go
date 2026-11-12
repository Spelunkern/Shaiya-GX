// Package effect reads the original effect libraries without interpreting unknown fields.
package effect

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// Library follows Parsec Shaiya/Eft: mesh/texture names, emitters and sequences.
type Library struct {
	Format           string
	Meshes, Textures []string
	Effects          []Emitter
	Sequences        []Sequence
}

type Emitter struct {
	Name                                                                     string
	Unk01, Unk02, Unk03, Loop, SrcBlend, Unk06, DstBlend, Unk08, Mesh, Unk10 int32
	FrameDelay                                                               float32
	Unk12, Unk13, Unk14, InitialDelay, Unk16, Unk17, Unk18                   float32
	OffsetFrame, Trembling, Position, Spread1, Spread2                       [3]float32
	BaseAxis, Unk20, Unk21                                                   int32
	UnkVec6                                                                  [3]float32
	RotationMin                                                              float32
	RotationRandom, RotationEnabled                                          int32
	RotationMax                                                              float32
	RotationAxis                                                             int32
	EF3Unk                                                                   [2]float32
	Colors                                                                   []ColorFrame
	Opacity                                                                  []OpacityFrame
	Sub3                                                                     []Sub3Frame
	Unk29, Unk30, Unk31, Unk32                                               int32
	Textures                                                                 []int32
}
type ColorFrame struct {
	RGBA [4]float32
	Time float32
}
type OpacityFrame struct{ Opacity, Time float32 }
type Sub3Frame struct{ Unk1, Unk2, Time float32 }
type Sequence struct {
	Name    string
	Records []Record
}
type Record struct {
	Effect int32
	Time   float32
}

type reader struct {
	b          []byte
	off, total int
	err        error
}

func (r *reader) word() uint32 {
	if r.err != nil {
		return 0
	}
	if len(r.b)-r.off < 4 {
		r.err = fmt.Errorf("EFT: truncated at %d", r.off)
		return 0
	}
	v := binary.LittleEndian.Uint32(r.b[r.off:])
	r.off += 4
	return v
}
func (r *reader) i32() int32 { return int32(r.word()) }
func (r *reader) f32() float32 {
	f := math.Float32frombits(r.word())
	if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
		r.err = fmt.Errorf("EFT: non-finite float at %d", r.off-4)
	}
	return f
}
func (r *reader) count(size int) int {
	n := int(r.i32())
	if r.err != nil {
		return 0
	}
	if n < 0 || n > 16384 || n > (len(r.b)-r.off)/size || r.total+n > 262144 {
		r.err = fmt.Errorf("EFT: invalid count %d at %d", n, r.off-4)
		return 0
	}
	r.total += n
	return n
}
func (r *reader) str() string {
	n := int(r.i32())
	if r.err != nil {
		return ""
	}
	if n < 0 || n > 1<<20 || n > len(r.b)-r.off {
		r.err = fmt.Errorf("EFT: invalid string length %d at %d", n, r.off-4)
		return ""
	}
	s := string(r.b[r.off : r.off+n])
	r.off += n
	return strings.TrimRight(s, "\x00")
}
func (r *reader) vec() [3]float32 { return [3]float32{r.f32(), r.f32(), r.f32()} }
func (r *reader) names() []string {
	a := make([]string, r.count(4))
	for i := range a {
		a[i] = r.str()
	}
	return a
}

func ParseEFT(b []byte) (*Library, error) {
	if len(b) < 19 || len(b) > 16<<20 {
		return nil, fmt.Errorf("EFT: invalid length %d", len(b))
	}
	format := string(b[:3])
	if format != "EFT" && format != "EF2" && format != "EF3" {
		return nil, fmt.Errorf("EFT: unknown signature %q", format)
	}
	r := &reader{b: b, off: 3}
	l := &Library{Format: format, Meshes: r.names(), Textures: r.names()}
	minimum := 212
	if format == "EF3" {
		minimum += 8
	}
	l.Effects = make([]Emitter, r.count(minimum))
	for i := range l.Effects {
		e := &l.Effects[i]
		e.Name = r.str()
		for _, p := range []*int32{&e.Unk01, &e.Unk02, &e.Unk03, &e.Loop, &e.SrcBlend, &e.Unk06, &e.DstBlend, &e.Unk08, &e.Mesh, &e.Unk10} {
			*p = r.i32()
		}
		for _, p := range []*float32{&e.FrameDelay, &e.Unk12, &e.Unk13, &e.Unk14, &e.InitialDelay, &e.Unk16, &e.Unk17, &e.Unk18} {
			*p = r.f32()
		}
		e.OffsetFrame, e.Trembling, e.Position, e.Spread1, e.Spread2 = r.vec(), r.vec(), r.vec(), r.vec(), r.vec()
		e.BaseAxis, e.Unk20, e.Unk21 = r.i32(), r.i32(), r.i32()
		e.UnkVec6 = r.vec()
		e.RotationMin, e.RotationRandom, e.RotationEnabled, e.RotationMax, e.RotationAxis = r.f32(), r.i32(), r.i32(), r.f32(), r.i32()
		if format == "EF3" {
			e.EF3Unk = [2]float32{r.f32(), r.f32()}
		}
		e.Colors = make([]ColorFrame, r.count(20))
		for j := range e.Colors {
			e.Colors[j] = ColorFrame{[4]float32{r.f32(), r.f32(), r.f32(), r.f32()}, r.f32()}
		}
		e.Opacity = make([]OpacityFrame, r.count(8))
		for j := range e.Opacity {
			e.Opacity[j] = OpacityFrame{r.f32(), r.f32()}
		}
		e.Sub3 = make([]Sub3Frame, r.count(12))
		for j := range e.Sub3 {
			e.Sub3[j] = Sub3Frame{r.f32(), r.f32(), r.f32()}
		}
		e.Unk29, e.Unk30, e.Unk31, e.Unk32 = r.i32(), r.i32(), r.i32(), r.i32()
		e.Textures = make([]int32, r.count(4))
		for j := range e.Textures {
			e.Textures[j] = r.i32()
		}
	}
	l.Sequences = make([]Sequence, r.count(8))
	for i := range l.Sequences {
		s := &l.Sequences[i]
		s.Name = r.str()
		s.Records = make([]Record, r.count(8))
		for j := range s.Records {
			s.Records[j] = Record{r.i32(), r.f32()}
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.off != len(b) {
		return nil, fmt.Errorf("EFT: %d trailing bytes", len(b)-r.off)
	}
	return l, nil
}
