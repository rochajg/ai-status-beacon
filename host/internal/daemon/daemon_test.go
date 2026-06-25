package daemon

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fakePort is a thread-safe mock serial port.
type fakePort struct {
	mu      sync.Mutex
	written []byte
}

func (f *fakePort) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.written = append(f.written, p...)
	return len(p), nil
}

func (f *fakePort) Close() error { return nil }

func (f *fakePort) Received() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return string(f.written)
}

// startDaemon launches the daemon in a goroutine and waits until the
// socket is accepting connections before returning.
func startDaemon(t *testing.T, fp *fakePort) (sock string, cancel context.CancelFunc) {
	t.Helper()
	sock = filepath.Join(t.TempDir(), "beacon.sock")
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		runWithOpener(ctx, sock, func() (portWriter, error) {
			return fp, nil
		})
	}()

	// Poll until the socket is accepting.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("unix", sock, 50*time.Millisecond); err == nil {
			c.Close()
			return sock, cancel
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("daemon did not start in time")
	return "", nil
}

func send(t *testing.T, sock, msg string) {
	t.Helper()
	c, err := net.DialTimeout("unix", sock, time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	c.Write([]byte(msg))
}

func sendAndRead(t *testing.T, sock, msg string) string {
	t.Helper()
	c, err := net.DialTimeout("unix", sock, time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	c.Write([]byte(msg))
	buf := make([]byte, 16)
	c.SetDeadline(time.Now().Add(300 * time.Millisecond))
	n, _ := c.Read(buf)
	return string(buf[:n])
}

func TestDaemonForwardsValidState(t *testing.T) {
	fp := &fakePort{}
	sock, cancel := startDaemon(t, fp)
	defer cancel()

	send(t, sock, "thinking\n")
	time.Sleep(50 * time.Millisecond)

	if got := fp.Received(); got != "thinking\n" {
		t.Fatalf("want %q, got %q", "thinking\n", got)
	}
}

func TestDaemonIgnoresInvalidState(t *testing.T) {
	fp := &fakePort{}
	sock, cancel := startDaemon(t, fp)
	defer cancel()

	send(t, sock, "launch_missiles\n")
	time.Sleep(50 * time.Millisecond)

	if got := fp.Received(); got != "" {
		t.Fatalf("expected nothing written to serial, got %q", got)
	}
}

func TestDaemonRepliesPONGOnPing(t *testing.T) {
	fp := &fakePort{}
	sock, cancel := startDaemon(t, fp)
	defer cancel()

	resp := sendAndRead(t, sock, "ping\n")
	if resp != "PONG\n" {
		t.Fatalf("want %q, got %q", "PONG\n", resp)
	}
	// ping must NOT reach the serial port
	time.Sleep(30 * time.Millisecond)
	if got := fp.Received(); got != "" {
		t.Fatalf("ping must not go to serial, got %q", got)
	}
}

func TestDaemonRemovesSocketOnShutdown(t *testing.T) {
	fp := &fakePort{}
	sock, cancel := startDaemon(t, fp)

	cancel()
	time.Sleep(150 * time.Millisecond)

	if _, err := net.DialTimeout("unix", sock, 100*time.Millisecond); err == nil {
		t.Fatal("socket still reachable after shutdown")
	}
}

func TestDaemonAcceptsSequentialStates(t *testing.T) {
	fp := &fakePort{}
	sock, cancel := startDaemon(t, fp)
	defer cancel()

	for _, s := range []string{"thinking", "waiting", "done", "idle"} {
		send(t, sock, s+"\n")
		time.Sleep(30 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	want := "thinking\nwaiting\ndone\nidle\n"
	if got := fp.Received(); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
