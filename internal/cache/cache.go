// Package cache provides an on-disk, etag-aware response cache for
// YouTube Data API list calls.
//
// Entries are keyed by a SHA-256 hash of (endpoint, sorted-params),
// stored as JSON files under os.UserCacheDir()/yt/. Each entry records the
// API-returned ETag and the raw response payload. Callers do a two-step
// dance:
//
//  1. Lookup(key) — returns the previously cached etag (if any) and the
//     stored payload bytes.
//  2. After making the API call with If-None-Match: <etag>, either reuse the
//     stored payload (on http.StatusNotModified) or call Store(key, etag,
//     freshPayload) with the fresh response.
//
// The cache only stores list responses — it makes no assumption about what
// "fresh" means beyond "the API returned 200 with a new etag".
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrMiss is returned by Lookup when no cached entry exists for the key.
var ErrMiss = errors.New("cache miss")

// Entry is the on-disk cache record.
type Entry struct {
	Etag    string          `json:"etag"`
	Payload json.RawMessage `json:"payload"`
}

// Key builds a stable cache key for an API call. params is keyed-value
// pairs (e.g. "playlistId", "PLxx", "pageToken", "..."); order does not
// matter. The endpoint is something like "playlistItems.list".
func Key(endpoint string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(endpoint)
	for _, k := range keys {
		b.WriteByte('\x1f')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// Dir returns the cache directory, creating it if needed.
func Dir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "yt")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func entryPath(key string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key+".json"), nil
}

// Lookup returns the stored entry for key. ErrMiss when not present.
func Lookup(key string) (*Entry, error) {
	path, err := entryPath(key)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrMiss
		}
		return nil, fmt.Errorf("cache read: %w", err)
	}
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, fmt.Errorf("cache decode: %w", err)
	}
	return &e, nil
}

// Store writes etag+payload to the cache atomically.
func Store(key, etag string, payload json.RawMessage) error {
	if etag == "" {
		// Without an etag we have no way to revalidate later — skip silently
		// so callers don't have to special-case this.
		return nil
	}
	path, err := entryPath(key)
	if err != nil {
		return err
	}
	data, err := json.Marshal(Entry{Etag: etag, Payload: payload})
	if err != nil {
		return fmt.Errorf("cache encode: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("cache tmp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("cache write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cache close: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cache chmod: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cache rename: %w", err)
	}
	return nil
}

// Clear deletes every cached entry. Returns the number of files removed.
func Clear() (int, error) {
	dir, err := Dir()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("cache list: %w", err)
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return removed, fmt.Errorf("cache remove %s: %w", name, err)
		}
		removed++
	}
	return removed, nil
}
