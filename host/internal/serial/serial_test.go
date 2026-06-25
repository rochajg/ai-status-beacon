package serial

import (
	"io"
	"testing"
	"time"
)

func TestFindPortEnvOverride(t *testing.T) {
	t.Setenv("BEACON_SERIAL_PORT", "/dev/fake42")
	if got := FindPort(); got != "/dev/fake42" {
		t.Fatalf("want /dev/fake42, got %q", got)
	}
}

func TestFindPortEmptyWithNoDevice(t *testing.T) {
	t.Setenv("BEACON_SERIAL_PORT", "")
	_ = FindPort() // must not panic; returns "" in CI (no hardware)
}

func TestWriteStateAppendsNewline(t *testing.T) {
	pr, pw := io.Pipe()
	mock := &mockPort{WriteCloser: pw}

	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := pr.Read(buf)
		done <- string(buf[:n])
	}()

	if err := writeState("thinking", mock, 300*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got := <-done; got != "thinking\n" {
		t.Fatalf("want %q, got %q", "thinking\n", got)
	}
}

func TestWriteStateReturnsErrorOnWriteFailure(t *testing.T) {
	mock := &mockPort{WriteCloser: failWriter{}}
	if err := writeState("done", mock, 300*time.Millisecond); err == nil {
		t.Fatal("expected error, got nil")
	}
}

type mockPort struct{ io.WriteCloser }

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func (failWriter) Close() error              { return nil }
