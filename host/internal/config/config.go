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

// validKeys maps dotted key names to validation functions.
// Returns an error if the value is invalid for that key.
var validKeys = map[string]func(string) error{
	"brightness":     validateBrightness,
	"color.thinking": validateColor,
	"color.waiting":  validateColor,
	"color.done":     validateColor,
	"color.error":    validateColor,
	"timing.done":    validateTimingMs,
	"timing.waiting": validateTimingMs,
	"buzzer.enabled": validateBool,
	"buzzer.freq":    validateFreq,
}

func validateBrightness(v string) error {
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 0 || n > 255 {
		return fmt.Errorf("brightness must be 0-255, got %q", v)
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
			return fmt.Errorf("color values must be 0-255, got %q", v)
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
		return fmt.Errorf("buzzer.freq must be 100-5000 Hz, got %q", v)
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
