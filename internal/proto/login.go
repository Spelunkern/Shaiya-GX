package proto

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/itemstack"
)

// LoginHello is the server's 0xA101 (plaintext): public RSA key.
// Client handler: dispatcher case 0xa101; Rapture cmd/rapture_login handshake().
type LoginHello struct {
	Unk        uint8
	ExponentLE []byte
	ModulusLE  []byte
}

func DecodeLoginHello(p []byte) (m LoginHello, err error) {
	r := NewReader(p)
	m.Unk = r.U8()
	el, ml := int(r.U8()), int(r.U8())
	m.ExponentLE = r.Bytes(el)
	m.ModulusLE = r.Bytes(ml)
	return m, r.Err()
}

// KeyReply is the client's 0xA101 (plaintext): size:u8 + RSA block.
// Writer 0x5992a0: always 0x83 bytes, a 128-byte block.
func KeyReply(block []byte) ([]byte, error) {
	if len(block) != 128 {
		return nil, fmt.Errorf("proto: RSA block is %d bytes, the client sends 128", len(block))
	}
	w := NewWriter(LoginHandshake)
	w.U8(128)
	w.Raw(block)
	return w.Bytes(), nil
}

// LoginRequestPacket is 0xA102 (writer 0x5992f0, 50 bytes): user[32] and
// password[16], both copied raw from NUL-padded buffers.
func LoginRequestPacket(user, password string) []byte {
	w := NewWriter(LoginRequest)
	w.Fixed(user, 32, true)
	w.Fixed(password, 16, false)
	return w.Bytes()
}

// LoginResult is the server's 0xA102.
type LoginResult struct {
	Result     uint8 // 0 = ok
	UserUID    uint32
	Permission uint8
	Token      [16]byte
}

func DecodeLoginResult(p []byte) (m LoginResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	if m.Result == 0 {
		m.UserUID = r.U32()
		m.Permission = r.U8()
		copy(m.Token[:], r.Bytes(16))
	}
	return m, r.Err()
}

// Server is one row of 0xA201.
type Server struct {
	ID       uint8
	Status   uint8
	Users    uint16
	Capacity uint16
	Name     string
}

func DecodeServerList(p []byte) ([]Server, error) {
	r := NewReader(p)
	n := int(r.U8())
	out := make([]Server, 0, n)
	for i := 0; i < n && r.Err() == nil; i++ {
		out = append(out, Server{ID: r.U8(), Status: r.U8(), Users: r.U16(), Capacity: r.U16(), Name: r.Fixed(32)})
	}
	return out, r.Err()
}

// ClientVersion identifies the GX uint16 item-count wire revision.
const ClientVersion uint32 = itemstack.ClientVersion

// SelectServerPacket is 0xA202 (writer 0x599440, 7 bytes).
func SelectServerPacket(serverID uint8, version uint32) []byte {
	w := NewWriter(SelectServer)
	w.U8(serverID)
	w.U32(version)
	return w.Bytes()
}

// SelectServerResult is the server's 0xA202: status and the world IPv4.
type SelectServerResult struct {
	Status uint8
	IPv4   [4]byte
}

func DecodeSelectServer(p []byte) (m SelectServerResult, err error) {
	r := NewReader(p)
	m.Status = r.U8()
	copy(m.IPv4[:], r.Bytes(4))
	return m, r.Err()
}

// WorldHelloPacket is the client's plaintext 0xA301 (writer 0x599480,
// 22 bytes): user uid and the login token.
func WorldHelloPacket(userUID uint32, token [16]byte) []byte {
	w := NewWriter(GameHandshake)
	w.U32(userUID)
	w.Raw(token[:])
	return w.Bytes()
}

// WorldHandshake is the server's 0xA301 (handler 0x5712f0): status 0 = ok,
// then the inbound table mode and seed for 0x401520.
type WorldHandshake struct {
	Status uint8
	Mode   uint8
	Seed   [16]byte
}

func DecodeWorldHandshake(p []byte) (m WorldHandshake, err error) {
	r := NewReader(p)
	m.Status = r.U8()
	m.Mode = r.U8()
	copy(m.Seed[:], r.Bytes(16))
	return m, r.Err()
}
