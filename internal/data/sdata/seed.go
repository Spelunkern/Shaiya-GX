// Package sdata reads the client's .SData tables. EP5 files are encrypted
// with KISA SEED (port of Parsec Cryptography/SEED.cs and SData.Decrypt,
// MIT licensed; see NOTICE).
package sdata

import (
	"encoding/binary"
	"errors"
)

const seedSignature = "0001CBCEBC5B2784D3FC9A2A9DB84D1C3FEB6E99"

// Decrypt returns the plain bytes of an SData file (unchanged when it is
// not SEED-encrypted).
func Decrypt(b []byte) ([]byte, error) {
	if len(b) < len(seedSignature) || string(b[:len(seedSignature)]) != seedSignature {
		return b, nil
	}
	if len(b) < 64 || len(b)%16 != 0 {
		return nil, errors.New("sdata: bad SEED length")
	}
	off := 40
	checksum := binary.LittleEndian.Uint32(b[off:])
	off += 4
	if checksum == 0 {
		off += 4
	}
	real := int(binary.LittleEndian.Uint32(b[off:]))
	enc := b[64:]
	out := make([]byte, len(enc))
	for i := 0; i+16 <= len(enc); i += 16 {
		decryptChunk(enc[i:i+16], out[i:i+16])
	}
	if real > len(out) {
		return nil, errors.New("sdata: real size beyond data")
	}
	return out[:real], nil
}

func be32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }

func rotl(x uint32, n uint) uint32 { return x<<n | x>>(32-n) }

func swap(v uint32) uint32 { return rotl(v, 8)&0x00ff00ff | rotl(v, 24)&0xff00ff00 }

func g(v uint32) uint32 {
	return seedSS[0][byte(v)] ^ seedSS[1][byte(v>>8)] ^ seedSS[2][byte(v>>16)] ^ seedSS[3][byte(v>>24)]
}

func round(l0, l1 *uint32, r0, r1 uint32, k int) {
	k0 := swap(be32(seedKey[k*4:]))
	k1 := swap(be32(seedKey[(k+1)*4:]))
	t0, t1 := r0^k0, r1^k1
	t1 ^= t0
	t1 = g(t1)
	t0 += t1
	t0 = g(t0)
	t1 += t0
	t1 = g(t1)
	t0 += t1
	*l0 ^= t0
	*l1 ^= t1
}

func decryptChunk(in, out []byte) {
	l0, l1, r0, r1 := be32(in), be32(in[4:]), be32(in[8:]), be32(in[12:])
	for k := 30; k >= 0; k -= 4 {
		round(&l0, &l1, r0, r1, k)
		round(&r0, &r1, l0, l1, k-2)
	}
	binary.BigEndian.PutUint32(out, r0)
	binary.BigEndian.PutUint32(out[4:], r1)
	binary.BigEndian.PutUint32(out[8:], l0)
	binary.BigEndian.PutUint32(out[12:], l1)
}
