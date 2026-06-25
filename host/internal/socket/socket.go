package socket

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// SendToSocket connects to the Unix socket and writes "state\n".
// Always closes the connection. Returns error on any failure.
func SendToSocket(state, socketPath string, timeout time.Duration) error {
	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))
	_, err = fmt.Fprintf(conn, "%s\n", state)
	return err
}

// PingSocket sends "ping\n" and expects "PONG\n" back from the daemon.
// Returns error if unreachable or response is not PONG.
func PingSocket(socketPath string, timeout time.Duration) error {
	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	if _, err := fmt.Fprint(conn, "ping\n"); err != nil {
		return err
	}

	sc := bufio.NewScanner(conn)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return err
		}
		return fmt.Errorf("daemon closed connection without responding")
	}
	if got := strings.TrimSpace(sc.Text()); got != "PONG" {
		return fmt.Errorf("expected PONG, got %q", got)
	}
	return nil
}
