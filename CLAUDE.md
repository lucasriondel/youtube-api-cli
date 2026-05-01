# CLAUDE.md

Project notes for Claude (and humans) working on this repo. Read this first.

## What this is

`yt` is an agent-friendly CLI for inspecting and reorganizing the user's YouTube account via the YouTube Data API v3. It's modeled on [`gogcli`](https://github.com/steipete/gogcli): stable structured output, scriptable commands, secrets in the OS keyring.

The intended driver is an LLM agent — every command should produce parseable output, behave idempotently when reasonable, and fail loudly with actionable messages.

## Hard constraints (read before designing features)

- **Watch Later and Watch History are NOT accessible via the API.** Google removed third-party access in 2016. `playlists.list` / `playlistItems.list` against `WL` returns empty or `playlistOperationUnsupported`. Do not waste effort trying to make this work through the official API. A separate browser-automation path is on the roadmap, opt-in only.
- **Auth is OAuth2, not API keys.** API keys only work for public read-only data. Anything touching the user's own playlists, ratings, subscriptions, etc. requires OAuth2. We use the **Desktop app** flow with a loopback redirect.
- **Quota matters.** Default project quota is 10,000 units/day. `playlistItems.insert` is 50 units, `search.list` is 100 units, most reads are 1. Sorting a 500-item playlist by re-inserting items costs 25,000 units — over quota. Any write-heavy command should:
  - Always support `--dry-run`
  - Batch where the API allows
  - Cache reads on disk when iterating
  - Document its expected quota cost in `--help`

## Stack

- **Go 1.26.2**, pinned in `.mise.toml`. Use `mise exec -- go ...` if `go` isn't on PATH.
- **Cobra** for CLI structure.
- **`google.golang.org/api/youtube/v3`** for the YouTube client.
- **`golang.org/x/oauth2`** + **`golang.org/x/oauth2/google`** for the OAuth2 flow.
- **`github.com/zalando/go-keyring`** for secret storage (Keychain / Secret Service / Credential Manager), with a `0600`-mode file fallback in the config dir.

## Layout

```
cmd/yt/main.go              # entrypoint, just calls cmd.NewRoot().Execute()
internal/cmd/               # cobra commands. one command group per file.
  root.go                   # root command + global flags (--json, --plain)
  auth.go                   # auth credentials | login | status | logout
  playlists.go              # playlists list (more to come)
internal/auth/              # OAuth2 + secret storage. NO cobra here.
  paths.go                  # ConfigDir() — ~/.config/yt
  storage.go                # SaveToken/LoadToken + SaveClientSecret/LoadClientSecret
  oauth.go                  # LoadConfig, Login (loopback flow), DefaultScopes
  random.go                 # randomState() for OAuth state param
internal/ytapi/             # authenticated youtube.Service factory
  client.go                 # New(ctx) — reads stored creds + token, returns *youtube.Service
  token_source.go           # persistingTokenSource — re-saves token on refresh
internal/output/            # rendering. NO domain logic here.
  output.go                 # Render(w, format, headers, rows, data) for table/json/plain
```

### Layout rules

- `internal/cmd/` is the only package that imports cobra. Don't leak cobra types into `auth/`, `ytapi/`, or `output/`.
- `internal/auth/` and `internal/ytapi/` must remain importable from non-CLI contexts (e.g. tests, future MCP server).
- One cobra command group per file in `internal/cmd/` (see `auth.go`, `playlists.go`). When a group grows, split subcommands into their own files (`playlists_list.go`, `playlists_show.go`, etc.) — mirrors gogcli's `internal/cmd/` style.
- Per the user's global preference: prefer multiple small files over one large file.

## Auth model

Mirrors gogcli's split between **client credentials** (per OAuth client) and **token** (per authorized account):

1. User downloads `client_secret_....json` from Google Cloud Console (Desktop app OAuth client).
2. `yt auth credentials <path>` validates and stores the JSON in the keyring under key `oauth2-client`.
3. `yt auth login` reads the stored credentials, runs the loopback flow, stores the resulting token under key `oauth2-token`.
4. `yt auth logout` clears the token only. `--all` also clears credentials.

Both keyring entries live under service `yt-cli` (constants in `internal/auth/paths.go`). The fallback file location is `os.UserConfigDir()/yt/{client_secret,token}.json` mode `0600`.

The `persistingTokenSource` in `internal/ytapi/token_source.go` ensures refreshed tokens are written back automatically — never fetch tokens directly from `oauth2.Config.TokenSource`, always go through `ytapi.New(ctx)`.

### Default scopes

`youtube.YoutubeForceSslScope` (covers read + write + ratings). Defined in `auth.DefaultScopes()`. If you add a command that needs a different scope, do **not** silently widen the default — surface it explicitly and force a re-login.

## Output conventions

Every command must respect the global `--json` and `--plain` flags via `output.FormatFromFlags(Globals.JSON, Globals.Plain)`:

- `--json`: pretty-printed JSON of the **raw API objects** (not a CLI-specific shape). Agents and `jq` consume this.
- `--plain`: tab-separated values, no header, stable for shell pipelines. Column order must match the table view.
- *(default)*: aligned table for humans.

When adding a new command:
1. Pass the raw API response slice as `data` to `output.Render` for the JSON case.
2. Build `rows [][]string` with the same columns as the table headers.
3. Don't add per-command output flags; extend the global ones if you need a new format.

## Quota budgeting (when you add write commands)

Cost reference: https://developers.google.com/youtube/v3/determine_quota_cost

Expensive operations to watch:
- `search.list` = 100
- `playlistItems.insert` / `update` / `delete` = 50 each
- `videos.insert` = 1600

Patterns to apply:
- For sort/reorder: compute the target order locally, then only `update` items whose position actually changes.
- For bulk reads: cache `playlistItems.list` responses on disk keyed by `(playlistID, etag)` so repeated invocations don't re-fetch.
- Always offer `--dry-run` that prints the planned mutations + total quota cost.

## Build, run, conventions

```sh
mise install                                # one-time, installs Go 1.26.2
mise exec -- go build -o bin/yt ./cmd/yt    # build
mise exec -- go test ./...                  # tests (none yet)
mise exec -- go mod tidy                    # after adding/removing deps
./bin/yt --help
```

After any change that adds/removes a Go import, run `go mod tidy`. The current `go.mod` has all deps marked `// indirect` because the initial scaffold used `go get` without immediate usage in non-test files at the time of resolution; this is harmless but tidy will normalize it.

### Style

- Errors wrap context: `fmt.Errorf("playlists.list: %w", err)`. Don't `panic` from CLI code.
- User-facing messages on `stderr` (use `cmd.OutOrStderr()` or `os.Stderr`); structured/parseable output on `stdout` (use `cmd.OutOrStdout()`).
- Don't print to `os.Stdout` directly from a cobra `RunE` — go through `cmd.OutOrStdout()` so tests can capture.
- Sentinel errors live next to the storage they relate to (see `auth.ErrNoToken`, `auth.ErrNoClientSecret`). Check with `errors.Is`.

## Roadmap (in priority order)

1. `playlists show <id>` / `create` / `update` / `delete`
2. `items list <playlist-id>` / `add` / `remove` / `move`
3. `items sort <playlist-id> --by=title|date|duration|channel` with mandatory `--dry-run` first
4. `liked list` (the `LL` playlist — still accessible via API)
5. `search "query"`
6. On-disk response cache (etag-aware) for read-heavy iteration
7. Watch Later workaround via browser automation (separate, opt-in command — design TBD)

## What NOT to do

- Don't add a "natural language" mode that embeds an LLM inside the CLI. Scope is structured commands; the agent lives outside.
- Don't introduce a config file format. State lives in the keyring (or fallback file). If a setting needs persistence, add a `yt config` subcommand later, but defer until there's a real need.
- Don't add per-command output flags (`--csv`, `--yaml`, etc.). Extend the global format set if needed.
- Don't bypass `ytapi.New` to talk to the YouTube API directly — the persisting token source is the only place refreshed tokens get saved back.
- Don't widen the OAuth scope silently. New scopes = new explicit re-login.
