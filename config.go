package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// config holds the user's settings. It lives in the user's roaming profile
// (%APPDATA%\claude-use\config.json on Windows), never next to the executable,
// so it works when the app is installed in a read-only folder such as Program Files.
type config struct {
	Language string     `json:"language,omitempty"`
	Position *windowPos `json:"position,omitempty"` // top-left corner in screen pixels; nil = default dock
}

type windowPos struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "claude-use", "config.json"), nil
}

// loadConfig returns the saved settings, or defaults if the file is missing or invalid.
func loadConfig() config {
	var c config
	path, err := configPath()
	if err != nil {
		return c
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &c)
	}
	return c
}

// save writes the settings atomically (temp file + rename).
func (c config) save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
