package daemon

import (
	"bufio"
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	goserial "go.bug.st/serial"

	"beacon/internal/serial"
)

const reconnectInterval = 2 * time.Second

var validStates = map[string]bool{
	"thinking": true, "waiting": true, "done": true,
	"idle": true, "error": true,
}

// portWriter is the minimal serial interface the daemon needs.
type portWriter interface {
	Write([]byte) (int, error)
	Close() error
}

// Run is the public entry point for `beacon daemon`.
// Blocks until ctx is cancelled (wired to SIGTERM/SIGINT in main.go).
func Run(ctx context.Context, socketPath string, baud int) {
	runWithOpener(ctx, socketPath, func() (portWriter, error) {
		port := serial.FindPort()
		if port == "" {
			return nil, os.ErrNotExist
		}
		return goserial.Open(port, &goserial.Mode{BaudRate: baud})
	})
}

// runWithOpener is the testable core. openPort is called for each
// (re)connection attempt.
func runWithOpener(ctx context.Context, socketPath string, openPort func() (portWriter, error)) {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		log.Printf("daemon: mkdir: %v", err)
		return
	}
	os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Printf("daemon: listen: %v", err)
		return
	}
	defer func() {
		ln.Close()
		os.Remove(socketPath)
	}()

	var (
		mu      sync.Mutex
		current portWriter
	)

	// First connect attempt (synchronous so tests can rely on it).
	if p, err := openPort(); err == nil {
		current = p
		log.Println("daemon: serial connected")
	}

	// Background reconnect loop.
	go func() {
		ticker := time.NewTicker(reconnectInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				needOpen := current == nil
				mu.Unlock()

				if needOpen {
					p, e := openPort()
					if e == nil {
						mu.Lock()
						if current == nil {
							current = p
							log.Println("daemon: serial reconnected")
						} else {
							p.Close()
						}
						mu.Unlock()
					}
				}
			}
		}
	}()

	log.Printf("daemon: listening on %s", socketPath)

	for {
		connCh := make(chan net.Conn, 1)
		go func() {
			c, e := ln.Accept()
			if e == nil {
				connCh <- c
			}
		}()

		select {
		case <-ctx.Done():
			mu.Lock()
			if current != nil {
				current.Close()
			}
			mu.Unlock()
			return
		case conn := <-connCh:
			go handle(conn, &mu, &current)
		}
	}
}

func handle(conn net.Conn, mu *sync.Mutex, current *portWriter) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(300 * time.Millisecond))

	sc := bufio.NewScanner(conn)
	if !sc.Scan() {
		return
	}
	line := strings.TrimSpace(sc.Text())
	if line == "" {
		return
	}
	cmd := strings.Fields(line)[0]

	if cmd == "ping" {
		conn.Write([]byte("PONG\n"))
		return
	}
	if !validStates[cmd] {
		log.Printf("daemon: ignored %q", cmd)
		return
	}

	mu.Lock()
	p := *current
	if p == nil {
		mu.Unlock()
		log.Println("daemon: serial unavailable, dropping")
		return
	}
	_, werr := p.Write([]byte(cmd + "\n"))
	if werr != nil {
		*current = nil
		log.Printf("daemon: write error: %v — will reconnect", werr)
	}
	mu.Unlock()
}
