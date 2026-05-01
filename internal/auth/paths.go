package auth

import (
	"os"
	"path/filepath"
)

const (
	keyringService = "yt-cli"
	keyringTokenKey = "oauth2-token"
	keyringClientKey = "oauth2-client"
)

// ConfigDir returns ~/.config/yt, creating it if needed.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "yt")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}
