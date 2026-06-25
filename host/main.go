package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"beacon/internal/config"
	"beacon/internal/daemon"
	"beacon/internal/serial"
	"beacon/internal/socket"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3"
var version = "dev"

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
	case "version", "--version", "-v":
		fmt.Println(version)
		os.Exit(0)

	case "daemon":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		daemon.Run(ctx, socketPath, baud)

	case "status":
		os.Exit(runStatus())

	case "config":
		os.Exit(runConfig(args[1:]))

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

// buildCommand assembles the extended serial command for state, resolving
// host config into key=value params. Falls back to bare "state\n" when
// no config keys are set for this state.
func buildCommand(state string, cfg config.Config) string {
	params := cfg.Resolve(state)
	if len(params) == 0 {
		return state + "\n"
	}
	return state + " " + strings.Join(params, " ") + "\n"
}

func runSocket(state string) {
	cfg, _ := config.Load(config.DefaultPath())
	cmd := buildCommand(state, cfg)
	_ = socket.SendToSocket(cmd, socketPath, timeout)
}

func runDirect(state string) {
	port := serial.FindPort()
	if port == "" {
		return
	}
	cfg, _ := config.Load(config.DefaultPath())
	cmd := buildCommand(state, cfg)
	_ = serial.SendDirectRaw(cmd, port, timeout)
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
	fmt.Printf(`AI Status Beacon %s

Usage:
  beacon <state>                      send via daemon (exit 0 always)
  beacon <state> --direct             send directly to serial (exit 0 always)
  beacon daemon                       run the daemon (blocking)
  beacon status                       daemon + device health (exit 0 or 1)
  beacon config get                   show current config
  beacon config set <key> <value>     write one config key
  beacon config reset                 restore firmware defaults
  beacon config path                  print config file path
  beacon version                      print version

States: thinking, waiting, done, idle, error
`, version)
}

func runConfig(args []string) int {
	cfgPath := config.DefaultPath()

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: beacon config <get|set|reset|path>")
		return 1
	}

	switch args[0] {
	case "path":
		fmt.Println(cfgPath)
		return 0

	case "reset":
		if err := config.Reset(cfgPath); err != nil {
			fmt.Fprintln(os.Stderr, "beacon config reset:", err)
			return 1
		}
		fmt.Println("Config reset to defaults.")
		return 0

	case "set":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "Usage: beacon config set <key> <value>")
			return 1
		}
		if err := config.Set(cfgPath, args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "beacon config set:", err)
			return 1
		}
		fmt.Printf("Set %s = %s\n", args[1], args[2])
		return 0

	case "get":
		cfg, err := config.Load(cfgPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "beacon config get:", err)
			return 1
		}
		printConfig(cfg, cfgPath)
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown config command %q\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: beacon config <get|set|reset|path>")
		return 1
	}
}

func printConfig(cfg config.Config, path string) {
	fmt.Printf("Config file: %s\n\n", path)

	printOptRGB := func(label string, v *config.RGB) {
		if v != nil {
			fmt.Printf("  %-22s %d,%d,%d\n", label, v.R, v.G, v.B)
		} else {
			fmt.Printf("  %-22s (firmware default)\n", label)
		}
	}
	printOptInt := func(label string, v *int) {
		if v != nil {
			fmt.Printf("  %-22s %d\n", label, *v)
		} else {
			fmt.Printf("  %-22s (firmware default)\n", label)
		}
	}
	printOptUint8 := func(label string, v *uint8) {
		if v != nil {
			fmt.Printf("  %-22s %d\n", label, *v)
		} else {
			fmt.Printf("  %-22s (firmware default)\n", label)
		}
	}
	printOptBool := func(label string, v *bool) {
		if v != nil {
			fmt.Printf("  %-22s %v\n", label, *v)
		} else {
			fmt.Printf("  %-22s (firmware default)\n", label)
		}
	}

	fmt.Println("Colors:")
	printOptRGB("color.thinking", cfg.Color.Thinking)
	printOptRGB("color.waiting", cfg.Color.Waiting)
	printOptRGB("color.done", cfg.Color.Done)
	printOptRGB("color.error", cfg.Color.Error)
	fmt.Println("Brightness:")
	printOptUint8("brightness", cfg.Brightness)
	fmt.Println("Timings:")
	printOptInt("timing.done (ms)", cfg.Timing.Done)
	printOptInt("timing.waiting (ms)", cfg.Timing.Waiting)
	fmt.Println("Buzzer:")
	printOptBool("buzzer.enabled", cfg.Buzzer.Enabled)
	printOptInt("buzzer.freq (Hz)", cfg.Buzzer.Freq)
}
