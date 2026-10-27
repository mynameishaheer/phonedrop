package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	appName     = "PhoneDrop"
	version     = "0.9.3" // reported by GET /v1/ping; keep winres/winres.json in step
	defaultPort = 47474
)

// settings: %LOCALAPPDATA%\PhoneDrop\settings.json. Start-at-sign-in isn't stored here; the
// Run key is the source of truth, so the installer and the panel can't disagree.
type settings struct {
	SaveDir  string `json:"save_dir"`  // empty: the user's Downloads folder
	AutoCopy bool   `json:"auto_copy"` // put received text on the clipboard
	Hotkey   bool   `json:"hotkey"`    // Ctrl+Alt+P sends the clipboard to the phone
	Encrypt  bool   `json:"encrypt"`   // TLS mode (beta, PROTOCOL.md §7)
	Port     int    `json:"port"`
	// Trusted: Public networks the user chose "Use here" on (Windows network ids).
	Trusted        []trustedNet `json:"trusted,omitempty"`
	PublicFirewall bool         `json:"public_firewall,omitempty"` // the Public-profile firewall rule was added
}

type trustedNet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func dataDir() string {
	base, err := os.UserCacheDir() // %LOCALAPPDATA% on Windows
	if err != nil {
		base = "."
	}
	return filepath.Join(base, appName)
}

func settingsPath() string { return filepath.Join(dataDir(), "settings.json") }

func defaultSettings() settings { return settings{AutoCopy: true, Hotkey: true, Port: defaultPort} }

func loadSettings() settings {
	s := defaultSettings()
	if b, err := os.ReadFile(settingsPath()); err == nil {
		json.Unmarshal(b, &s)
	}
	if s.Port <= 0 || s.Port > 65535 {
		s.Port = defaultPort
	}
	return s
}

func (s settings) save() error {
	if err := os.MkdirAll(dataDir(), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := settingsPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, settingsPath())
}

// saveDir: where received files go.
func (s settings) saveDir() string {
	if s.SaveDir != "" {
		return s.SaveDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return dataDir()
	}
	return filepath.Join(home, "Downloads")
}
