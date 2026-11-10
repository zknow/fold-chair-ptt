package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	AlwaysOnTop    bool   `json:"alwaysOnTop"`
	Theme          string `json:"theme"`
	FontSize       int    `json:"fontSize"`
	PollIntervalMs int    `json:"pollIntervalMs"`
	MaxMessages    int    `json:"maxMessages"`
}

func Default() Settings {
	return Settings{
		AlwaysOnTop:    true,
		Theme:          "light",
		FontSize:       13,
		PollIntervalMs: 3000,
		MaxMessages:    500,
	}
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fold-chair-ptt", "config.json"), nil
}

func Load() Settings {
	s := Default()
	path, err := configPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return Normalize(s)
}

func Normalize(s Settings) Settings {
	if s.PollIntervalMs < 1000 {
		s.PollIntervalMs = 3000
	}
	if s.MaxMessages < 50 {
		s.MaxMessages = 500
	}
	if s.FontSize < 10 || s.FontSize > 24 {
		s.FontSize = 13
	}
	if s.Theme != "dark" {
		s.Theme = "light"
	}
	return s
}

func Save(s Settings) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
