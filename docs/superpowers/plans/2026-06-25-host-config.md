# Status Beacon — Host Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-host TOML configuration (`~/.beacon/config.toml`) so the CLI resolves colors, timings, and buzzer settings per machine and sends them as extended serial parameters to the firmware.

**Architecture:** New `internal/config` Go package loads `~/.beacon/config.toml`, exposes `Resolve(state)` that returns only the set keys as `key=value` strings. The CLI builds `"thinking color=0,100,255 brightness=200\n"` instead of `"thinking\n"` when config is present. Daemon is unchanged — it forwards verbatim. `beacon config get|set|reset|path` manages the TOML file.

**Tech Stack:** Go 1.25, `github.com/pelletier/go-toml/v2` (TOML parser/writer).

## Global Constraints

- Config file path: `~/.beacon/config.toml`
- TOML color format: `"R,G,B"` string (e.g. `"0,100,255"`)
- Valid states for color: `thinking`, `waiting`, `done`, `error`
- Valid dotted keys: `brightness`, `color.thinking`, `color.waiting`, `color.done`, `color.error`, `timing.done`, `timing.waiting`, `buzzer.enabled`, `buzzer.freq`
- `brightness`: int 0–255
- `color.*`: three ints 0–255, comma-separated, no spaces
- `timing.*`: int ≥ 0 (milliseconds)
- `buzzer.enabled`: `true`/`false`/`1`/`0`
- `buzzer.freq`: int 100–5000 (Hz)
- `beacon config set` errors exit 1 with message on stderr; list valid keys
- All state-sending commands (`beacon <state>`) remain exit 0 always
- Module: `beacon` at `host/`; run tests with `cd host && go test ./...`
- Commits: Conventional Commits in English
- Co-author every commit: `Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>`

---

## File Map

```
host/
  go.mod                              ← add go-toml/v2 dependency
  go.sum                              ← updated
  main.go                             ← add "config" subcommand dispatch; wire Resolve() into runSocket/runDirect
  internal/
    config/
      config.go                       ← Config struct, Load, Set, Reset, Path, Resolve
      config_test.go                  ← unit tests (no filesystem side effects — use t.TempDir())
```

---

### Task 1: `internal/config` — Load, Path, Resolve

**Files:**
- Create: `host/internal/config/config.go`
- Create: `host/internal/config/config_test.go`
- Modify: `host/go.mod` (add go-toml/v2)

**Produces:**
- `config.DefaultPath() string`
- `config.Load(path string) (Config, error)` — returns empty Config on file-not-found
- `(Config) Resolve(state string) []string` — returns `["color=R,G,B", "brightness=N", ...]`, omitting unset keys

- [ ] **Step 1: Add go-toml/v2 dependency**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go get github.com/pelletier/go-toml/v2
```

Expected: `go.mod` updated with `github.com/pelletier/go-toml/v2`.

- [ ] **Step 2: Write failing tests — `host/internal/config/config_test.go`**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTOML(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Brightness != nil || cfg.Color.Thinking != nil {
		t.Fatal("expected empty config")
	}
}

func TestLoadBrightness(t *testing.T) {
	p := writeTOML(t, t.TempDir(), "brightness = 180\n")
	cfg, _ := Load(p)
	if cfg.Brightness == nil || *cfg.Brightness != 180 {
		t.Fatalf("want 180, got %v", cfg.Brightness)
	}
}

func TestLoadColor(t *testing.T) {
	p := writeTOML(t, t.TempDir(), "[color]\nthinking = \"0,100,255\"\n")
	cfg, _ := Load(p)
	if cfg.Color.Thinking == nil {
		t.Fatal("thinking color not loaded")
	}
	c := cfg.Color.Thinking
	if c.R != 0 || c.G != 100 || c.B != 255 {
		t.Fatalf("want 0,100,255 got %d,%d,%d", c.R, c.G, c.B)
	}
}

func TestLoadBuzzer(t *testing.T) {
	p := writeTOML(t, t.TempDir(), "[buzzer]\nenabled = false\nfreq = 880\n")
	cfg, _ := Load(p)
	if cfg.Buzzer.Enabled == nil || *cfg.Buzzer.Enabled != false {
		t.Fatal("buzzer.enabled not loaded")
	}
	if cfg.Buzzer.Freq == nil || *cfg.Buzzer.Freq != 880 {
		t.Fatal("buzzer.freq not loaded")
	}
}

func TestResolveEmptyConfigBareCommand(t *testing.T) {
	var cfg Config
	if params := cfg.Resolve("thinking"); len(params) != 0 {
		t.Fatalf("expected empty params, got %v", params)
	}
}

func TestResolveColorOnly(t *testing.T) {
	r, g, b := uint8(0), uint8(100), uint8(255)
	var cfg Config
	cfg.Color.Thinking = &RGB{r, g, b}
	params := cfg.Resolve("thinking")
	if len(params) != 1 || params[0] != "color=0,100,255" {
		t.Fatalf("unexpected params: %v", params)
	}
}

func TestResolveFullThinking(t *testing.T) {
	br := uint8(180)
	var cfg Config
	cfg.Color.Thinking = &RGB{0, 100, 255}
	cfg.Brightness = &br
	params := cfg.Resolve("thinking")
	// order: color, brightness, timing, buzzer, freq
	want := map[string]bool{"color=0,100,255": true, "brightness=180": true}
	got := map[string]bool{}
	for _, p := range params {
		got[p] = true
	}
	for k := range want {
		if !got[k] {
			t.Fatalf("missing param %q in %v", k, params)
		}
	}
}

func TestResolveTimingForDone(t *testing.T) {
	ms := 60000
	var cfg Config
	cfg.Timing.Done = &ms
	params := cfg.Resolve("done")
	found := false
	for _, p := range params {
		if p == "timing=60000" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing timing param in %v", params)
	}
}

func TestResolveTimingNotAddedForThinking(t *testing.T) {
	ms := 60000
	var cfg Config
	cfg.Timing.Done = &ms
	for _, p := range cfg.Resolve("thinking") {
		if len(p) > 6 && p[:7] == "timing=" {
			t.Fatalf("timing should not appear for thinking: %v", p)
		}
	}
}

func TestResolveBuzzerEnabled(t *testing.T) {
	enabled := false
	var cfg Config
	cfg.Buzzer.Enabled = &enabled
	params := cfg.Resolve("waiting")
	found := false
	for _, p := range params {
		if p == "buzzer=0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing buzzer=0 in %v", params)
	}
}

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}
}
```

- [ ] **Step 3: Run tests — expect compile failure**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go test ./internal/config/... 2>&1 | head -5
```

Expected: `no Go files` or `undefined`.

- [ ] **Step 4: Implement `host/internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// RGB is a WS2812 color stored as a TOML string "R,G,B".
type RGB struct{ R, G, B uint8 }

func (rgb *RGB) UnmarshalText(text []byte) error {
	_, err := fmt.Sscanf(string(text), "%d,%d,%d", &rgb.R, &rgb.G, &rgb.B)
	return err
}

func (rgb RGB) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%d,%d,%d", rgb.R, rgb.G, rgb.B)), nil
}

// Config holds the per-host beacon configuration. All fields are optional
// (nil pointer = unset = firmware uses its compiled default).
type Config struct {
	Brightness *uint8 `toml:"brightness,omitempty"`
	Color      struct {
		Thinking *RGB `toml:"thinking,omitempty"`
		Waiting  *RGB `toml:"waiting,omitempty"`
		Done     *RGB `toml:"done,omitempty"`
		Error    *RGB `toml:"error,omitempty"`
	} `toml:"color,omitempty"`
	Timing struct {
		Done    *int `toml:"done,omitempty"`
		Waiting *int `toml:"waiting,omitempty"`
	} `toml:"timing,omitempty"`
	Buzzer struct {
		Enabled *bool `toml:"enabled,omitempty"`
		Freq    *int  `toml:"freq,omitempty"`
	} `toml:"buzzer,omitempty"`
}

// DefaultPath returns ~/.beacon/config.toml.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".beacon", "config.toml")
}

// Load reads path and returns a Config. Returns an empty Config (no error) if
// the file does not exist.
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	return cfg, toml.Unmarshal(data, &cfg)
}

// Resolve returns the key=value parameter strings for state, omitting keys that
// are not set in cfg. The firmware applies its compiled defaults for omitted keys.
// Parameter order: color, brightness, timing, buzzer, freq.
func (c Config) Resolve(state string) []string {
	var params []string

	if color := c.colorForState(state); color != nil {
		params = append(params, fmt.Sprintf("color=%d,%d,%d", color.R, color.G, color.B))
	}
	if c.Brightness != nil {
		params = append(params, fmt.Sprintf("brightness=%d", *c.Brightness))
	}
	switch state {
	case "done":
		if c.Timing.Done != nil {
			params = append(params, fmt.Sprintf("timing=%d", *c.Timing.Done))
		}
	case "waiting":
		if c.Timing.Waiting != nil {
			params = append(params, fmt.Sprintf("timing=%d", *c.Timing.Waiting))
		}
	}
	if c.Buzzer.Enabled != nil {
		v := 0
		if *c.Buzzer.Enabled {
			v = 1
		}
		params = append(params, fmt.Sprintf("buzzer=%d", v))
	}
	if c.Buzzer.Freq != nil {
		params = append(params, fmt.Sprintf("freq=%d", *c.Buzzer.Freq))
	}
	return params
}

func (c Config) colorForState(state string) *RGB {
	switch state {
	case "thinking":
		return c.Color.Thinking
	case "waiting":
		return c.Color.Waiting
	case "done":
		return c.Color.Done
	case "error":
		return c.Color.Error
	}
	return nil
}
```

The `Set` function (added in Task 2) uses the same `toml` import already declared above.

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go test ./internal/config/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add host/go.mod host/go.sum host/internal/config/
git commit -m "$(cat <<'EOF'
feat(config): add internal/config — Load, Resolve per-state params

Config struct with optional fields (nil = unset = firmware default).
RGB uses encoding.TextUnmarshaler for TOML "R,G,B" string format.
Resolve() returns only set keys as key=value params for the firmware.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: `internal/config` — Set and Reset

**Files:**
- Modify: `host/internal/config/config.go` (add `Set`, `Reset`)
- Modify: `host/internal/config/config_test.go` (add tests)

**Produces:**
- `config.Set(path, key, value string) error`
- `config.Reset(path string) error`

- [ ] **Step 1: Add tests for Set and Reset**

Append to `host/internal/config/config_test.go`:

```go
func TestSetCreatesFileAndSetsKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")

	if err := Set(p, "brightness", "180"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load(p)
	if cfg.Brightness == nil || *cfg.Brightness != 180 {
		t.Fatalf("want 180, got %v", cfg.Brightness)
	}
}

func TestSetColorThinking(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")

	if err := Set(p, "color.thinking", "0,100,255"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load(p)
	if cfg.Color.Thinking == nil {
		t.Fatal("color.thinking not set")
	}
	if cfg.Color.Thinking.R != 0 || cfg.Color.Thinking.G != 100 || cfg.Color.Thinking.B != 255 {
		t.Fatalf("unexpected color: %+v", cfg.Color.Thinking)
	}
}

func TestSetPreservesExistingKeys(t *testing.T) {
	dir := t.TempDir()
	p := writeTOML(t, dir, "brightness = 200\n")

	if err := Set(p, "color.thinking", "0,100,255"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load(p)
	if cfg.Brightness == nil || *cfg.Brightness != 200 {
		t.Fatal("existing brightness lost after Set")
	}
}

func TestSetBuzzerEnabled(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")

	if err := Set(p, "buzzer.enabled", "false"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load(p)
	if cfg.Buzzer.Enabled == nil || *cfg.Buzzer.Enabled != false {
		t.Fatal("buzzer.enabled not set to false")
	}
}

func TestSetRejectsUnknownKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := Set(p, "color.rainbow", "1,2,3"); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestSetRejectsInvalidBrightness(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := Set(p, "brightness", "999"); err == nil {
		t.Fatal("expected error for out-of-range brightness")
	}
}

func TestSetRejectsInvalidColor(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := Set(p, "color.thinking", "not-a-color"); err == nil {
		t.Fatal("expected error for invalid color")
	}
}

func TestResetDeletesFile(t *testing.T) {
	dir := t.TempDir()
	p := writeTOML(t, dir, "brightness = 180\n")

	if err := Reset(p); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load(p) // missing file → empty config
	if cfg.Brightness != nil {
		t.Fatal("expected empty config after reset")
	}
}

func TestResetNoErrorIfFileAbsent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := Reset(p); err != nil {
		t.Fatalf("unexpected error resetting absent file: %v", err)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL (Set/Reset undefined)**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go test ./internal/config/... 2>&1 | grep -E "FAIL|undefined"
```

Expected: `undefined: Set` and `undefined: Reset`.

- [ ] **Step 3: Implement `Set` and `Reset` in `host/internal/config/config.go`**

Remove the `encoding/json` import and `var _ = json.Marshal` line added as a placeholder in Task 1. Add these functions and the `strings` import (already present):

```go
// validKeys maps dotted key names to validation functions.
// Returns an error if the value is invalid for that key.
var validKeys = map[string]func(string) error{
	"brightness":      validateBrightness,
	"color.thinking":  validateColor,
	"color.waiting":   validateColor,
	"color.done":      validateColor,
	"color.error":     validateColor,
	"timing.done":     validateTimingMs,
	"timing.waiting":  validateTimingMs,
	"buzzer.enabled":  validateBool,
	"buzzer.freq":     validateFreq,
}

func validateBrightness(v string) error {
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 0 || n > 255 {
		return fmt.Errorf("brightness must be 0–255, got %q", v)
	}
	return nil
}

func validateColor(v string) error {
	var r, g, b int
	if _, err := fmt.Sscanf(v, "%d,%d,%d", &r, &g, &b); err != nil {
		return fmt.Errorf("color must be R,G,B (e.g. 0,100,255), got %q", v)
	}
	for _, c := range []int{r, g, b} {
		if c < 0 || c > 255 {
			return fmt.Errorf("color values must be 0–255, got %q", v)
		}
	}
	return nil
}

func validateTimingMs(v string) error {
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 0 {
		return fmt.Errorf("timing must be a non-negative integer (ms), got %q", v)
	}
	return nil
}

func validateBool(v string) error {
	switch strings.ToLower(v) {
	case "true", "false", "1", "0":
		return nil
	}
	return fmt.Errorf("must be true/false/1/0, got %q", v)
}

func validateFreq(v string) error {
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 100 || n > 5000 {
		return fmt.Errorf("buzzer.freq must be 100–5000 Hz, got %q", v)
	}
	return nil
}

// Set writes a single dotted key to the TOML config file, creating it if needed.
// Returns an error for unknown keys or invalid values.
func Set(path, key, value string) error {
	validate, ok := validKeys[key]
	if !ok {
		keys := make([]string, 0, len(validKeys))
		for k := range validKeys {
			keys = append(keys, k)
		}
		return fmt.Errorf("unknown config key %q\nvalid keys: %s", key, strings.Join(keys, ", "))
	}
	if err := validate(value); err != nil {
		return err
	}

	// Load existing file as a raw map so we can update one key.
	raw := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = toml.Unmarshal(data, &raw)
	}

	// Navigate/create nested map for dotted key.
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 1 {
		raw[key] = tomlValue(key, value)
	} else {
		sub, _ := raw[parts[0]].(map[string]any)
		if sub == nil {
			sub = map[string]any{}
		}
		sub[parts[1]] = tomlValue(key, value)
		raw[parts[0]] = sub
	}

	// Marshal and write.
	data, err := toml.Marshal(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// tomlValue converts the string value to the native Go type for the key.
func tomlValue(key, value string) any {
	switch key {
	case "brightness", "timing.done", "timing.waiting", "buzzer.freq":
		var n int
		fmt.Sscanf(value, "%d", &n)
		return n
	case "buzzer.enabled":
		return value == "true" || value == "1"
	default: // color.* — store as string
		return value
	}
}

// Reset deletes the config file. Returns nil if the file does not exist.
func Reset(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
```

Also remove the `encoding/json` import and `var _ = json.Marshal` line from Task 1.

- [ ] **Step 4: Run all tests — expect PASS**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go test ./internal/config/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add host/internal/config/
git commit -m "$(cat <<'EOF'
feat(config): add Set and Reset with key validation

Set reads existing TOML, updates one dotted key, writes back.
Validates all keys and values before writing. Unknown keys error.
Reset deletes the file; no-op if absent.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: `beacon config` subcommand in main.go

**Files:**
- Modify: `host/main.go`

**Consumes:**
- `config.DefaultPath() string` — Task 1
- `config.Load(path string) (Config, error)` — Task 1
- `config.Set(path, key, value string) error` — Task 2
- `config.Reset(path string) error` — Task 2

**Produces:**
- `beacon config get` — prints effective config (all set keys + firmware defaults for unset)
- `beacon config set <key> <value>` — writes one key
- `beacon config reset` — deletes the file
- `beacon config path` — prints the file path

- [ ] **Step 1: Add `beacon config` dispatch to main.go**

In `host/main.go`, after the existing imports, add:

```go
import (
    // ... existing imports ...
    "beacon/internal/config"
)
```

In `main()`, add a `case "config":` before `default:`:

```go
case "config":
    os.Exit(runConfig(args[1:]))
```

Add `runConfig` and helper functions at the bottom of `host/main.go`:

```go
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
```

Also update `printUsage()` to include the config subcommand:

```go
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
```

- [ ] **Step 2: Build and verify**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go build -o /tmp/beacon-test .
/tmp/beacon-test config path
```

Expected: prints `~/.beacon/config.toml` (full expanded path).

```bash
/tmp/beacon-test config set color.thinking 0,100,255
/tmp/beacon-test config get
/tmp/beacon-test config reset
```

- [ ] **Step 3: Run all tests**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go test ./... -timeout 30s
```

Expected: all packages pass.

- [ ] **Step 4: Rebuild and install**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go build -o ~/.local/bin/beacon .
beacon config path
```

- [ ] **Step 5: Commit**

```bash
git add host/main.go
git commit -m "$(cat <<'EOF'
feat(config): add beacon config get|set|reset|path subcommand

Dispatches to runConfig() which reads/writes ~/.beacon/config.toml.
get prints all keys with (firmware default) for unset values.
set validates key + value before writing. reset removes the file.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Wire config resolution into state commands

**Files:**
- Modify: `host/main.go` (update `runSocket` and `runDirect`)

**Consumes:**
- `config.DefaultPath() string` — Task 1
- `config.Load(path string) (Config, error)` — Task 1
- `(Config) Resolve(state string) []string` — Task 1

**Produces:**
- `buildCommand(state string, cfg config.Config) string` — returns `"state key=val ...\n"`
- `runSocket` and `runDirect` use `buildCommand` instead of bare `state + "\n"`

- [ ] **Step 1: Add `buildCommand` and update `runSocket`/`runDirect`**

In `host/main.go`:

```go
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
```

Add `"strings"` to the import block.

Update `runSocket`:
```go
func runSocket(state string) {
	cfg, _ := config.Load(config.DefaultPath())
	cmd := buildCommand(state, cfg)
	_ = socket.SendToSocket(cmd, socketPath, timeout)
}
```

Update `runDirect`:
```go
func runDirect(state string) {
	port := serial.FindPort()
	if port == "" {
		return
	}
	cfg, _ := config.Load(config.DefaultPath())
	cmd := buildCommand(state, cfg)
	_ = serial.SendDirectRaw(cmd, port, timeout)
}
```

> **Note:** `SendDirect` currently takes `state string` not the full command. You need to add `SendDirectRaw(cmd, port string, timeout time.Duration) error` to `internal/serial/serial.go` that writes the raw string (instead of building `state + "\n"` internally). Or rename the existing function. See Step 2.

- [ ] **Step 2: Add `SendDirectRaw` to `host/internal/serial/serial.go`**

Add after `SendDirect`:

```go
// SendDirectRaw opens the serial port and writes cmd verbatim (no newline added).
// Used by the CLI when the command already includes params and a trailing newline.
func SendDirectRaw(cmd, port string, timeout time.Duration) error {
	p, err := goserial.Open(port, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { _, err := fmt.Fprint(p, cmd); errCh <- err }()
	select {
	case err := <-errCh:
		p.Close()
		return err
	case <-ctx.Done():
		p.Close()
		<-errCh
		return ctx.Err()
	}
}
```

Also add a test to `host/internal/serial/serial_test.go`:

```go
func TestSendDirectRawWritesVerbatim(t *testing.T) {
	pr, pw := io.Pipe()
	mock := &mockPort{WriteCloser: pw}

	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := pr.Read(buf)
		done <- string(buf[:n])
	}()

	if err := sendDirectRaw("thinking color=0,100,255\n", mock, 300*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got := <-done; got != "thinking color=0,100,255\n" {
		t.Fatalf("want %q, got %q", "thinking color=0,100,255\n", got)
	}
}
```

Add an unexported `sendDirectRaw` that accepts a `portWriter` for testability (same pattern as `writeState`):

```go
func sendDirectRaw(cmd string, p portWriter, timeout time.Duration) error {
	defer p.Close()
	_, err := fmt.Fprint(p, cmd)
	return err
}
```

And `SendDirectRaw` calls it via the real serial port.

- [ ] **Step 3: Run all tests**

```bash
cd /Users/jorocha/pessoal/ai-status-beacon/host
go test ./... -timeout 30s
```

Expected: all pass.

- [ ] **Step 4: Smoke test**

```bash
beacon config set color.thinking 0,100,255
beacon config set brightness 180

# With daemon running:
beacon thinking
# Expected: firmware receives "thinking color=0,100,255 brightness=180\n"
# (verify by checking daemon logs or firmware serial output)
```

- [ ] **Step 5: Commit**

```bash
git add host/main.go host/internal/serial/
git commit -m "$(cat <<'EOF'
feat(config): wire config resolution into state commands

runSocket and runDirect now build extended commands via buildCommand()
which resolves ~/.beacon/config.toml into key=value params.
Bare state\n sent when no config is set (backwards compatible).
Added SendDirectRaw + sendDirectRaw to serial package for raw command.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```
