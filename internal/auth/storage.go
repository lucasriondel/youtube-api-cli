package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// ErrNoToken is returned when no token has been stored yet.
var ErrNoToken = errors.New("no token stored; run `yt auth login` first")

// ErrNoClientSecret is returned when client credentials haven't been stored yet.
var ErrNoClientSecret = errors.New("no client credentials stored; run `yt auth credentials <path/to/client_secret.json>` first")

// SaveClientSecret stores the raw client_secret JSON in the OS keyring,
// falling back to a 0600 file in the config dir if the keyring is unavailable.
func SaveClientSecret(raw []byte) error {
	if err := keyring.Set(keyringService, keyringClientKey, string(raw)); err == nil {
		return nil
	}
	return writeFallback("client_secret.json", raw)
}

// LoadClientSecret retrieves the previously stored client_secret JSON.
func LoadClientSecret() ([]byte, error) {
	if raw, err := keyring.Get(keyringService, keyringClientKey); err == nil {
		return []byte(raw), nil
	}
	data, err := readFallback("client_secret.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoClientSecret
		}
		return nil, err
	}
	return data, nil
}

// DeleteClientSecret removes stored client credentials from both backends.
func DeleteClientSecret() error {
	_ = keyring.Delete(keyringService, keyringClientKey)
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "client_secret.json")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// SaveToken stores a token in the OS keyring, falling back to a 0600 file
// in the config dir if the keyring is unavailable.
func SaveToken(tok *oauth2.Token) error {
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, keyringTokenKey, string(data)); err == nil {
		return nil
	}
	return writeFallback("token.json", data)
}

// LoadToken retrieves a previously stored token.
func LoadToken() (*oauth2.Token, error) {
	if raw, err := keyring.Get(keyringService, keyringTokenKey); err == nil {
		var tok oauth2.Token
		if err := json.Unmarshal([]byte(raw), &tok); err != nil {
			return nil, fmt.Errorf("decode keyring token: %w", err)
		}
		return &tok, nil
	}
	data, err := readFallback("token.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoToken
		}
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("decode token file: %w", err)
	}
	return &tok, nil
}

// DeleteToken removes the stored token from both keyring and fallback file.
func DeleteToken() error {
	_ = keyring.Delete(keyringService, keyringTokenKey)
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "token.json")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func writeFallback(name string, data []byte) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o600)
}

func readFallback(name string) ([]byte, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(dir, name))
}
