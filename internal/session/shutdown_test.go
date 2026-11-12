package session

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/netx"
)

func tcpWorld(t *testing.T, serve func(net.Conn)) *World {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		serve(conn)
	}()
	c, err := netx.Dial(l.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	w := startWorld(c)
	t.Cleanup(func() { w.Close() })
	return w
}

func TestShutdownDeliversLastRequestAndWaitsForServerClose(t *testing.T) {
	ready, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	request, _ := netx.AppendFrame(nil, []byte{7, 1})
	w := tcpWorld(t, func(conn net.Conn) {
		got, err := io.ReadAll(conn)
		if err != nil || !bytes.Equal(got, request) {
			t.Errorf("last request lost before EOF: %x, %v", got, err)
			return
		}
		frame, _ := netx.AppendFrame(nil, []byte{1, 2, 3})
		for i := 0; i < 300; i++ {
			if _, err := conn.Write(frame); err != nil {
				t.Error(err)
				return
			}
		}
		close(ready)
		<-release
	})
	if err := w.Send([]byte{7, 1}); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- w.Shutdown(2 * time.Second) }()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("server did not receive half-close or packets stopped draining")
	}
	select {
	case err := <-result:
		t.Fatal("shutdown finished before server cleanup", err)
	default:
	}
	// A value also releases the peer without double-closing the cleanup channel.
	release <- struct{}{}
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish after the server closed")
	}
}

func TestShutdownTimeoutReleasesReader(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	w := tcpWorld(t, func(conn net.Conn) { io.Copy(io.Discard, conn); <-release })
	if err := w.Shutdown(30 * time.Millisecond); err == nil {
		t.Fatal("unresponsive server shutdown succeeded")
	}
	select {
	case <-w.done:
	case <-time.After(time.Second):
		t.Fatal("timeout leaked the reader")
	}
}

func TestCloseReleasesReaderWithFullPacketQueue(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	w := tcpWorld(t, func(conn net.Conn) {
		frame, _ := netx.AppendFrame(nil, []byte{1, 2})
		for i := 0; i < 300; i++ {
			if _, err := conn.Write(frame); err != nil {
				return
			}
		}
		<-release
	})
	deadline := time.Now().Add(time.Second)
	for len(w.Packets) < cap(w.Packets) {
		if time.Now().After(deadline) {
			t.Fatal("fixture did not fill the queue")
		}
		time.Sleep(time.Millisecond)
	}
	w.Close()
	select {
	case <-w.done:
	case <-time.After(time.Second):
		t.Fatal("full packet queue leaked the reader after close")
	}
}
