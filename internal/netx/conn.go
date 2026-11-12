package netx

import (
	"bufio"
	"encoding/binary"
	"net"
	"sync"
	"time"
)

// Opcodes that drive the inbound transport mode (dispatcher 0x59cf70).
const (
	opLogoutAck = 0x0107 // fun_004015a0: back to the AES stream
	opSkillBar  = 0x010B // fun_00591dd0 -> fun_00401570: switch to the table
)

// Conn is one TCP session with the server. Write may be called from any
// goroutine; Read must be called from a single reader goroutine.
type Conn struct {
	nc net.Conn
	r  *bufio.Reader

	wmu sync.Mutex
	out *CTR // nil: plaintext

	in        *CTR
	inEnabled bool
	table     *XORTable
	mode      uint8
	useTable  bool
}

func Dial(addr string, timeout time.Duration) (*Conn, error) {
	nc, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	if tcp, ok := nc.(*net.TCPConn); ok {
		tcp.SetNoDelay(true)
	}
	return &Conn{nc: nc, r: bufio.NewReaderSize(nc, 64<<10)}, nil
}

func (c *Conn) Close() error { return c.nc.Close() }

// CloseWrite delivers EOF after queued requests, leaving the read side open.
func (c *Conn) CloseWrite() error {
	if tcp, ok := c.nc.(*net.TCPConn); ok {
		return tcp.CloseWrite()
	}
	return c.nc.Close()
}
func (c *Conn) RemoteAddr() net.Addr { return c.nc.RemoteAddr() }

// SetReadDeadline forwards to the socket.
func (c *Conn) SetReadDeadline(t time.Time) error { return c.nc.SetReadDeadline(t) }

// EnableAES installs both streams (0x401000 at login, 0x4013a0 at world).
func (c *Conn) EnableAES(key, counter [16]byte) {
	c.wmu.Lock()
	c.out = NewCTR(key, counter)
	c.wmu.Unlock()
	c.in = NewCTR(key, counter)
	c.inEnabled = true
}

// SetTableMode records the 0xA301 mode and seed (0x401520).
func (c *Conn) SetTableMode(mode uint8, seed [16]byte) {
	c.table = NewXORTable(seed)
	c.mode = mode
	if mode == 0 {
		c.inEnabled = false
	}
}

// Write frames and (if enabled) encrypts one payload.
func (c *Conn) Write(payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	body := append([]byte(nil), payload...)
	if c.out != nil {
		c.out.XOR(body)
	}
	frame, err := AppendFrame(nil, body)
	if err != nil {
		return err
	}
	_, err = c.nc.Write(frame)
	return err
}

// Read returns the next decoded payload (opcode + body).
func (c *Conn) Read() ([]byte, error) {
	p, err := ReadFrame(c.r)
	if err != nil {
		return nil, err
	}
	switch {
	case c.useTable:
		c.table.XOR(p)
	case c.inEnabled && c.in != nil:
		c.in.XOR(p)
	}
	if len(p) >= 2 {
		switch binary.LittleEndian.Uint16(p) {
		case opSkillBar: // fun_00401570
			if c.mode <= 1 {
				c.inEnabled = false
			} else if c.mode != 3 && c.table != nil {
				c.useTable = true
			}
		case opLogoutAck: // fun_004015a0
			c.useTable = false
			c.inEnabled = c.mode != 0
		}
	}
	return p, nil
}
