# yt — agent-friendly YouTube CLI

A command-line tool for inspecting and reorganizing your YouTube playlists via the [YouTube Data API v3](https://developers.google.com/youtube/v3/docs). Inspired by [`gogcli`](https://github.com/steipete/gogcli) — designed to be scriptable and pleasant to drive from an LLM agent.

> **Note on Watch Later.** Google removed third-party API access to the *Watch Later* and *History* playlists in 2016. This CLI works against your user-created playlists and (later) your *Liked Videos* playlist (`LL`). A non-API workaround for Watch Later is on the roadmap.

## Status

Early scaffold. Implemented so far:

- `yt auth credentials <path>` / `auth login` / `auth status` / `auth logout` (OAuth2 desktop flow, loopback redirect, secrets stored in OS keyring with encrypted-file fallback)
- `yt playlists list` / `yt playlists show <id>` (with `--json` and `--plain`)
- `yt items list <playlist-id>` (paginates the full playlist)

## Install

Requires Go 1.26+. With [`mise`](https://mise.jdx.dev) the version is pinned in `.mise.toml`:

```sh
mise install
go build -o bin/yt ./cmd/yt
```

## Quick Start

### 1. Get OAuth2 credentials

Create OAuth2 credentials in the Google Cloud Console:

1. Open https://console.cloud.google.com/apis/credentials and create (or pick) a project.
2. Enable the **YouTube Data API v3**: https://console.cloud.google.com/apis/api/youtube.googleapis.com
3. Configure the OAuth consent screen: https://console.cloud.google.com/auth/branding (External, add yourself as a Test user).
4. Create an OAuth client at https://console.cloud.google.com/auth/clients:
   - **Create Client → Application type: Desktop app**
   - Download the JSON file (named like `client_secret_....apps.googleusercontent.com.json`).

### 2. Store credentials

```sh
yt auth credentials ~/Downloads/client_secret_....json
```

The JSON contents are stored in your OS keyring (Keychain on macOS, Secret Service on Linux, Credential Manager on Windows), with a `0600`-mode file fallback in the config dir if the keyring is unavailable.

### 3. Authorize your account

```sh
yt auth login
```

Your browser opens, you grant access, and the CLI captures the OAuth redirect on a loopback port. The refresh token is stored alongside the credentials.

### 4. Try it

```sh
yt auth status
yt playlists list
yt playlists list --json
yt playlists list --plain | awk -F'\t' '{print $2}'
```

To wipe everything and start over: `yt auth logout --all`.

## Project layout

```
cmd/yt/                 # entrypoint (main.go)
internal/cmd/           # cobra commands
internal/auth/          # OAuth2 flow + token storage
internal/ytapi/         # authenticated YouTube service factory
internal/output/        # --json / --plain / table formatters
```

## Output flags (every command)

- `--json` — pretty-printed JSON of the raw API objects, for piping into `jq` or feeding an agent.
- `--plain` — tab-separated values, no header, stable for shell scripts.
- *(default)* — aligned table for humans.

## Roadmap

- `yt playlists create` / `update` / `delete`
- `yt items add` / `remove` / `move`
- `yt items sort <playlist-id> --by=title|date|duration|channel` (with `--dry-run`)
- `yt liked list` (the `LL` playlist)
- `yt search "query"`
- Quota-aware batching and on-disk response cache
- Watch Later workaround (browser-side) — separate command, opt-in
