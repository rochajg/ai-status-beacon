package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"beacon/internal/daemon"
	"beacon/internal/serial"
	"beacon/internal/socket"
)

const (
	baud    = 115200
	timeout = 300 * time.Millisecond
)

var (
	socketPath  string
	validStates = map[string]bool{
		"thinking": true, "waiting": true, "done": true,
		"idle": true, "error": true, "ping": true,
	}
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "beacon: cannot determine home directory:", err)
		os.Exit(1)
	}
	socketPath = filepath.Join(home, ".beacon", "beacon.sock")
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printUsage()
		os.Exit(0)
	}

	switch args[0] {
	case "daemon":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		daemon.Run(ctx, socketPath, baud)

	case "status":
		os.Exit(runStatus())

	default:
		state := args[0]
		if !validStates[state] {
			os.Exit(0) // unknown state — silent, never block hooks
		}
		direct := len(args) > 1 && args[1] == "--direct"
		if direct {
			runDirect(state)
		} else {
			runSocket(state)
		}
		os.Exit(0)
	}
}

func runSocket(state string) {
	_ = socket.SendToSocket(state, socketPath, timeout)
}

func runDirect(state string) {
	port := serial.FindPort()
	if port == "" {
		return
	}
	_ = serial.SendDirect(state, port, timeout)
}

func runStatus() int {
	daemonUp := socket.PingSocket(socketPath, timeout) == nil
	devicePort := serial.FindPort()
	deviceUp := devicePort != ""

	sym := func(up bool) string {
		if up {
			return "●"
		}
		return "○"
	}
	label := func(up bool) string {
		if up {
			return "running"
		}
		return "stopped"
	}

	if daemonUp {
		fmt.Printf("daemon   %s %s    (%s)\n", sym(true), label(true), socketPath)
	} else {
		fmt.Printf("daemon   %s %s\n", sym(false), label(false))
	}

	if deviceUp {
		fmt.Printf("device   %s connected  (%s)\n", sym(true), devicePort)
	} else {
		fmt.Printf("device   %s not found\n", sym(false))
	}

	if daemonUp && deviceUp {
		return 0
	}
	return 1
}

func printUsage() {
	fmt.Println(`Usage:
  beacon <state>             send state via daemon socket (exit 0 always)
  beacon <state> --direct    send state directly to serial (exit 0 always)
  beacon daemon              run the daemon (blocking)
  beacon status              show daemon and device health (exit 0 or 1)

States: thinking, waiting, done, idle, error, ping`)
}
