package serial

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	goserial "go.bug.st/serial"
)

// portWriter abstracts the serial port for testing.
type portWriter interface {
	Write([]byte) (int, error)
	Close() error
}

// FindPort returns the first /dev/cu.usbmodem* device, or "".
// Respects BEACON_SERIAL_PORT env override.
func FindPort() string {
	if p := os.Getenv("BEACON_SERIAL_PORT"); p != "" {
		return p
	}
	matches, _ := filepath.Glob("/dev/cu.usbmodem*")
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// SendDirect opens the serial port, writes "state\n", and closes.
// The timeout is honoured via a context deadline that cancels the write if it
// does not complete in time (go.bug.st/serial does not expose SetWriteDeadline,
// so we enforce the budget with a goroutine + context).
func SendDirect(state, port string, timeout time.Duration) error {
	p, err := goserial.Open(port, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- writeState(state, p)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Abort the blocked write by closing the port, then drain the result.
		p.Close()
		<-errCh
		return ctx.Err()
	}
}

// writeState is the testable core: writes "state\n" to any portWriter and
// closes it when done.
func writeState(state string, p portWriter) error {
	defer p.Close()
	_, err := fmt.Fprintf(p, "%s\n", state)
	return err
}

// SendDirectRaw opens the serial port and writes cmd verbatim (no newline added).
// cmd must already include the trailing newline. Used by the CLI when the command
// already includes params assembled by buildCommand.
func SendDirectRaw(cmd, port string, timeout time.Duration) error {
	p, err := goserial.Open(port, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- sendDirectRaw(cmd, p, timeout)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		p.Close()
		<-errCh
		return ctx.Err()
	}
}

// sendDirectRaw is the testable core: writes cmd verbatim to any portWriter and
// closes it when done.
func sendDirectRaw(cmd string, p portWriter, _ time.Duration) error {
	defer p.Close()
	_, err := fmt.Fprint(p, cmd)
	return err
}
