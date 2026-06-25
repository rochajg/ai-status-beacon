package socket

import (
	"net"
	"path/filepath"
	"testing"
	"time"
)

const timeout = 300 * time.Millisecond

func serve(t *testing.T, sock string, handler func(net.Conn)) {
	t.Helper()
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handler(c)
		}
	}()
}

func TestSendToSocketWritesStateLine(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "beacon.sock")
	received := make(chan string, 1)

	serve(t, sock, func(c net.Conn) {
		defer c.Close()
		buf := make([]byte, 64)
		n, _ := c.Read(buf)
		received <- string(buf[:n])
	})

	if err := SendToSocket("thinking", sock, timeout); err != nil {
		t.Fatal(err)
	}
	if got := <-received; got != "thinking\n" {
		t.Fatalf("want %q, got %q", "thinking\n", got)
	}
}

func TestSendToSocketErrorWhenNoServer(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "missing.sock")
	if err := SendToSocket("thinking", sock, 50*time.Millisecond); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPingSocketReturnsPONG(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "beacon.sock")

	serve(t, sock, func(c net.Conn) {
		defer c.Close()
		buf := make([]byte, 16)
		c.Read(buf)
		c.Write([]byte("PONG\n"))
	})

	if err := PingSocket(sock, timeout); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestPingSocketErrorWhenNoServer(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "missing.sock")
	if err := PingSocket(sock, 50*time.Millisecond); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPingSocketErrorWhenNoResponse(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "beacon.sock")

	serve(t, sock, func(c net.Conn) {
		buf := make([]byte, 16)
		c.Read(buf)
		c.Close() // accept but send nothing
	})

	if err := PingSocket(sock, timeout); err == nil {
		t.Fatal("expected error when no PONG received")
	}
}
