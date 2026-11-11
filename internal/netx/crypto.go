// Package netx implements the client side of the Shaiya EP5.4 transport:
// length-prefixed frames, the 0xA101 RSA key exchange, the AES-128-CTR
// stream with a little-endian counter and the world's inbound table XOR.
//
// Sources: shaiya-client-decomp src/client/crypto/{bootstrap,bootstrap_crypto,
// transport}.cpp (Game.exe 0x401000..0x4015a0) and src/client/network/
// {socket_transport,transport_seed}.cpp; cross-checked with Rapture
// internal/crypto.
package netx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"
	"slices"
)

// Material is the 32 bytes derived at login: AES key then counter/nonce.
type Material [32]byte

func (m Material) Key() (k [16]byte)   { copy(k[:], m[:16]); return }
func (m Material) Nonce() (n [16]byte) { copy(n[:], m[16:]); return }

// WorldCounter is the counter both world streams restart from
// (0x4013a0: SHA-256 of the login nonce, first 16 bytes).
func (m Material) WorldCounter() (c [16]byte) {
	n := m.Nonce()
	d := sha256.Sum256(n[:])
	copy(c[:], d[:16])
	return
}

func reversed(b []byte) []byte { r := slices.Clone(b); slices.Reverse(r); return r }

// KeyExchange builds the 0xA101 reply for the server's public key
// (little-endian exponent and modulus). It returns the RSA block in
// little-endian whole-modulus width and the derived AES material.
//
// 0x401160: 128 random bytes, the top bit of the last (most significant)
// byte cleared, HMAC-SHA256(key=random, msg=modulus LE).
func KeyExchange(exponentLE, modulusLE []byte) (block []byte, m Material, err error) {
	if len(modulusLE) == 0 || len(modulusLE) > 256 || len(exponentLE) == 0 || len(exponentLE) > 256 {
		return nil, m, errors.New("netx: bad RSA key size")
	}
	var secret [128]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return nil, m, err
	}
	secret[127] &= 0x7f
	// Rapture recovers the secret with big.Int and drops trailing zero
	// bytes, which would change the HMAC key; keep the top byte nonzero
	// (still a valid value for the original client's rule).
	if secret[127] == 0 {
		secret[127] = 1
	}
	e := new(big.Int).SetBytes(reversed(exponentLE))
	n := new(big.Int).SetBytes(reversed(modulusLE))
	c := new(big.Int).Exp(new(big.Int).SetBytes(reversed(secret[:])), e, n)
	block = make([]byte, len(modulusLE))
	cb := c.Bytes() // big-endian, at most len(modulus) bytes
	for i, b := range cb {
		block[len(cb)-1-i] = b
	}
	mac := hmac.New(sha256.New, secret[:])
	mac.Write(modulusLE)
	copy(m[:], mac.Sum(nil))
	return block, m, nil
}

// CTR is AES-128 in counter mode with a little-endian counter
// (aes_transform_ctr). Encryption and decryption are the same operation.
type CTR struct {
	block   cipher.Block
	counter [16]byte
	mask    [16]byte
	pos     int
}

func NewCTR(key, counter [16]byte) *CTR {
	b, err := aes.NewCipher(key[:])
	if err != nil {
		panic(err) // unreachable: the key is always 16 bytes
	}
	return &CTR{block: b, counter: counter, pos: 16}
}

// XOR transforms p in place.
func (c *CTR) XOR(p []byte) {
	for i := range p {
		if c.pos == 16 {
			c.block.Encrypt(c.mask[:], c.counter[:])
			for j := range c.counter {
				c.counter[j]++
				if c.counter[j] != 0 {
					break
				}
			}
			c.pos = 0
		}
		p[i] ^= c.mask[c.pos]
		c.pos++
	}
}

// XORTable is the world's inbound "mode 2" table (0x401520): SHA-256 of the
// 16-byte seed, then each next 32 bytes are SHA-256 of the previous 16.
// A payload of n bytes is XORed with table[n : 2n]; the original table is
// 4096 bytes, longer payloads extend the same chain (as Rapture does).
type XORTable struct{ buf []byte }

func NewXORTable(seed [16]byte) *XORTable {
	t := &XORTable{}
	d := sha256.Sum256(seed[:])
	t.buf = append(t.buf, d[:]...)
	t.grow(4096)
	return t
}

func (t *XORTable) grow(n int) {
	for len(t.buf) < n {
		d := sha256.Sum256(t.buf[len(t.buf)-16:])
		t.buf = append(t.buf, d[:]...)
	}
}

// XOR transforms p in place.
func (t *XORTable) XOR(p []byte) {
	t.grow(2 * len(p))
	for i := range p {
		p[i] ^= t.buf[len(p)+i]
	}
}
