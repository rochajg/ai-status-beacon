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

