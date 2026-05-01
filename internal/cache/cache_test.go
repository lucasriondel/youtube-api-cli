package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("HOME", dir)
	return filepath.Join(dir, "yt")
}

func TestKeyStableAcrossParamOrder(t *testing.T) {
	a := Key("playlistItems.list", map[string]string{"playlistId": "PL1", "pageToken": ""})
	b := Key("playlistItems.list", map[string]string{"pageToken": "", "playlistId": "PL1"})
	if a != b {
		t.Fatalf("expected stable key regardless of map order, got %q vs %q", a, b)
	}
}

func TestKeyDistinguishesParams(t *testing.T) {
	a := Key("playlistItems.list", map[string]string{"playlistId": "PL1"})
	b := Key("playlistItems.list", map[string]string{"playlistId": "PL2"})
	if a == b {
		t.Fatalf("expected different keys for different playlistIds, got %q", a)
	}
}

func TestStoreAndLookupRoundTrip(t *testing.T) {
	withTempCache(t)
	key := Key("test.endpoint", map[string]string{"x": "1"})
	payload := json.RawMessage(`{"hello":"world"}`)
	if err := Store(key, "etag-abc", payload); err != nil {
		t.Fatalf("Store: %v", err)
	}
	entry, err := Lookup(key)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if entry.Etag != "etag-abc" {
		t.Errorf("etag = %q, want etag-abc", entry.Etag)
	}
	if string(entry.Payload) != `{"hello":"world"}` {
		t.Errorf("payload = %s, want {\"hello\":\"world\"}", entry.Payload)
	}
}

func TestStoreSkipsEmptyEtag(t *testing.T) {
	withTempCache(t)
	key := Key("test.endpoint", map[string]string{"x": "noetag"})
	if err := Store(key, "", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("Store with empty etag returned error: %v", err)
	}
	if _, err := Lookup(key); !errors.Is(err, ErrMiss) {
		t.Fatalf("expected ErrMiss after storing empty etag, got %v", err)
	}
}

func TestLookupMissReturnsErrMiss(t *testing.T) {
	withTempCache(t)
	if _, err := Lookup("nonexistent"); !errors.Is(err, ErrMiss) {
		t.Fatalf("expected ErrMiss, got %v", err)
	}
}

func TestClearRemovesEntries(t *testing.T) {
	cacheDir := withTempCache(t)
	keys := []string{
		Key("a", map[string]string{"k": "1"}),
		Key("b", map[string]string{"k": "2"}),
	}
	for _, k := range keys {
		if err := Store(k, "etag", json.RawMessage(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	n, err := Clear()
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if n != len(keys) {
		t.Errorf("removed %d, want %d", n, len(keys))
	}
	entries, _ := os.ReadDir(cacheDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			t.Errorf("expected no .json files left, found %s", e.Name())
		}
	}
}
