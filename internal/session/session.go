// Package session drives a connection through the login server and hands
// it over to the world server: 0xA101 key exchange, 0xA102 login, 0xA201
// server list, 0xA202 world address, then the world's 0xA301 handshake.
package session

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Spelunkern/shaiya-go-client/internal/netx"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
)

const DialTimeout = 10 * time.Second

// LoginError carries the server's 0xA102 result code.
type LoginError struct{ Code uint8 }

func (e LoginError) Error() string { return fmt.Sprintf("login rejected (code %d)", e.Code) }

// Login is an authenticated login-server connection.
type Login struct {
	conn     *netx.Conn
	material netx.Material
	Result   proto.LoginResult
	Servers  []proto.Server
}

func readOp(c *netx.Conn, want proto.Opcode, timeout time.Duration) ([]byte, error) {
	c.SetReadDeadline(time.Now().Add(timeout))
	defer c.SetReadDeadline(time.Time{})
	for {
		p, err := c.Read()
		if err != nil {
			return nil, err
		}
		if proto.Op(p) == want {
			return p, nil
		}
	}
}

// Dial connects to the login server and authenticates.
func Dial(addr, user, password string) (*Login, error) {
	c, err := netx.Dial(addr, DialTimeout)
	if err != nil {
		return nil, err
	}
	l := &Login{conn: c}
	if err := l.authenticate(user, password); err != nil {
		c.Close()
		return nil, err
	}
	return l, nil
}

func (l *Login) authenticate(user, password string) error {
	p, err := readOp(l.conn, proto.LoginHandshake, DialTimeout)
	if err != nil {
		return fmt.Errorf("waiting for 0xA101: %w", err)
	}
	hello, err := proto.DecodeLoginHello(p)
	if err != nil {
		return err
	}
	block, m, err := netx.KeyExchange(hello.ExponentLE, hello.ModulusLE)
	if err != nil {
		return err
	}
	reply, err := proto.KeyReply(block)
	if err != nil {
		return err
	}
	if err := l.conn.Write(reply); err != nil {
		return err
	}
	l.material = m
	l.conn.EnableAES(m.Key(), m.Nonce())
	if err := l.conn.Write(proto.LoginRequestPacket(user, password)); err != nil {
		return err
	}
	p, err = readOp(l.conn, proto.LoginRequest, DialTimeout)
	if err != nil {
		return fmt.Errorf("waiting for 0xA102: %w", err)
	}
	if l.Result, err = proto.DecodeLoginResult(p); err != nil {
		return err
	}
	if l.Result.Result != 0 {
		return LoginError{l.Result.Result}
	}
	p, err = readOp(l.conn, proto.ServerList, DialTimeout)
	if err != nil {
		return fmt.Errorf("waiting for 0xA201: %w", err)
	}
	l.Servers, err = proto.DecodeServerList(p)
	return err
}

// SelectServer asks for a world's address and closes the login connection.
func (l *Login) SelectServer(id uint8, version uint32) (net.IP, error) {
	defer l.conn.Close()
	if err := l.conn.Write(proto.SelectServerPacket(id, version)); err != nil {
		return nil, err
	}
	p, err := readOp(l.conn, proto.SelectServer, DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("waiting for 0xA202: %w", err)
	}
	r, err := proto.DecodeSelectServer(p)
	if err != nil {
		return nil, err
	}
	if r.Status != 0 {
		return nil, fmt.Errorf("server %d unavailable (status %d)", id, r.Status)
	}
	return net.IPv4(r.IPv4[0], r.IPv4[1], r.IPv4[2], r.IPv4[3]), nil
}

// World is the game-server connection. Packets delivers every decoded
// payload in order; it is closed when the connection ends, after which Err
// reports why.
type World struct {
	conn    *netx.Conn
	Packets <-chan []byte
	err     error
	done    chan struct{}
}

var ErrHandshake = errors.New("world handshake rejected")

// DialWorld connects to ip:port and performs the 0xA301 handshake.
func (l *Login) DialWorld(ip net.IP, port int) (*World, error) {
	c, err := netx.Dial(net.JoinHostPort(ip.String(), strconv.Itoa(port)), DialTimeout)
	if err != nil {
		return nil, err
	}
	if err := c.Write(proto.WorldHelloPacket(l.Result.UserUID, l.Result.Token)); err != nil {
		c.Close()
		return nil, err
	}
	// 0x4013a0: both streams restart from SHA-256(login nonce).
	c.EnableAES(l.material.Key(), l.material.WorldCounter())
	p, err := readOp(c, proto.GameHandshake, DialTimeout)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("waiting for 0xA301: %w", err)
	}
	hs, err := proto.DecodeWorldHandshake(p)
	if err != nil || hs.Status != 0 {
		c.Close()
		return nil, ErrHandshake
	}
	c.SetTableMode(hs.Mode, hs.Seed)
	ch := make(chan []byte, 256)
	w := &World{conn: c, Packets: ch, done: make(chan struct{})}
	go func() {
		defer close(ch)
		defer close(w.done)
		for {
			p, err := c.Read()
			if err != nil {
				w.err = err
				return
			}
			ch <- p
		}
	}()
	return w, nil
}

// Send writes one payload; a nil payload (refused by its builder) is ignored.
func (w *World) Send(p []byte) error {
	if p == nil {
		return nil
	}
	return w.conn.Write(p)
}

func (w *World) Close() error { return w.conn.Close() }

// Err is valid after Packets is closed.
func (w *World) Err() error { <-w.done; return w.err }
