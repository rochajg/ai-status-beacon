package config

import (
	"fmt"
	"os"
	"path/filepath"

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
