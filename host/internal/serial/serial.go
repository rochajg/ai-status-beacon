package serial

import (
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
// Always closes the port, even on error.
func SendDirect(state, port string, _ time.Duration) error {
	p, err := goserial.Open(port, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		return err
	}
	return writeState(state, p, 0)
}

// writeState is the testable core: writes "state\n" to any portWriter.
func writeState(state string, p portWriter, _ time.Duration) error {
	defer p.Close()
	_, err := fmt.Fprintf(p, "%s\n", state)
	return err
}
