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
